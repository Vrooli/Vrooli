package main

import (
	"github.com/vrooli/cli-core/cliapp"
	"github.com/vrooli/cli-core/cliutil"

	"test-genie/cli/execute"
	"test-genie/cli/generate"
	"test-genie/cli/runlocal"
	"test-genie/cli/status"
)

const (
	appName    = "test-genie"
	appVersion = "0.3.0"

	defaultAPIBase = ""
)

var (
	buildFingerprint = "unknown"
	buildTimestamp   = "unknown"
	buildSourceRoot  = ""
)

// App is the main application container.
type App struct {
	core *cliapp.ScenarioApp

	// Domain clients
	statusClient   *status.Client
	generateClient *generate.Client
	executeClient  *execute.Client
	runlocalClient *runlocal.Client
}

// NewApp creates a new CLI application instance.
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
		AllowAnonymous:     true,
	})
	if err != nil {
		return nil, err
	}

	app := &App{
		core:           core,
		statusClient:   status.NewClient(core.APIClient),
		generateClient: generate.NewClient(core.APIClient),
		executeClient:  execute.NewClient(core.APIClient, core.HTTPClient),
		runlocalClient: runlocal.NewClient(core.APIClient),
	}
	app.core.SetCommands(app.registerCommands())
	return app, nil
}

// Run executes the CLI with the given arguments.
func (a *App) Run(args []string) error {
	return a.core.CLI.Run(args)
}

// HTTPClient returns the underlying HTTP client for advanced use.
func (a *App) HTTPClient() *cliutil.HTTPClient {
	return a.core.HTTPClient
}

func (a *App) registerCommands() []cliapp.CommandGroup {
	suites := cliapp.CommandGroup{
		Title: "Suites",
		Commands: []cliapp.Command{
			{
				Name:        "generate",
				NeedsAPI:    true,
				Description: "Queue suite generation",
				Run:         func(args []string) error { return generate.Run(a.generateClient, args) },
			},
			{
				Name:        "execute",
				NeedsAPI:    true,
				Description: "Execute a suite for a scenario",
				Run:         func(args []string) error { return execute.Run(a.executeClient, a.HTTPClient(), args) },
			},
		},
	}
	local := cliapp.CommandGroup{
		Title: "Local",
		Commands: []cliapp.Command{
			{
				Name:        "run-tests",
				NeedsAPI:    true,
				Description: "Trigger scenario-local test runner",
				Run:         func(args []string) error { return runlocal.Run(a.runlocalClient, args) },
			},
		},
	}
	system := cliapp.CommandGroup{
		Title: "System",
		Commands: []cliapp.Command{
			{
				Name:        "status",
				NeedsAPI:    true,
				Description: "Check Test Genie health",
				Run:         func(args []string) error { return status.Run(a.statusClient) },
			},
		},
	}
	config := cliapp.CommandGroup{
		Title: "Configuration",
		Commands: []cliapp.Command{
			a.core.ConfigureCommand([]string{"api_base"}, []string{"token", "api_token"}),
		},
	}
	return []cliapp.CommandGroup{suites, local, system, config}
}
