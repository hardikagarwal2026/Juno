package dispatcher

import (
	"Distributed_Job_Dispatcher/internal/job"
	"fmt"
	"time"
	"context"
)

// StartDispatcher streams jobs into the queue at a controlled rate
func StartDispatcher(ctx context.Context, jobs []*job.Job, queue chan *job.Job, rate <-chan time.Time) {
	for _, j := range jobs {
		select {
		case <-ctx.Done():
			fmt.Println("[Dispatcher] Shutdown signal received. Stopping dispatch...")
			close(queue)
			return
		case <-rate:
			fmt.Printf("[Dispatcher] Dispatching Job #%d\n", j.ID)
			queue <- j
		}
	}
	close(queue) // All jobs sent
}
