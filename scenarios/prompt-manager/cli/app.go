package main

import (
	"github.com/vrooli/cli-core/cliapp"

	"prompt-manager/cli/domains"
	"prompt-manager/cli/internal/appctx"
)

const (
	appName        = "prompt-manager"
	appVersion     = "2.0.0"
	defaultAPIBase = ""
)

var (
	buildFingerprint = "unknown"
	buildTimestamp   = "unknown"
	buildSourceRoot  = ""
)

// App wraps the cli-core ScenarioApp with prompt-manager specific functionality.
type App struct {
	core *cliapp.ScenarioApp
	ctx  appctx.Runtime
}

// NewApp creates a new prompt-manager CLI application.
func NewApp() (*App, error) {
	app := &App{}
	core, err := cliapp.NewStandardScenarioApp(cliapp.StandardScenarioOptions{
		Name:             appName,
		Version:          appVersion,
		Description:      "Personal Prompt Manager CLI - manage skills, tags, members, and more",
		DefaultAPIBase:   defaultAPIBase,
		ExtraAPIEnvVars:  []string{"API_BASE_URL", "VITE_API_BASE_URL"},
		BuildFingerprint: buildFingerprint,
		BuildTimestamp:   buildTimestamp,
		BuildSourceRoot:  buildSourceRoot,
		AllowAnonymous:   true,
		CommandGroups: func(core *cliapp.ScenarioApp) []cliapp.CommandGroup {
			app.core = core
			app.ctx = appctx.Runtime{Core: core}
			return domains.CommandGroups(app.ctx)
		},
	})
	if err != nil {
		return nil, err
	}
	app.core = core
	app.ctx = appctx.Runtime{Core: core}
	return app, nil
}

// Run executes the CLI with the given arguments.
func (a *App) Run(args []string) error {
	return a.core.CLI.Run(args)
}
