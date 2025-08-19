## Juno: Concurrent Job Dispatcher

I built Juno to really understand Go’s concurrency model by writing something real: a rate-limited dispatcher that feeds a retry-capable worker pool via channels, with graceful shutdowns, per-job timeouts, and simple stats. This is a small codebase on purpose so I can explain the why behind every piece.

### Overview
- **Dispatcher**: throttles jobs with a rate limiter and sends them to a worker pool
- **Workers**: process jobs with per-job timeouts, retries with exponential backoff, and result reporting
- **Graceful shutdown**: context-based cancellation and orderly channel closing

### Quick start
- Requires Go (matches `go.mod`): 1.24.x
- Run: `go run .`
- Optional knobs (in `main.go`):
  - **Worker count**: `const workerCount = 4`
  - **Rate limit**: `rate.NewLimiter(rate.Limit(5), 5)`
  - **Demo stop timer**: `time.AfterFunc(8*time.Second, ...)`
  - **Retry limit**: `attempt < 3` (in `internal/worker/pool.go`)
  - **Per-job timeout**: `1 * time.Second` (in `internal/worker/pool.go`)

### Big-picture flow
```
Jobs [] ---> Dispatcher (rate limited) ---> queue (chan *job.Job) ---> [Worker 1..N] ---> results (chan *job.Job)
                                               ^                             |
                                               |                             |
                                               +----- retryQueue <-----------+
```
1) I create a root context and listen for Ctrl+C to cancel cleanly (`shutdown.Listen`).
2) I generate demo jobs and set up channels: `queue`, `retryQueue`, `results`.
3) The dispatcher pushes jobs into `queue` with a rate limiter.
4) N workers pull from `queue`, process with a timeout, and either:
   - send to `results` on success, or
   - push to `retryQueue` with exponential backoff (until a retry limit).
5) A forwarder pumps `retryQueue` back into `queue`.
6) After a short demo window, I stop retries, wait for the forwarder, then close `queue` so workers drain and exit. I close `results` last and print stats.

---

## Concepts I wanted to learn (and how I used them here)

### Concurrency vs Parallelism
- **Why I care**: Concurrency is about structuring a program as independent tasks that can make progress without blocking each other. Parallelism is doing those tasks at the same time on multiple cores. I wanted to think and code in terms of concurrency, not rely on parallelism.
- **How in Juno**: The dispatcher, workers, retry forwarder, and result collector each run in separate goroutines. They’re concurrent by design. If your CPU has multiple cores, the Go runtime may run them in parallel, but correctness doesn’t depend on it.

### Goroutines
- **Why**: They’re ultra-light threads managed by the Go runtime. Spawning many is cheap, which is perfect for orchestrators (dispatcher/forwarder) and workers.
- **How**: I run the dispatcher, the retry forwarder, the results collector, and each worker as a goroutine so they can work independently.

### Channels
- **Why**: Channels make communication explicit and safe. They help avoid shared-memory pain and make fan-out/fan-in patterns readable.
- **How**:
  - `queue` is the main work channel that workers range over.
  - `retryQueue` is where failed/timeouts go for another attempt.
  - `results` is where successful jobs are reported and logged.
  - A tiny goroutine forwards from `retryQueue` back into `queue`, so all work enters through a single lane that workers consume.

### Synchronization and WaitGroups
- **Why**: I want deterministic shutdown without races or leaked goroutines.
- **How**:
  - A `WaitGroup` (`wg`) tracks worker lifetimes. When the `queue` closes and workers finish, `wg` drops to zero and I know it’s safe to proceed.
  - A second `WaitGroup` (`retryWG`) ensures the retry forwarder has flushed everything from `retryQueue` back to `queue` before I close `queue`.
  - For stats (`Processed`/`Failed`) I used a small `sync.Mutex`. It’s simple and clear for counters.

### Contexts: cancellation and timeouts
- **Why**: Context gives me a unified way to cancel work and set deadlines. It’s the standard Go way to propagate “stop now” signals.
- **How**:
  - Root `context.WithCancel` is triggered by Ctrl+C (SIGINT/SIGTERM) in `internal/shutdown`.
  - Each job uses `context.WithTimeout(..., 1*time.Second)` so no single job can hang a worker forever. I race the job’s result against the timeout.

### Rate limiting (backpressure at the source)
- **Why**: To avoid flooding the worker pool and to simulate real systems that have to throttle producers.
- **How**: The dispatcher calls `limiter.Wait(ctx)` from `golang.org/x/time/rate)` before sending each job into `queue`. If the context is canceled, the dispatcher exits and doesn’t leave half-sent work behind.

### Retries with exponential backoff and jitter
- **Why**: Transient failures happen. Backoff spreads out retries so they don’t stampede.
- **How**: On failure/timeout, I increment the job’s retry count and, if still under the limit, compute backoff as `base * 2^attempt + jitter`, sleep, and re-queue the job via `retryQueue`.

### Worker pool design
- **Why**: Fan-out work across multiple goroutines, each with bounded execution time.
- **How**:
  - Workers `range` over `queue` until it’s closed.
  - For each job, set status to Running, process with a per-job timeout, and either:
    - on success: mark Success and send to `results`;
    - on error/timeout: mark Failed/TimedOut, bump retry count, back off, and send to `retryQueue` if below limit.

### Graceful shutdown plan
- **Why**: Unplanned exits can corrupt state, leak goroutines, or deadlock.
- **How**:
  - In the demo, I stop producing after 8s for a clean end (a stand-in for a real "producers done" signal).
  - I close `retryQueue`, wait for the retry forwarder to flush (`retryWG.Wait()`), then close `queue` so workers naturally exit when the channel drains.
  - After workers finish (`wg.Wait()`), I close `results` so the collector exits and then print final stats.

---

## Good parts
- I use channels for both fan-out (`queue`) and fan-in (`results`), with a side-lane for retries. This shows comfort with channel topologies.
- Backpressure is handled at the source with a rate limiter; I don’t rely on growing backlogs to absorb load.
- Context is used for both global shutdown and per-job timeouts—a unified cancellation model.
- Retries use exponential backoff with jitter to avoid synchronization storms.
- Channel ownership and closing are centralized in `main.go`, which prevents sends on closed channels.
- Minimal but correct synchronization: `WaitGroup`s for lifecycle; a `Mutex` for counters.
- Clean package boundaries that mirror runtime responsibilities: `dispatcher`, `worker`, `job`, `shutdown`.

---

## Try it out
- See more retries: increase the failure rate in `internal/job/processor.go` or decrease the per-job timeout.
- Stress the pool: increase dispatch rate or the number of jobs.

## File map
- `main.go`: Orchestration; wires channels, starts goroutines, manages shutdown.
- `internal/dispatcher/dispatcher.go`: Rate-limited feed into `queue`.
- `internal/worker/pool.go`: Worker pool with timeout, retry/backoff, and stats.
- `internal/job/job.go`: Job model and status enum.
- `internal/job/processor.go`: Simulated work and random failures.
- `internal/shutdown/shutdown.go`: Ctrl+C listener that cancels the root context.



