package main

import (
	"Distributed_Job_Dispatcher/internal/dispatcher"
	"Distributed_Job_Dispatcher/internal/job"
	"Distributed_Job_Dispatcher/internal/shutdown"
	"Distributed_Job_Dispatcher/internal/worker"
	"context"
	"fmt"
	"sync"
	"time"
	"golang.org/x/time/rate"
	"flag"

)

//creating the flags for job buffer size and result channel buffer size
var (
	jobChanBufferSize    = flag.Int("jobbuffer", 5, "Buffer size for job channel")
	resultChanBufferSize = flag.Int("resultbuffer", 5, "Buffer size for result channel")
)

func main() {
	flag.Parse()
	// Create cancellable context for system-wide shutdown control
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	// Listen for SIGINT (Ctrl+C) and call cancel() on interrupt
	shutdown.Listen(cancel)

	// Create channels
	jobQueue := make(chan *job.Job, *jobChanBufferSize)   // Job queue shared between dispatcher and workers
	results := make(chan *job.Job, *resultChanBufferSize)    // Results channel (successful jobs)
	// rate := time.Tick(300 * time.Millisecond) // Rate limiter: 1 job every 300ms

    //for requeuing
	var requeueWg sync.WaitGroup
	requeueWg.Add(1)

	//creating a retry queue for failed or tmed out jobs
	retryqueue := make(chan *job.Job, *jobChanBufferSize)

	// this go routines continuously listens on retryqueue, whenever jobs comes, it sends to to jobqueue
	go func(){
		defer requeueWg.Done()
		for j := range retryqueue {
			if j.Retries >= 3 {
					fmt.Printf("[Requeue] Dropping Job #%d from retryQueue — exceeded retries\n", j.ID)
					continue
			}
			jobQueue <- j 
		}
	}()


	//creating a burstlimiter, it creates a burst in starting, meaning allowing initial number of requests, and then after than channel is free
	limiter := rate.NewLimiter(rate.Every(300*time.Millisecond), 9)




	// Stats tracker for processed/failed jobs
	stats := &worker.Stats{}

	// Prepare list of jobs to dispatch
	var jobList []*job.Job
	for i := 1; i <= 20; i++ {
		jobList = append(jobList, &job.Job{
			ID:      i,
			Payload: fmt.Sprintf("Payload-%d", i),
			Status:  job.Pending,
		})
	}

	// Start dispatcher in a goroutine
	// Pass ctx so it stops if shutdown is triggered
	go dispatcher.StartDispatcher(ctx, jobList, jobQueue, limiter)

	// Start a pool of workers
	const numWorkers = 10
	var wg sync.WaitGroup

	for i := 1; i <= numWorkers; i++ {
		wg.Add(1)
		go worker.StartWorker(i, jobQueue,retryqueue, results, &wg, stats)
	}

	// Wait until all workers have finished
	wg.Wait()
    close(retryqueue)
	requeueWg.Wait()
	// Close results channel once workers are done
	close(results)

	// Print final job stats
	fmt.Println("\n========== FINAL REPORT ==========")
	fmt.Printf("Processed Jobs : %d\n", stats.Processed)
	fmt.Printf("Failed Jobs    : %d\n", stats.Failed)
	fmt.Println("==================================")
}
