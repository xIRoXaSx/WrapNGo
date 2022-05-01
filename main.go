package main

import (
	"CloudTransferTasks/config"
	"fmt"
	"log"
	"os"
	"strings"
	"sync"
)

func init() {
	// Create new config if not already existing.
	path, err := config.NewConfig()
	if err != nil {
		log.Fatal(err)
	}
	if path == "" {
		log.Printf("Please modify the created config and restart. Path of config: %s\n", path)
		os.Exit(0)
	}

	// Load config values.
	err = config.LoadConfig()
	if err != nil {
		log.Fatal(err)
		return
	}
}

func main() {
	args := os.Args
	errs := make([]error, 0)
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
			fmt.Printf("Starting Task \"%s\" in the background.\n", t.Name)
			task := t
			go func() {
				defer wg.Done()
				err := RunTask(task)
				if err != nil {
					errs = append(errs, err)
					log.Println(err)
				}
			}()
		}
		wg.Wait()
		if len(errs) > 0 {
			os.Exit(1)
		}
	}
}
