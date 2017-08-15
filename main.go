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

func receive(stopChan chan bool) {

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
		case <-stopChan:
			log.Printf("[R] Stopping; waiting for the last task...")
			<-errChan
			break SERVICE_LOOP
		}
	}
}

func forward(stopChan chan bool) {

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
		case <-stopChan:
			log.Printf("[F] Stopping; waiting for the last task...")
			<-errChan
			break SERVICE_LOOP
		}
	}
}

func main() {
	var wg sync.WaitGroup
	sigChan := make(chan os.Signal)
	stopChan := make(chan bool)

	signal.Notify(sigChan, os.Interrupt, os.Kill)
	go func() {
		sig := <-sigChan
		log.Printf("RECEIVED A SIGNAL: %s(%d)", sig, sig)
		stopChan <- true
		stopChan <- true
	}()

	wg.Add(2)
	go func() {
		defer wg.Done()
		receive(stopChan)
	}()
	go func() {
		defer wg.Done()
		forward(stopChan)
	}()
	wg.Wait()

	log.Printf("Exiting")
}
