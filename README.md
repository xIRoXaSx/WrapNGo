# WrapNGo
<img src="https://user-images.githubusercontent.com/38859398/167641088-6eed9457-2540-454d-bb38-a21700fd4c51.png" alt="WrapNGo icon made by LilliaKurako" style="float: right" align="right"/>  

## About
Started as a simple backup solution, WrapNGo evolved into a fully configurable process wrapper.  
This small and efficient wrapper allows you to run multiple operations either sequential or in parallel.  
Depending on your needs, WrapNGo can be set up to handle simple or complex tasks for single or recurring actions.

***

## Installation
For the installation, simply download the latest release from the [release section](https://github.com/xIRoXaSx/WrapNGo/releases).  
If you want to build the project yourself: install go, clone the repository, change into the cloned repo directory and build it via `go build` / `go install`.  
If you want to disable CGO (not necessary for this repository), you can use `CGO_ENABLED=0 go build` instead.

To use WrapNGo or any other executable from any directory, you can set up the `PATH` variable as described in these stackoverflow posts:

| Operating Systems                               |
|-------------------------------------------------|
| [Windows](https://stackoverflow.com/a/44272417) |
| [OSX](https://stackoverflow.com/a/16288733)     |
| [Linux](https://stackoverflow.com/a/14638025)   |

***

## Usage
You can either call the program without any arguments to use the interactive mode or call it with the name of the Task you wish to execute (`WrapNGo <NameOfTheTaskToStart>`).
![WrapNGo-Interactive](https://user-images.githubusercontent.com/38859398/167660363-3911b453-4a7d-40fc-97e8-dfc3429d55b5.gif)

***

## Configuration
Each task can consist of 3 main components:
```
Task
  ├ PreOperations
  ├ Job
  └ PostOperations
```

Each `PreOperations` and `PostOperations` section can consist of 0 or more operations:
```
Task
  ├ PreOperations
    ├ Operation 1
    ├ Operation 2
    └ Operation 3
  ├ Job
  └ PostOperations
```

Down below, you can find the default config which will be generated after starting the executable the first time.  
If you feel more comfortable using YAML instead, you can start the program again (without any arguments) and select 
`Create main yaml config (config.yaml)` in the interactive menu.  
Afterwards you can open either of the configurations inside your local config folder:

| Operating System | Path of the configuration                                      | Shorthand                                         |
|------------------|----------------------------------------------------------------|---------------------------------------------------|
| Windows          | `C:\Users\%USERNAME%\AppData\Roaming\WrapNGo\config.json`      | %AppData%\WrapNGo\config.json                     |
| OSX              | `/Users/$USER/Library/Application Support/wrapngo/config.json` | ~/Library/Application Support/wrapngo/config.json |
| Linux            | `/home/$USER/.config/wrapngo/config.json`                      | ~/.config/wrapngo/config.json                     |

### Notice
For Windows users: The described path assumes that your Windows drive is `C:`. Change drive-letter correspondingly!

### Explanation
The following table explains what each property inside the config does:

| Property name                              | Description                                                                                                                           |
|--------------------------------------------|---------------------------------------------------------------------------------------------------------------------------------------|
| GeneralSettings.GlobalCommand              | This is the binary / program path which will be used as a fallback (whenever the Task's command is unset / empty)                     |
| GeneralSettings.Debug                      | If set to `true`, more information will be printed to have a much simpler debugging experience                                        |
| GeneralSettings.CaseSensitiveJobNames      | If set to `true`, tasks will only be executed if the given argument matches the case sensitive task name                              |
| GeneralSettings.DateFormat                 | The general date and time format for the `%Date%` placeholder                                                                         |
| GlobalDynamic                              | Same as `Tasks.Dynamic` but can be accessed by every task, operation and configuration file.                                          |
| Tasks.Name                                 | The name of the task. Used for calling each task (`./WrapNGo <TaskName>`)                                                             |
| Tasks.Command                              | The job's command, script or executable path to use                                                                                   |
| Tasks.Dynamic                              | This section allows you to create your own variables to use as placeholders to organize your commands                                 |
| Tasks.Arguments                            | These are the arguments to use with the provided `Command` property                                                                   |
| Tasks.StopIfUnsuccessful                   | Whether to stop the execution of all parallelized `PreOperations` and `PostOperations` if the job fails                               |
| Tasks.Compression.PathToCompress           | If set, the given path will be compressed into a *.tar.gz file before the job starts                                                  |
| Tasks.Compression.OutputPath               | If set, the compressed archive will be placed into the given path. If empty the parent directory of the source will be used instead   |
| Tasks.Compression.InMemoryCompressionLimit | The maximum allowed file / dir size in order to use in-memory compression. If size is larger, compression will append to archive file |
| Tasks.Compression.OverwriteCompressed      | Whether the compressed content of `PathToCompress` should be overwritten or not                                                       |
| Tasks.Compression.RetainStructure          | Whether the compressed archive will keep the original path inside the archive or just its content                                     |
| Tasks.RemovePathAfterJobCompletes          | If set, the given path will be removed after the job completes                                                                        |
| Tasks.AllowParallelOperationsRun           | Whether the `PreOperations` should run in parallel (job won't wait for all `PreOperations` to finish)                                 |
| Tasks.Operations.Enabled                   | Whether the corresponding `PreOperation` / `PostOperation` should be activated                                                        |
| Tasks.Operations.StopIfUnsuccessful        | Whether the corresponding `PreOperation` / `PostOperation` should cause the task to fail (= exit code 1) on error                     |
| Tasks.Operations.SecondsUntilTimeout       | The amount of seconds after which the `PreOperation` / `PostOperation` should be considered as failed                                 |
| Tasks.Operations.IgnoreTimeout             | Whether the configured timeout (`Tasks.Operations.SecondsUntilTimeout`) should be ignored / disabled                                  |
| Tasks.Operations.CaptureStdOut             | Whether the output of the `PreOperation` / `PostOperation` process should be logged to the console                                    |
| Tasks.Operations.Command                   | Same functionality as `Tasks.Command`                                                                                                 |
| Tasks.Operations.Arguments                 | Same functionality as `Tasks.Arguments`                                                                                               |

### Notice
When using multiple configurations, only the `GeneralSettings` of the main file (`config.json` / `config.yaml`) will be applied.  

`Tasks.Compression.PathToCompress`: If you want to use this feature to compress any directory / file and want to retrieve the name of the archive,
simply use the [placeholder](#placeholders) `%PathToCompress%`.  

`Tasks.Compression.InMemoryCompressionLimit`: You can use any available number (must fit into the current available RAM) via `B` (byte), `KB`, `MB` or `GB`.  
Decimal places are not allowed, use the smaller unit instead!   
E.g.: `InMemoryCompressionLimit: 512MB`, `InMemoryCompressionLimit: 1GB`, ...  

`Tasks.Compression.RetainStructure`: If you set `RetainStructure` to true the output archive will keep the path to the source file.  
E.g.: `PathToCompress: /path/to/file/to/compress` will also include the `/path/to/file/to` directory structure.

`GlobalDynamic`: When using this property across multiple configurations, every unique property will be added to the collection.  
You can use them across every config.

### Multiple configuration files
If you like to split the configuration into individual files (to separate them into logical collections for instance), you can create
multiple configurations (yaml or json) inside your local config folder (check table down below).
As an example, we want to separate all "startup" tasks from the "background" tasks:

#### startup.json
```json
{
  "Tasks": [
    {
      "Name": "StartupTask1",
      "Command": "SomeCommandToStart"
    },
    {
      "Name": "StartupTask2",
      "Command": "SomeOtherCommandToStart"
    } 
  ]
}
```

#### background.json
```json
{
  "Tasks": [
    {
      "Name": "BackgroundTask1",
      "Command": "SomeCommandForTheBackground"
    },
    {
      "Name": "BackgroundTask2",
      "Command": "SomeOtherCommandForTheBackground"
    } 
  ]
}
```

***

## Placeholders
The integrated placeholders will simplify your configuration even more!  
To use placeholders, you can specify any **Task** property between two `%`-signs inside the properties listed down below.  
An example for such placeholders in a simplified version of a task:
```json
{
  "Command": "mv",
  "Arguments": [
    "%Compression.PathToCompress%",
    "%Dynamic.Destination%"
  ],
  "Compression": {
    "PathToCompress": "Some/Path/To/Compress",
    "InMemoryCompressionLimit": "1GB"
  }
}
```

This will move the compressed archive to the location defined in the `Task.Dynamic.Destination` property.  
You are allowed to use any placeholder from the Task itself (even inside the `Operations`) but you cannot use placeholders of `Operations`.

There are a few additional placeholders / placeholder functions as well:

| Placeholder      | Description                                                                                                                        |
|------------------|------------------------------------------------------------------------------------------------------------------------------------|
| %Date%           | The current date of the corresponding execution. The format of `GeneralSettings.DateFormat` will be used                           |
| %Date(<FORMAT>)% | The current date of the corresponding execution. Replace `<FORMAT>` with the desired date and time [format](#date-and-time-format) |
| %Env(<NAME>)%    | The environmental variable's value. Replace `<NAME>` with the provided & accessible env. variable name                             |

Inside each of the following properties placeholders can be used:
- `Command`
- `Arguments`
- `PathToCompress`
- `InMemoryCompressionLimit`
- `RemovePathAfterJobCompletes`

### Date and time format
If you want to use a customized date and time format, you can have a look at the following table.  
Formats are **case-sensitive**!

| Format | Description                                    | Example |
|--------|------------------------------------------------|---------|
| YYYY   | The current year in 4 digits                   | 2022    |
| YYY    | The current year day                           | 130     |
| YY     | The current year in 2 digits                   | 22      |
| MMMM   | The current month in letters                   | May     |
| MMM    | The current month's first 3 letters            | May     |
| MM     | The current month in 2 digits                  | 05      |
| M      | The current month in 1 digit (if applicable)   | 5       |
| DDDD   | The current day in letters                     | Tuesday |
| DDD    | The current day's first 3 letters              | Tue     |
| DD     | The current day in 2 digits                    | 10      |
| D      | The current day in 1 digit (if applicable)     | 10      |
| hha    | The current time in am / pm format             | 3:04PM  |
| hh     | The current 24 hour formatted hour in 2 digits | 14      |
| h      | The current 24 hour formatted hour in 1 digits | 14      |
| mm     | The current minute in 2 digits                 | 41      |
| m      | The current minute in 1 digit                  | 41      |
| ss     | The current second in 2 digits                 | 09      |
| s      | The current second in 1 digit                  | 9       |
| ms     | The current millisecond                        | 219     |


### Default configuration
#### JSON format
```json
{
  "GeneralSettings": {
    "GlobalCommand": "your-program-to-wrap",
    "Debug": false,
    "CaseSensitiveJobNames": false,
    "DateFormat": "YYYY-MM-DD_hh-mm-ss"
  },
  "GlobalDynamic": {
    "Description": "Here you can specify global dynamics to use as placeholders."
  },
  "Tasks": [
    {
      "Name": "ShortNameOfTask",
      "Command": "Binary/command",
      "Dynamic": {
        "Description": "Define your own placeholders here and use the placeholder with %Dynamic.Name%",
        "Destination": "Some/Destination/Path",
        "Source": "Some/Source/Path"
      },
      "Arguments": [
        "--SomeArgument",
        "--another=Argument",
        "--Argument 3"
      ],
      "StopIfUnsuccessful": true,
      "RemovePathAfterJobCompletes": "",
      "AllowParallelOperationsRun": false,
      "Compression": {
        "PathToCompress": "",
        "OutputPath": "",
        "InMemoryCompressionLimit": "1GB",
        "OverwriteCompressed": false,
        "RetainStructure": false
      },
      "PreOperations": [
        {
          "Enabled": false,
          "StopIfUnsuccessful": true,
          "SecondsUntilTimeout": 3,
          "IgnoreTimeout": false,
          "CaptureStdOut": true,
          "Command": "Call-Another-Program-Or-Script-Before-Main-Program-Ran",
          "Arguments": [
            "Description: Arguments can be used inside your called script / application.",
            "StartedAt: %Date%",
            "Command: %Command%",
            "Source: %Dynamic.Source%",
            "Destination: %Dynamic.Destination%"
          ]
        }
      ],
      "PostOperations": [
        {
          "Enabled": false,
          "StopIfUnsuccessful": true,
          "SecondsUntilTimeout": 3,
          "IgnoreTimeout": false,
          "CaptureStdOut": true,
          "Command": "Call-Another-Program-Or-Script-After-Main-Program-Ran",
          "Arguments": [
            "Description: Arguments can be used inside your called script / application.",
            "StartedAt: %Date%",
            "Command: %Command%",
            "Source: %Dynamic.Source%",
            "Destination: %Dynamic.Destination%"
          ]
        }
      ]
    }
  ]
}
```

#### YAML format
```yaml
GeneralSettings:
  GlobalCommand: your-program-to-wrap
  Debug: false
  CaseSensitiveJobNames: false
  DateFormat: YYYY-MM-DD_hh-mm-ss
GlobalDynamic:
  Description: Here you can specify global dynamics to use as placeholders.
Tasks:
  - Name: ShortNameOfTask
    Command: Binary/command
    Dynamic:
      Description: Define your own placeholders here and use the placeholder with %Dynamic.Name%
      Destination: Some/Destination/Path
      Source: Some/Source/Path
    Arguments:
      - --SomeArgument
      - --another=Argument
      - --Argument 3
    StopIfUnsuccessful: true
    RemovePathAfterJobCompletes: ""
    AllowParallelOperationsRun: false
    Compression:
      PathToCompress: ""
      OutputPath: ""
      InMemoryCompressionLimit: 1GB
      OverwriteCompressed: false
      RetainStructure: false
    PreOperations:
      - Enabled: false
        StopIfUnsuccessful: true
        SecondsUntilTimeout: 3
        IgnoreTimeout: false
        CaptureStdOut: true
        Command: Call-Another-Program-Or-Script-Before-Main-Program-Ran
        Arguments:
          - 'Description: Arguments can be used inside your called script / application.'
          - 'StartedAt: %Date%'
          - 'Command: %Command%'
          - 'Source: %Dynamic.Source%'
          - 'Destination: %Dynamic.Destination%'
    PostOperations:
      - Enabled: false
        StopIfUnsuccessful: true
        SecondsUntilTimeout: 3
        IgnoreTimeout: false
        CaptureStdOut: true
        Command: Call-Another-Program-Or-Script-After-Main-Program-Ran
        Arguments:
          - 'Description: Arguments can be used inside your called script / application.'
          - 'StartedAt: %Date%'
          - 'Command: %Command%'
          - 'Source: %Dynamic.Source%'
          - 'Destination: %Dynamic.Destination%'
```

***

## Credits
Special thanks to [@LilliaKurako](https://twitter.com/LilliaKurako) for the amazing artwork!

## Used libraries
| Library   | Use                                 | Maintainer  | Repository                                      |
|-----------|-------------------------------------|-------------|-------------------------------------------------|
| Survey v2 | Interactive menu for the executable | AlecAivazis | [GitHub](https://github.com/AlecAivazis/survey) |
