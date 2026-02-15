// Package main provides the Swarm Manager CLI.
package main

import (
	"github.com/vrooli/cli-core/cliapp"
	"github.com/vrooli/cli-core/cliutil"
)

const (
	appName        = "swarm-manager"
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
	env := cliapp.StandardScenarioEnv(appName, cliapp.ScenarioEnvOptions{
		ExtraAPIEnvVars:     []string{"API_BASE_URL", "VITE_API_BASE_URL"},
		ExtraAPIPortEnvVars: []string{"API_PORT"},
	})
	core, err := cliapp.NewScenarioApp(cliapp.ScenarioOptions{
		Name:              appName,
		Version:           appVersion,
		Description:       "Swarm Manager CLI",
		DefaultAPIBase:    defaultAPIBase,
		APIEnvVars:        env.APIEnvVars,
		APIPortEnvVars:    env.APIPortEnvVars,
		APIPortDetector:   cliutil.DetectPortFromVrooli(appName, "API_PORT"),
		ConfigDirEnvVars:  env.ConfigDirEnvVars,
		SourceRootEnvVars: env.SourceRootEnvVars,
		TokenEnvVars:      env.TokenEnvVars,
		BuildFingerprint:  buildFingerprint,
		BuildTimestamp:    buildTimestamp,
		BuildSourceRoot:   buildSourceRoot,
		AllowAnonymous:    true,
	})
	if err != nil {
		return nil, err
	}

	app := &App{core: core}
	app.core.SetCommands(app.registerCommands())
	return app, nil
}

func (a *App) Run(args []string) error {
	return a.core.CLI.Run(args)
}

func (a *App) registerCommands() []cliapp.CommandGroup {
	health := cliapp.CommandGroup{
		Title: "Health",
		Commands: []cliapp.Command{
			{Name: "status", NeedsAPI: true, Description: "Check API health", Run: a.cmdStatus},
		},
	}

	backlog := cliapp.CommandGroup{
		Title: "Backlog",
		Commands: []cliapp.Command{
			{Name: "backlog list", NeedsAPI: true, Description: "List backlog items (args: [kinds])", Run: a.cmdBacklogList},
			{Name: "backlog get", NeedsAPI: true, Description: "Get a backlog item (args: <kind> <name>)", Run: a.cmdBacklogGet},
			{Name: "backlog create", NeedsAPI: true, Description: "Create a backlog item (args: <json-or-@file>)", Run: a.cmdBacklogCreate},
			{Name: "backlog update", NeedsAPI: true, Description: "Update a backlog item (args: <kind> <name> <json-or-@file>)", Run: a.cmdBacklogUpdate},
			{Name: "backlog delete", NeedsAPI: true, Description: "Delete a backlog item (args: <kind> <name>)", Run: a.cmdBacklogDelete},
			{Name: "backlog files", NeedsAPI: true, Description: "List backlog item files (args: <kind> <name>)", Run: a.cmdBacklogFiles},
			{Name: "backlog queue", NeedsAPI: true, Description: "Queue a backlog item (args: <kind> <name> [--mode ... --delay-seconds ...])", Run: a.cmdBacklogQueue},
			{Name: "backlog research", NeedsAPI: true, Description: "Spawn research agent for a backlog item (args: <kind> <name> [json-or-@file])", Run: a.cmdBacklogResearch},
			{Name: "backlog convert", NeedsAPI: true, Description: "Convert a backlog item (args: <kind> <name> <target-kind> [target-name])", Run: a.cmdBacklogConvert},
		},
	}

	scenarios := cliapp.CommandGroup{
		Title: "Scenarios",
		Commands: []cliapp.Command{
			{Name: "scenarios list", NeedsAPI: true, Description: "List all scenarios", Run: a.cmdScenariosList},
			{Name: "scenarios get", NeedsAPI: true, Description: "Get scenario details (args: <name>)", Run: a.cmdScenariosGet},
			{Name: "scenarios update", NeedsAPI: true, Description: "Update scenario metadata (args: <name> <json-or-@file>)", Run: a.cmdScenariosUpdate},
			{Name: "scenarios delete", NeedsAPI: true, Description: "Delete a scenario (args: <name> [--archive])", Run: a.cmdScenariosDelete},
			{Name: "scenarios files", NeedsAPI: true, Description: "List scenario files (args: <name>)", Run: a.cmdScenariosFiles},
			{Name: "scenarios start", NeedsAPI: true, Description: "Start a scenario (args: <name>)", Run: a.cmdScenariosStart},
			{Name: "scenarios stop", NeedsAPI: true, Description: "Stop a scenario (args: <name>)", Run: a.cmdScenariosStop},
			{Name: "scenarios restart", NeedsAPI: true, Description: "Restart a scenario (args: <name>)", Run: a.cmdScenariosRestart},
		},
	}

	settings := cliapp.CommandGroup{
		Title: "Settings",
		Commands: []cliapp.Command{
			{Name: "settings get", NeedsAPI: true, Description: "Get current settings", Run: a.cmdSettingsGet},
			{Name: "settings update", NeedsAPI: true, Description: "Update settings (args: <json-or-@file>)", Run: a.cmdSettingsUpdate},
		},
	}

	queue := cliapp.CommandGroup{
		Title: "Queue",
		Commands: []cliapp.Command{
			{Name: "queue list", NeedsAPI: true, Description: "List queue items", Run: a.cmdQueueList},
			{Name: "queue create", NeedsAPI: true, Description: "Create a queue item (args: <kind> [payload-json-or-@file])", Run: a.cmdQueueCreate},
			{Name: "queue delete", NeedsAPI: true, Description: "Delete a queue item (args: <id>)", Run: a.cmdQueueDelete},
		},
	}

	execution := cliapp.CommandGroup{
		Title: "Execution",
		Commands: []cliapp.Command{
			{Name: "execution list", NeedsAPI: true, Description: "List execution runs", Run: a.cmdExecutionList},
			{Name: "execution get", NeedsAPI: true, Description: "Get execution details (args: <execution-id>)", Run: a.cmdExecutionGet},
			{Name: "execution policy get", NeedsAPI: true, Description: "Get execution policy defaults", Run: a.cmdExecutionPolicyGet},
			{Name: "execution policy update", NeedsAPI: true, Description: "Update execution policy defaults (flags: --mode --delay-seconds)", Run: a.cmdExecutionPolicyUpdate},
			{Name: "execution start", NeedsAPI: true, Description: "Start an execution (args: <execution-id>)", Run: a.cmdExecutionStart},
			{Name: "execution cancel", NeedsAPI: true, Description: "Cancel an execution (args: <execution-id>)", Run: a.cmdExecutionCancel},
			{Name: "execution retry", NeedsAPI: true, Description: "Retry a failed execution (args: <execution-id>)", Run: a.cmdExecutionRetry},
		},
	}

	config := cliapp.CommandGroup{
		Title: "Configuration",
		Commands: []cliapp.Command{
			a.core.ConfigureCommand([]string{"api_base"}, []string{"token", "api_token"}),
		},
	}

	return []cliapp.CommandGroup{health, backlog, scenarios, settings, queue, execution, config}
}
