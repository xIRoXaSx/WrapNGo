package main

import (
	config "CloudTransferTasks/config"
	"bytes"
	"fmt"
	"log"
	"os"
	"os/exec"
	"os/signal"
	"reflect"
	"regexp"
	"strings"
	"syscall"
	"time"
)

const (
	jobPreOperation  = "PreOperation"
	jobPostOperation = "PostOperation"
)

// RunTask will execute the given Task.
// It will start the Pre- and Post-Operations as well as the job.
func RunTask(t config.Task) (err error) {
	usrItr := make(chan os.Signal, 1)
	opItr := make(chan error, 1)
	signal.Notify(usrItr, syscall.SIGINT, syscall.SIGTERM)

	// Execute PreOperation if available.
	if t.PreOperation != nil && t.PreOperation.Enabled {
		if t.PreOperation.AllowParallelRun {
			go func() {
				err = runOperation(t.PreOperation, t.Name, jobPreOperation)
				if err != nil {
					log.Println(err)
				}
				opItr <- err
			}()
		} else {
			err = runOperation(t.PreOperation, t.Name, jobPreOperation)
			if err != nil {
				log.Println(err)
				if t.StopIfOperationFailed {
					return
				}
			}
		}
	}

	// Run the defined job.
	err = runJob(t, usrItr)
	if err != nil {
		log.Println(err)
		if t.StopIfTaskFailed {
			return
		}
	}

	// Execute the PostOperation if available.
	if t.PostOperation != nil && t.PostOperation.Enabled {
		err = runOperation(t.PostOperation, t.Name, jobPostOperation)
		if err != nil {
			log.Println(err)
			return
		}
		log.Printf("%s: %s finished\n", t.Name, jobPostOperation)
	}
	log.Printf("%s: Task %s finished\n", t.Name, jobPostOperation)
	return
}

// runJob executes the actual rclone action.
func runJob(t config.Task, itrChan chan os.Signal) (err error) {
	opItr := make(chan error, 1)
	job := make(chan error)
	args := make([]string, 1)
	args[0] = t.Action
	args = append(args, t.Source, t.Destination)

	// Since flags can contain spaces, separate them
	// and append them to the args slice.
	for _, f := range t.StartFlags {
		flags := strings.Split(f, " ")
		args = append(args, flags...)
	}

	// To be able to use the Task's FileTypes slice,
	// append it to the args via the include / exclude flag.
	if len(t.FileTypes) > 0 {
		appendType := "--exclude"
		if !t.FileTypesAsBlacklist {
			appendType = "--include"
		}

		// Since user can use spaces between each filetype,
		// make sure to separate each one via comma to keep things standardized.
		args = append(args, appendType)
		ft := make([]string, 0)
		for _, f := range t.FileTypes {
			ft = append(ft, escapeSplit(f, "\\", " ")...)
		}
		args = append(args, fmt.Sprintf("{%v}", strings.Join(ft, ",")))
	}

	buf := &bytes.Buffer{}
	c := exec.Command(config.Current().GeneralSettings.BinaryPath, args...)
	c.Stdout = os.Stdout
	c.Stderr = buf
	err = c.Start()
	if err != nil {
		log.Println(err)
	}

	go func() {
		log.Printf("%s: Executing job...\n", t.Name)
		job <- c.Wait()
	}()

	select {
	case <-itrChan:
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

		if t.StopIfOperationFailed {
			return
		}
	case err = <-job:
	}
	if err != nil {
		return fmt.Errorf("%s: %v: %v", t.Name, ErrTaskFailed, buf)
	}

	log.Printf("Task \"%s\" completed successfully", t.Name)
	return
}

// replacePlaceholders checks the given strings for placeholders and replaces them accordingly.
func replacePlaceholders(t config.Task, values ...string) (replaced []string) {
	dateFormat := config.Current().GeneralSettings.DateFormat
	fElem := reflect.ValueOf(&t).Elem()
	for _, v := range values {
		// Check for date placeholders.
		if dateFormat != "" {
			reg, err := regexp.Compile(fmt.Sprintf("(?i)(%sDate%s)", config.PlaceholderChar, config.PlaceholderChar))
			if err != nil {
				continue
			}
			found := reg.FindStringSubmatch(v)
			if len(found) > 0 {
				v = strings.Replace(v, found[1], time.Now().Format(dateFormat), -1)
			}
		}

		for i := 0; i < fElem.NumField(); i++ {
			fName := fElem.Type().Field(i).Name
			fVal := fmt.Sprintf("%s", fElem.Field(i))
			reg, err := regexp.Compile(fmt.Sprintf("(?i)(%s%s%s)", config.PlaceholderChar, fName, config.PlaceholderChar))
			if err != nil {
				continue
			}
			found := reg.FindStringSubmatch(v)
			if len(found) > 0 {
				replaced = append(replaced, strings.Replace(v, found[1], fVal, -1))
			}
		}
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

// omitEmpty returns a new slice that does not contain empty values.
func omitEmpty(values []string) (val []string) {
	for _, v := range values {
		if strings.TrimSpace(v) != "" {
			val = append(val, v)
		}
	}
	return
}

// escapeSplit will check the value for an escaped sequence before splitting to omit wrong splits.
// escapeSeq is the string that should prevent the split.
// separator is the string that is used to split.
// 	Example: escapeSplit("Escaped\\ space, not escaped", "\\", " ")
// 	Will produce []string{"Escaped space,", "not", "escaped"}
func escapeSplit(value, escapeSeq, separator string) (values []string) {
	values = make([]string, 0)
	token := "\x00"
	replacedValue := strings.ReplaceAll(value, escapeSeq+separator, token)
	replaced := omitEmpty(strings.Split(replacedValue, separator))
	for _, ftToken := range replaced {
		split := strings.Split(ftToken, " ")
		for i := 0; i < len(split); i++ {
			values = append(values, strings.ReplaceAll(split[i], token, separator))
		}
	}
	return
}
