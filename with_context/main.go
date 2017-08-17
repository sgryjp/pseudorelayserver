package main

import (
	"context"
	"errors"
	"log"
	"math/rand"
	"os"
	"os/signal"
	"sync"
	"time"
)

const taskTimeout = 3 * time.Second

// execTask sleeps for randomly determined duration.
// It sometimes fail and returns an error.
func execTask(ctx context.Context) error {
	// Do pseudo-task. Here, it is just a "sleep".
	n := 500 + rand.Intn(3500)
	timer := time.NewTimer(time.Duration(n) * time.Millisecond)
	select {
	case <-ctx.Done():
		// Cancel the pseudo-task here.
		timer.Stop()
		return ctx.Err()
	case <-timer.C:
		// Do nothing here. Proceed to the following code
	}

	// Return result of the task. Here, failure means the random number is a
	// multiples of 9.
	if (n % 9) == 0 {
		return errors.New("bad luck")
	}
	return nil
}

func receiver(ctx context.Context) {
	errChan := make(chan error)
	for i := 0; ; i++ {
		log.Printf("[R] Executing a task %d...", i)
		go func() {
			newCtx, cancel := context.WithTimeout(ctx, taskTimeout)
			defer cancel()
			errChan <- execTask(newCtx)
		}()
		select {
		case err := <-errChan:
			switch err {
			default:
				log.Printf("[R] Task failed: %v", err)
			case context.DeadlineExceeded:
				log.Printf("[R] Task timed out")
			case context.Canceled:
				log.Printf("[R] Task canceled")
				return
			case nil:
				log.Printf("[R] Task succeeded.")
			}
			// We should not receive from ctx.Done() here. If ctx was
			// canceled, it's child context is also canceled so the execTask()
			// should finish in no time.
		}
	}
}

func forwarder(ctx context.Context) {
	errChan := make(chan error)
	for i := 0; ; i++ {
		log.Printf("[F] Executing a task %d...", i)
		go func() {
			newCtx, cancel := context.WithTimeout(ctx, taskTimeout)
			defer cancel()
			errChan <- execTask(newCtx)
		}()
		select {
		case err := <-errChan:
			switch err {
			default:
				log.Printf("[F] Task failed: %v", err)
			case context.DeadlineExceeded:
				log.Printf("[F] Task timed out")
			case context.Canceled:
				log.Printf("[F] Task canceled")
				return
			case nil:
				log.Printf("[F] Task succeeded.")
			}
			// We should not receive from ctx.Done() here. If ctx was
			// canceled, it's child context is also canceled so the execTask()
			// should finish in no time.
		}
	}
}

func main() {
	var wg sync.WaitGroup
	sigChan := make(chan os.Signal)
	ctx, cancel := context.WithCancel(context.Background())

	log.SetFlags(log.LstdFlags | log.Lmicroseconds)

	// Broadcast shutdown event if signal received.
	signal.Notify(sigChan, os.Interrupt, os.Kill)
	go func() {
		sig := <-sigChan
		log.Printf("RECEIVED A SIGNAL: %s(%d)", sig, sig)
		cancel()
	}()

	// Start two child goroutines to execute distinct tasks.
	wg.Add(2)
	go func() {
		defer wg.Done()
		receiver(ctx)
	}()
	go func() {
		defer wg.Done()
		forwarder(ctx)
	}()
	wg.Wait() // Wait for end of both children

	log.Printf("Exiting")
}
