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

// execTask sleeps for randomly determined duration.
// It sometimes fail and returns an error.
func execTask(timeout time.Duration, shutdown chan struct{}) error {
	// Do pseudo-task. Here, it is just a "sleep".
	n := 500 + rand.Intn(3500)
	done := time.After(time.Duration(n) * time.Millisecond)
	select {
	case <-shutdown:
		return errors.New("shutdown") // Shutdown signaled. Canceling task...
	case <-time.After(timeout):
		return errors.New("timed out")
	case <-done:
		// Do nothing here. Proceed to the following code
	}

	// Return result of the task. Here, failure means the random number is a multiples of 9.
	if (n % 9) == 0 {
		return errors.New("bad luck")
	}
	return nil
}

func receive(shutdown chan struct{}) {
	errChan := make(chan error)
	for i := 0; ; i++ {
		log.Printf("[R] Executing a task %d...", i)
		go func() {
			errChan <- execTask(taskTimeout, shutdown)
		}()
		select {
		case err := <-errChan:
			if err != nil {
				log.Printf("[R] Task failed: %v", err)
			} else {
				log.Printf("[R] Done.")
			}
		case <-shutdown:
			log.Printf("[R] Stopping; waiting for the last task...")
			<-errChan
			return
		}
	}
}

func forward(shutdown chan struct{}) {
	errChan := make(chan error)
	for i := 0; ; i++ {
		log.Printf("[F] Executing a task %d...", i)
		go func() {
			errChan <- execTask(taskTimeout, shutdown)
		}()
		select {
		case err := <-errChan:
			if err != nil {
				log.Printf("[F] Task failed: %v", err)
			} else {
				log.Printf("[F] Done.")
			}
		case <-shutdown:
			log.Printf("[F] Stopping; waiting for the last task...")
			<-errChan
			return
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
		receive(shutdown)
	}()
	go func() {
		defer wg.Done()
		forward(shutdown)
	}()
	wg.Wait()

	log.Printf("Exiting")
}
