package main

import (
	"WrapNGo/config"
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
				opItr <- runOperation(t.PreOperation, usrItr, t, jobPreOperation)
			}()
		} else {
			err = runOperation(t.PreOperation, usrItr, t, jobPreOperation)
			if err != nil {
				if t.StopIfOperationFailed {
					return
				}
			}
		}
	}

	// Run the defined job.
	err = runJob(t, usrItr, opItr)
	if err != nil {
		if t.StopIfJobFailed {
			return
		}
		log.Println(err)
		err = nil
	}

	// Execute the PostOperation if available.
	if t.PostOperation != nil && t.PostOperation.Enabled {
		err = runOperation(t.PostOperation, usrItr, t, jobPostOperation)
		if err != nil {
			return fmt.Errorf("%s: %s: %v", t.Name, jobPostOperation, err)
		}
		log.Printf("%s: %s finished\n", t.Name, jobPostOperation)
	}
	log.Printf("%s: Task finished\n", t.Name)
	return
}

// runJob executes the actual binary action.
func runJob(t config.Task, itrChan chan os.Signal, opItr chan error) (err error) {
	job := make(chan error)
	args := make([]string, 1)
	args[0] = t.Action

	// Compress source if enabled.
	if t.CompressToTarBeforeHand {
		var path string
		path, err = Compress(t.Source, t.OverwriteCompressedTar)
		if err != nil && t.StopIfJobFailed {
			return
		}
		t.Source = path
	}
	args = append(args, replacePlaceholders(t, t.Source)[0], replacePlaceholders(t, t.Destination)[0])

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

	args = replacePlaceholders(t, args...)
	buf := &bytes.Buffer{}
	c := exec.Command(config.Current().GeneralSettings.BinaryPath, args...)
	c.Stdout = os.Stdout
	c.Stderr = buf
	err = c.Start()
	if err != nil {
		log.Println(fmt.Sprintf("%s: %v", t.Name, err))
	}

	go func() {
		log.Printf("%s: Executing job...\n", t.Name)
		job <- c.Wait()
	}()

	select {
	case <-itrChan:
		err = c.Process.Kill()
		if err != nil {
			log.Println(err)
		}
		return fmt.Errorf("%s: %v", t.Name, ErrUserInterrupt)
	case <-opItr:
		err = c.Process.Kill()
		if err != nil {
			log.Println(err)
		}
		if t.RemoveAfterJobCompletes {
			err = os.Remove(t.Source)
			if err != nil {
				log.Println(err)
			}
		}
		return fmt.Errorf("%s: %v", t.Name, ErrOperationFailed)
	case err = <-job:
	}

	if t.RemoveAfterJobCompletes {
		err = os.Remove(t.Source)
		if err != nil {
			log.Println(err)
		}
	}

	if err != nil {
		return fmt.Errorf("%s: %v: %v", t.Name, ErrJobFailed, buf)
	}

	log.Printf("Job \"%s\" completed successfully", t.Name)
	return
}

// runOperation runs the given operation and blocks until it has finished.
func runOperation(o *config.Operation, itrChan chan os.Signal, t config.Task, oType string) (err error) {
	log.Printf("%s: Executing %s\n", t.Name, oType)
	c := exec.Command(o.Command, replacePlaceholders(t, o.Arguments...)...)
	if o.CaptureStdOut {
		c.Stdout = os.Stdout
	}

	done := make(chan error, 1)
	err = c.Start()
	if err != nil {
		log.Println(fmt.Sprintf("%s: %s: %v", t.Name, oType, err))
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
		return fmt.Errorf("%s: %s - %v", t.Name, oType, ErrTimeout)

	// User has interrupted, stop command execution.
	case <-itrChan:
		err = c.Process.Kill()
		return fmt.Errorf("%s: %s - %v", t.Name, oType, ErrUserInterrupt)

	// Command finished.
	case err = <-done:
	}

	if err != nil {
		return fmt.Errorf("%s: %s: executed operation caught an error: %v", t.Name, oType, err)
	}
	return
}

// replacePlaceholders checks the given strings for placeholders and replaces them accordingly.
func replacePlaceholders(t config.Task, values ...string) (replaced []string) {
	tm := time.Now()
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
				v, err = parseDate(tm, dateFormat)
			}
		}

		// Check for single date placeholders (e.g %Date(YYYY-MM-DD)%)
		dtFmt := regexp.QuoteMeta("(") + "(.*)" + regexp.QuoteMeta(")")
		reg, err := regexp.Compile(
			fmt.Sprintf("(?i)(%sDate%s%s)", config.PlaceholderChar, dtFmt, config.PlaceholderChar),
		)
		if err != nil {
			continue
		}
		found := reg.FindStringSubmatch(v)
		if len(found) > 0 {
			var parsed string
			parsed, err = parseDate(tm, found[2])
			if err != nil {
				return
			}
			v = strings.ReplaceAll(v, found[1], parsed)
		}

		// Dynamic placeholders.
		for i := 0; i < fElem.NumField(); i++ {
			fName := fElem.Type().Field(i).Name
			fVal := fmt.Sprintf("%s", fElem.Field(i))
			reg, err = regexp.Compile(fmt.Sprintf("(?i)(%s%s%s)", config.PlaceholderChar, fName, config.PlaceholderChar))
			if err != nil {
				continue
			}
			found = reg.FindStringSubmatch(v)
			if len(found) > 0 {
				v = strings.Replace(v, found[1], fVal, -1)
			}
		}
		replaced = append(replaced, v)
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
