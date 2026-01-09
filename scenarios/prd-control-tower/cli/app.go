package main

import (
	"time"

	"github.com/vrooli/cli-core/cliapp"
	"github.com/vrooli/cli-core/cliutil"
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
	env := cliapp.StandardScenarioEnv(appName, cliapp.ScenarioEnvOptions{})
	core, err := cliapp.NewScenarioApp(cliapp.ScenarioOptions{
		Name:               appName,
		Version:            appVersion,
		Description:        "PRD Control Tower CLI",
		DefaultAPIBase:     defaultAPIBase,
		DefaultHTTPTimeout: 300 * time.Second, // AI generation can take several minutes
		APIEnvVars:         env.APIEnvVars,
		APIPortEnvVars:     env.APIPortEnvVars,
		APIPortDetector:    cliutil.DetectPortFromVrooli(appName, "API_PORT"),
		ConfigDirEnvVars:   env.ConfigDirEnvVars,
		SourceRootEnvVars:  env.SourceRootEnvVars,
		TokenEnvVars:       env.TokenEnvVars,
		HTTPTimeoutEnvVars: env.HTTPTimeoutEnvVars,
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
	})
	if err != nil {
		return nil, err
	}

	app := &App{core: core}
	app.services = NewServices(app.core.APIClient)
	app.core.SetCommands(app.registerCommands())
	return app, nil
}

func (a *App) registerCommands() []cliapp.CommandGroup {
	health := cliapp.CommandGroup{
		Title: "Health",
		Commands: []cliapp.Command{
			{Name: "status", NeedsAPI: true, Description: "Check API health", Run: func(args []string) error { return a.cmdStatus(args) }},
		},
	}
	drafts := cliapp.CommandGroup{
		Title: "Drafts",
		Commands: []cliapp.Command{
			{Name: "list-drafts", NeedsAPI: true, Description: "List PRD drafts", Run: a.cmdListDrafts},
		},
	}
	prds := cliapp.CommandGroup{
		Title: "PRDs",
		Commands: []cliapp.Command{
			{Name: "prd", NeedsAPI: true, Description: "PRD management (generate, validate, fix)", Run: a.cmdPRD},
		},
	}
	requirements := cliapp.CommandGroup{
		Title: "Requirements",
		Commands: []cliapp.Command{
			{Name: "requirements", NeedsAPI: true, Description: "Requirements management (generate, validate)", Run: a.cmdRequirements},
		},
	}
	config := cliapp.CommandGroup{
		Title: "Configuration",
		Commands: []cliapp.Command{
			a.core.ConfigureCommand(nil, nil),
		},
	}
	return []cliapp.CommandGroup{health, drafts, prds, requirements, config}
}

func (a *App) Run(args []string) error {
	return a.core.CLI.Run(args)
}
