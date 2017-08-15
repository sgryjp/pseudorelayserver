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

// execTask sleeps for randomly determined duration.
// It sometimes fail and returns an error.
func execTask() error {
	n := 500 + rand.Intn(2500)
	time.Sleep(time.Duration(n) * time.Millisecond)
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
			errChan <- execTask()
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
	for i := 0; ; i++ {
		log.Printf("[F] Executing a task %d...", i)
		go func() {
			errChan <- execTask()
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
