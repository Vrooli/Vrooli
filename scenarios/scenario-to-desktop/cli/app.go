package main

import (
	"time"

	"scenario-to-desktop/cli/domains"
	"scenario-to-desktop/cli/internal/support"

	"github.com/vrooli/cli-core/cliapp"
)

const (
	appName        = "scenario-to-desktop"
	appVersion     = "1.0.0"
	defaultAPIBase = ""
)

var (
	buildFingerprint = "unknown"
	buildTimestamp   = "unknown"
	buildSourceRoot  = ""
)

type App struct {
	core *cliapp.ScenarioApp
}

func boolPtr(v bool) *bool { return &v }

func NewApp() (*App, error) {
	app := &App{}
	core, err := cliapp.NewStandardScenarioApp(cliapp.StandardScenarioOptions{
		Name:                 appName,
		Version:              appVersion,
		Description:          "Convert Vrooli scenarios into cross-platform desktop applications",
		DefaultAPIBase:       defaultAPIBase,
		ExtraAPIEnvVars:      []string{"API_BASE_URL"},
		BuildFingerprint:     buildFingerprint,
		BuildTimestamp:       buildTimestamp,
		BuildSourceRoot:      buildSourceRoot,
		AllowAnonymous:       true,
		DefaultHTTPTimeout:   15 * time.Minute,
		IncludeStatusCommand: boolPtr(false),
		CommandGroups: func(core *cliapp.ScenarioApp) []cliapp.CommandGroup {
			app.core = core
			return domains.CommandGroups(app.dependencies())
		},
		SubcommandGroups: func(core *cliapp.ScenarioApp) []cliapp.SubcommandGroup {
			app.core = core
			return domains.SubcommandGroups(app.dependencies())
		},
	})
	if err != nil {
		return nil, err
	}
	app.core = core
	return app, nil
}

// Run executes the CLI with provided arguments.
func (a *App) Run(args []string) error {
	return a.core.CLI.Run(args)
}

func (a *App) dependencies() support.Dependencies {
	return support.Dependencies{
		Core: func() *cliapp.ScenarioApp {
			return a.core
		},
	}
}
