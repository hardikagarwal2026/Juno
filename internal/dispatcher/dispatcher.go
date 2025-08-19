package dispatcher

import (
	"Juno/internal/job"
	"context"
	"fmt"
	"golang.org/x/time/rate"
)

// Feed jobs into the worker queue, but pace them with a rate limiter so I don't overwhelm workers.
func StartDispatcher(ctx context.Context, jobs []*job.Job, queue chan *job.Job, limiter *rate.Limiter) {
	for _, j := range jobs {
		select {
		case <-ctx.Done():
			fmt.Println("[Dispatcher] Shutdown signal received. Stopping dispatch...")
			close(queue)
			return
		default:
			// Wait for a rate-limiter token; this blocks so I keep the send pace under control.
			if err := limiter.Wait(ctx); err != nil {
				fmt.Println("[Dispatcher] Context canceled while waiting for rate limiter")
				close(queue)
				return
			}
			// Just logging that I got a token from the limiter.
			fmt.Printf("[Limiter] Dispatch allowed — token acquired for Job #%d\n", j.ID)
			// Send the job into the worker queue.
			fmt.Printf("[Dispatcher] Dispatching Job #%d\n", j.ID)
			queue <- j
		}
	}
	// I don't close the queue here because retries also send into it.
	// The main function owns closing the queue when all producers are done.
}
