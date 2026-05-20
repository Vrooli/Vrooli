package topcli

import (
	"io"

	"github.com/vrooli/vrooli/internal/cli/commandtree"
)

type CommandID string

const (
	CommandSetup        CommandID = "setup"
	CommandDevelop      CommandID = "develop"
	CommandBuild        CommandID = "build"
	CommandClean        CommandID = "clean"
	CommandStatus       CommandID = "status"
	CommandStop         CommandID = "stop"
	CommandBackup       CommandID = "backup"
	CommandRestore      CommandID = "restore"
	CommandScenario     CommandID = "scenario"
	CommandPackage      CommandID = "package"
	CommandResource     CommandID = "resource"
	CommandRuntime      CommandID = "runtime"
	CommandCleanup      CommandID = "cleanup"
	CommandDoctor       CommandID = "doctor"
	CommandOrphans      CommandID = "orphans"
	CommandLocks        CommandID = "locks"
	CommandDiagnosePort CommandID = "diagnose-port"
	CommandContract     CommandID = "contract"
	CommandPlans        CommandID = "plans"
	CommandHygiene      CommandID = "hygiene"
	CommandSharedDrift  CommandID = "check-shared-drift"
	CommandLifecycle    CommandID = "lifecycle"
	CommandAuth         CommandID = "auth"
)

func CommandSpecs() []commandtree.Spec[CommandID] {
	return []commandtree.Spec[CommandID]{
		{Name: string(CommandSetup), Group: "Lifecycle Commands", Summary: "Initialize the development environment", Handler: CommandSetup, Suggestable: true, RootPolicy: commandtree.RootPolicy{RequiresRoot: true, CanRunWithoutRoot: HelpOnlyWithoutRoot}},
		{Name: string(CommandDevelop), Group: "Lifecycle Commands", Summary: "Start development servers", Handler: CommandDevelop, Suggestable: true, RootPolicy: commandtree.RootPolicy{RequiresRoot: true, CanRunWithoutRoot: HelpOnlyWithoutRoot}},
		{Name: string(CommandBuild), Group: "Lifecycle Commands", Summary: "Build project-level binaries", Handler: CommandBuild, Suggestable: true, RootPolicy: commandtree.RootPolicy{RequiresRoot: true, CanRunWithoutRoot: HelpOnlyWithoutRoot}},
		{Name: string(CommandClean), Group: "Lifecycle Commands", Summary: "Clean build artifacts", Handler: CommandClean, Suggestable: true, RootPolicy: commandtree.RootPolicy{RequiresRoot: true, CanRunWithoutRoot: HelpOnlyWithoutRoot}},
		{Name: string(CommandStatus), Group: "Lifecycle Commands", Summary: "Show system health and status overview", Handler: CommandStatus, Suggestable: true, RootPolicy: commandtree.RootPolicy{RequiresRoot: true, CanRunWithoutRoot: HelpOnlyWithoutRoot}},
		{Name: string(CommandStop), Group: "Lifecycle Commands", Summary: "Stop all or specific components", Handler: CommandStop, Suggestable: true, RootPolicy: commandtree.RootPolicy{RequiresRoot: true, CanRunWithoutRoot: HelpOnlyWithoutRoot}},
		{Name: string(CommandBackup), Group: "Lifecycle Commands", Summary: "Run the project backup lifecycle when defined", Handler: CommandBackup, Suggestable: true, RootPolicy: commandtree.RootPolicy{RequiresRoot: true, CanRunWithoutRoot: HelpOnlyWithoutRoot}},
		{Name: string(CommandRestore), Group: "Lifecycle Commands", Summary: "Run the project restore lifecycle when defined", Handler: CommandRestore, Suggestable: true, RootPolicy: commandtree.RootPolicy{RequiresRoot: true, CanRunWithoutRoot: HelpOnlyWithoutRoot}},
		{Name: string(CommandScenario), Group: "Scenario Management", Summary: "Manage scenarios from their source locations", Handler: CommandScenario, Suggestable: true, RootPolicy: commandtree.RootPolicy{RequiresRoot: true, CanRunWithoutRoot: ScenarioCanRunWithoutRoot}},
		{Name: string(CommandPackage), Group: "Package Governance", Summary: "Manage governed shared packages", Handler: CommandPackage, Suggestable: true, RootPolicy: commandtree.RootPolicy{RequiresRoot: true, CanRunWithoutRoot: ListOrHelpWithoutRoot}},
		{Name: string(CommandResource), Group: "Resource Management", Summary: "Manage local resources and dependency services", Handler: CommandResource, Suggestable: true, RootPolicy: commandtree.RootPolicy{RequiresRoot: true, CanRunWithoutRoot: ListOrHelpWithoutRoot}},
		{Name: string(CommandRuntime), Group: "Runtime Management", Summary: "Manage the scenario runtime supervisor", Handler: CommandRuntime, Suggestable: true, RootPolicy: commandtree.RootPolicy{RequiresRoot: true, CanRunWithoutRoot: ListOrHelpWithoutRoot}},
		{Name: string(CommandCleanup), Group: "Maintenance Commands", Summary: "Clean up orphans, stale registry claims, and legacy lock artifacts", Handler: CommandCleanup, Suggestable: true, RootPolicy: commandtree.RootPolicy{RequiresRoot: true, CanRunWithoutRoot: ListOrHelpWithoutRoot}},
		{Name: string(CommandDoctor), Group: "Maintenance Commands", Summary: "Run environment and tool diagnostics", Handler: CommandDoctor, Suggestable: true, RootPolicy: commandtree.RootPolicy{RequiresRoot: true, CanRunWithoutRoot: HelpOnlyWithoutRoot}},
		{Name: string(CommandOrphans), Group: "Maintenance Commands", Summary: "Inspect or clean orphaned Vrooli processes", Handler: CommandOrphans, Suggestable: true, RootPolicy: commandtree.RootPolicy{RequiresRoot: true, CanRunWithoutRoot: HelpOnlyWithoutRoot}},
		{Name: string(CommandLocks), Group: "Maintenance Commands", Summary: "Inspect runtime registry claims and legacy lock artifacts", Handler: CommandLocks, Suggestable: true, RootPolicy: commandtree.RootPolicy{RequiresRoot: true, CanRunWithoutRoot: HelpOnlyWithoutRoot}},
		{Name: string(CommandDiagnosePort), Group: "Maintenance Commands", Summary: "Diagnose port conflicts using registry claims and listener evidence", Handler: CommandDiagnosePort, Suggestable: true, RootPolicy: commandtree.RootPolicy{RequiresRoot: true, CanRunWithoutRoot: HelpOnlyWithoutRoot}},
		{Name: string(CommandContract), Group: "Maintenance Commands", Summary: "Inspect and validate the repository contract", Handler: CommandContract, Suggestable: true},
		{Name: string(CommandPlans), Group: "Maintenance Commands", Summary: "Manage user-scoped implementation plans", Handler: CommandPlans, Suggestable: true},
		{Name: string(CommandHygiene), Group: "Maintenance Commands", Summary: "Run repository hygiene checks", Handler: CommandHygiene, Suggestable: true},
		{Name: string(CommandSharedDrift), Group: "Maintenance Commands", Summary: "Check dependent scenarios for stale shared-package state", Handler: CommandSharedDrift, Suggestable: true},
		{Name: string(CommandAuth), Group: "Maintenance Commands", Summary: "Report sign-in state for host tools (buf, future: claude/codex/gh/...)", Handler: CommandAuth, Suggestable: true},
		{Name: string(CommandLifecycle), Group: "Maintenance Commands", Summary: "Internal lifecycle command plumbing", Handler: CommandLifecycle, Hidden: true, Suggestable: false, RootPolicy: commandtree.RootPolicy{RequiresRoot: true, CanRunWithoutRoot: HelpOnlyWithoutRoot}},
	}
}

