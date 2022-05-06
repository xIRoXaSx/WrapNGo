package main

import (
	"WrapNGo/config"
	"WrapNGo/logger"
	"WrapNGo/parsing"
	"bytes"
	"fmt"
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

var (
	wildcardReg = regexp.QuoteMeta("(") + "(.*)" + regexp.QuoteMeta(")")
	dateReg     = regexp.MustCompile(fmt.Sprintf("(?i)(%sDate%s)", config.PlaceholderChar, config.PlaceholderChar))
	mapReg      = regexp.MustCompile("map\\[(.*)]")
	dynamicReg  = regexp.MustCompile("%Dynamic\\.(.*?)%")
	dateFuncReg = regexp.MustCompile(
		fmt.Sprintf("(?i)(%sDate%s%s)", config.PlaceholderChar, wildcardReg, config.PlaceholderChar),
	)
	envFuncReg = regexp.MustCompile(
		fmt.Sprintf("(?i)(%sEnv%s%s)", config.PlaceholderChar, wildcardReg, config.PlaceholderChar),
	)
)

// RunTask will execute the given Task.
// It will start the Pre- and Post-Operations as well as the job.
func RunTask(t *config.Task) (err error) {
	usrItr := make(chan os.Signal, 1)
	opItr := make(chan error, 1)
	signal.Notify(usrItr, syscall.SIGINT, syscall.SIGTERM)

	// Execute PreOperations if available.
	if t.AllowParallelOperationsRun {
		for i, preOp := range t.PreOperations {
			if !preOp.Enabled {
				continue
			}

			go func(o config.Operation, num int) {
				opItr <- runOperation(o, *t, usrItr, jobPreOperation, num)
			}(preOp, i+1)
		}
	} else {
		for i, preOp := range t.PreOperations {
			if !preOp.Enabled {
				continue
			}

			err = runOperation(preOp, *t, usrItr, jobPreOperation, i+1)
			if err != nil {
				if preOp.StopIfUnsuccessful {
					return
				}
			}
		}
	}

	// Run the defined job.
	err = runJob(t, usrItr, opItr)
	if err != nil {
		if t.StopIfUnsuccessful {
			return
		}
		logger.Error(err)
		err = nil
	}

	// Execute the PostOperations if available.
	for i, postOp := range t.PostOperations {
		if !postOp.Enabled {
			continue
		}

		err = runOperation(postOp, *t, usrItr, jobPostOperation, i+1)
		if err != nil && postOp.StopIfUnsuccessful {
			return
		}
	}
	return
}

// runJob executes the actual binary action.
func runJob(t *config.Task, itrChan chan os.Signal, opItr chan error) (err error) {
	job := make(chan error)
	args := make([]string, 1)
	args[0] = t.Command

	// Compress source if enabled.
	t.CompressPathToTarBeforeHand = replacePlaceholders(*t, t.CompressPathToTarBeforeHand)[0]
	if t.CompressPathToTarBeforeHand != "" {
		var path string
		path, err = compress(t.CompressPathToTarBeforeHand, t.OverwriteCompressed)

		// Only write back if compressing was successful.
		if err != nil && t.StopIfUnsuccessful {
			return
		}
		t.CompressPathToTarBeforeHand = path
	}
	args = append(args, replacePlaceholders(*t, t.Command)[0], replacePlaceholders(*t, t.Command)[0])

	// Since flags can contain spaces, separate them
	// and append them to the args slice.
	for _, f := range t.Arguments {
		flags := strings.Split(f, " ")
		args = append(args, flags...)
	}

	args = replacePlaceholders(*t, args...)
	buf := &bytes.Buffer{}
	cmd := config.Current().GeneralSettings.GlobalCommand
	if t.Command != "" {
		cmd = t.Command
	}

	c := exec.Command(cmd, args...)
	c.Stdout = os.Stdout
	c.Stderr = buf
	err = c.Start()
	if err != nil {
		logger.Errorf("%s: %v", t.Name, err)
	}

	go func() {
		logger.Infof("%s: Executing job...\n", t.Name)
		job <- c.Wait()
	}()

	// Anonymous function which tries to remove the given path
	// 3 times after the job completes.
	removePath := func(path string) {
		if path != "" {
			_, err := os.Stat(path)
			if err != nil {
				logger.Error(err)
				return
			}
			for i := 0; i < 3; i++ {
				time.Sleep(500 * time.Millisecond)
				err = os.Remove(path)
				if err == nil {
					return
				}
			}
			logger.Error(err)
		}
		return
	}

	t.RemovePathAfterJobCompletes = replacePlaceholders(*t, t.RemovePathAfterJobCompletes)[0]
	select {
	case <-itrChan:
		err = c.Process.Kill()
		if err != nil {
			logger.Error(err)
		}
		return fmt.Errorf("%s: %v", t.Name, ErrUserInterrupt)
	case <-opItr:
		err = c.Process.Kill()
		if err != nil {
			logger.Error(err)
		}
		removePath(t.RemovePathAfterJobCompletes)
		return fmt.Errorf("%s: %v: %v", t.Name, jobPreOperation, ErrOperationFailed)
	case err = <-job:
	}

	removePath(t.RemovePathAfterJobCompletes)
	if err != nil {
		return fmt.Errorf("%s: %v: %s", t.Name, ErrJobFailed, strings.TrimSuffix(buf.String(), "\n"))
	}

	logger.Infof("Job \"%s\" completed successfully", t.Name)
	return
}

// runOperation runs the given operation and blocks until it has finished.
func runOperation(o config.Operation, t config.Task, itrChan chan os.Signal, oType string, oNum int) (err error) {
	logger.Infof("%s: Executing %s #%d\n", t.Name, oType, oNum)
	c := exec.Command(o.Command, replacePlaceholders(t, o.Arguments...)...)
	if o.CaptureStdOut {
		c.Stdout = logger.OperationWriter()
	}

	done := make(chan error, 1)
	err = c.Start()
	if err != nil {
		return
	}

	go func() {
		done <- c.Wait()
	}()

	var timeout <-chan time.Time
	if o.SecondsUntilTimeout > 0 && !o.IgnoreTimeout {
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
	var (
		err      error
		fieldReg *regexp.Regexp
		date     string
	)
	for _, v := range values {
		// Check for date placeholders.
		if dateFormat != "" {
			found := dateReg.FindStringSubmatch(v)
			if len(found) > 0 {
				date, err = parsing.ParseDate(tm, dateFormat)
				if err != nil {
					return
				}
				v = strings.Replace(v, found[0], date, -1)
			}
		}

		// Check for single date placeholders (e.g %Date(YYYY-MM-DD)%)
		found := dateFuncReg.FindStringSubmatch(v)
		if len(found) > 0 {
			var parsed string
			parsed, err = parsing.ParseDate(tm, found[2])
			if err != nil {
				return
			}
			v = strings.ReplaceAll(v, found[1], parsed)
		}

		// Check for env placeholders.
		found = envFuncReg.FindStringSubmatch(v)
		if len(found) > 0 {
			env := os.Getenv(found[2])
			v = strings.ReplaceAll(v, found[1], env)
		}

		// Task dependent placeholders.
		for i := 0; i < fElem.NumField(); i++ {
			fName := fElem.Type().Field(i).Name
			fVal := fmt.Sprintf("%s", fElem.Field(i))
			fieldReg, err = regexp.Compile(fmt.Sprintf("(?i)(%s%s%s)", config.PlaceholderChar, fName, config.PlaceholderChar))
			if err != nil {
				continue
			}
			found = fieldReg.FindStringSubmatch(v)
			if len(found) > 0 {
				v = strings.Replace(v, found[1], fVal, -1)
			}
			if fName != "Dynamic" {
				continue
			}

			// Dynamic placeholders.
			found = mapReg.FindStringSubmatch(fVal)
			if len(found) > 0 {
				foundMatches := dynamicReg.FindAllStringSubmatch(v, -1)
				valMap := escapeSplit(found[1], "\\", ":")
				for j := 0; j < len(foundMatches); j++ {
					for m := 0; m < len(valMap); m += 2 {
						if strings.ToLower(foundMatches[j][1]) != strings.ToLower(valMap[m]) {
							continue
						}
						if m+1 >= len(valMap) {
							break
						}
						v = strings.Replace(v, foundMatches[j][0], valMap[m+1], -1)
					}
				}
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
//  Example: escapeSplit("Escaped\\ space, not escaped", "\\", " ")
//  Will produce []string{"Escaped space,", "not", "escaped"}
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
