package main

import (
	"fmt"
	"os"
	"path/filepath"
	"strings"

	"github.com/vrooli/cli-core/cliutil"
	"scenario-completeness-scoring/cli/format"
)

const (
	appVersion     = "0.1.0"
	defaultAPIBase = ""

	genericSourceRootEnvVar = "VROOLI_CLI_SOURCE_ROOT"
	legacySourceRootEnvVar  = "SCENARIO_COMPLETENESS_SCORING_CLI_SOURCE_ROOT"
)

var (
	buildFingerprint = "unknown"
	buildTimestamp   = "unknown"
	buildSourceRoot  = ""
)

type Config struct {
	APIBase string `json:"api_base"`
	Token   string `json:"token,omitempty"`
}

type App struct {
	configStore     *cliutil.ConfigFile
	config          Config
	apiOverride     string
	client          *cliutil.HTTPClient
	api             *APIClient
	configDirectory string
	staleChecker    *cliutil.StaleChecker
	commands        []commandDefinition
	commandGroups   []commandGroup
	colorEnabled    bool
}

type commandDefinition struct {
	Name     string
	Aliases  []string
	NeedsAPI bool
	Handler  func([]string) error
}

type commandGroup struct {
	Title    string
	Commands []commandDefinition
}

func (a *App) describeCommand(name string) string {
	switch name {
	case "help":
		return "Show this help message"
	case "version":
		return "Show CLI version"
	case "configure":
		return "View or update local CLI settings (api_base, token)"
	case "status":
		return "Check API & collector health"
	case "collectors":
		return "Show collector health status"
	case "circuit-breaker":
		return "View or reset the circuit breaker status"
	case "scores":
		return "List completeness scores for all scenarios"
	case "score":
		return "Show detailed score for a scenario"
	case "calculate":
		return "Force score recalculation and save history"
	case "history":
		return "View score history"
	case "trends":
		return "View trend analysis for a scenario"
	case "what-if":
		return "Run hypothetical improvement analysis"
	case "recommend":
		return "Get prioritized improvement recommendations"
	case "config":
		return "Show server scoring configuration"
	case "presets":
		return "List configuration presets"
	case "preset":
		return "Apply a configuration preset"
	default:
		return ""
	}
}

func main() {
	app, err := NewApp()
	if err != nil {
		fmt.Fprintf(os.Stderr, "Error: %v\n", err)
		os.Exit(1)
	}
	if err := app.Run(os.Args[1:]); err != nil {
		fmt.Fprintf(os.Stderr, "Error: %v\n", err)
		os.Exit(1)
	}
}

func NewApp() (*App, error) {
	dir, err := os.UserHomeDir()
	if err != nil {
		return nil, fmt.Errorf("resolve home directory: %w", err)
	}
	configDir := filepath.Join(dir, ".scenario-completeness-scoring")
	configStore, err := cliutil.NewConfigFile(filepath.Join(configDir, "config.json"))
	if err != nil {
		return nil, err
	}
	cfg := Config{}
	if err := configStore.Load(&cfg); err != nil {
		return nil, err
	}
	colorEnabled := os.Getenv("NO_COLOR") == ""
	app := &App{
		configStore:     configStore,
		config:          cfg,
		apiOverride:     "",
	client:          cliutil.NewHTTPClient(cliutil.HTTPClientOptions{}),
	configDirectory: configDir,
	staleChecker:    cliutil.NewStaleChecker("scenario-completeness-scoring", buildFingerprint, buildTimestamp, buildSourceRoot, genericSourceRootEnvVar, legacySourceRootEnvVar),
	colorEnabled:    colorEnabled,
}
app.api = NewAPIClient(app.client, app.buildAPIBaseOptions, func() string { return app.config.Token })
app.commandGroups, app.commands = app.registerCommands()
return app, nil
}

func (a *App) saveConfig() error {
	return a.configStore.Save(a.config)
}

