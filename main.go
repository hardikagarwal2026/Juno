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
)

func main() {
	// Create cancellable context for system-wide shutdown control
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	// Listen for SIGINT (Ctrl+C) and call cancel() on interrupt
	shutdown.Listen(cancel)

	// Create channels
	jobQueue := make(chan *job.Job, 10)   // Job queue shared between dispatcher and workers
	results := make(chan *job.Job, 10)    // Results channel (successful jobs)
	rate := time.Tick(300 * time.Millisecond) // Rate limiter: 1 job every 300ms

	// Stats tracker for processed/failed jobs
	stats := &worker.Stats{}

	// Prepare list of jobs to dispatch
	var jobList []*job.Job
	for i := 1; i <= 15; i++ {
		jobList = append(jobList, &job.Job{
			ID:      i,
			Payload: fmt.Sprintf("Payload-%d", i),
			Status:  job.Pending,
		})
	}

	// Start dispatcher in a goroutine
	// Pass ctx so it stops if shutdown is triggered
	go dispatcher.StartDispatcher(ctx, jobList, jobQueue, rate)

	// Start a pool of workers
	const numWorkers = 4
	var wg sync.WaitGroup

	for i := 1; i <= numWorkers; i++ {
		wg.Add(1)
		go worker.StartWorker(i, jobQueue, results, &wg, stats)
	}

	// Wait until all workers have finished
	wg.Wait()

	// Close results channel once workers are done
	close(results)

	// Print final job stats
	fmt.Println("\n========== FINAL REPORT ==========")
	fmt.Printf("Processed Jobs : %d\n", stats.Processed)
	fmt.Printf("Failed Jobs    : %d\n", stats.Failed)
	fmt.Println("==================================")
}
