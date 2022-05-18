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
	wildcardReg      = regexp.QuoteMeta("(") + "(.*)" + regexp.QuoteMeta(")")
	dateReg          = regexp.MustCompile(fmt.Sprintf("(?i)(%sDate%s)", config.PlaceholderChar, config.PlaceholderChar))
	mapReg           = regexp.MustCompile("\"(.+?)\":\"(.+?)\"[,}]")
	dynamicReg       = regexp.MustCompile("%Dynamic\\.(.*?)%")
	globalDynamicReg = regexp.MustCompile("%GlobalDynamic\\.(.*?)%")
	dateFuncReg      = regexp.MustCompile(
		fmt.Sprintf("(?i)(%sDate%s%s)", config.PlaceholderChar, wildcardReg, config.PlaceholderChar),
	)
	envFuncReg = regexp.MustCompile(
		fmt.Sprintf("(?i)(%sEnv%s%s)", config.PlaceholderChar, wildcardReg, config.PlaceholderChar),
	)
)

// RunTask will execute the given Task.
// It will start the Pre- and Post-Operations as well as the job.
func RunTask(t *config.Task, globalDynamic map[string]any) (err error) {
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
				opItr <- runOperation(o, *t, globalDynamic, usrItr, jobPreOperation, num)
			}(preOp, i+1)
		}
	} else {
		for i, preOp := range t.PreOperations {
			if !preOp.Enabled {
				continue
			}

			err = runOperation(preOp, *t, globalDynamic, usrItr, jobPreOperation, i+1)
			if err != nil {
				if preOp.StopIfUnsuccessful {
					return
				}
			}
		}
	}

	// Run the defined job.
	err = runJob(t, globalDynamic, usrItr, opItr)
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

		err = runOperation(postOp, *t, globalDynamic, usrItr, jobPostOperation, i+1)
		if err != nil && postOp.StopIfUnsuccessful {
			return
		}
	}
	return
}

// runJob executes the actual binary action.
func runJob(t *config.Task, globalDynamic map[string]any, itrChan chan os.Signal, opItr chan error) (err error) {
	job := make(chan error)
	args := make([]string, 1)
	args[0] = replacePlaceholders(*t, globalDynamic, t.Command)[0]

	// Compress source if enabled.
	t.CompressPathToTarBeforeHand = replacePlaceholders(*t, globalDynamic, t.CompressPathToTarBeforeHand)[0]
	if t.CompressPathToTarBeforeHand != "" {
		var path string
		path, err = compress(t.CompressPathToTarBeforeHand, t.OverwriteCompressed)

		// Only write back if compressing was successful.
		if err != nil && t.StopIfUnsuccessful {
			return
		}
		t.CompressPathToTarBeforeHand = path
	}

	// Since flags can contain spaces, separate them
	// and append them to the args slice.
	for _, f := range t.Arguments {
		flags := strings.Split(f, " ")
		args = append(args, flags...)
	}

	args = replacePlaceholders(*t, globalDynamic, args...)
	buf := &bytes.Buffer{}
	cmd := config.Current().GeneralSettings.GlobalCommand
	if t.Command != "" {
		cmd = t.Command
	}

	c := exec.Command(cmd, args...)
	c.Stdout = logger.JobWriter()
	c.Stdin = os.Stdin
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

	t.RemovePathAfterJobCompletes = replacePlaceholders(*t, globalDynamic, t.RemovePathAfterJobCompletes)[0]
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
func runOperation(o config.Operation, t config.Task, globalDynamic map[string]any, itrChan chan os.Signal, oType string, oNum int) (err error) {
	logger.Infof("%s: Executing %s #%d\n", t.Name, oType, oNum)
	c := exec.Command(replacePlaceholders(t, globalDynamic, o.Command)[0], replacePlaceholders(t, globalDynamic, o.Arguments...)...)
	c.Stdin = os.Stdin
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
func replacePlaceholders(t config.Task, globalDynamic map[string]any, values ...string) (replaced []string) {
	tm := time.Now()
	dateFormat := config.Current().GeneralSettings.DateFormat
	fElem := reflect.ValueOf(&t).Elem()
	if len(values) < 1 {
		return
	}

	replaceDynamics := func(regex *regexp.Regexp, mapString, v string) (replaced string) {
		replaced = v
		foundMatches := regex.FindAllStringSubmatch(replaced, -1)
		if len(foundMatches) < 1 {
			return
		}

		found := mapReg.FindAllStringSubmatch(mapString, -1)
		if len(found) > 0 {
			for _, f := range found {
				for _, fm := range foundMatches {
					if strings.ToLower(fm[1]) != strings.ToLower(f[1]) {
						continue
					}
					replaced = strings.Replace(replaced, fm[0], f[2], -1)
				}
			}
		}
		return
	}

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

			fieldReg, err = regexp.Compile(fmt.Sprintf("(?i)(%s%s%s)", config.PlaceholderChar, fName, config.PlaceholderChar))
			if err != nil {
				continue
			}
			found = fieldReg.FindStringSubmatch(v)
			if len(found) > 0 {
				fVal := fmt.Sprintf("%s", fElem.Field(i))
				v = strings.Replace(v, found[1], fVal, -1)
			}
			if fName != "Dynamic" {
				continue
			}

			// Dynamic placeholders.
			fVal := fmt.Sprintf("%#v", fElem.Field(i))
			v = replaceDynamics(dynamicReg, fVal, v)
		}

		// Dynamic placeholders.
		v = replaceDynamics(globalDynamicReg, fmt.Sprintf("%#v", globalDynamic), v)
		replaced = append(replaced, v)
	}
	return
}
