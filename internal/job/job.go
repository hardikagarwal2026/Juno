package job

import (
	"fmt"
	"sync/atomic"
)

type Status string

//creating enums for the job status of type Status
const(
	Pending Status = "pending"
	Running Status = "running"
	Success Status = "success"
	Failed Status = "failed"
	TimedOut Status = "timed_out"
)

//job struct consist of id, type of data, no of retries, and its status
type Job struct {
	ID int
	Payload string
	Retries int32
	Status Status
}

//this function is use to mark the no. of retries the job gone through
func (j *Job) MarkRetry() {
	atomic.AddInt32(&j.Retries, 1)// use atomic for prevent from dirty read write, or increment the counter by one every timw
}

//func to display the status of a job
func(j * Job) LogStatus() {
	fmt.Printf("[Job %d] Status: %s | Retries: %d\n", j.ID, j.Status, j.Retries)
}

