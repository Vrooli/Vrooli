package main

import (
	"fmt"
	"os"
	"strings"

	"github.com/vrooli/cli-core/cliapp"
	"github.com/vrooli/cli-core/cliutil"
	"scenario-completeness-scoring/cli/format"
)

const (
	appVersion     = "0.1.0"
	defaultAPIBase = ""

	genericSourceRootEnvVar = "VROOLI_CLI_SOURCE_ROOT"
	legacySourceRootEnvVar  = "SCENARIO_COMPLETENESS_SCORING_CLI_SOURCE_ROOT"
	configDirEnvVar         = "SCENARIO_COMPLETENESS_SCORING_CONFIG_DIR"
	configDirGenericEnvVar  = "VROOLI_CLI_CONFIG_DIR"
)

var (
	buildFingerprint = "unknown"
	buildTimestamp   = "unknown"
	buildSourceRoot  = ""
)

type App struct {
	configStore *cliutil.ConfigFile
	config      cliutil.APIConfig
	apiOverride string
	client      *cliutil.HTTPClient
	api         *cliutil.APIClient
	services    *Services
	cli         *cliapp.App
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
	configStore, cfg, err := cliutil.LoadAPIConfig("scenario-completeness-scoring", configDirEnvVar, configDirGenericEnvVar)
	if err != nil {
		return nil, err
	}
	app := &App{
		configStore: configStore,
		config:      cfg,
		apiOverride: "",
		client:      cliutil.NewHTTPClient(cliutil.HTTPClientOptions{}),
	}
	app.api = cliutil.NewAPIClient(app.client, app.buildAPIBaseOptions, func() string { return app.config.Token })
	app.services = NewServices(app.api)
	app.cli = cliapp.NewApp(cliapp.AppOptions{
		Name:         "scenario-completeness-scoring",
		Version:      appVersion,
		Description:  "scenario-completeness-scoring CLI",
		Commands:     app.registerCommands(),
		APIOverride:  &app.apiOverride,
		ColorEnabled: cliapp.DefaultColorEnabled(),
		OnColor:      format.SetColorEnabled,
		StaleChecker: cliutil.NewStaleChecker("scenario-completeness-scoring", buildFingerprint, buildTimestamp, buildSourceRoot, genericSourceRootEnvVar, legacySourceRootEnvVar),
		Preflight:    app.preflightAPI,
	})
	return app, nil
}

func (a *App) saveConfig() error {
	return a.configStore.Save(a.config)
}

func (a *App) registerCommands() []cliapp.CommandGroup {
	health := cliapp.CommandGroup{
		Title: "Health",
		Commands: []cliapp.Command{
			{Name: "status", NeedsAPI: true, Description: "Check API & collector health", Run: func(args []string) error { return a.cmdStatus() }},
			{Name: "collectors", NeedsAPI: true, Description: "Show collector health status", Run: func(args []string) error { return a.cmdCollectors() }},
			{Name: "circuit-breaker", NeedsAPI: true, Description: "View or reset the circuit breaker status", Run: a.cmdCircuitBreaker},
		},
	}
	scoring := cliapp.CommandGroup{
		Title: "Scoring",
		Commands: []cliapp.Command{
			{Name: "scores", NeedsAPI: true, Description: "List completeness scores for all scenarios", Run: a.cmdScores},
			{Name: "score", NeedsAPI: true, Description: "Show detailed score for a scenario", Run: a.cmdScore},
			{Name: "calculate", NeedsAPI: true, Description: "Force score recalculation and save history", Run: a.cmdCalculate},
			{Name: "history", NeedsAPI: true, Description: "View score history", Run: a.cmdHistory},
			{Name: "trends", NeedsAPI: true, Description: "View trend analysis for a scenario", Run: a.cmdTrends},
			{Name: "what-if", NeedsAPI: true, Description: "Run hypothetical improvement analysis", Run: a.cmdWhatIf},
			{Name: "recommend", NeedsAPI: true, Description: "Get prioritized improvement recommendations", Run: a.cmdRecommend},
		},
	}
	config := cliapp.CommandGroup{
		Title: "Configuration",
		Commands: []cliapp.Command{
			{Name: "config", NeedsAPI: true, Description: "Show server scoring configuration", Run: a.cmdConfig},
			{Name: "presets", NeedsAPI: true, Description: "List configuration presets", Run: func(args []string) error { return a.cmdPresets() }},
			{Name: "preset", NeedsAPI: true, Description: "Apply a configuration preset", Run: a.cmdPreset},
			{Name: "configure", NeedsAPI: false, Description: "View or update local CLI settings (api_base, token)", Run: a.cmdConfigure},
		},
	}
	return []cliapp.CommandGroup{health, scoring, config}
}

func (a *App) Run(args []string) error {
	return a.cli.Run(args)
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

func (a *App) preflightAPI(cmd cliapp.Command, global cliapp.GlobalOptions) error {
	if !cmd.NeedsAPI {
		return nil
	}
	_, err := cliutil.ValidateAPIBase(a.buildAPIBaseOptions())
	return err
}