func (a *App) registerCommands() ([]commandGroup, []commandDefinition) {
	health := []commandDefinition{
		{Name: "status", NeedsAPI: true, Handler: func(args []string) error { return a.cmdStatus() }},
		{Name: "collectors", NeedsAPI: true, Handler: func(args []string) error { return a.cmdCollectors() }},
		{Name: "circuit-breaker", NeedsAPI: true, Handler: a.cmdCircuitBreaker},
	}
	scoring := []commandDefinition{
		{Name: "scores", NeedsAPI: true, Handler: a.cmdScores},
		{Name: "score", NeedsAPI: true, Handler: a.cmdScore},
		{Name: "calculate", NeedsAPI: true, Handler: a.cmdCalculate},
		{Name: "history", NeedsAPI: true, Handler: a.cmdHistory},
		{Name: "trends", NeedsAPI: true, Handler: a.cmdTrends},
		{Name: "what-if", NeedsAPI: true, Handler: a.cmdWhatIf},
		{Name: "recommend", NeedsAPI: true, Handler: a.cmdRecommend},
	}
	config := []commandDefinition{
		{Name: "config", NeedsAPI: true, Handler: a.cmdConfig},
		{Name: "presets", NeedsAPI: true, Handler: func(args []string) error { return a.cmdPresets() }},
		{Name: "preset", NeedsAPI: true, Handler: a.cmdPreset},
		{Name: "configure", NeedsAPI: false, Handler: a.cmdConfigure},
	}
	meta := []commandDefinition{
		{Name: "help", Aliases: []string{"--help", "-h"}, NeedsAPI: false, Handler: func(args []string) error {
			a.printHelp()
			return nil
		}},
		{Name: "version", Aliases: []string{"--version", "-v"}, NeedsAPI: false, Handler: func(args []string) error {
			fmt.Printf("scenario-completeness-scoring CLI version %s\n", appVersion)
			return nil
		}},
	}

	var all []commandDefinition
	appendAll := func(cmds []commandDefinition) {
		all = append(all, cmds...)
	}
	appendAll(meta)
	appendAll(health)
	appendAll(scoring)
	appendAll(config)

	groups := []commandGroup{
		{Title: "Meta", Commands: meta},
		{Title: "Health", Commands: health},
		{Title: "Scoring", Commands: scoring},
		{Title: "Configuration", Commands: config},
	}
	return groups, all
}

func (a *App) findCommand(name string) (commandDefinition, bool) {
	for _, cmd := range a.commands {
		if cmd.Name == name {
			return cmd, true
		}
		for _, alias := range cmd.Aliases {
			if alias == name {
				return cmd, true
			}
		}
	}
	return commandDefinition{}, false
}

func (a *App) Run(args []string) error {
	if len(args) == 0 {
		a.printHelp()
		return nil
	}
	remaining, err := a.consumeGlobalFlags(args)
	if err != nil {
		return err
	}
	format.SetColorEnabled(a.colorEnabled)
	if len(remaining) == 0 {
		a.printHelp()
		return nil
	}
	cmd, ok := a.findCommand(remaining[0])
	if !ok {
		return fmt.Errorf("Unknown command: %s", remaining[0])
	}
	if cmd.NeedsAPI {
		a.warnIfBinaryStale()
	}
	return cmd.Handler(remaining[1:])
}

func (a *App) consumeGlobalFlags(args []string) ([]string, error) {
	remaining := make([]string, 0, len(args))
	for i := 0; i < len(args); i++ {
		if args[i] == "--api-base" {
			if i+1 >= len(args) {
				return nil, fmt.Errorf("missing value for --api-base")
			}
			a.apiOverride = args[i+1]
			i++
			continue
		}
		if args[i] == "--no-color" {
			a.colorEnabled = false
			continue
		}
		if args[i] == "--color" {
			a.colorEnabled = true
			continue
		}
		remaining = append(remaining, args[i])
	}
	return remaining, nil
}

func (a *App) printHelp() {
	fmt.Print(`scenario-completeness-scoring CLI

Usage:
  scenario-completeness-scoring <command> [options]

Global Options:
  --api-base <url>   Override API base URL (default: auto-detected)
  --no-color         Disable ANSI color output (or set NO_COLOR)

Commands:
`)
	for _, group := range a.commandGroups {
		fmt.Printf("  %s\n", group.Title)
		for _, cmd := range group.Commands {
			fmt.Printf("    %-28s %s\n", cmd.Name, a.describeCommand(cmd.Name))
		}
		fmt.Println()
	}

	fmt.Print(`Environment:
  VROOLI_CLI_SOURCE_ROOT       Override the source path used for stale-build detection
  SCENARIO_COMPLETENESS_SCORING_CLI_SOURCE_ROOT
                               Legacy override for stale-build detection
  NO_COLOR                     Disable ANSI color output
`)
}

type multiValue []string

func (m *multiValue) String() string {
	return strings.Join(*m, ",")
}

func (m *multiValue) Set(value string) error {
	*m = append(*m, value)
	return nil
}

func getString(m map[string]interface{}, key string) string {
	if value, ok := m[key].(string); ok {
		return value
	}
	return ""
}
