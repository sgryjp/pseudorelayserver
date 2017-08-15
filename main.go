package main

import (
	"fmt"
	"log"
	"math/rand"
	"net"
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

func doSomeTask(errChan chan net.Error) {
	const maximum = 2000 * time.Millisecond

	n := time.Duration(rand.Intn(3000))
	if n < maximum {
		time.Sleep(n * time.Millisecond)
		errChan <- nil
	} else {
		time.Sleep(maximum * time.Millisecond)
		errChan <- &myError{
			error:     fmt.Errorf("Operation timed out"),
			isTimeout: true,
		}
	}
}

func receive(shutdown chan struct{}) {

SERVICE_LOOP:
	for {
		errChan := make(chan net.Error)

		go doSomeTask(errChan)
		select {
		case err := <-errChan:
			switch {
			case err == nil:
				log.Printf("[R] Received")
			case err.Timeout():
				log.Printf("[R] Timed out")
			default:
				log.Printf("[R] Failed to receive: %v", err)
			}
		case <-shutdown:
			log.Printf("[R] Stopping; waiting for the last task...")
			<-errChan
			break SERVICE_LOOP
		}
	}
}

func forward(shutdown chan struct{}) {

SERVICE_LOOP:
	for {
		errChan := make(chan net.Error)

		go doSomeTask(errChan)
		select {
		case err := <-errChan:
			switch {
			case err == nil:
				log.Printf("[F] Forwarded")
			case err.Timeout():
				log.Printf("[F] Timed out")
			default:
				log.Printf("[F] Failed to forward: %v", err)
			}
		case <-shutdown:
			log.Printf("[F] Stopping; waiting for the last task...")
			<-errChan
			break SERVICE_LOOP
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
