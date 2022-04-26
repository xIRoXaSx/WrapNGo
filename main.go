package main

import (
	"CloudTransferTasks/config"
	"log"
	"os"
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

}
