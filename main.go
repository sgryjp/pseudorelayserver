package main

import (
	"fmt"
	"log"
	"math/rand"
	"os"
	"os/signal"
	"sync"
	"time"
)

// Conforms to net.Error
type myError struct {
	error
	isTimeout bool
}

func (e *myError) Timeout() bool {
	return e.isTimeout
}

func (e *myError) Temporary() bool {
	return false
}

func doSomeTask() error {
	const maximum = 2000 * time.Millisecond

	n := time.Duration(rand.Intn(3000))
	if n < maximum {
		time.Sleep(n * time.Millisecond)
		return nil
	}
	time.Sleep(maximum * time.Millisecond)
	return &myError{
		error:     fmt.Errorf("Operation timed out"),
		isTimeout: true,
	}
}

func receive(shutdown chan struct{}) {
	errChan := make(chan error)
	for {
		log.Printf("[R] Executing a task...")
		go func() {
			errChan <- doSomeTask()
		}()
		select {
		case err := <-errChan:
			if err != nil {
				log.Printf("[R] Task failed: %v", err)
			} else {
				log.Printf("[F] Done.")
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
	for {
		log.Printf("[F] Executing a task...")
		go func() {
			errChan <- doSomeTask()
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
