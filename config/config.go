package config

import (
	"encoding/json"
	"errors"
	"os"
	"path/filepath"
	"runtime"
	"strings"
)

const (
	fileName = "config.yml"
	dirName  = "CloudTransferTasks"
)

type GeneralSettings struct {
	BinaryPath string `json:"BinaryPath"`
	Debug      bool   `json:"Debug"`
}

type Operation struct {
	Enabled              bool     `json:"Enabled"`
	FailIfNotSuccessful  bool     `json:"FailIfNotSuccessful"`
	Command              string   `json:"Command"`
	SecondsUntilTimeout  int      `json:"SecondsUntilTimeout"`
	ContinueAfterTimeout bool     `json:"ContinueAfterTimeout"`
	Arguments            []string `json:"Arguments"`
}

type Job struct {
	Name          string     `json:"Name"`
	Source        string     `json:"Source"`
	Destination   string     `json:"Destination"`
	Action        string     `json:"Action"`
	FileTypes     []string   `json:"FileTypes"`
	StartFlags    []string   `json:"StartFlags"`
	PreOperation  *Operation `json:"PreOperation"`
	PostOperation *Operation `json:"PostOperation"`
}

type Config struct {
	GeneralSettings GeneralSettings `json:"GeneralSettings"`
	Jobs            []Job           `json:"Jobs"`
}

func defaultConfig() *Config {
	return &Config{
		GeneralSettings: GeneralSettings{
			BinaryPath: "/path/to/rclone",
			Debug:      false,
		},
		Jobs: []Job{
			{
				Name:        "ShortNameOfTask",
				Source:      "source",
				Destination: "SomeDrive:Destination/Path",
				Action:      "copy",
				FileTypes:   []string{"*.png", "*.jpg", "*.gif"},
				StartFlags:  []string{"-P", "--retries 5", "--transfers 3"},
				PreOperation: &Operation{
					Enabled:              false,
					FailIfNotSuccessful:  true,
					Command:              "Call-Another-Program-Or-Script-Before-Rclone-Ran",
					SecondsUntilTimeout:  3,
					ContinueAfterTimeout: false,
					Arguments: []string{
						"Description: Arguments can be used inside your called script / application.",
						"StartedAt: <Date>",
						"CurrentAction: <Action>",
						"Source: <Source>",
						"Destination: <Destination>",
					},
				},
				PostOperation: &Operation{
					Enabled:              false,
					FailIfNotSuccessful:  true,
					Command:              "Call-Another-Program-Or-Script-After-Rclone-Ran",
					SecondsUntilTimeout:  3,
					ContinueAfterTimeout: false,
					Arguments: []string{
						"Description: Arguments can be used inside your called script / application.",
						"StartedAt: <Date>",
						"CurrentAction: <Action>",
						"Source: <Source>",
						"Destination: <Destination>",
					},
				},
			},
		},
	}
}

// NewConfig creates a new config if it does not already exist.
func NewConfig() (err error) {
	fullPath, err := configPath()
	if err != nil {
		return
	}

	// Create the folder which contains the config.
	p, err := os.UserConfigDir()
	if err != nil {
		return
	}
	p = filepath.Join(p, dirName)
	_, err = os.Stat(p)
	if errors.Is(err, os.ErrNotExist) {
		err = os.Mkdir(p, 0700)
		if err != nil {
			return
		}
	}

	_, err = os.Stat(fullPath)
	if errors.Is(err, os.ErrNotExist) {
		defConf := defaultConfig()
		var confBytes []byte
		confBytes, err = json.Marshal(defConf)
		if err != nil {
			return
		}

		err = os.WriteFile(fullPath, confBytes, 0500)
		if err != nil {
			return
		}
	}
	return
}

// LoadConfig loads the local configuration into the config type.
func LoadConfig() (conf Config, err error) {
	fullPath, err := configPath()
	if err != nil {
		return
	}

	b, err := os.ReadFile(fullPath)
	if err != nil {
		return
	}

	err = json.Unmarshal(b, &conf)
	if err != nil {
		return
	}
	return
}

func configPath() (p string, err error) {
	p, err = os.UserConfigDir()
	if err != nil {
		return
	}

	// Config dir to lower case if not Windows machine.
	dir := dirName
	if runtime.GOOS != "windows" {
		dir = strings.ToLower(dir)
	}

	p = filepath.Join(p, dir, fileName)
	return
}