func HelpOnlyWithoutRoot(args []string) bool {
	return commandtree.WantsHelp(args)
}

func ListOrHelpWithoutRoot(args []string) bool {
	return len(args) == 0 || commandtree.WantsHelp(args)
}

func ScenarioCanRunWithoutRoot(args []string) bool {
	return len(args) == 0 || commandtree.WantsHelp(args)
}

func RenderMainHelp(w io.Writer, specs []commandtree.Spec[CommandID]) {
	commandtree.RenderHelp(w, commandtree.Help{
		Title: "                          ___\n" +
			" _   _ _ __ ___   ___    / (_)\n" +
			"| | | | '__/ _ \\\\ / _ \\\\  / /| |\n" +
			"| |_| | | | (_) | (_) |/ / | |\n" +
			" \\___/|_|  \\___/ \\___//_/  |_|\n" +
			"                                   ",
		Description:  "Vrooli CLI - AI Platform Management Tool",
		Usage:        "vrooli <command> [options]",
		Options:      GlobalOptions(),
		Examples:     []string{"vrooli <command> --help"},
		Notes:        []string{"Documentation: docs/"},
		DefaultGroup: "",
	}, specs)
}

func GlobalOptions() []commandtree.OptionArg {
	return []commandtree.OptionArg{
		{Name: "--help", Aliases: []string{"-h"}, Description: "Show help for a command"},
		{Name: "--version", Aliases: []string{"-v"}, Description: "Show version information"},
		{Name: "--json", Description: "Emit JSON output when supported by the selected command"},
		{Name: "--verbose", Description: "Enable verbose command output"},
		{Name: "--no-color", Description: "Disable ANSI color output"},
		{Name: "--no-stale-check", Description: "Skip the Go source freshness check"},
	}
}
