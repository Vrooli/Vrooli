package main

import (
	"fmt"
	"plan-manager/cli/domains"

	"github.com/vrooli/cli-core/cliapp"
)

const (
	appName        = "plan-manager"
	appVersion     = "0.1.0"
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

func NewApp() (*App, error) {
	app := &App{}
	// Manifest parse / binding wiring is a programmer error caught at NewApp
	// time. It must still fail loudly on the first CLI invocation, but a panic
	// in a library entrypoint gives the caller a stack trace instead of a
	// message and denies it any chance to report cleanly. The builder callback
	// cannot return an error, so capture it and propagate through NewApp, which
	// already has an error return.
	var subcommandGroupErr error
	subcommandGroups := func(core *cliapp.ScenarioApp) []cliapp.SubcommandGroup {
		groups, err := domains.SubcommandGroups(core, manifestBytes)
		if err != nil {
			subcommandGroupErr = err
			return nil
		}
		return groups
	}
	core, err := cliapp.NewStandardScenarioApp(cliapp.StandardScenarioOptions{
		Name:               appName,
		Version:            appVersion,
		Description:        "Plan Manager CLI",
		DefaultAPIBase:     defaultAPIBase,
		ExtraAPIEnvVars:    []string{"API_BASE_URL", "VITE_API_BASE_URL"},
		BuildFingerprint:   buildFingerprint,
		BuildTimestamp:     buildTimestamp,
		BuildSourceRoot:    buildSourceRoot,
		AllowAnonymous:     true,
		CommandGroups:      domains.CommandGroups,
		SubcommandGroups:   subcommandGroups,
		UnknownCommandHint: planManagerCommandHint,
	})
	if err != nil {
		return nil, err
	}
	if subcommandGroupErr != nil {
		return nil, fmt.Errorf("build plan-manager subcommand groups: %w", subcommandGroupErr)
	}
	app.core = core
	return app, nil
}

func (a *App) Run(args []string) error {
	return a.core.CLI.Run(args)
}
