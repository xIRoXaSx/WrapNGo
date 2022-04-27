package config

import (
	"encoding/json"
	"errors"
	"os"
	"path/filepath"
	"runtime"
	"strings"
	"sync"
)

const (
	fileName        = "config.yml"
	dirName         = "CloudTransferTasks"
	placeholderChar = "%"
)

var config = &Config{
	GeneralSettings: GeneralSettings{},
	Jobs:            []Job{},
	Mutex:           &sync.Mutex{},
}

type GeneralSettings struct {
	BinaryPath            string `json:"BinaryPath"`
	Debug                 bool   `json:"Debug"`
	CaseSensitiveJobNames bool   `json:"CaseSensitiveJobNames"`
}

// The Operation type contains information for a single Job operation.
// Each Job can contain up to 2 Jobs (Pre- and Post-operation).
type Operation struct {
	Enabled              bool     `json:"Enabled"`
	AllowParallelRun     bool     `json:"AllowParallelRun"`
	FailIfNotSuccessful  bool     `json:"FailIfNotSuccessful"`
	SecondsUntilTimeout  int      `json:"SecondsUntilTimeout"`
	ContinueAfterTimeout bool     `json:"ContinueAfterTimeout"`
	Command              string   `json:"Command"`
	Arguments            []string `json:"Arguments"`
	CaptureStdOut        bool     `json:"CaptureStdOut"`
}

// The Job type contains information for a single job.
// The Config contains n Jobs.
type Job struct {
	Name                  string     `json:"Name"`
	Source                string     `json:"Source"`
	Destination           string     `json:"Destination"`
	Action                string     `json:"Action"`
	FileTypes             []string   `json:"FileTypes"`
	StartFlags            []string   `json:"StartFlags"`
	StopIfOperationFailed bool       `json:"StopIfOperationFailed"`
	PreOperation          *Operation `json:"PreOperation"`
	PostOperation         *Operation `json:"PostOperation"`
}

// The Config type contains all the information used inside this project.
type Config struct {
	GeneralSettings GeneralSettings `json:"GeneralSettings"`
	Jobs            []Job           `json:"Jobs"`
	*sync.Mutex
}

// defaultConfig defines the default configuration.
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
						"StartedAt: " + formatPlaceholder("Date"),
						"CurrentAction: " + formatPlaceholder("Action"),
						"Source: " + formatPlaceholder("Source"),
						"Destination: " + formatPlaceholder("Destination"),
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
						"StartedAt: " + formatPlaceholder("Date"),
						"CurrentAction: " + formatPlaceholder("Action"),
						"Source: " + formatPlaceholder("Source"),
						"Destination: " + formatPlaceholder("Destination"),
					},
				},
			},
		},
	}
}

// NewConfig creates a new config if it does not already exist.
func NewConfig() (path string, err error) {
	path, err = configPath()
	if err != nil {
		return
	}

	// Create the folder which contains the config.
	p, err := os.UserConfigDir()
	if err != nil {
		return
	}
	p = filepath.Join(p, configDirPath())
	_, err = os.Stat(p)
	if errors.Is(err, os.ErrNotExist) {
		err = os.Mkdir(p, 0700)
		if err != nil {
			return
		}
	}

	_, err = os.Stat(path)
	if errors.Is(err, os.ErrNotExist) {
		defConf := defaultConfig()
		var b []byte
		b, err = json.MarshalIndent(defConf, "", "\t")
		if err != nil {
			return
		}

		err = os.WriteFile(path, b, 0700)
		if err != nil {
			return
		}
	}
	return
}

// LoadConfig loads the local configuration into the config type.
func LoadConfig() (err error) {
	fullPath, err := configPath()
	if err != nil {
		return
	}

	b, err := os.ReadFile(fullPath)
	if err != nil {
		return
	}

	config.Lock()
	err = json.Unmarshal(b, &config)
	config.Unlock()
	if err != nil {
		return
	}
	return
}

// configPath returns the full path of the configuration file.
func configPath() (p string, err error) {
	p, err = os.UserConfigDir()
	if err != nil {
		return
	}

	// Config dir to lower case if not Windows machine.
	dir := configDirPath()
	p = filepath.Join(p, dir, fileName)
	return
}

// configDirPath returns the dirName specifically formatted for the current OS.
func configDirPath() (dir string) {
	dir = dirName
	if runtime.GOOS != "windows" {
		dir = strings.ToLower(dir)
	}
	return
}

// formatPlaceholder formats the given key to a placeholder.
func formatPlaceholder(key string) string {
	return placeholderChar + key + placeholderChar
}

func Current() *Config {
	return config
}
