package main

import (
	"github.com/vrooli/cli-core/cliapp"

	"ecosystem-manager/cli/domains"
	"ecosystem-manager/cli/internal/appctx"
)

const (
	appName        = "ecosystem-manager"
	appVersion     = "1.0.0"
	defaultAPIBase = ""
)

var (
	buildFingerprint = "unknown"
	buildTimestamp   = "unknown"
	buildSourceRoot  = ""
)

// App wraps the shared scenario app scaffold for ecosystem-manager.
type App struct {
	core *cliapp.ScenarioApp
}

// NewApp creates a new ecosystem-manager CLI application.
func NewApp() (*App, error) {
	app := &App{}
	core, err := cliapp.NewStandardScenarioApp(cliapp.StandardScenarioOptions{
		Name:             appName,
		Version:          appVersion,
		Description:      "Ecosystem Manager CLI - manage tasks, queue, and auto-steer profiles",
		DefaultAPIBase:   defaultAPIBase,
		ExtraAPIEnvVars:  []string{"API_BASE_URL", "VITE_API_BASE_URL"},
		APIPrefix:        "/api",
		BuildFingerprint: buildFingerprint,
		BuildTimestamp:   buildTimestamp,
		BuildSourceRoot:  buildSourceRoot,
		AllowAnonymous:   true,
		CommandGroups: func(core *cliapp.ScenarioApp) []cliapp.CommandGroup {
			app.core = core
			return domains.CommandGroups(appctx.New(core))
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
