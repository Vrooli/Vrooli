package main

import (
	"test-genie/cli/domains"
	"test-genie/cli/execute"
	"test-genie/cli/generate"
	"test-genie/cli/internal/deps"
	"test-genie/cli/playbooksseed"
	"test-genie/cli/runlocal"
	"test-genie/cli/status"
	"test-genie/cli/uismoke"

	"github.com/vrooli/cli-core/cliapp"
)

const (
	appName    = "test-genie"
	appVersion = "0.3.0"

	defaultAPIBase = ""
)

var (
	buildFingerprint = "unknown"
	buildTimestamp   = "unknown"
	buildSourceRoot  = ""
)

func boolPtr(v bool) *bool { return &v }

// App is the main application container.
type App struct {
	core *cliapp.ScenarioApp
}

// NewApp creates a new CLI application instance.
func NewApp() (*App, error) {
	app := &App{}
	core, err := cliapp.NewStandardScenarioApp(cliapp.StandardScenarioOptions{
		Name:                 appName,
		Version:              appVersion,
		Description:          "Test Genie CLI",
		DefaultAPIBase:       defaultAPIBase,
		BuildFingerprint:     buildFingerprint,
		BuildTimestamp:       buildTimestamp,
		BuildSourceRoot:      buildSourceRoot,
		FreshnessInputs:      []string{"api/**", "cli/**", ".vrooli/service.json", ".vrooli/testing.json"},
		AllowAnonymous:       true,
		IncludeStatusCommand: boolPtr(true),
		CommandGroups: func(core *cliapp.ScenarioApp) []cliapp.CommandGroup {
			app.core = core
			return domains.CommandGroups(deps.Runtime{
				Generate:   generate.NewClient(core.APIClient),
				Execute:    execute.NewClient(core.APIClient, core.HTTPClient),
				RunLocal:   runlocal.NewClient(core.APIClient),
				UISmoke:    uismoke.NewClient(core.APIClient),
				Seed:       playbooksseed.NewClient(core.APIClient),
				Status:     status.NewClient(core.APIClient),
				HTTPClient: core.HTTPClient,
			})
		},
	})
	if err != nil {
		return nil, err
	}

	app.core = core
	return app, nil
}

// Run executes the CLI with the given arguments.
func (a *App) Run(args []string) error {
	return a.core.CLI.Run(args)
}
