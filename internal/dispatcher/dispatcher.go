package dispatcher

import (
	"Distributed_Job_Dispatcher/internal/job"
	"context"
	"fmt"
	"golang.org/x/time/rate"
)

// StartDispatcher streams jobs into the queue at a controlled bursty rate
func StartDispatcher(ctx context.Context, jobs []*job.Job, queue chan *job.Job, limiter *rate.Limiter) {
	for _, j := range jobs {
		select {
		case <-ctx.Done():
			fmt.Println("[Dispatcher] Shutdown signal received. Stopping dispatch...")
			close(queue)
			return

		// limiter.Wait(ctx) blocks until a token is available.
		// If context is canceled (e.g., Ctrl+C), it exits cleanly.
		// Otherwise, it lets job go through only when allowed by rate limiter.
		default:
			if err := limiter.Wait(ctx); err != nil {
				fmt.Println("[Dispatcher] Context canceled while waiting for rate limiter")
				close(queue)
				return
			}

			// Log when a token is successfully acquired from the limiter
			fmt.Printf("[Limiter] Dispatch allowed — token acquired for Job #%d\n", j.ID)

			// Dispatch job to the worker queue
			fmt.Printf("[Dispatcher] Dispatching Job #%d\n", j.ID)
			queue <- j
		}
	}

	//Once all jobs are dispatched, close the job queue so workers stop
	close(queue)
}
