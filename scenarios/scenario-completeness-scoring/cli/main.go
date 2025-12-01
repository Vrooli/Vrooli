package main

import (
	"fmt"
	"os"
	"path/filepath"
	"strings"

	"github.com/vrooli/cli-core/cliutil"
)

const (
	appVersion     = "0.1.0"
	defaultAPIBase = "http://localhost:17777"

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
	configDirectory string
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
	return &App{
		configStore:     configStore,
		config:          cfg,
		apiOverride:     "",
		client:          cliutil.NewHTTPClient(cliutil.HTTPClientOptions{}),
		configDirectory: configDir,
	}, nil
}

func (a *App) saveConfig() error {
	return a.configStore.Save(a.config)
}

func (a *App) Run(args []string) error {
	if len(args) == 0 {
		a.printHelp()
		return nil
	}
	remaining := a.consumeGlobalFlags(args)
	if len(remaining) == 0 {
		a.printHelp()
		return nil
	}
	switch remaining[0] {
	case "help", "--help", "-h":
		a.printHelp()
		return nil
	case "version", "--version", "-v":
		fmt.Printf("scenario-completeness-scoring CLI version %s\n", appVersion)
		return nil
	case "configure":
		return a.cmdConfigure(remaining[1:])
	case "status":
		return a.cmdStatus()
	case "scores":
		return a.cmdScores(remaining[1:])
	case "score":
		return a.cmdScore(remaining[1:])
	case "calculate":
		return a.cmdCalculate(remaining[1:])
	case "history":
		return a.cmdHistory(remaining[1:])
	case "trends":
		return a.cmdTrends(remaining[1:])
	case "what-if":
		return a.cmdWhatIf(remaining[1:])
	case "config":
		return a.cmdConfig(remaining[1:])
	case "presets":
		return a.cmdPresets()
	case "preset":
		return a.cmdPreset(remaining[1:])
	case "collectors":
		return a.cmdCollectors()
	case "circuit-breaker":
		return a.cmdCircuitBreaker(remaining[1:])
	case "recommend":
		return a.cmdRecommend(remaining[1:])
	default:
		return fmt.Errorf("Unknown command: %s", remaining[0])
	}
}

func (a *App) consumeGlobalFlags(args []string) []string {
	var rest []string
	for i := 0; i < len(args); i++ {
		if args[i] == "--api-base" && i+1 < len(args) {
			a.apiOverride = args[i+1]
			i++
			continue
		}
		rest = args[i:]
		break
	}
	if rest == nil {
		return []string{}
	}
	return rest
}

func (a *App) printHelp() {
	fmt.Print(`scenario-completeness-scoring CLI

Usage:
  scenario-completeness-scoring <command> [options]

Global Options:
  --api-base <url>   Override API base URL (default: auto-detected)

Commands:
  help                         Show this help message
  version                      Show CLI version
  configure [key value]        View or update local CLI settings (api_base, token)
  status                       Check API & collector health
  scores [--json]              List completeness scores for all scenarios
  score <scenario> [--json --verbose --metrics]
                               Show detailed score for a scenario
  calculate <scenario> [--source name] [--tag value] [--json]
                               Force score recalculation and save history
  history <scenario> [--limit N] [--source name] [--tag value] [--json]
                               View score history
  trends <scenario> [options]  View trend analysis for a scenario
  what-if <scenario> [--file path] [--json]
                               Run hypothetical improvement analysis
  config                       Show server scoring configuration
  config set --file path|--json '{...}'
                               Update server configuration
  presets                      List configuration presets
  preset apply <name>          Apply a configuration preset
  collectors                   Show collector health status
  circuit-breaker [reset]      View or reset the circuit breaker status
  recommend <scenario> [--json]
                               Get prioritized improvement recommendations

Environment:
  VROOLI_CLI_SOURCE_ROOT       Override the source path used for stale-build detection
  SCENARIO_COMPLETENESS_SCORING_CLI_SOURCE_ROOT
                               Legacy override for stale-build detection
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
