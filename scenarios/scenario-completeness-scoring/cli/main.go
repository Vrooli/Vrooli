package main

import (
	"fmt"
	"os"

	"github.com/vrooli/cli-core/cliapp"
	"github.com/vrooli/cli-core/cliutil"
	"scenario-completeness-scoring/cli/format"
)

const (
	appName        = "scenario-completeness-scoring"
	appVersion     = "0.1.0"
	defaultAPIBase = ""
)

var (
	buildFingerprint = "unknown"
	buildTimestamp   = "unknown"
	buildSourceRoot  = ""
)

type App struct {
	core     *cliapp.ScenarioApp
	services *Services
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
	env := cliapp.StandardScenarioEnv(appName, cliapp.ScenarioEnvOptions{
		ExtraAPIEnvVars: []string{"SCORING_API_BASE"},
	})
	core, err := cliapp.NewScenarioApp(cliapp.ScenarioOptions{
		Name:               appName,
		Version:            appVersion,
		Description:        "scenario-completeness-scoring CLI",
		DefaultAPIBase:     defaultAPIBase,
		APIEnvVars:         env.APIEnvVars,
		APIPortEnvVars:     env.APIPortEnvVars,
		APIPortDetector:    cliutil.DetectPortFromVrooli(appName, "API_PORT"),
		ConfigDirEnvVars:   env.ConfigDirEnvVars,
		SourceRootEnvVars:  env.SourceRootEnvVars,
		TokenEnvVars:       env.TokenEnvVars,
		HTTPTimeoutEnvVars: env.HTTPTimeoutEnvVars,
		OnColor:            format.SetColorEnabled,
		BuildFingerprint:   buildFingerprint,
		BuildTimestamp:     buildTimestamp,
		BuildSourceRoot:    buildSourceRoot,
		AllowAnonymous:     true,
	})
	if err != nil {
		return nil, err
	}
	app := &App{core: core}
	app.services = NewServices(app.core.APIClient)
	app.core.SetCommands(app.registerCommands())
	return app, nil
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
			a.core.ConfigureCommand(nil, nil),
			{Name: "config", NeedsAPI: true, Description: "Show server scoring configuration", Run: a.cmdConfig},
			{Name: "presets", NeedsAPI: true, Description: "List configuration presets", Run: func(args []string) error { return a.cmdPresets() }},
			{Name: "preset", NeedsAPI: true, Description: "Apply a configuration preset", Run: a.cmdPreset},
		},
	}
	return []cliapp.CommandGroup{health, scoring, config}
}

func (a *App) Run(args []string) error {
	return a.core.CLI.Run(args)
}
