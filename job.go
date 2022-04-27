package main

import (
	"CloudTransferTasks/config"
	"bytes"
	"fmt"
	"log"
	"os"
	"os/exec"
	"os/signal"
	"strings"
	"syscall"
	"time"
)

const (
	jobPreOperation  = "PreOperation"
	jobPostOperation = "PostOperation"
)

// RunJob will execute the defined Job in a separate goroutine.
func RunJob(j config.Job) (err error) {
	usrItr := make(chan os.Signal, 1)
	opItr := make(chan error, 1)
	job := make(chan error)
	signal.Notify(usrItr, syscall.SIGINT, syscall.SIGTERM)

	// Execute PreOperation if available.
	if j.PreOperation != nil && j.PreOperation.Enabled {
		if j.PreOperation.AllowParallelRun {
			go func() {
				err = runOperation(j.PreOperation, j.Name, jobPreOperation)
				if err != nil {
					log.Println(err)
					opItr <- err
				}
			}()
		} else {
			err = runOperation(j.PreOperation, j.Name, jobPreOperation)
			if err != nil {
				log.Println(err)
				if j.StopIfOperationFailed {
					return
				}
			}
		}
	}

	// Execute job.
	args := make([]string, 1)
	args[0] = j.Action

	// Since flags can contain spaces, separate them
	// and append them to the args slice.
	for _, f := range j.StartFlags {
		flags := strings.Split(f, " ")
		args = append(args, flags...)
	}
	args = append(args, j.Source, j.Destination)

	buf := &bytes.Buffer{}
	c := exec.Command(config.Current().GeneralSettings.BinaryPath, args...)
	c.Stdout = os.Stdout
	c.Stderr = buf
	err = c.Start()
	if err != nil {
		log.Println(err)
	}

	go func() {
		log.Printf("%s: Executing job...\n", j.Name)
		job <- c.Wait()
	}()

	select {
	case <-usrItr:
		log.Println(ErrUserInterrupt)
		err = c.Process.Kill()
		if err != nil {
			log.Println(err)
		}
		return
	case <-opItr:
		log.Println(ErrOperationInterrupt)
		err = c.Process.Kill()
		if err != nil {
			log.Println(err)
		}

		if j.StopIfOperationFailed {
			return
		}
	case err = <-job:
		if err != nil {
			if j.StopIfOperationFailed {
				log.Printf("%s: %v\n", ErrOperationFailed, err)
			} else {
				log.Printf("%v: %v\n", err, buf)
			}
			return
		}
		log.Printf("Job \"%s\" completed successfully", j.Name)
	}

	if j.PostOperation != nil && j.PostOperation.Enabled {
		err = runOperation(j.PostOperation, j.Name, jobPostOperation)
		if err != nil {
			log.Println(err)
			return
		}
		log.Printf("%s: %s finished\n", j.Name, jobPostOperation)
	}
	return
}

// runOperation runs the given operation and blocks until it has finished.
func runOperation(o *config.Operation, jobName, oType string) (err error) {
	log.Printf("%s: Executing %s\n", jobName, oType)
	c := exec.Command(o.Command, o.Arguments...)
	if o.CaptureStdOut {
		c.Stdout = os.Stdout
	}

	done := make(chan error, 1)
	err = c.Start()
	if err != nil {
		log.Println(err)
		return
	}

	go func() {
		done <- c.Wait()
	}()

	var timeout <-chan time.Time
	if o.SecondsUntilTimeout > 0 {
		timeout = time.After(time.Duration(o.SecondsUntilTimeout) * time.Second)
	}

	select {
	// If timeout reached, stop the command execution.
	case <-timeout:
		err = c.Process.Kill()
		return fmt.Errorf("%s: %v", jobName, ErrTimeout)
	// Command finished.
	case err = <-done:
	}

	if err != nil {
		return fmt.Errorf("%s: executed operation caught an error: %v", jobName, err)
	}
	return
}
