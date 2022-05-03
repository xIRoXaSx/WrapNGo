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
	fileName        = "config.json"
	dirName         = "WrapNGo"
	PlaceholderChar = "%"
)

var config = &Config{
	GeneralSettings: GeneralSettings{},
	Tasks:           []Task{},
	Mutex:           &sync.Mutex{},
}

type GeneralSettings struct {
	BinaryPath            string `json:"BinaryPath"`
	Debug                 bool   `json:"Debug"`
	CaseSensitiveJobNames bool   `json:"CaseSensitiveJobNames"`
	DateFormat            string `json:"DateFormat"`
}

// The Operation type contains information for a single Task operation.
// Each Task can contain up to 2 Tasks (Pre- and Post-operation).
type Operation struct {
	Enabled              bool     `json:"Enabled"`
	FailIfNotSuccessful  bool     `json:"FailIfNotSuccessful"`
	SecondsUntilTimeout  int      `json:"SecondsUntilTimeout"`
	ContinueAfterTimeout bool     `json:"ContinueAfterTimeout"`
	CaptureStdOut        bool     `json:"CaptureStdOut"`
	Command              string   `json:"Command"`
	Arguments            []string `json:"Arguments"`
}

// The Task type contains information for a single job.
// The Config contains n Tasks.
type Task struct {
	Name                       string      `json:"Name"`
	Source                     string      `json:"Source"`
	Destination                string      `json:"Destination"`
	Action                     string      `json:"Action"`
	FileTypesAsBlacklist       bool        `json:"FileTypesAsBlacklist"`
	FileTypes                  []string    `json:"FileTypes"`
	StartFlags                 []string    `json:"StartFlags"`
	StopIfJobFailed            bool        `json:"StopIfJobFailed"`
	CompressToTarBeforeHand    bool        `json:"CompressToTarBeforeHand"`
	OverwriteCompressedTar     bool        `json:"OverwriteCompressedTar"`
	RemoveAfterJobCompletes    bool        `json:"RemoveAfterJobCompletes"`
	AllowParallelOperationsRun bool        `json:"AllowParallelOperationsRun"`
	PreOperations              []Operation `json:"PreOperations"`
	PostOperations             []Operation `json:"PostOperations"`
}

// The Config type contains all the information used inside this project.
type Config struct {
	GeneralSettings GeneralSettings `json:"GeneralSettings"`
	Tasks           []Task          `json:"Tasks"`
	*sync.Mutex
}

// defaultConfig defines the default configuration.
func defaultConfig() *Config {
	return &Config{
		GeneralSettings: GeneralSettings{
			BinaryPath: "your-program-to-wrap",
			DateFormat: "YYYY-MM-DD_hh-mm-ss",
		},
		Tasks: []Task{
			{
				Name:            "ShortNameOfTask",
				Source:          "source",
				Destination:     "SomeDrive:Destination/Path",
				Action:          "copy",
				StopIfJobFailed: true,
				FileTypes:       []string{"*.png", "*.jpg", "*.gif"},
				StartFlags:      []string{"-P", "--retries 5", "--transfers 3"},
				PreOperations: []Operation{
					{
						FailIfNotSuccessful: true,
						CaptureStdOut:       true,
						Command:             "Call-Another-Program-Or-Script-Before-Main-Program-Ran",
						SecondsUntilTimeout: 3,
						Arguments: []string{
							"Description: Arguments can be used inside your called script / application.",
							"StartedAt: " + formatPlaceholder("Date"),
							"CurrentAction: " + formatPlaceholder("Action"),
							"Source: " + formatPlaceholder("Source"),
							"Destination: " + formatPlaceholder("Destination"),
						},
					},
				},
				PostOperations: []Operation{
					{
						FailIfNotSuccessful: true,
						CaptureStdOut:       true,
						Command:             "Call-Another-Program-Or-Script-After-Main-Program-Ran",
						SecondsUntilTimeout: 3,
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
	return PlaceholderChar + key + PlaceholderChar
}

func Current() *Config {
	return config
}
