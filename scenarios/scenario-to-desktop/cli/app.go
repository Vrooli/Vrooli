package main

import (
	"time"

	"scenario-to-desktop/cli/deploytarget"
	"scenario-to-desktop/cli/pipeline"
	"scenario-to-desktop/cli/signing"
	"scenario-to-desktop/cli/system"
	"scenario-to-desktop/cli/telemetry"

	"github.com/vrooli/cli-core/cliapp"
	"github.com/vrooli/cli-core/cliutil"
)

const (
	appName        = "scenario-to-desktop"
	appVersion     = "1.0.0"
	defaultAPIBase = ""
)

var (
	buildFingerprint = "unknown"
	buildTimestamp   = "unknown"
	buildSourceRoot  = ""
)

// App wires the cli-core scaffolding to the scenario-specific commands.
type App struct {
	core         *cliapp.ScenarioApp
	pipeline     *pipeline.Commands
	signing      *signing.Commands
	telemetry    *telemetry.Commands
	system       *system.Commands
	deployTarget *deploytarget.Commands
}

// NewApp constructs the CLI application.
func NewApp() (*App, error) {
	env := cliapp.StandardScenarioEnv(appName, cliapp.ScenarioEnvOptions{
		ExtraAPIEnvVars:     []string{"API_BASE_URL"},
		ExtraAPIPortEnvVars: []string{"API_PORT"},
	})
	core, err := cliapp.NewScenarioApp(cliapp.ScenarioOptions{
		Name:              appName,
		Version:           appVersion,
		Description:       "Convert Vrooli scenarios into cross-platform desktop applications",
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
		// Longer timeout for blocking pipeline operations (default 30s is too short)
		DefaultHTTPTimeout: 15 * time.Minute,
	})
	if err != nil {
		return nil, err
	}

	app := &App{
		core:         core,
		pipeline:     pipeline.New(core.APIClient),
		signing:      signing.New(core.APIClient),
		telemetry:    telemetry.New(core.APIClient),
		system:       system.New(core.APIClient),
		deployTarget: deploytarget.New(core.APIClient),
	}
	app.core.SetCommandsWithSubgroups(app.registerCommands(), app.registerSubcommandGroups())
	return app, nil
}

// Run executes the CLI with provided arguments.
func (a *App) Run(args []string) error {
	return a.core.CLI.Run(args)
}

func (a *App) registerCommands() []cliapp.CommandGroup {
	// Flat commands for simple operations
	health := cliapp.CommandGroup{
		Title: "Health & Status",
		Commands: []cliapp.Command{
			{Name: "status", NeedsAPI: true, Description: "Check API health and system status", Run: a.system.Status},
		},
	}

	templates := cliapp.CommandGroup{
		Title: "Templates",
		Commands: []cliapp.Command{
			{Name: "templates", NeedsAPI: true, Description: "List available desktop templates", Run: a.system.TemplatesList},
			{Name: "template", NeedsAPI: true, Description: "Get template details: template <type>", Run: a.system.TemplateGet},
		},
	}

	records := cliapp.CommandGroup{
		Title: "Desktop Records",
		Commands: []cliapp.Command{
			{Name: "records", NeedsAPI: true, Description: "List desktop generation records", Run: a.system.RecordsList},
			{Name: "records-move", NeedsAPI: true, Description: "Move desktop wrapper: records-move <id> [--target <path>]", Run: a.system.RecordsMove},
			{Name: "records-delete", NeedsAPI: true, Description: "Delete desktop app: records-delete <scenario>", Run: a.system.RecordsDelete},
		},
	}

	download := cliapp.CommandGroup{
		Title: "Download",
		Commands: []cliapp.Command{
			{Name: "download", NeedsAPI: true, Description: "Download built package: download <scenario> <platform> [--output <path>]", Run: a.system.Download},
		},
	}

	scenarios := cliapp.CommandGroup{
		Title: "Scenarios",
		Commands: []cliapp.Command{
			{Name: "desktop-status", NeedsAPI: true, Description: "List desktop build status and artifacts", Run: a.system.DesktopStatus},
		},
	}

	config := cliapp.CommandGroup{
		Title: "Configuration",
		Commands: []cliapp.Command{
			a.core.ConfigureCommand([]string{"api_base"}, []string{"token", "api_token"}),
		},
	}

	return []cliapp.CommandGroup{health, templates, records, download, scenarios, config}
}

