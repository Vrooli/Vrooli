// Package pipeline provides CLI commands for pipeline management.
package pipeline

import (
	"scenario-to-desktop/cli/internal/support"

	"github.com/vrooli/cli-core/cliapp"
)

// Commands provides pipeline CLI commands.
type Commands struct {
	rpc pipelineRPC
}

// New creates a new pipeline Commands instance.
func New(deps support.Dependencies) *Commands {
	return &Commands{rpc: newPipelineRPC(deps.ScenarioApp())}
}

func Register(deps support.Dependencies) cliapp.SubcommandGroup {
	cmds := New(deps)
	return cliapp.SubcommandGroup{
		Name:        "pipeline",
		Description: "Build pipeline operations (run 'pipeline help' for details)",
		NeedsAPI:    true,
		Subcommands: []cliapp.Command{
			(cliapp.Command{Name: "run", Description: "Start a new pipeline: run <scenario> [--stages ...] [--platforms ...]", Args: pipelineRunArgs()}).WithPrimitive(cmds.runPrimitive()),
			(cliapp.Command{Name: "status", Description: "Get pipeline status: status <id>", Args: pipelineIDArgs()}).WithPrimitive(cmds.statusPrimitive()),
			(cliapp.Command{Name: "resume", Description: "Resume a stopped pipeline: resume <id>", Args: pipelineIDArgs()}).WithPrimitive(cmds.resumePrimitive()),
			(cliapp.Command{Name: "cancel", Description: "Cancel a running pipeline: cancel <id>", Args: pipelineIDArgs()}).WithPrimitive(cmds.cancelPrimitive()),
			(cliapp.Command{Name: "list", Description: "List all pipelines"}).WithPrimitive(cmds.listPrimitive()),
			(cliapp.Command{Name: "active", Description: "Get active pipeline for scenario: active <scenario>", Args: pipelineScenarioArgs()}).WithPrimitive(cmds.activePrimitive()),
			(cliapp.Command{Name: "create", Description: "Create new pipeline for scenario: create <scenario>", Args: pipelineScenarioArgs()}).WithPrimitive(cmds.createPrimitive()),
			(cliapp.Command{Name: "reset", Description: "Reset active pipeline for scenario: reset <scenario>", Args: pipelineScenarioArgs()}).WithPrimitive(cmds.resetPrimitive()),
			(cliapp.Command{Name: "history", Description: "Get pipeline history: history <scenario> [--limit N]", Args: pipelineScenarioArgs(cliapp.Flag{Name: "limit", Default: "10", Description: "Maximum history entries"})}).WithPrimitive(cmds.historyPrimitive()),
			(cliapp.Command{Name: "start", Description: "Start active pipeline: start <scenario> [--stages ...] [--platforms ...]", Args: pipelineRunArgs()}).WithPrimitive(cmds.startPrimitive()),
			(cliapp.Command{Name: "gate", Description: "Show approval gate status: gate <id>", Args: pipelineIDArgs()}).WithPrimitive(cmds.gatePrimitive()),
		},
	}
}

func pipelineIDArgs() cliapp.ArgSchema {
	return cliapp.ArgSchema{Positionals: []cliapp.Positional{{Name: "pipeline-id", Required: true, Description: "Pipeline identifier"}}}
}

func pipelineScenarioArgs(flags ...cliapp.Flag) cliapp.ArgSchema {
	return cliapp.ArgSchema{Positionals: []cliapp.Positional{{Name: "scenario", Required: true, Description: "Scenario to package"}}, Flags: flags}
}

func pipelineRunArgs() cliapp.ArgSchema {
	return pipelineScenarioArgs(
		cliapp.Flag{Name: "stages", Description: "Comma-separated stages: bundle,preflight,generate,build,smoketest,deploy"},
		cliapp.Flag{Name: "platforms", Description: "Comma-separated platforms: win,mac,linux"},
		cliapp.Flag{Name: "deployment-mode", Default: "bundled", Values: []string{"bundled", "proxy"}, Description: "Deployment mode"},
		cliapp.Flag{Name: "location-mode", Values: []string{"proper", "staging", "temp"}, Description: "Output location"},
		cliapp.Flag{Name: "resource-artifact-root", Description: "Verified signed resource-artifact directory"},
		cliapp.Flag{Name: "tool-artifact-root", Description: "Verified signed vendored tool-artifact directory"},
		cliapp.Flag{Name: "artifact-trust-mode", Values: []string{"development-local", "production"}, Description: "Required with resource-artifact-root; development-local is non-promotable, production requires release-manifest signature"},
		cliapp.Flag{Name: "update-provider", Values: []string{"generic", "none"}, Description: "Auto-update provider embedded in the generated app"},
		cliapp.Flag{Name: "update-url", Description: "Generic update-feed base URL; HTTP is limited to development-local evidence"},
		cliapp.Flag{Name: "update-channel", Description: "Auto-update channel (default: stable)"},
		cliapp.Flag{Name: "update-auto-check", Bool: true, Description: "Check the configured update feed when the packaged app starts"},
		cliapp.Flag{Name: "clean", Bool: true, Description: "Clean existing desktop output first"},
		cliapp.Flag{Name: "version", Description: "Run-only version override"},
	)
}
