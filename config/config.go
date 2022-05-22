package config

import (
	"encoding/json"
	"errors"
	"fmt"
	"io/fs"
	"log"
	"os"
	"path/filepath"
	"runtime"
	"strings"
	"sync"

	"gopkg.in/yaml.v3"
)

const (
	jsonExtension   = ".json"
	yamlExtension   = ".yaml"
	fileBaseName    = "config"
	fileNameJson    = fileBaseName + jsonExtension
	fileNameYaml    = fileBaseName + yamlExtension
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
	GlobalCommand         string `json:"GlobalCommand" yaml:"GlobalCommand"`
	Debug                 bool   `json:"Debug" yaml:"Debug"`
	CaseSensitiveJobNames bool   `json:"CaseSensitiveJobNames" yaml:"CaseSensitiveJobNames"`
	DateFormat            string `json:"DateFormat" yaml:"DateFormat"`
}

// The Operation type contains information for a single Task operation.
// Each Task can contain up to 2 Tasks (Pre- and Post-operation).
type Operation struct {
	Enabled             bool     `json:"Enabled" yaml:"Enabled"`
	StopIfUnsuccessful  bool     `json:"StopIfUnsuccessful" yaml:"StopIfUnsuccessful"`
	SecondsUntilTimeout int      `json:"SecondsUntilTimeout" yaml:"SecondsUntilTimeout"`
	IgnoreTimeout       bool     `json:"IgnoreTimeout" yaml:"IgnoreTimeout"`
	CaptureStdOut       bool     `json:"CaptureStdOut" yaml:"CaptureStdOut"`
	Command             string   `json:"Command" yaml:"Command"`
	Arguments           []string `json:"Arguments" yaml:"Arguments"`
}

// The Task type contains information for a single job.
// The Config contains n Tasks.
type Task struct {
	Name                        string         `json:"Name" yaml:"Name"`
	Command                     string         `json:"Command" yaml:"Command"`
	Dynamic                     map[string]any `json:"Dynamic" yaml:"Dynamic"`
	Arguments                   []string       `json:"Arguments" yaml:"Arguments"`
	StopIfUnsuccessful          bool           `json:"StopIfUnsuccessful" yaml:"StopIfUnsuccessful"`
	CompressPathToTarBeforeHand string         `json:"CompressPathToTarBeforeHand" yaml:"CompressPathToTarBeforeHand"`
	OverwriteCompressed         bool           `json:"OverwriteCompressed" yaml:"OverwriteCompressed"`
	RemovePathAfterJobCompletes string         `json:"RemovePathAfterJobCompletes" yaml:"RemovePathAfterJobCompletes"`
	AllowParallelOperationsRun  bool           `json:"AllowParallelOperationsRun" yaml:"AllowParallelOperationsRun"`
	PreOperations               []Operation    `json:"PreOperations" yaml:"PreOperations"`
	PostOperations              []Operation    `json:"PostOperations" yaml:"PostOperations"`
}

// The Config type contains all the information used inside this project.
type Config struct {
	GeneralSettings GeneralSettings `json:"GeneralSettings" yaml:"GeneralSettings"`
	GlobalDynamic   map[string]any  `json:"GlobalDynamic" yaml:"GlobalDynamic"`
	Tasks           []Task          `json:"Tasks" yaml:"Tasks"`
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
func NewConfig(overwrite, isYaml bool) (path string, created bool, err error) {
	path, err = FullPath(isYaml)
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
		if isYaml {
			b, err = yaml.Marshal(defConf)
		} else {
			b, err = json.MarshalIndent(defConf, "", "\t")
		}
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

// LoadJson loads the given file to the in memory config.
func LoadJson(path string, isMain bool) (err error) {
	var conf Config
	err = conf.LoadInto(path, false)
	if err != nil {
		return
	}

	config.Lock()
	defer config.Unlock()

	config.Tasks = append(config.Tasks, conf.Tasks...)
	addGlobalDynamics(conf.GlobalDynamic)
	if isMain {
		config.GeneralSettings = conf.GeneralSettings
	}
	return
}

// LoadYaml loads the given file to the in memory config.
func LoadYaml(path string, isMain bool) (err error) {
	var conf Config
	err = conf.LoadInto(path, true)
	if err != nil {
		return
	}

	config.Lock()
	defer config.Unlock()

	config.Tasks = append(config.Tasks, conf.Tasks...)
	addGlobalDynamics(conf.GlobalDynamic)
	if isMain {
		config.GeneralSettings = conf.GeneralSettings
	}
	return
}

// addGlobalDynamics adds all values of m to config if not already set.
// This implementation is not thread-safe.
func addGlobalDynamics(m map[string]any) {
	if m == nil {
		return
	}

	for k, v := range m {
		_, ok := config.GlobalDynamic[k]
		if ok {
			log.Printf("GlobalDynamic '%s' has already been set, skipping\n", k)
			continue
		}
		config.GlobalDynamic[k] = v
	}
}

func (c *Config) LoadInto(path string, isYaml bool) (err error) {
	b, err := os.ReadFile(path)
	if err != nil {
		return
	}
	if c.Mutex == nil {
		c.Mutex = &sync.Mutex{}
	}
	c.Lock()
	defer c.Unlock()

	if isYaml {
		err = yaml.Unmarshal(b, &c)
		return
	}
	err = json.Unmarshal(b, &c)
	return
}

func LoadAll() (err error) {
	p, err := os.UserConfigDir()
	if err != nil {
		return
	}

	// Config dir to lower case if not Windows machine.
	dir := configDirPath()
	p = filepath.Join(p, dir)
	if err != nil {
		return
	}

	mainFound := false
	err = filepath.Walk(p, func(path string, info fs.FileInfo, err error) (wErr error) {
		stat, wErr := os.Stat(path)
		if wErr != nil {
			return wErr
		}

		if !stat.IsDir() {
			name := stat.Name()

			isYaml := strings.HasSuffix(strings.ToLower(name), yamlExtension)
			isJson := strings.HasSuffix(strings.ToLower(name), jsonExtension)
			if !isYaml && !isJson {
				return
			}

			if isYaml {
				isMain := name == fileNameYaml
				if isMain {
					mainFound = true
				}
				wErr = LoadYaml(path, isMain)
			} else {
				isMain := name == fileNameJson
				if isMain {
					mainFound = true
				}
				wErr = LoadJson(path, isMain)
			}
			if wErr != nil {
				return fmt.Errorf("unable to load %s: %v\n", name, wErr)
			}
		}
		return
	})
	if err != nil {
		return
	}

	if !mainFound {
		log.Println("main config could not be found, please ensure 'config.json' / 'config.yaml' is available")
	}
	return
}

// FullPath returns the full path of the configuration file.
func FullPath(isYaml bool) (p string, err error) {
	p, err = os.UserConfigDir()
	if err != nil {
		return
	}

	// Config dir to lower case if not Windows machine.
	dir := configDirPath()
	if isYaml {
		p = filepath.Join(p, dir, fileNameYaml)
		return
	}
	p = filepath.Join(p, dir, fileNameJson)
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
