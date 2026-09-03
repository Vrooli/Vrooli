package topcli

import (
	"io"

	"github.com/vrooli/vrooli/internal/cli/commandtree"
)

type CommandID string

const (
	CommandSetup            CommandID = "setup"
	CommandDevelop          CommandID = "develop"
	CommandBuild            CommandID = "build"
	CommandClean            CommandID = "clean"
	CommandStatus           CommandID = "status"
	CommandStop             CommandID = "stop"
	CommandBackup           CommandID = "backup"
	CommandRestore          CommandID = "restore"
	CommandAgent            CommandID = "agent"
	CommandScenario         CommandID = "scenario"
	CommandPackage          CommandID = "package"
	CommandResource         CommandID = "resource"
	CommandRuntime          CommandID = "runtime"
	CommandCleanup          CommandID = "cleanup"
	CommandDoctor           CommandID = "doctor"
	CommandOrphans          CommandID = "orphans"
	CommandLocks            CommandID = "locks"
	CommandDiagnosePort     CommandID = "diagnose-port"
	CommandContract         CommandID = "contract"
	CommandHygiene          CommandID = "hygiene"
	CommandLifecycle        CommandID = "lifecycle"
	CommandAuth             CommandID = "auth"
	CommandRecovery         CommandID = "recovery"
	CommandHost             CommandID = "host"
	CommandHostWatchdog     CommandID = "host-watchdog"
	CommandWorkload         CommandID = "workload"
	CommandCapacity         CommandID = "capacity"
	CommandCapability       CommandID = "capability"
	CommandTuning           CommandID = "tuning"
	CommandCredentials      CommandID = "credentials"
	CommandReleaseAuthority CommandID = "release-authority"
	CommandBreakGlass       CommandID = "break-glass"
	CommandSupervisionSet   CommandID = "supervision-set"
	CommandUninstall        CommandID = "uninstall"
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
		{Name: string(CommandAgent), Group: "Interactive Commands", Summary: "Launch an allowlisted coding-agent CLI", Handler: CommandAgent, Suggestable: true, RootPolicy: commandtree.RootPolicy{RequiresRoot: false, CanRunWithoutRoot: nil}},
		{Name: string(CommandScenario), Group: "Scenario Management", Summary: "Manage scenarios from their source locations", Handler: CommandScenario, Suggestable: true, RootPolicy: commandtree.RootPolicy{RequiresRoot: true, CanRunWithoutRoot: ScenarioCanRunWithoutRoot}},
		{Name: string(CommandPackage), Group: "Package Governance", Summary: "Manage governed shared packages", Handler: CommandPackage, Suggestable: true, RootPolicy: commandtree.RootPolicy{RequiresRoot: true, CanRunWithoutRoot: ListOrHelpWithoutRoot}},
		{Name: string(CommandResource), Group: "Resource Management", Summary: "Manage local resources and dependency services", Handler: CommandResource, Suggestable: true, RootPolicy: commandtree.RootPolicy{RequiresRoot: true, CanRunWithoutRoot: ListOrHelpWithoutRoot}},
		{Name: string(CommandRuntime), Group: "Runtime Management", Summary: "Manage the scenario runtime supervisor", Handler: CommandRuntime, Suggestable: true, RootPolicy: commandtree.RootPolicy{RequiresRoot: true, CanRunWithoutRoot: ListOrHelpWithoutRoot}},
		{Name: string(CommandCleanup), Group: "Maintenance Commands", Summary: "Clean up orphans and stale registry claims", Handler: CommandCleanup, Suggestable: true, RootPolicy: commandtree.RootPolicy{RequiresRoot: true, CanRunWithoutRoot: ListOrHelpWithoutRoot}},
		{Name: string(CommandDoctor), Group: "Maintenance Commands", Summary: "Run environment and tool diagnostics", Handler: CommandDoctor, Suggestable: true, RootPolicy: commandtree.RootPolicy{RequiresRoot: true, CanRunWithoutRoot: HelpOnlyWithoutRoot}},
		{Name: string(CommandOrphans), Group: "Maintenance Commands", Summary: "Inspect or clean orphaned Vrooli processes", Handler: CommandOrphans, Suggestable: true, RootPolicy: commandtree.RootPolicy{RequiresRoot: true, CanRunWithoutRoot: HelpOnlyWithoutRoot}},
		{Name: string(CommandLocks), Group: "Maintenance Commands", Summary: "Inspect runtime registry claims", Handler: CommandLocks, Suggestable: true, RootPolicy: commandtree.RootPolicy{RequiresRoot: true, CanRunWithoutRoot: HelpOnlyWithoutRoot}},
		{Name: string(CommandDiagnosePort), Group: "Maintenance Commands", Summary: "Diagnose port conflicts using registry claims and listener evidence", Handler: CommandDiagnosePort, Suggestable: true, RootPolicy: commandtree.RootPolicy{RequiresRoot: true, CanRunWithoutRoot: HelpOnlyWithoutRoot}},
		{Name: string(CommandContract), Group: "Maintenance Commands", Summary: "Inspect and validate the repository contract", Handler: CommandContract, Suggestable: true},
		{Name: string(CommandHygiene), Group: "Maintenance Commands", Summary: "Run repository hygiene checks", Handler: CommandHygiene, Suggestable: true},
		{Name: string(CommandAuth), Group: "Maintenance Commands", Summary: "Report sign-in state for host tools (buf, future: claude/codex/gh/...)", Handler: CommandAuth, Suggestable: true},
		{Name: string(CommandRecovery), Group: "Maintenance Commands", Summary: "Baseline Modes recovery floor: restore points and engagement manifests", Handler: CommandRecovery, Suggestable: true},
		{Name: string(CommandHost), Group: "Maintenance Commands", Summary: "Inspect local host inventory via the shared Go collector", Handler: CommandHost, Suggestable: true},
		{Name: string(CommandHostWatchdog), Group: "Maintenance Commands", Summary: "Run the disk-floor watchdog and report pressure to storage-manager", Handler: CommandHostWatchdog, Suggestable: true, RootPolicy: commandtree.RootPolicy{RequiresRoot: false, CanRunWithoutRoot: func([]string) bool { return true }}},
		{Name: string(CommandWorkload), Group: "Maintenance Commands", Summary: "List host workloads and their Vrooli ownership posture", Handler: CommandWorkload, Suggestable: true, RootPolicy: commandtree.RootPolicy{RequiresRoot: false, CanRunWithoutRoot: ListOrHelpWithoutRoot}},
		{Name: string(CommandCapacity), Group: "Maintenance Commands", Summary: "Arbitrate host resource capacity (GPU VRAM/RAM/CPU) via the claim ledger", Handler: CommandCapacity, Suggestable: true},
		{Name: string(CommandCapability), Group: "Maintenance Commands", Summary: "Read the manifest-derived cross-platform capability ledger", Handler: CommandCapability, Suggestable: true},
		{Name: string(CommandTuning), Group: "Configuration Commands", Summary: "Inspect operator-visible control-plane timing levers", Handler: CommandTuning, Suggestable: true},
		{Name: string(CommandUninstall), Group: "Maintenance Commands", Summary: "Plan and apply removal of recorded Vrooli installation artifacts", Handler: CommandUninstall, Suggestable: true},
		{Name: string(CommandCredentials), Group: "Configuration Commands", Summary: "Provision and inspect credentials through the native secure-store authority", Handler: CommandCredentials, Suggestable: true},
		{Name: string(CommandReleaseAuthority), Group: "Configuration Commands", Summary: "Create, inspect, and use the managed release signing authority", Handler: CommandReleaseAuthority, Suggestable: true},
		{Name: string(CommandBreakGlass), Group: "Configuration Commands", Summary: "Provision and issue target-bound break-glass credentials", Handler: CommandBreakGlass, Suggestable: true},
		{Name: string(CommandSupervisionSet), Group: "Configuration Commands", Summary: "Show the manifest-derived supervision set and attribution", Handler: CommandSupervisionSet, Suggestable: true, RootPolicy: commandtree.RootPolicy{RequiresRoot: true, CanRunWithoutRoot: HelpOnlyWithoutRoot}, Args: commandtree.ArgSchema{Options: []commandtree.OptionArg{{Name: "--kind", ValueName: "scenario|resource", Description: "Filter members by kind"}, commandtree.JSONOption()}}},
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

// RenderMainHelp renders the root help. notes are appended after the
// standard documentation note; the root runner passes the retired-flags line
// so `vrooli help` documents the tolerance grace.
func RenderMainHelp(w io.Writer, specs []commandtree.Spec[CommandID], notes ...string) {
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
		Notes:        append([]string{"Documentation: docs/"}, notes...),
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
	}
}
