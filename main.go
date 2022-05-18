package main

import (
	"WrapNGo/config"
	"WrapNGo/logger"
	"fmt"
	"log"
	"os"
	"strings"
	"sync"

	"github.com/AlecAivazis/survey/v2"
)

func init() {
	// Create new config if not already existing.
	path, created, err := config.NewConfig(false)
	if err != nil {
		log.Fatal(err)
	}
	if created {
		log.Printf("Please modify the created config and restart. Path of config: %s\n", path)
		os.Exit(0)
	}

	// Load config values.
	err = config.LoadConfig()
	if err != nil {
		log.Fatal(err)
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
	for {
		var opt int
		err := survey.AskOne(&survey.Select{
			Message: "Please select an option",
			Options: []string{
				"List tasks",
				"Execute task",
				"Regenerate default config",
				"Exit",
			},
			Default: 0,
		}, &opt)
		if err != nil {
			logger.Fatal(ErrUserInterrupt)
		}

		switch opt {
		case 0:
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
		case 1:
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
		case 2:
			// Regenerate config.
			var (
				confirm bool
				path    string
			)
			err = survey.AskOne(&survey.Confirm{
				Message: "Are you sure? This will overwrite your current config!",
			}, &confirm)
			if err != nil {
				logger.Fatal(err)
			}

			if !confirm {
				continue
			}

			path, _, err = config.NewConfig(true)
			if err != nil {
				logger.Fatalf("could not create new config: %v", err)
			}
			logger.Infof("Please modify the created config and restart. Path of config: %s\n", path)
		case 3:
			// Exit.
			return
		}
		fmt.Println()
	}
}
