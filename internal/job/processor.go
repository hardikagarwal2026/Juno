package job

import (
	"fmt"
	"math/rand"
	"time"
)

// Pretend to do the actual work for a job.
func Process(j *Job) error {
	fmt.Printf("Job:%d starting with Payload: %s\n", j.ID, j.Payload)

	// Take 100–999ms to "do work" so timing looks real.
	worktime := time.Duration(rand.Intn(900)+100) * time.Millisecond
	time.Sleep(worktime)

	// Fail 20% of the time so I can exercise retries and timeouts.
	if rand.Float32() < 0.2 {
		return fmt.Errorf("simulated failure")
	}

	fmt.Printf("processor Job %d completed successfully\n", j.ID)
	return nil
}
