package main

import (
	"WrapNGo/config"
	"WrapNGo/logger"
	"os"
	"strings"
	"sync"
)

func init() {
	// Create new config if not already existing.
	path, err := config.NewConfig()
	if err != nil {
		logger.Fatal(err)
	}
	if path == "" {
		logger.Infof("Please modify the created config and restart. Path of config: %s\n", path)
		os.Exit(0)
	}

	// Load config values.
	err = config.LoadConfig()
	if err != nil {
		logger.Fatal(err)
		return
	}

	// Create a new logger.
	logger.NewInstance(config.Current().GeneralSettings.Debug)
}

func main() {
	args := os.Args
	numErr := 0
	if len(args) > 1 {
		conf := config.Current()
		tasks := make([]config.Task, 0)
		if conf.GeneralSettings.CaseSensitiveJobNames {
			for _, t := range conf.Tasks {
				if t.Name != args[1] {
					continue
				}
				tasks = append(tasks, t)
			}
		} else {
			for _, t := range conf.Tasks {
				if strings.ToLower(t.Name) != strings.ToLower(args[1]) {
					continue
				}
				tasks = append(tasks, t)
			}
		}

		// Run each task in a separate goroutine to parallelize.
		wg := sync.WaitGroup{}
		for _, t := range tasks {
			wg.Add(1)
			logger.Infof("Starting Task \"%s\" in the background.\n", t.Name)
			go func(t config.Task) {
				defer wg.Done()
				err := RunTask(&t)
				if err != nil {
					logger.Error(err)
					numErr++
				}
				logger.Infof("%s: Task finished\n", t.Name)
			}(t)
		}
		wg.Wait()
		if numErr > 0 {
			os.Exit(1)
		}
	}
}
