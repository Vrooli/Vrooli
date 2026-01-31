package main

import (
	"scenario-to-desktop/cli/distribution"
	"scenario-to-desktop/cli/pipeline"
	"scenario-to-desktop/cli/signing"
	"scenario-to-desktop/cli/system"
	"scenario-to-desktop/cli/telemetry"
	"time"

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
	distribution *distribution.Commands
	signing      *signing.Commands
	telemetry    *telemetry.Commands
	system       *system.Commands
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
		distribution: distribution.New(core.APIClient),
		signing:      signing.New(core.APIClient),
		telemetry:    telemetry.New(core.APIClient),
		system:       system.New(core.APIClient),
	}
	app.core.SetCommands(app.registerCommands())
	return app, nil
}

// Run executes the CLI with provided arguments.
func (a *App) Run(args []string) error {
	return a.core.CLI.Run(args)
}

func (a *App) registerCommands() []cliapp.CommandGroup {
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

	pipelineGroup := cliapp.CommandGroup{
		Title: "Pipeline",
		Commands: []cliapp.Command{
			{Name: "pipeline-run", NeedsAPI: true, Description: "Start a new pipeline: pipeline-run <scenario> [--stages ...] [--platforms ...] [--wait] [--timeout N]", Run: a.pipeline.Run},
			{Name: "pipeline-status", NeedsAPI: true, Description: "Get pipeline status: pipeline-status <id> [--verbose]", Run: a.pipeline.Status},
			{Name: "pipeline-resume", NeedsAPI: true, Description: "Resume a stopped pipeline: pipeline-resume <id>", Run: a.pipeline.Resume},
			{Name: "pipeline-cancel", NeedsAPI: true, Description: "Cancel a running pipeline: pipeline-cancel <id>", Run: a.pipeline.Cancel},
			{Name: "pipeline-list", NeedsAPI: true, Description: "List all pipelines", Run: a.pipeline.List},
			{Name: "pipeline-active", NeedsAPI: true, Description: "Get active pipeline for scenario: pipeline-active <scenario>", Run: a.pipeline.Active},
			{Name: "pipeline-create", NeedsAPI: true, Description: "Create new pipeline for scenario: pipeline-create <scenario>", Run: a.pipeline.Create},
			{Name: "pipeline-reset", NeedsAPI: true, Description: "Reset active pipeline for scenario: pipeline-reset <scenario>", Run: a.pipeline.Reset},
			{Name: "pipeline-history", NeedsAPI: true, Description: "Get pipeline history: pipeline-history <scenario> [--limit N]", Run: a.pipeline.History},
			{Name: "pipeline-start", NeedsAPI: true, Description: "Start active pipeline: pipeline-start <scenario> [--stages ...] [--platforms ...] [--wait] [--timeout N]", Run: a.pipeline.Start},
		},
	}

	telemetryGroup := cliapp.CommandGroup{
		Title: "Telemetry",
		Commands: []cliapp.Command{
			{Name: "telemetry-ingest", NeedsAPI: true, Description: "Ingest telemetry from file: telemetry-ingest <scenario> --file <path>", Run: a.telemetry.Ingest},
			{Name: "telemetry-summary", NeedsAPI: true, Description: "Get telemetry summary: telemetry-summary <scenario>", Run: a.telemetry.Summary},
			{Name: "telemetry-insights", NeedsAPI: true, Description: "Get telemetry insights: telemetry-insights <scenario>", Run: a.telemetry.Insights},
			{Name: "telemetry-tail", NeedsAPI: true, Description: "Get recent telemetry: telemetry-tail <scenario> [--limit N]", Run: a.telemetry.Tail},
			{Name: "telemetry-download", NeedsAPI: true, Description: "Download telemetry file: telemetry-download <scenario> [--output <path>]", Run: a.telemetry.Download},
			{Name: "telemetry-delete", NeedsAPI: true, Description: "Delete telemetry: telemetry-delete <scenario>", Run: a.telemetry.Delete},
		},
	}

	signingGroup := cliapp.CommandGroup{
		Title: "Code Signing",
		Commands: []cliapp.Command{
			{Name: "signing-get", NeedsAPI: true, Description: "Get signing config: signing-get <scenario>", Run: a.signing.Get},
			{Name: "signing-set", NeedsAPI: true, Description: "Set signing config: signing-set <scenario> --config <json>", Run: a.signing.Set},
			{Name: "signing-delete", NeedsAPI: true, Description: "Delete signing config: signing-delete <scenario>", Run: a.signing.Delete},
			{Name: "signing-validate", NeedsAPI: true, Description: "Validate signing config: signing-validate <scenario>", Run: a.signing.Validate},
			{Name: "signing-ready", NeedsAPI: true, Description: "Check signing readiness: signing-ready <scenario>", Run: a.signing.Ready},
			{Name: "signing-prerequisites", NeedsAPI: true, Description: "List available signing tools", Run: a.signing.Prerequisites},
			{Name: "signing-discover", NeedsAPI: true, Description: "Discover certificates: signing-discover <platform>", Run: a.signing.Discover},
			{Name: "signing-generate-key", NeedsAPI: true, Description: "Generate Linux GPG key: signing-generate-key <scenario> --name <name> --email <email>", Run: a.signing.GenerateKey},
		},
	}

	distributionGroup := cliapp.CommandGroup{
		Title: "Distribution",
		Commands: []cliapp.Command{
			{Name: "dist-targets", NeedsAPI: true, Description: "List distribution targets", Run: a.distribution.TargetsList},
			{Name: "dist-target-get", NeedsAPI: true, Description: "Get distribution target: dist-target-get <name>", Run: a.distribution.TargetGet},
			{Name: "dist-target-create", NeedsAPI: true, Description: "Create distribution target: dist-target-create --config <json>", Run: a.distribution.TargetCreate},
			{Name: "dist-target-update", NeedsAPI: true, Description: "Update distribution target: dist-target-update <name> --config <json>", Run: a.distribution.TargetUpdate},
			{Name: "dist-target-delete", NeedsAPI: true, Description: "Delete distribution target: dist-target-delete <name>", Run: a.distribution.TargetDelete},
			{Name: "dist-target-test", NeedsAPI: true, Description: "Test distribution target: dist-target-test <name>", Run: a.distribution.TargetTest},
			{Name: "dist-validate", NeedsAPI: true, Description: "Validate all distribution targets", Run: a.distribution.Validate},
			{Name: "dist-check-credentials", NeedsAPI: true, Description: "Check distribution credentials", Run: a.distribution.CheckCredentials},
			{Name: "distribute", NeedsAPI: true, Description: "Start distribution: distribute <scenario> --artifacts <paths>", Run: a.distribution.Distribute},
			{Name: "dist-status", NeedsAPI: true, Description: "Get distribution status: dist-status <id>", Run: a.distribution.Status},
			{Name: "dist-cancel", NeedsAPI: true, Description: "Cancel distribution: dist-cancel <id>", Run: a.distribution.Cancel},
			{Name: "dist-list", NeedsAPI: true, Description: "List all distributions", Run: a.distribution.List},
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

	wine := cliapp.CommandGroup{
		Title: "Wine (Windows builds on Linux)",
		Commands: []cliapp.Command{
			{Name: "wine-check", NeedsAPI: true, Description: "Check Wine installation status", Run: a.system.WineCheck},
			{Name: "wine-install", NeedsAPI: true, Description: "Install Wine: wine-install --method <flatpak|appimage>", Run: a.system.WineInstall},
			{Name: "wine-status", NeedsAPI: true, Description: "Get Wine install status: wine-status <id>", Run: a.system.WineStatus},
		},
	}

	config := cliapp.CommandGroup{
		Title: "Configuration",
		Commands: []cliapp.Command{
			a.core.ConfigureCommand([]string{"api_base"}, []string{"token", "api_token"}),
		},
	}

	return []cliapp.CommandGroup{health, templates, pipelineGroup, telemetryGroup, signingGroup, distributionGroup, records, download, wine, config}
}
