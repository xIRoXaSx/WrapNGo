package config

import (
	"encoding/json"
	"errors"
	"log"
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
	GlobalDynamic:   map[string]any{},
	Tasks:           []Task{},
	Mutex:           &sync.Mutex{},
}

type GeneralSettings struct {
	GlobalCommand         string `json:"GlobalCommand"`
	Debug                 bool   `json:"Debug"`
	CaseSensitiveJobNames bool   `json:"CaseSensitiveJobNames"`
	DateFormat            string `json:"DateFormat"`
}

// The Operation type contains information for a single Task operation.
// Each Task can contain up to 2 Tasks (Pre- and Post-operation).
type Operation struct {
	Enabled             bool     `json:"Enabled"`
	StopIfUnsuccessful  bool     `json:"StopIfUnsuccessful"`
	SecondsUntilTimeout int      `json:"SecondsUntilTimeout"`
	IgnoreTimeout       bool     `json:"IgnoreTimeout"`
	CaptureStdOut       bool     `json:"CaptureStdOut"`
	Command             string   `json:"Command"`
	Arguments           []string `json:"Arguments"`
}

// The Task type contains information for a single job.
// The Config contains n Tasks.
type Task struct {
	Name                        string         `json:"Name"`
	Command                     string         `json:"Command"`
	Dynamic                     map[string]any `json:"Dynamic"`
	Arguments                   []string       `json:"Arguments"`
	StopIfUnsuccessful          bool           `json:"StopIfUnsuccessful"`
	CompressPathToTarBeforeHand string         `json:"CompressPathToTarBeforeHand"`
	OverwriteCompressed         bool           `json:"OverwriteCompressed"`
	RemovePathAfterJobCompletes string         `json:"RemovePathAfterJobCompletes"`
	AllowParallelOperationsRun  bool           `json:"AllowParallelOperationsRun"`
	PreOperations               []Operation    `json:"PreOperations"`
	PostOperations              []Operation    `json:"PostOperations"`
}

// The Config type contains all the information used inside this project.
type Config struct {
	GeneralSettings GeneralSettings `json:"GeneralSettings"`
	GlobalDynamic   map[string]any  `json:"GlobalDynamic"`
	Tasks           []Task          `json:"Tasks"`
	*sync.Mutex
}

// defaultConfig defines the default configuration.
func defaultConfig() *Config {
	return &Config{
		GeneralSettings: GeneralSettings{
			GlobalCommand: "your-program-to-wrap",
			DateFormat:    "YYYY-MM-DD_hh-mm-ss",
		},
		GlobalDynamic: map[string]any{
			"Description": "Here you can specify global dynamics to use as placeholders.",
		},
		Tasks: []Task{
			{
				Name:               "ShortNameOfTask",
				Command:            "Binary/command",
				StopIfUnsuccessful: true,
				Dynamic: map[string]any{
					"Description": "Define your own placeholders here and use the placeholder with %Dynamic.Name%",
					"Source":      "Some/Source/Path",
					"Destination": "Some/Destination/Path",
				},
				Arguments: []string{"-P", "--retries 5", "--transfers 3"},
				PreOperations: []Operation{
					{
						StopIfUnsuccessful:  true,
						CaptureStdOut:       true,
						Command:             "Call-Another-Program-Or-Script-Before-Main-Program-Ran",
						SecondsUntilTimeout: 3,
						Arguments: []string{
							"Description: Arguments can be used inside your called script / application.",
							"StartedAt: " + formatPlaceholder("Date"),
							"Command: " + formatPlaceholder("Command"),
							"Source: " + formatPlaceholder("Dynamic.Source"),
							"Destination: " + formatPlaceholder("Dynamic.Destination"),
						},
					},
				},
				PostOperations: []Operation{
					{
						StopIfUnsuccessful:  true,
						CaptureStdOut:       true,
						Command:             "Call-Another-Program-Or-Script-After-Main-Program-Ran",
						SecondsUntilTimeout: 3,
						Arguments: []string{
							"Description: Arguments can be used inside your called script / application.",
							"StartedAt: " + formatPlaceholder("Date"),
							"Command: " + formatPlaceholder("Command"),
							"Source: " + formatPlaceholder("Dynamic.Source"),
							"Destination: " + formatPlaceholder("Dynamic.Destination"),
						},
					},
				},
			},
		},
	}
}

// NewConfig creates a new config.
func NewConfig(overwrite bool) (path string, created bool, err error) {
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

	if err == nil && overwrite {
		err = os.Remove(path)
		if err != nil {
			log.Fatal("unable to remove config file")
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
		created = true
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
