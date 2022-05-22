package main

import (
	"WrapNGo/config"
	"WrapNGo/logger"
	"errors"
	"fmt"
	"log"
	"os"
	"strings"
	"sync"

	"github.com/AlecAivazis/survey/v2"
)

func init() {
	// Create new config if not already existing.
	created := createConf(false, false)
	if created {
		os.Exit(0)
	}

	// Load config values.
	err := config.LoadAll()
	if err != nil {
		log.Fatalf("%s: %s", ErrInitializing, err)
		return
	}

	// Create a new logger.
	logger.NewInstance(config.Current().GeneralSettings.Debug)
}

func main() {
	args := os.Args
	if len(args) < 2 {
		interactive()
		return
	}

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

		if len(tasks) < 1 {
			logger.Warn("no such task found.")
			return
		}

		// Run each task in a separate goroutine to parallelize.
		wg := sync.WaitGroup{}
		for _, t := range tasks {
			wg.Add(1)
			logger.Infof("Starting Task \"%s\" in the background.\n", t.Name)
			go func(t config.Task) {
				defer wg.Done()
				err := RunTask(&t, conf.GlobalDynamic)
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
		return
	}
	logger.Info("Please provide a command or task name.")
}

func interactive() {
	const (
		listTasks   = "List tasks"
		executeTask = "Execute tasks"
		createJson  = "Create main json config (config.json)"
		createYaml  = "Create main yaml config (config.yaml)"
		regen       = "Regenerate main configs"
		exit        = "Exit"
	)

	for {
		jPath, err := config.FullPath(false)
		if err != nil {
			logger.Error(err)
			continue
		}

		yPath, err := config.FullPath(true)
		if err != nil {
			logger.Error(err)
			continue
		}

		opts := []string{
			listTasks,
			executeTask,
		}

		// Append options dynamically.
		_, err = os.Stat(jPath)
		if errors.Is(err, os.ErrNotExist) {
			opts = append(opts, createJson)
		}
		_, err = os.Stat(yPath)
		if errors.Is(err, os.ErrNotExist) {
			opts = append(opts, createYaml)
		}
		opts = append(opts, regen, exit)

		opt := ""
		err = survey.AskOne(&survey.Select{
			Message: "Please select an option",
			Options: opts,
			Default: 0,
		}, &opt)
		if err != nil {
			logger.Fatal(ErrUserInterrupt)
		}

		switch opt {
		case listTasks:
			// List all tasks.
			cfg := config.Current()
			tskStr := "tasks are"
			if len(cfg.Tasks) == 1 {
				tskStr = "task is"
			}

			logger.Infof("Currently %d %s stored:\n", len(cfg.Tasks), tskStr)
			for i := 0; i < len(cfg.Tasks); i++ {
				logger.Infof("\t> %s: %s", cfg.Tasks[i].Name, cfg.Tasks[i].Command)
			}
		case executeTask:
			// Execute selected task.
			ind := 0
			cfg := config.Current()
			tasks := make([]string, len(cfg.Tasks))
			for i := 0; i < len(cfg.Tasks); i++ {
				tasks[i] = cfg.Tasks[i].Name
			}

			err = survey.AskOne(&survey.Select{
				Options: tasks,
				Message: "Choose a task to execute",
			}, &ind)
			if err != nil {
				logger.Fatal(err)
			}

			os.Args = append(os.Args, cfg.Tasks[ind].Name)
			main()
		case createJson:
			createConf(false, false)
		case createYaml:
			createConf(false, true)
		case regen:
			// Regenerate config.
			confirm := false
			err = survey.AskOne(&survey.Confirm{
				Message: "Are you sure? This will overwrite your current main configs!",
			}, &confirm)
			if err != nil {
				logger.Fatal(err)
			}
			if !confirm {
				continue
			}
			createConf(true, false)
			createConf(true, true)
		case exit:
			// Exit.
			return
		}
		fmt.Println()
	}
}

// createConf is a small convenience wrapper to create a new config in the desired format.
func createConf(overwrite, isYaml bool) (created bool) {
	path, created, err := config.NewConfig(overwrite, isYaml)
	if err != nil {
		log.Fatal(err)
	}
	if created {
		log.Printf("Please modify the created config and restart. Path of config: %s\n", path)
	}
	return
}
