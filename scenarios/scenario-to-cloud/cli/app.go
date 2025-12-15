package main

import (
	"time"

	"github.com/vrooli/cli-core/cliapp"
	"github.com/vrooli/cli-core/cliutil"
)

const (
	appName        = "scenario-to-cloud"
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
	api  *cliutil.APIClient
}

func NewApp() (*App, error) {
	env := cliapp.StandardScenarioEnv(appName, cliapp.ScenarioEnvOptions{})
	core, err := cliapp.NewScenarioApp(cliapp.ScenarioOptions{
		Name:               appName,
		Version:            appVersion,
		Description:        "scenario-to-cloud CLI",
		DefaultAPIBase:     defaultAPIBase,
		APIEnvVars:         env.APIEnvVars,
		APIPortEnvVars:     env.APIPortEnvVars,
		APIPortDetector:    cliutil.DetectPortFromVrooli(appName, "API_PORT"),
		ConfigDirEnvVars:   env.ConfigDirEnvVars,
		SourceRootEnvVars:  env.SourceRootEnvVars,
		TokenEnvVars:       env.TokenEnvVars,
		HTTPTimeoutEnvVars: env.HTTPTimeoutEnvVars,
		BuildFingerprint:   buildFingerprint,
		BuildTimestamp:     buildTimestamp,
		BuildSourceRoot:    buildSourceRoot,
		DefaultHTTPTimeout: 2 * time.Minute,
		AllowAnonymous:     true,
	})
	if err != nil {
		return nil, err
	}

	app := &App{
		core: core,
		api:  core.APIClient,
	}
	app.core.SetCommands(app.registerCommands())
	return app, nil
}

func (a *App) registerCommands() []cliapp.CommandGroup {
	health := cliapp.CommandGroup{
		Title: "Health",
		Commands: []cliapp.Command{
			{Name: "status", NeedsAPI: true, Description: "Check API and dependency health", Run: a.cmdStatus},
		},
	}

	deploy := cliapp.CommandGroup{
		Title: "Deploy",
		Commands: []cliapp.Command{
			{Name: "manifest-validate", NeedsAPI: true, Description: "Validate a cloud manifest JSON file", Run: a.cmdManifestValidate},
			{Name: "plan", NeedsAPI: true, Description: "Generate a deploy plan from a cloud manifest JSON file", Run: a.cmdPlan},
		},
	}

	config := cliapp.CommandGroup{
		Title: "Configuration",
		Commands: []cliapp.Command{
			a.core.ConfigureCommand([]string{"api_base"}, []string{"token", "api_token"}),
		},
	}

	return []cliapp.CommandGroup{health, deploy, config}
}

func (a *App) Run(args []string) error {
	return a.core.CLI.Run(args)
}
