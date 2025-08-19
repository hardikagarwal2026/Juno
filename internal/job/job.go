package job

import (
	"fmt"
	"sync/atomic"
)

type Status string

// Status values for a job.
const (
	Pending  Status = "pending"
	Running  Status = "running"
	Success  Status = "success"
	Failed   Status = "failed"
	TimedOut Status = "timed_out"
)

// A Job is my unit of work: id, payload, retry count, and current status.
type Job struct {
	ID      int
	Payload string
	Retries int32
	Status  Status
}

// Bump the retry count atomically (safe across goroutines).
func (j *Job) MarkRetry() {
	atomic.AddInt32(&j.Retries, 1)
}

// Print a friendly status line for this job.
func (j *Job) LogStatus() {
	fmt.Printf("[Job %d] Status: %s | Retries: %d\n", j.ID, j.Status, j.Retries)
}
