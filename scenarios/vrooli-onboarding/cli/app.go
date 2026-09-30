package main

import (
	"vrooli-onboarding/cli/domains"

	"github.com/vrooli/cli-core/cliapp"
)

const (
	appName        = "vrooli-onboarding"
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
	commandGroups := func(core *cliapp.ScenarioApp) []cliapp.CommandGroup {
		groups, _, err := domains.ManifestCommandGroups(core, manifestBytes)
		if err != nil {
			panic(err)
		}
		return groups
	}
	subcommandGroups := func(core *cliapp.ScenarioApp) []cliapp.SubcommandGroup {
		_, groups, err := domains.ManifestCommandGroups(core, manifestBytes)
		if err != nil {
			panic(err)
		}
		return groups
	}
	core, err := cliapp.NewStandardScenarioApp(cliapp.StandardScenarioOptions{
		Name:             appName,
		Version:          appVersion,
		Description:      "Vrooli Onboarding CLI",
		APIPrefix:        "/api",
		DefaultAPIBase:   defaultAPIBase,
		ExtraAPIEnvVars:  []string{"API_BASE_URL", "VITE_API_BASE_URL"},
		ExtraTokenEnvVars: []string{"VROOLI_AUTH_LOCAL_TOKEN"},
		BuildFingerprint: buildFingerprint,
		BuildTimestamp:   buildTimestamp,
		BuildSourceRoot:  buildSourceRoot,
		AllowAnonymous:   true,
		CommandGroups:    commandGroups,
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
