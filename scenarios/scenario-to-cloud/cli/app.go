package main

import (
	"scenario-to-cloud/cli/domains"
	"time"

	"github.com/vrooli/cli-core/cliapp"
)

const (
	appName        = "scenario-to-cloud"
	appVersion     = "0.2.0"
	defaultAPIBase = ""
)

var (
	buildFingerprint = "unknown"
	buildTimestamp   = "unknown"
	buildSourceRoot  = ""
)

// App is the main CLI application container.
type App struct {
	core *cliapp.ScenarioApp
}

// NewApp creates a new CLI application instance.
func NewApp() (*App, error) {
	core, err := cliapp.NewStandardScenarioApp(cliapp.StandardScenarioOptions{
		Name:               appName,
		Version:            appVersion,
		Description:        "scenario-to-cloud CLI",
		DefaultAPIBase:     defaultAPIBase,
		BuildFingerprint:   buildFingerprint,
		BuildTimestamp:     buildTimestamp,
		BuildSourceRoot:    buildSourceRoot,
		DefaultHTTPTimeout: 2 * time.Minute,
		AllowAnonymous:     true,
		CommandGroups:      domains.CommandGroups,
	})
	if err != nil {
		return nil, err
	}

	return &App{core: core}, nil
}

// Run executes the CLI with the given arguments.
func (a *App) Run(args []string) error {
	return a.core.CLI.Run(args)
}
