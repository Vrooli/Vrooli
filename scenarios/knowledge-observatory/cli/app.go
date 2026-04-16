package main

import (
	"knowledge-observatory/cli/domains"
	"knowledge-observatory/cli/internal/support"

	"github.com/vrooli/cli-core/cliapp"
)

const (
	appName        = "knowledge-observatory"
	appVersion     = "2.0.0"
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
	core, err := cliapp.NewStandardScenarioApp(cliapp.StandardScenarioOptions{
		Name:             appName,
		Version:          appVersion,
		Description:      "Knowledge Observatory CLI - search, ingest, and graph knowledge",
		DefaultAPIBase:   defaultAPIBase,
		ExtraAPIEnvVars:  []string{"API_BASE_URL", "VITE_API_BASE_URL"},
		BuildFingerprint: buildFingerprint,
		BuildTimestamp:   buildTimestamp,
		BuildSourceRoot:  buildSourceRoot,
		AllowAnonymous:   true,
		CommandGroups: func(core *cliapp.ScenarioApp) []cliapp.CommandGroup {
			app.core = core
			return domains.CommandGroups(app.dependencies())
		},
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

func (a *App) dependencies() support.Dependencies {
	return support.Dependencies{
		Core: func() *cliapp.ScenarioApp {
			return a.core
		},
	}
}

func (a *App) commandGroups() []cliapp.CommandGroup {
	return domains.CommandGroups(a.dependencies())
}
