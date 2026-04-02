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
	core      *cliapp.ScenarioApp
	globalDry bool // set by preflight from --dry-run global flag
}

func NewApp() (*App, error) {
	env := cliapp.StandardScenarioEnv(appName, cliapp.ScenarioEnvOptions{
		ExtraAPIEnvVars: []string{"API_BASE_URL", "VITE_API_BASE_URL"},
	})
	app := &App{}
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
		Preflight: func(_ cliapp.Command, global cliapp.GlobalOptions, _ *cliapp.ScenarioApp) error {
			app.globalDry = global.DryRun
			return nil
		},
	})
	if err != nil {
		return nil, err
	}

	app.core = core
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
			{Name: "overview", NeedsAPI: true, Description: "Full backlog situational awareness [--format json|markdown]", Run: a.cmdOverview},
		},
	}

	config := cliapp.CommandGroup{
		Title: "Configuration",
		Commands: []cliapp.Command{
			a.core.ConfigureCommand([]string{"api_base"}, []string{"token", "api_token"}),
		},
	}

	migration := cliapp.CommandGroup{
		Title: "Migration",
		Commands: []cliapp.Command{
			{Name: "migrate-workshop", Description: "Migrate backlog items from clarify/suggest/enhance to workshop/plan.md [--dry-run] [--root PATH]", Run: a.cmdMigrateWorkshop},
		},
	}

	return []cliapp.CommandGroup{health, config, migration}
}

