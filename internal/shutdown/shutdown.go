package shutdown

import (
	"os"
	"os/signal"
	"syscall"
	"context"
)

//we can Trigger a cancellation via context.CancelFunc()
func Listen(cancel context.CancelFunc) {
	c := make(chan os.Signal, 1)

	signal.Notify(c, os.Interrupt, syscall.SIGTERM)

	go func(){
		<- c
		println("\n[Shutdown] Caught interrupt. Gracefully stopping...")
		cancel()
	}()
}