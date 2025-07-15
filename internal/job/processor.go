package job

import (
	"fmt"
	"math/rand"
	"time"
)


//it simulated a job being processed
func Process(j *Job) error {
	fmt.Printf("Job:%d starting with Payload: %s\n",j.ID, j.Payload )
	//time.Duration give 100ms to 999ms
	worktime := time.Duration(rand.Intn(900)+100) * time.Millisecond
	//it randomly sleeps for some worktime(100ms-1000ms)
	time.Sleep(worktime)

	//it randomly fails for 20% of the jobs, if satisfies the condition
	if rand.Float32() < 0.2 {
		return fmt.Errorf("For simulating failure")
	}

	fmt.Printf("processor Job %d completed successfully\n", j.ID)
	return nil

}

