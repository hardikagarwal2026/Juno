package shutdown

import (
	"context"
	"os"
	"os/signal"
	"syscall"
)

// I listen for Ctrl+C (SIGINT/SIGTERM) and call cancel() so everything can shut down cleanly.
func Listen(cancel context.CancelFunc) {
	c := make(chan os.Signal, 1)
	signal.Notify(c, os.Interrupt, syscall.SIGTERM)
	go func() {
		<-c
		println("\n[Shutdown] Caught interrupt. Gracefully stopping...")
		cancel()
	}()
}
