package main

import (
	"time"

	"prd-control-tower/cli/domains"
	"prd-control-tower/cli/internal/support"

	"github.com/vrooli/cli-core/cliapp"
)

const (
	appName        = "prd-control-tower"
	appVersion     = "1.0.0"
	defaultAPIBase = "http://localhost:18600"
)

var (
	buildFingerprint = "unknown"
	buildTimestamp   = "unknown"
	buildSourceRoot  = ""
)

type App struct {
	core     *cliapp.ScenarioApp
	services *Services
}

func NewApp() (*App, error) {
	app := &App{}
	core, err := cliapp.NewStandardScenarioApp(cliapp.StandardScenarioOptions{
		Name:               appName,
		Version:            appVersion,
		Description:        "PRD Control Tower CLI",
		DefaultAPIBase:     defaultAPIBase,
		DefaultHTTPTimeout: 300 * time.Second, // AI generation can take several minutes
		Preflight: func(cmd cliapp.Command, global cliapp.GlobalOptions, app *cliapp.ScenarioApp) error {
			if !cmd.NeedsAPI {
				return nil
			}
			return ensureScenarioAPIReady(app, global, appName)
		},
		BuildFingerprint: buildFingerprint,
		BuildTimestamp:   buildTimestamp,
		BuildSourceRoot:  buildSourceRoot,
		AllowAnonymous:   true,
		CommandGroups: func(core *cliapp.ScenarioApp) []cliapp.CommandGroup {
			app.core = core
			return app.customCommandGroups()
		},
	})
	if err != nil {
		return nil, err
	}

	app.core = core
	app.services = NewServices(app.core.APIClient)
	return app, nil
}

func (a *App) customCommandGroups() []cliapp.CommandGroup {
	return domains.CommandGroups(a.dependencies())
}

func (a *App) commandGroups() []cliapp.CommandGroup {
	return append(a.core.StandardBaseCommandGroups(cliapp.StandardBaseCommandOptions{}), a.customCommandGroups()...)
}

func (a *App) Run(args []string) error {
	return a.core.CLI.Run(args)
}

func (a *App) dependencies() support.Dependencies {
	return support.Dependencies{
		ListDrafts:   a.cmdListDrafts,
		PRD:          a.cmdPRD,
		Requirements: a.cmdRequirements,
	}
}
