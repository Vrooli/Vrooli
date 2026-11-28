package main

import (
	"sync"

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
	commandGroups, subcommandGroups := cachedManifestGroupCallbacks(func(core *cliapp.ScenarioApp) ([]cliapp.CommandGroup, []cliapp.SubcommandGroup, error) {
		return domains.ManifestCommandGroups(core, manifestBytes)
	})
	core, err := cliapp.NewStandardScenarioApp(cliapp.StandardScenarioOptions{
		Name:              appName,
		Version:           appVersion,
		Description:       "Vrooli Onboarding CLI",
		APIPrefix:         "/api",
		DefaultAPIBase:    defaultAPIBase,
		ExtraAPIEnvVars:   []string{"API_BASE_URL", "VITE_API_BASE_URL"},
		ExtraTokenEnvVars: []string{"VROOLI_AUTH_LOCAL_TOKEN"},
		BuildFingerprint:  buildFingerprint,
		BuildTimestamp:    buildTimestamp,
		BuildSourceRoot:   buildSourceRoot,
		AllowAnonymous:    true,
		CommandGroups:     commandGroups,
		SubcommandGroups:  subcommandGroups,
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

type manifestGroupLoader func(*cliapp.ScenarioApp) ([]cliapp.CommandGroup, []cliapp.SubcommandGroup, error)

func cachedManifestGroupCallbacks(load manifestGroupLoader) (func(*cliapp.ScenarioApp) []cliapp.CommandGroup, func(*cliapp.ScenarioApp) []cliapp.SubcommandGroup) {
	var once sync.Once
	var commandGroups []cliapp.CommandGroup
	var subcommandGroups []cliapp.SubcommandGroup
	var loadErr error
	ensureLoaded := func(core *cliapp.ScenarioApp) {
		once.Do(func() {
			commandGroups, subcommandGroups, loadErr = load(core)
		})
		if loadErr != nil {
			panic(loadErr)
		}
	}
	return func(core *cliapp.ScenarioApp) []cliapp.CommandGroup {
			ensureLoaded(core)
			return commandGroups
		}, func(core *cliapp.ScenarioApp) []cliapp.SubcommandGroup {
			ensureLoaded(core)
			return subcommandGroups
		}
}
