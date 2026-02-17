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
	app.core.SetCommandsWithSubgroups(app.registerCommands(), app.registerSubcommandGroups())
	return app, nil
}

func (a *App) Run(args []string) error {
	return a.core.CLI.Run(args)
}

func (a *App) registerCommands() []cliapp.CommandGroup {
	health := cliapp.CommandGroup{
		Title: "Health",
		Commands: []cliapp.Command{
			{Name: "status", Aliases: []string{"health"}, NeedsAPI: true, Description: "Check API health and readiness", Run: a.cmdStatus},
		},
	}

	config := cliapp.CommandGroup{
		Title: "Configuration",
		Commands: []cliapp.Command{
			a.core.ConfigureCommand([]string{"api_base"}, []string{"token", "api_token"}),
		},
	}

	return []cliapp.CommandGroup{health, config}
}

func (a *App) registerSubcommandGroups() []cliapp.SubcommandGroup {
	return []cliapp.SubcommandGroup{
		{
			Name:        "backlog",
			Description: "Backlog item management",
			Subcommands: []cliapp.Command{
				{Name: "list", NeedsAPI: true, Description: "List backlog items (use backlog get <kind> <name> for full details)", Run: a.cmdBacklogList},
				{Name: "get", NeedsAPI: true, Description: "Get full backlog item details (args: <kind> <name>)", Run: a.cmdBacklogGet},
				{Name: "create", NeedsAPI: true, Description: "Create a backlog item (args: <json-or-@file>)", Run: a.cmdBacklogCreate},
				{Name: "update", NeedsAPI: true, Description: "Update a backlog item (args: <kind> <name> <json-or-@file>)", Run: a.cmdBacklogUpdate},
				{Name: "delete", NeedsAPI: true, Description: "Delete a backlog item (args: <kind> <name>)", Run: a.cmdBacklogDelete},
				{Name: "files", NeedsAPI: true, Description: "List backlog item files (args: <kind> <name>)", Run: a.cmdBacklogFiles},
				{Name: "file", NeedsAPI: true, Description: "Backlog file subcommands (args: <get|upload> ...)", Run: a.cmdBacklogFile},
				{Name: "file-get", NeedsAPI: true, Description: "Get a file from a backlog item (args: <kind> <name> <path>)", Run: a.cmdBacklogFileGet},
				{Name: "file-upload", NeedsAPI: true, Description: "Upload a file to a backlog item (args: <kind> <name> <local-file>)", Run: a.cmdBacklogFileUpload},
				{Name: "queue", NeedsAPI: true, Description: "Queue a backlog item for execution (args: <kind> <name>)", Run: a.cmdBacklogQueue},
				{Name: "research", NeedsAPI: true, Description: "Spawn research agent (args: <kind> <name> [json-or-@file])", Run: a.cmdBacklogResearch},
				{Name: "prompt-trace", NeedsAPI: true, Description: "Get latest backlog research prompt trace (args: <kind> <name>)", Run: a.cmdBacklogPromptTrace},
				{Name: "convert", NeedsAPI: true, Description: "Convert backlog item kind (args: <kind> <name> <target-kind> [target-name])", Run: a.cmdBacklogConvert},
			},
		},
		{
			Name:        "scenarios",
			Description: "Scenario catalog and lifecycle",
			Subcommands: []cliapp.Command{
				{Name: "list", NeedsAPI: true, Description: "List scenarios (use scenarios get <name> for full details)", Run: a.cmdScenariosList},
				{Name: "get", NeedsAPI: true, Description: "Get scenario details (args: <name>)", Run: a.cmdScenariosGet},
				{Name: "update", NeedsAPI: true, Description: "Update scenario metadata (args: <name> <json-or-@file>)", Run: a.cmdScenariosUpdate},
				{Name: "delete", NeedsAPI: true, Description: "Delete a scenario (args: <name> [--archive])", Run: a.cmdScenariosDelete},
				{Name: "files", NeedsAPI: true, Description: "List scenario files (args: <name>)", Run: a.cmdScenariosFiles},
				{Name: "spec-sync-archive", NeedsAPI: true, Description: "Queue spec-sync-archive execution (args: <name>)", Run: a.cmdScenariosSpecSyncArchive},
				{Name: "start", NeedsAPI: true, Description: "Start a scenario (args: <name>)", Run: a.cmdScenariosStart},
				{Name: "stop", NeedsAPI: true, Description: "Stop a scenario (args: <name>)", Run: a.cmdScenariosStop},
				{Name: "restart", NeedsAPI: true, Description: "Restart a scenario (args: <name>)", Run: a.cmdScenariosRestart},
			},
		},
		{
			Name:        "settings",
			Description: "Scenario settings",
			Subcommands: []cliapp.Command{
				{Name: "get", NeedsAPI: true, Description: "Get current settings", Run: a.cmdSettingsGet},
				{Name: "update", NeedsAPI: true, Description: "Update settings (args: <json-or-@file>)", Run: a.cmdSettingsUpdate},
			},
		},
		{
			Name:        "queue",
			Description: "Execution queue operations",
			Subcommands: []cliapp.Command{
				{Name: "list", NeedsAPI: true, Description: "List queue items (use queue delete <id> to remove)", Run: a.cmdQueueList},
				{Name: "create", NeedsAPI: true, Description: "Create a queue item (args: <kind> [payload-json-or-@file])", Run: a.cmdQueueCreate},
				{Name: "delete", NeedsAPI: true, Description: "Delete a queue item (args: <id>)", Run: a.cmdQueueDelete},
			},
		},
		{
			Name:        "execution",
			Description: "Execution run controls",
			Subcommands: []cliapp.Command{
				{Name: "list", NeedsAPI: true, Description: "List execution runs (use execution get <execution-id> for full details)", Run: a.cmdExecutionList},
				{Name: "get", NeedsAPI: true, Description: "Get execution details (args: <execution-id>)", Run: a.cmdExecutionGet},
				{Name: "create", NeedsAPI: true, Description: "Create execution from backlog item", Run: a.cmdExecutionCreate},
				{Name: "policy-get", NeedsAPI: true, Description: "Get execution policy defaults", Run: a.cmdExecutionPolicyGet},
				{Name: "policy-update", NeedsAPI: true, Description: "Update execution policy defaults (flags: --mode --delay-seconds)", Run: a.cmdExecutionPolicyUpdate},
				{Name: "prompt-trace", NeedsAPI: true, Description: "Get execution prompt trace (args: <execution-id>)", Run: a.cmdExecutionPromptTrace},
				{Name: "start", NeedsAPI: true, Description: "Start an execution (args: <execution-id>)", Run: a.cmdExecutionStart},
				{Name: "cancel", NeedsAPI: true, Description: "Cancel an execution (args: <execution-id>)", Run: a.cmdExecutionCancel},
				{Name: "retry", NeedsAPI: true, Description: "Retry a failed execution (args: <execution-id>)", Run: a.cmdExecutionRetry},
			},
		},
		{
			Name:        "prompts",
			Description: "Prompt bindings and skill operations",
			Subcommands: []cliapp.Command{
				{Name: "map", NeedsAPI: true, Description: "List prompt trigger-to-skill bindings", Run: a.cmdPromptsMap},
				{Name: "skills", NeedsAPI: true, Description: "List prompt skills used by swarm-manager", Run: a.cmdPromptsSkills},
				{Name: "skill-get", NeedsAPI: true, Description: "Get prompt skill details (args: <skill-id>)", Run: a.cmdPromptsSkillGet},
				{Name: "skill-update", NeedsAPI: true, Description: "Update prompt skill fields (args: <skill-id> <json-or-@file>)", Run: a.cmdPromptsSkillUpdate},
				{Name: "skill-versions", NeedsAPI: true, Description: "Get prompt skill version history (args: <skill-id>)", Run: a.cmdPromptsSkillVersions},
				{Name: "skill-revert", NeedsAPI: true, Description: "Revert prompt skill to version (args: <skill-id> <version>)", Run: a.cmdPromptsSkillRevert},
				{Name: "preview", NeedsAPI: true, Description: "Render a skill prompt with variables (args: <skill-id>)", Run: a.cmdPromptsPreview},
				{Name: "simulate", NeedsAPI: true, Description: "Simulate selected prompt for a workload kind (args: <kind>)", Run: a.cmdPromptsSimulate},
			},
		},
		{
			Name:        "agent-manager",
			Description: "Agent-manager integration status",
			Subcommands: []cliapp.Command{
				{Name: "status", NeedsAPI: true, Description: "Get agent-manager availability and profile status", Run: a.cmdAgentManagerStatus},
			},
		},
	}
}
