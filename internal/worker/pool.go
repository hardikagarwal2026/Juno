package worker

import (
	"context"
	"fmt"
	"math/rand"
	"sync"
	"time"

	"Juno/internal/job"
)

func calculateBackoff(attempt int) time.Duration {
	base := 200 * time.Millisecond
	jitter := time.Duration(rand.Intn(100)) * time.Millisecond
	return base*time.Duration(1<<attempt) + jitter
}

type Stats struct {
	Processed int
	Failed    int
	Mutex     sync.Mutex
}

func StartWorker(
	id int,
	jobs <-chan *job.Job,
	retryqueue chan<- *job.Job,
	results chan<- *job.Job,
	wg *sync.WaitGroup,
	stats *Stats,
) {
	// When this worker is done, say goodbye and decrement the WaitGroup.
	defer func() {
		fmt.Printf("[Worker %d] Exiting\n", id)
		wg.Done()
	}()

	// Keep pulling jobs from the queue until it closes.
	for j := range jobs {
		// Mark the job as running so I can track its state.
		j.Status = job.Running

		// Give each job up to 1s to finish.
		ctx, cancel := context.WithTimeout(context.Background(), 1*time.Second)
		resultChan := make(chan error, 1)

		// Process in a goroutine so I can race the result vs the timeout.
		go func() {
			resultChan <- job.Process(j)
		}()

		select {
		case <-ctx.Done():
			j.Status = job.TimedOut
			j.MarkRetry()
			attempt := int(j.Retries)

			stats.Mutex.Lock()
			stats.Failed++
			stats.Mutex.Unlock()

			fmt.Printf("[Worker %d] Job #%d timed out | Attempt %d\n", id, j.ID, attempt)
			if attempt < 3 {
				backoff := calculateBackoff(attempt)
				fmt.Printf("[Worker %d] Retrying Job #%d after %v backoff\n", id, j.ID, backoff)
				time.Sleep(backoff)
				j.Status = job.Pending
				retryqueue <- j
			} else {
				fmt.Printf("[Worker %d] Job #%d exceeded retry limit. Dropping.\n", id, j.ID)
			}

		case err := <-resultChan:
			if err != nil {
				j.Status = job.Failed
				j.MarkRetry()
				attempt := int(j.Retries)

				stats.Mutex.Lock()
				stats.Failed++
				stats.Mutex.Unlock()

				fmt.Printf("[Worker %d] Job #%d failed: %v\n", id, j.ID, err)
				if attempt < 3 {
					backoff := calculateBackoff(attempt)
					fmt.Printf("[Worker %d] Retrying Job #%d after %v backoff\n", id, j.ID, backoff)
					time.Sleep(backoff)
					j.Status = job.Pending
					retryqueue <- j
				} else {
					fmt.Printf("[Worker %d] Job #%d exceeded retry limit. Dropping.\n", id, j.ID)
				}
			} else {
				j.Status = job.Success
				stats.Mutex.Lock()
				stats.Processed++
				stats.Mutex.Unlock()
				results <- j
			}
		}

		// Always cancel to free the timer and context resources.
		cancel()
	}
}
