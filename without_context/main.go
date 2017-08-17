package main

import (
	"errors"
	"log"
	"math/rand"
	"os"
	"os/signal"
	"sync"
	"time"
)

const taskTimeout = 3 * time.Second

var errShutdown = errors.New("shutdown")
var errTimeout = errors.New("timed out")

// execTask sleeps for randomly determined duration.
// It sometimes fail and returns an error.
func execTask(timeout time.Duration, shutdown chan struct{}) error {
	// Do pseudo-task. Here, it is just a "sleep".
	n := 500 + rand.Intn(3500)
	done := time.After(time.Duration(n) * time.Millisecond)
	select {
	case <-shutdown:
		return errShutdown
	case <-time.After(timeout):
		return errTimeout
	case <-done:
		// Do nothing here. Proceed to the following code
	}

	// Return result of the task. Here, failure means the random number is a
	// multiples of 9.
	if (n % 9) == 0 {
		return errors.New("bad luck")
	}
	return nil
}

func receiver(shutdown chan struct{}) {
	errChan := make(chan error)
	for i := 0; ; i++ {
		log.Printf("[R] Executing a task %d...", i)
		go func() {
			errChan <- execTask(taskTimeout, shutdown)
		}()
		select {
		case err := <-errChan:
			switch err {
			default:
				log.Printf("[R] Task failed: %v", err)
			case errTimeout:
				log.Printf("[R] Task timed out")
			case errShutdown:
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

func forwarder(shutdown chan struct{}) {
	errChan := make(chan error)
	for i := 0; ; i++ {
		log.Printf("[F] Executing a task %d...", i)
		go func() {
			errChan <- execTask(taskTimeout, shutdown)
		}()
		select {
		case err := <-errChan:
			switch err {
			default:
				log.Printf("[F] Task failed: %v", err)
			case errTimeout:
				log.Printf("[F] Task timed out")
			case errShutdown:
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
	shutdown := make(chan struct{})

	log.SetFlags(log.LstdFlags | log.Lmicroseconds)

	// Broadcast shutdown event if signal received.
	signal.Notify(sigChan, os.Interrupt, os.Kill)
	go func() {
		sig := <-sigChan
		log.Printf("RECEIVED A SIGNAL: %s(%d)", sig, sig)
		close(shutdown)
	}()

	// Start two child goroutines to execute distinct tasks.
	wg.Add(2)
	go func() {
		defer wg.Done()
		receiver(shutdown)
	}()
	go func() {
		defer wg.Done()
		forwarder(shutdown)
	}()
	wg.Wait() // Wait for end of both children

	log.Printf("Exiting")
}
