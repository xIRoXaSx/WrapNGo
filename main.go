package main

import (
	"CloudTransferTasks/config"
	"log"
	"os"
	"strings"
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
	if len(args) > 1 {
		conf := config.Current()
		if conf.GeneralSettings.CaseSensitiveJobNames {
			for _, j := range conf.Jobs {
				if j.Name != args[1] {
					continue
				}
				err := RunJob(j)
				if err != nil {
					log.Println(err)
					continue
				}
			}
			return
		}
		for _, j := range conf.Jobs {
			if strings.ToLower(j.Name) != strings.ToLower(args[1]) {
				continue
			}
			err := RunJob(j)
			if err != nil {
				log.Println(err)
				continue
			}
		}
	}
}
