package main

import (
	"context"
	"fmt"
	"math/rand"
	"sync"
	"time"

	"Juno/internal/dispatcher"
	"Juno/internal/job"
	"Juno/internal/shutdown"
	"Juno/internal/worker"

	"golang.org/x/time/rate"
)

func main() {
	rand.Seed(time.Now().UnixNano())

	// Create a root context and hook Ctrl+C so I can shut everything down gracefully
	ctx, cancel := context.WithCancel(context.Background())
	shutdown.Listen(cancel)

	// Seed a bunch of demo jobs I'll dispatch
	jobs := make([]*job.Job, 0, 20)
	for i := 1; i <= 20; i++ {
		jobs = append(jobs, &job.Job{
			ID:      i,
			Payload: fmt.Sprintf("payload-%d", i),
			Status:  job.Pending,
		})
	}

	//channels main queue, retry queue, and a results stream
	queue := make(chan *job.Job, 16)
	retryQueue := make(chan *job.Job, 16)
	results := make(chan *job.Job, 32)

	// Forward anything in the retry queue back into the main queue
	var retryWG sync.WaitGroup
	retryWG.Add(1)
	go func() {
		defer retryWG.Done()
		for j := range retryQueue {
			select {
			case <-ctx.Done():
				return
			case queue <- j:
			}
		}
	}()

	// Pace dispatching at 5 jobs/sec with a burst of 5
	limiter := rate.NewLimiter(rate.Limit(5), 5)

	// Kick off the dispatcher that feeds the queue
	go dispatcher.StartDispatcher(ctx, jobs, queue, limiter)

	// Spin up a small worker pool.
	const workerCount = 4
	var wg sync.WaitGroup
	stats := &worker.Stats{}
	wg.Add(workerCount)
	for i := 1; i <= workerCount; i++ {
		go worker.StartWorker(i, queue, retryQueue, results, &wg, stats)
	}

	// Collect and print job results as they come in.
	doneCollect := make(chan struct{})
	go func() {
		for r := range results {
			r.LogStatus()
		}
		close(doneCollect)
	}()

	// I want workers to exit only after the queues are drained
	// I'll stop producing after a bit and then close the queues in order

	// after 8s, stop accepting new work and start shutdown.
	time.AfterFunc(8*time.Second, func() {
		// First, stop generating more retries.
		close(retryQueue)
		// Wait until pending retries are forwarded back to the main queue
		retryWG.Wait()
		// No more producers now; close the main queue so workers finish once it's drained
		close(queue)
	})

	// Wait for all workers to finish
	wg.Wait()
	// Close results so the collector can finish
	close(results)
	<-doneCollect

	fmt.Printf("\n== Final Stats ==\nProcessed: %d  Failed: %d\n", stats.Processed, stats.Failed)
}
