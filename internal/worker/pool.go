package worker

import (
	"Distributed_Job_Dispatcher/internal/job"
	"context"
	"fmt"
	"sync"
	"time"
)

type Stats struct {
	Processed int
	Failed int
	Mutex sync.Mutex
}

func StartWorker(id int, jobs <-chan *job.Job,results chan<- *job.Job, wg *sync.WaitGroup, stats *Stats) {
	defer wg.Done() //it is used as a signal that the worker has finished its work

	//iterating through all the jobs 
	for j:= range jobs {
		//to mark the status of the job
		j.Status = job.Running

		//creating a deadline for our operation, any job that receives the context can stop early if timeout expires, now our setup is 1 sec
		ctx , cancel := context.WithTimeout(context.Background(), 1*time.Second)

		resultChan := make(chan error, 1)

		//Processing the job and passing it to the to the result channel, in separate go routine
		go func(){
			resultChan <- job.Process(j)
		}()

		select{
			//if timeout occurs first(ctx.done receives a channel), then we'll update the status as timedout, increse the returies, lock to update the stats, update it as failed(no.of times) 
			case <- ctx.Done():
				j.Status = job.TimedOut
				j.MarkRetry()
				stats.Mutex.Lock()
				stats.Failed++
				stats.Mutex.Unlock()
				fmt.Printf("Worker %d Job timed out \n",j.ID)
		    
			//if it recieves the result in the channel, it mifht be failed aor success
			case err:= <- resultChan:
				// if there is errror, we increse the retries, mark it as failed
				if err!=nil {
					j.Status = job.Failed
					j.MarkRetry()
					stats.Mutex.Lock()
					stats.Failed++
					stats.Mutex.Unlock()
					fmt.Printf("[Worker %d] Job #%d failed: %v\n", id, j.ID, err)
				} else {
					j.Status = job.Success
					stats.Mutex.Lock()
					stats.Processed++
					stats.Mutex.Unlock()
					results <-j 
				}
		}
        // to release the resources associated with context.Context
		cancel()

	}
}