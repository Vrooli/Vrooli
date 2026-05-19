package main

import (
	"react-component-library/cli/domains"

	"github.com/vrooli/cli-core/cliapp"
)

const (
	appName        = "react-component-library"
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
	subcommandGroups := func(core *cliapp.ScenarioApp) []cliapp.SubcommandGroup {
		groups, err := domains.SubcommandGroups(core, manifestBytes)
		if err != nil {
			panic(err)
		}
		return groups
	}
	core, err := cliapp.NewStandardScenarioApp(cliapp.StandardScenarioOptions{
		Name:             appName,
		Version:          appVersion,
		Description:      "React Component Library CLI",
		DefaultAPIBase:   defaultAPIBase,
		ExtraAPIEnvVars:  []string{"API_BASE_URL", "VITE_API_BASE_URL"},
		BuildFingerprint: buildFingerprint,
		BuildTimestamp:   buildTimestamp,
		BuildSourceRoot:  buildSourceRoot,
		AllowAnonymous:   true,
		CommandGroups:    domains.CommandGroups,
		SubcommandGroups: subcommandGroups,
	})
	if err != nil {
		return nil, err
	}
	app.core = core
	return app, nil
}

func (a *App) Run(args []string) error {
	return a.core.CLI.Run(args)
}
