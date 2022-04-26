package main

import (
	"CloudTransferTasks/config"
	"github.com/desertbit/closer"
	"io"
	"log"
	"os"
	"os/exec"
	"os/signal"
	"syscall"
	"time"
)

// RunJob will execute the defined Job in a separate goroutine.
func RunJob(j *config.Job) {
	cl := closer.New()
	defer func() {
		cErr := cl.Close()
		if cErr != nil {
			log.Println(cErr)
		}
	}()

	itr := make(chan os.Signal, 1)
	signal.Notify(itr, syscall.SIGINT, syscall.SIGTERM)
	go func() {
		select {
		// User interrupted, return.
		case <-itr:
			return
		// Closer got closed, run PostOperation if available.
		case <-cl.CloseChan():
			if j.PostOperation != nil && j.PostOperation.Enabled {
				runOperation(j.PostOperation)
			}
		}
	}()
}

// runOperation runs the given operation.
func runOperation(o *config.Operation) {
	c := exec.Command(o.Command, o.Arguments...)
	if o.CaptureStdOut {
		c.Stdout = io.MultiWriter(os.Stdout)
	}

	err := c.Start()
	done := make(chan error)

	// Block until finished.
	go func() {
		done <- c.Wait()
	}()

	var tc <-chan time.Time
	if o.SecondsUntilTimeout > 0 {
		tc = time.After(time.Duration(o.SecondsUntilTimeout) * time.Second)
	}

	select {
	// If timeout reached, stop the command execution.
	case <-tc:
		log.Println("Timeout reached... Force stopping")
		err = c.Process.Kill()
	// Command finished.
	case err = <-done:
	}

	if err != nil {
		log.Printf("Executed operation caught an error: %v", err.Error())
		return
	}

	log.Println("Operation executed successfully")
}
