package worker

import (
	"Distributed_Job_Dispatcher/internal/job"
	"context"
	"fmt"
	"math/rand"
	"sync"
	"time"
)


func calculateBackoff(attempt int) time.Duration {
	base := 200 * time.Millisecond
	jitter := time.Duration(rand.Intn(100)) * time.Millisecond
    return base * (1 << attempt) + jitter
}

type Stats struct {
	Processed int
	Failed int
	Mutex sync.Mutex
}

func StartWorker(id int, jobs <-chan *job.Job, retryqueue chan<- *job.Job, results chan<- *job.Job, wg *sync.WaitGroup, stats *Stats) {
	//it is used as a signal that the worker has finished its work
	defer func() {
        fmt.Printf("[Worker %d] Exiting\n", id)
        wg.Done()
    }()

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

		    
			//if it recieves the result in the channel, it mifht be failed aor success
			case err:= <- resultChan:
				// if there is errror, we increse the retries, mark it as failed
				if err!=nil {
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
					}else{
						fmt.Printf("[Worker %d] Job #%d exceeded retry limit. Dropping.\n", id, j.ID)
					}

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