func (a *App) registerSubcommandGroups() []cliapp.SubcommandGroup {
	// Pipeline subcommands
	pipelineGroup := cliapp.SubcommandGroup{
		Name:        "pipeline",
		Description: "Build pipeline operations (run 'pipeline help' for details)",
		NeedsAPI:    true,
		Subcommands: []cliapp.Command{
			{Name: "run", Description: "Start a new pipeline: run <scenario> [--stages ...] [--platforms ...] [--wait]", Run: a.pipeline.Run},
			{Name: "status", Description: "Get pipeline status: status <id> [--verbose]", Run: a.pipeline.Status},
			{Name: "resume", Description: "Resume a stopped pipeline: resume <id>", Run: a.pipeline.Resume},
			{Name: "cancel", Description: "Cancel a running pipeline: cancel <id>", Run: a.pipeline.Cancel},
			{Name: "list", Description: "List all pipelines", Run: a.pipeline.List},
			{Name: "active", Description: "Get active pipeline for scenario: active <scenario>", Run: a.pipeline.Active},
			{Name: "create", Description: "Create new pipeline for scenario: create <scenario>", Run: a.pipeline.Create},
			{Name: "reset", Description: "Reset active pipeline for scenario: reset <scenario>", Run: a.pipeline.Reset},
			{Name: "history", Description: "Get pipeline history: history <scenario> [--limit N]", Run: a.pipeline.History},
			{Name: "start", Description: "Start active pipeline: start <scenario> [--stages ...] [--platforms ...] [--wait]", Run: a.pipeline.Start},
		},
	}

	// Telemetry subcommands
	telemetryGroup := cliapp.SubcommandGroup{
		Name:        "telemetry",
		Description: "Deployment telemetry (run 'telemetry help' for details)",
		NeedsAPI:    true,
		Subcommands: []cliapp.Command{
			{Name: "ingest", Description: "Ingest telemetry from file: ingest <scenario> --file <path>", Run: a.telemetry.Ingest},
			{Name: "summary", Description: "Get telemetry summary: summary <scenario>", Run: a.telemetry.Summary},
			{Name: "insights", Description: "Get AI-generated insights: insights <scenario>", Run: a.telemetry.Insights},
			{Name: "tail", Description: "Get recent telemetry: tail <scenario> [--limit N]", Run: a.telemetry.Tail},
			{Name: "download", Description: "Download telemetry file: download <scenario> [--output <path>]", Run: a.telemetry.Download},
			{Name: "delete", Description: "Delete telemetry: delete <scenario>", Run: a.telemetry.Delete},
		},
	}

	// Signing subcommands
	signingGroup := cliapp.SubcommandGroup{
		Name:        "signing",
		Description: "Code signing configuration (run 'signing help' for details)",
		NeedsAPI:    true,
		Subcommands: []cliapp.Command{
			{Name: "get", Description: "Get signing config: get <scenario>", Run: a.signing.Get},
			{Name: "set", Description: "Set signing config: set <scenario> --config <json>", Run: a.signing.Set},
			{Name: "delete", Description: "Delete signing config: delete <scenario>", Run: a.signing.Delete},
			{Name: "validate", Description: "Validate signing config: validate <scenario>", Run: a.signing.Validate},
			{Name: "ready", Description: "Check signing readiness: ready <scenario>", Run: a.signing.Ready},
			{Name: "prerequisites", Description: "List available signing tools", Run: a.signing.Prerequisites},
			{Name: "discover", Description: "Discover certificates: discover <platform>", Run: a.signing.Discover},
			{Name: "generate-key", Description: "Generate Linux GPG key: generate-key <scenario> --name <name> --email <email>", Run: a.signing.GenerateKey},
		},
	}

	// Wine subcommands
	wineGroup := cliapp.SubcommandGroup{
		Name:        "wine",
		Description: "Wine for Windows builds on Linux (run 'wine help' for details)",
		NeedsAPI:    true,
		Subcommands: []cliapp.Command{
			{Name: "check", Description: "Check Wine installation status", Run: a.system.WineCheck},
			{Name: "install", Description: "Install Wine: install --method <flatpak|appimage>", Run: a.system.WineInstall},
			{Name: "status", Description: "Get Wine install status: status <id>", Run: a.system.WineStatus},
		},
	}

	// Deploy target subcommands
	deployTargetGroup := cliapp.SubcommandGroup{
		Name:        "deploy-target",
		Description: "Manage LPBS deploy targets (run 'deploy-target help' for details)",
		NeedsAPI:    true,
		Subcommands: []cliapp.Command{
			{Name: "list", Description: "List saved deploy targets", Run: a.deployTarget.List},
			{Name: "add", Description: "Add/update deploy target: add <name> --scenario <s> --profile <p> [--label <l>]", Run: a.deployTarget.Add},
			{Name: "remove", Description: "Remove deploy target: remove <name>", Run: a.deployTarget.Remove},
			{Name: "test", Description: "Test deploy target session: test <name> [--require-service-auth]", Run: a.deployTarget.Test},
			{Name: "doctor", Description: "Diagnose deploy target readiness: doctor <name>", Run: a.deployTarget.Doctor},
		},
	}

	return []cliapp.SubcommandGroup{pipelineGroup, telemetryGroup, signingGroup, deployTargetGroup, wineGroup}
}
