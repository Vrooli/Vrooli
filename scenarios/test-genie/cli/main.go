package main

import (
	"fmt"
	"os"

	"github.com/vrooli/cli-core/cliapp"
	"github.com/vrooli/cli-core/cliutil"
)

const (
	appName    = "test-genie"
	appVersion = "0.2.0"

	defaultAPIBase = ""
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

func main() {
	app, err := NewApp()
	if err != nil {
		fmt.Fprintf(os.Stderr, "Error: %v\n", err)
		os.Exit(1)
	}
	if err := app.Run(os.Args[1:]); err != nil {
		fmt.Fprintf(os.Stderr, "Error: %v\n", err)
		os.Exit(1)
	}
}

func NewApp() (*App, error) {
	env := cliapp.StandardScenarioEnv(appName, cliapp.ScenarioEnvOptions{})
	core, err := cliapp.NewScenarioApp(cliapp.ScenarioOptions{
		Name:               appName,
		Version:            appVersion,
		Description:        "Test Genie CLI",
		DefaultAPIBase:     defaultAPIBase,
		APIEnvVars:         env.APIEnvVars,
		APIPortEnvVars:     env.APIPortEnvVars,
		APIPortDetector:    cliutil.DetectPortFromVrooli(appName, "API_PORT"),
		ConfigDirEnvVars:   env.ConfigDirEnvVars,
		SourceRootEnvVars:  env.SourceRootEnvVars,
		TokenEnvVars:       env.TokenEnvVars,
		HTTPTimeoutEnvVars: env.HTTPTimeoutEnvVars,
		TokenKeys:          []string{"token", "api_token"},
		BuildFingerprint:   buildFingerprint,
		BuildTimestamp:     buildTimestamp,
		BuildSourceRoot:    buildSourceRoot,
	})
	if err != nil {
		return nil, err
	}
	app := &App{core: core}
	app.services = NewServices(app.core.APIClient)
	app.core.SetCommands(app.registerCommands())
	return app, nil
}

func (a *App) Run(args []string) error {
	return a.core.CLI.Run(args)
}

func (a *App) registerCommands() []cliapp.CommandGroup {
	exec := cliapp.CommandGroup{
		Title: "Suites",
		Commands: []cliapp.Command{
			{Name: "generate", NeedsAPI: true, Description: "Queue suite generation", Run: a.cmdGenerate},
			{Name: "execute", NeedsAPI: true, Description: "Execute a suite for a scenario", Run: a.cmdExecute},
		},
	}
	local := cliapp.CommandGroup{
		Title: "Local",
		Commands: []cliapp.Command{
			{Name: "run-tests", NeedsAPI: true, Description: "Trigger scenario-local test runner", Run: a.cmdRunTests},
		},
	}
	system := cliapp.CommandGroup{
		Title: "System",
		Commands: []cliapp.Command{
			{Name: "status", NeedsAPI: true, Description: "Check Test Genie health", Run: func(args []string) error { return a.cmdStatus() }},
		},
	}
	config := cliapp.CommandGroup{
		Title: "Configuration",
		Commands: []cliapp.Command{
			a.core.ConfigureCommand([]string{"api_base"}, []string{"token", "api_token"}),
		},
	}
	return []cliapp.CommandGroup{exec, local, system, config}
}