func (a *App) registerSubcommandGroups() []cliapp.SubcommandGroup {
	return []cliapp.SubcommandGroup{
		{
			Name:        "backlog",
			Description: "Backlog item management",
			Subcommands: []cliapp.Command{
				{Name: "list", NeedsAPI: true, Description: "List backlog items [--kind KIND]", Run: a.cmdBacklogList},
				{Name: "get", NeedsAPI: true, Description: "Get full backlog item details (--kind KIND --name NAME)", Run: a.cmdBacklogGet},
				{Name: "create", NeedsAPI: true, Description: "Create a backlog item (--data JSON)", Run: a.cmdBacklogCreate},
				{Name: "update", NeedsAPI: true, Description: "Update a backlog item (--kind KIND --name NAME --data JSON)", Run: a.cmdBacklogUpdate},
				{Name: "delete", NeedsAPI: true, Description: "Delete a backlog item (--kind KIND --name NAME)", Run: a.cmdBacklogDelete},
				{Name: "workshop-reset", NeedsAPI: true, Description: "Reset all workshop data for a backlog item (--kind KIND --name NAME)", Run: a.cmdBacklogWorkshopReset},
				{Name: "files", NeedsAPI: true, Description: "List backlog item files (--kind KIND --name NAME)", Run: a.cmdBacklogFiles},
				{Name: "file-get", NeedsAPI: true, Description: "Get a file from a backlog item (--kind KIND --name NAME --path PATH)", Run: a.cmdBacklogFileGet},
				{Name: "file-upload", NeedsAPI: true, Description: "Upload a file to a backlog item (--kind KIND --name NAME --path PATH --file FILE|--content CONTENT)", Run: a.cmdBacklogFileUpload},
				{Name: "process-preflight", NeedsAPI: true, Description: "Check processing readiness (--kind KIND --name NAME)", Run: a.cmdBacklogProcessPreflight},
				{Name: "queue", NeedsAPI: true, Description: "Preview/queue a backlog item (--kind KIND --name NAME [--execute] [--force])", Run: a.cmdBacklogQueue},
				{Name: "research", NeedsAPI: true, Description: "Spawn research agent (--kind KIND --name NAME [--data JSON])", Run: a.cmdBacklogResearch},
				{Name: "prompt-trace", NeedsAPI: true, Description: "Get latest backlog research prompt trace (--kind KIND --name NAME)", Run: a.cmdBacklogPromptTrace},
				{Name: "batch-create", NeedsAPI: true, Description: "Batch create backlog items from a plan file (--file items.json [--preview])", Run: a.cmdBacklogBatchCreate},
				{Name: "batch-queue", NeedsAPI: true, Description: "Batch queue backlog items (--items kind/name,kind/name [--execute] [--force] [--mode MODE])", Run: a.cmdBacklogBatchQueue},
				{Name: "export", NeedsAPI: true, Description: "Export backlog items to markdown for offline editing", Run: a.cmdBacklogExport},
				{Name: "import", NeedsAPI: true, Description: "Import edited markdown back into the backlog (--file FILE)", Run: a.cmdBacklogImport},
				{Name: "clarify", NeedsAPI: true, Description: "Start a decision clarification (--kind KIND --name NAME --round N --item ID [--message MSG])", Run: a.cmdBacklogClarify},
				{Name: "clarify-get", NeedsAPI: true, Description: "Get a clarification thread (--kind KIND --name NAME --thread ID)", Run: a.cmdBacklogClarifyGet},
				{Name: "clarify-continue", NeedsAPI: true, Description: "Continue a clarification thread (--kind KIND --name NAME --thread ID --message MSG)", Run: a.cmdBacklogClarifyContinue},
				{Name: "clarify-action", NeedsAPI: true, Description: "Apply post-clarification action (--kind KIND --name NAME --thread ID --action ACTION)", Run: a.cmdBacklogClarifyAction},
			},
		},
		{
			Name:        "scenarios",
			Description: "Scenario catalog and lifecycle",
			Subcommands: []cliapp.Command{
				{Name: "list", NeedsAPI: true, Description: "List scenarios [--search, --status, --tags, --sort, --order]", Run: a.cmdScenariosList},
				{Name: "get", NeedsAPI: true, Description: "Get scenario details (--name NAME)", Run: a.cmdScenariosGet},
				{Name: "update", NeedsAPI: true, Description: "Update scenario metadata (--name NAME --data JSON)", Run: a.cmdScenariosUpdate},
				{Name: "delete", NeedsAPI: true, Description: "Delete a scenario (--name NAME [--archive])", Run: a.cmdScenariosDelete},
				{Name: "files", NeedsAPI: true, Description: "List scenario files (--name NAME)", Run: a.cmdScenariosFiles},
				{Name: "spec-sync-archive", NeedsAPI: true, Description: "Queue spec-sync-archive execution (--name NAME)", Run: a.cmdScenariosSpecSyncArchive},
				{Name: "start", NeedsAPI: true, Description: "Start a scenario (--name NAME)", Run: a.cmdScenariosStart},
				{Name: "stop", NeedsAPI: true, Description: "Stop a scenario (--name NAME)", Run: a.cmdScenariosStop},
				{Name: "restart", NeedsAPI: true, Description: "Restart a scenario (--name NAME)", Run: a.cmdScenariosRestart},
			},
		},
		{
			Name:        "settings",
			Description: "Scenario settings",
			Subcommands: []cliapp.Command{
				{Name: "get", NeedsAPI: true, Description: "Get current settings", Run: a.cmdSettingsGet},
				{Name: "update", NeedsAPI: true, Description: "Update settings (--data JSON)", Run: a.cmdSettingsUpdate},
			},
		},
		{
			Name:        "queue",
			Description: "Execution queue operations",
			Subcommands: []cliapp.Command{
				{Name: "list", NeedsAPI: true, Description: "List queue items", Run: a.cmdQueueList},
				{Name: "create", NeedsAPI: true, Description: "Create a queue item (--kind KIND [--data JSON])", Run: a.cmdQueueCreate},
				{Name: "delete", NeedsAPI: true, Description: "Delete a queue item (--id ID)", Run: a.cmdQueueDelete},
			},
		},
		{
			Name:        "execution",
			Description: "Execution run controls",
			Subcommands: []cliapp.Command{
				{Name: "list", NeedsAPI: true, Description: "List execution runs", Run: a.cmdExecutionList},
				{Name: "get", NeedsAPI: true, Description: "Get execution details (--id ID)", Run: a.cmdExecutionGet},
				{Name: "create", NeedsAPI: true, Description: "Create execution from backlog item (--kind KIND --name NAME)", Run: a.cmdExecutionCreate},
				{Name: "policy-get", NeedsAPI: true, Description: "Get execution policy defaults (via settings)", Run: a.cmdExecutionPolicyGet},
				{Name: "policy-update", NeedsAPI: true, Description: "Update execution policy defaults (--mode MODE --delay-seconds N [--auto-fixup] [--max-fixup-attempts N])", Run: a.cmdExecutionPolicyUpdate},
				{Name: "prompt-trace", NeedsAPI: true, Description: "Get execution prompt trace (--id ID)", Run: a.cmdExecutionPromptTrace},
				{Name: "start", NeedsAPI: true, Description: "Start an execution (--id ID)", Run: a.cmdExecutionStart},
				{Name: "cancel", NeedsAPI: true, Description: "Cancel an execution (--id ID)", Run: a.cmdExecutionCancel},
				{Name: "retry", NeedsAPI: true, Description: "Retry a failed execution (--id ID)", Run: a.cmdExecutionRetry},
			},
		},
		{
			Name:        "prompts",
			Description: "Prompt catalog and skill operations",
			Subcommands: []cliapp.Command{
				{Name: "catalog", NeedsAPI: true, Description: "List the swarm-manager prompt catalog", Run: a.cmdPromptsCatalog},
				{Name: "skills", NeedsAPI: true, Description: "List prompt skills used by swarm-manager", Run: a.cmdPromptsSkills},
				{Name: "skill-get", NeedsAPI: true, Description: "Get prompt skill details (--id ID)", Run: a.cmdPromptsSkillGet},
				{Name: "skill-update", NeedsAPI: true, Description: "Update prompt skill fields (--id ID --data JSON)", Run: a.cmdPromptsSkillUpdate},
				{Name: "skill-versions", NeedsAPI: true, Description: "Get prompt skill version history (--id ID)", Run: a.cmdPromptsSkillVersions},
				{Name: "skill-revert", NeedsAPI: true, Description: "Revert prompt skill to version (--id ID --version VERSION)", Run: a.cmdPromptsSkillRevert},
				{Name: "preview", NeedsAPI: true, Description: "Render a skill prompt with variables (--id ID)", Run: a.cmdPromptsPreview},
				{Name: "simulate", NeedsAPI: true, Description: "Simulate selected prompt for a workload kind (--kind KIND)", Run: a.cmdPromptsSimulate},
			},
		},
		{
			Name:        "initiatives",
			Description: "Initiative management",
			Subcommands: []cliapp.Command{
				{Name: "list", NeedsAPI: true, Description: "List initiatives [--json]", Run: a.cmdInitiativesList},
				{Name: "get", NeedsAPI: true, Description: "Get initiative details (--name NAME) [--json]", Run: a.cmdInitiativesGet},
				{Name: "create", NeedsAPI: true, Description: "Create initiative (--data JSON) [--json]", Run: a.cmdInitiativesCreate},
				{Name: "update", NeedsAPI: true, Description: "Update initiative (--name NAME --data JSON) [--json]", Run: a.cmdInitiativesUpdate},
				{Name: "delete", NeedsAPI: true, Description: "Delete initiative (--name NAME)", Run: a.cmdInitiativesDelete},
				{Name: "add-items", NeedsAPI: true, Description: "Add items to initiative (--name NAME --items kind/name,...) [--json]", Run: a.cmdInitiativesAddItems},
				{Name: "remove-items", NeedsAPI: true, Description: "Remove items from initiative (--name NAME --items kind/name,...) [--json]", Run: a.cmdInitiativesRemoveItems},
				{Name: "files", NeedsAPI: true, Description: "List files in an initiative (--name NAME) [--json]", Run: a.cmdInitiativesFiles},
				{Name: "file-get", NeedsAPI: true, Description: "Get a file from an initiative (--name NAME --path PATH) [--out FILE] [--json]", Run: a.cmdInitiativesFileGet},
				{Name: "file-upload", NeedsAPI: true, Description: "Upload a file to an initiative (--name NAME --path PATH) (--stdin|--file|--content)", Run: a.cmdInitiativesFileUpload},
				{Name: "file-op", NeedsAPI: true, Description: "File operation on initiative (--name NAME --op OP --source PATH) [--dest PATH] [--json]", Run: a.cmdInitiativesFileOp},
			},
		},
		{
			Name:        "captures",
			Description: "Quick-capture management",
			Subcommands: []cliapp.Command{
				{Name: "list", NeedsAPI: true, Description: "List captures [--json]", Run: a.cmdCapturesList},
				{Name: "create", NeedsAPI: true, Description: "Create a capture (--text TEXT [--file FILE]...) [--json]", Run: a.cmdCapturesCreate},
				{Name: "get", NeedsAPI: true, Description: "Get capture details (--id ID) [--json]", Run: a.cmdCapturesGet},
				{Name: "delete", NeedsAPI: true, Description: "Delete a capture (--id ID)", Run: a.cmdCapturesDelete},
				{Name: "classify", NeedsAPI: true, Description: "Trigger classification for a capture (--id ID) [--json]", Run: a.cmdCapturesClassify},
			},
		},
		{
			Name:        "agent-manager",
			Description: "Agent-manager integration",
			Subcommands: []cliapp.Command{
				{Name: "status", NeedsAPI: true, Description: "Get agent-manager availability and profile status", Run: a.cmdAgentManagerStatus},
				{Name: "run-get", NeedsAPI: true, Description: "Get run status (--id ID) [--json]", Run: a.cmdAgentManagerRunGet},
				{Name: "run-stop", NeedsAPI: true, Description: "Stop a run (--id ID) [--json]", Run: a.cmdAgentManagerRunStop},
			},
		},
		{
			Name:        "stats",
			Description: "Event-driven analytics and metrics",
			Subcommands: []cliapp.Command{
				{Name: "summary", NeedsAPI: true, Description: "Full stats dashboard", Run: a.cmdStatsSummary},
				{Name: "throughput", NeedsAPI: true, Description: "Throughput metrics", Run: a.cmdStatsThroughput},
				{Name: "blocking", NeedsAPI: true, Description: "Blocking analysis", Run: a.cmdStatsBlocking},
				{Name: "initiatives", NeedsAPI: true, Description: "Initiative health", Run: a.cmdStatsInitiatives},
				{Name: "agent", NeedsAPI: true, Description: "Agent efficiency metrics", Run: a.cmdStatsAgent},
			},
		},
	}
}
