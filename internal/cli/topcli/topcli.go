package topcli

import (
	"fmt"
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
	CommandInfo         CommandID = "info"
	CommandScenario     CommandID = "scenario"
	CommandPackage      CommandID = "package"
	CommandResource     CommandID = "resource"
	CommandCleanup      CommandID = "cleanup"
	CommandDoctor       CommandID = "doctor"
	CommandOrphans      CommandID = "orphans"
	CommandLocks        CommandID = "locks"
	CommandDiagnosePort CommandID = "diagnose-port"
	CommandContract     CommandID = "contract"
	CommandLifecycle    CommandID = "lifecycle"
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
		{Name: string(CommandInfo), Group: "Context Commands", Summary: "Show consolidated project briefing", Handler: CommandInfo, Suggestable: true, RootPolicy: commandtree.RootPolicy{RequiresRoot: true, CanRunWithoutRoot: HelpOnlyWithoutRoot}},
		{Name: string(CommandScenario), Group: "Scenario Management", Summary: "Manage scenarios from their source locations", Handler: CommandScenario, Suggestable: true, RootPolicy: commandtree.RootPolicy{RequiresRoot: true, CanRunWithoutRoot: ScenarioCanRunWithoutRoot}},
		{Name: string(CommandPackage), Group: "Package Governance", Summary: "Manage governed shared packages", Handler: CommandPackage, Suggestable: true, RootPolicy: commandtree.RootPolicy{RequiresRoot: true, CanRunWithoutRoot: ListOrHelpWithoutRoot}},
		{Name: string(CommandResource), Group: "Resource Management", Summary: "Manage local resources and dependency services", Handler: CommandResource, Suggestable: true, RootPolicy: commandtree.RootPolicy{RequiresRoot: true, CanRunWithoutRoot: ListOrHelpWithoutRoot}},
		{Name: string(CommandCleanup), Group: "Maintenance Commands", Summary: "Clean up orphans and stale locks", Handler: CommandCleanup, Suggestable: true, RootPolicy: commandtree.RootPolicy{RequiresRoot: true, CanRunWithoutRoot: ListOrHelpWithoutRoot}},
		{Name: string(CommandDoctor), Group: "Maintenance Commands", Summary: "Run environment and tool diagnostics", Handler: CommandDoctor, Suggestable: true, RootPolicy: commandtree.RootPolicy{RequiresRoot: true, CanRunWithoutRoot: HelpOnlyWithoutRoot}},
		{Name: string(CommandOrphans), Group: "Maintenance Commands", Summary: "Inspect or clean orphaned Vrooli processes", Handler: CommandOrphans, Suggestable: true, RootPolicy: commandtree.RootPolicy{RequiresRoot: true, CanRunWithoutRoot: HelpOnlyWithoutRoot}},
		{Name: string(CommandLocks), Group: "Maintenance Commands", Summary: "Inspect or clean stale port lock files", Handler: CommandLocks, Suggestable: true, RootPolicy: commandtree.RootPolicy{RequiresRoot: true, CanRunWithoutRoot: HelpOnlyWithoutRoot}},
		{Name: string(CommandDiagnosePort), Group: "Maintenance Commands", Summary: "Diagnose port conflicts and stale lock ownership", Handler: CommandDiagnosePort, Suggestable: true, RootPolicy: commandtree.RootPolicy{RequiresRoot: true, CanRunWithoutRoot: HelpOnlyWithoutRoot}},
		{Name: string(CommandContract), Group: "Maintenance Commands", Summary: "Inspect and validate the repository contract", Handler: CommandContract, Suggestable: true},
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
	_, _ = fmt.Fprintln(w, "                          ___")
	_, _ = fmt.Fprintln(w, " _   _ _ __ ___   ___    / (_)")
	_, _ = fmt.Fprintln(w, "| | | | '__/ _ \\ / _ \\  / /| |")
	_, _ = fmt.Fprintln(w, "| |_| | | | (_) | (_) |/ / | |")
	_, _ = fmt.Fprintln(w, " \\___/|_|  \\___/ \\___//_/  |_|")
	_, _ = fmt.Fprintln(w, "                                   ")
	_, _ = fmt.Fprintln(w, "Vrooli CLI - AI Platform Management Tool")
	_, _ = fmt.Fprintln(w)
	_, _ = fmt.Fprintln(w, "USAGE:")
	_, _ = fmt.Fprintln(w, "    vrooli <command> [options]")
	_, _ = fmt.Fprintln(w)
	commandtree.RenderGroups(w, commandtree.VisibleEntries(specs, ""))
	_, _ = fmt.Fprintln(w)
	_, _ = fmt.Fprintln(w, "OPTIONS:")
	_, _ = fmt.Fprintln(w, "    --help, -h          Show help for a command")
	_, _ = fmt.Fprintln(w, "    --version, -v       Show version information")
	_, _ = fmt.Fprintln(w, "    --json              Emit JSON output when supported by the selected command")
	_, _ = fmt.Fprintln(w, "    --verbose           Enable verbose command output")
	_, _ = fmt.Fprintln(w, "    --no-color          Disable ANSI color output")
	_, _ = fmt.Fprintln(w, "    --no-stale-check    Skip the Go source freshness check")
	_, _ = fmt.Fprintln(w)
	_, _ = fmt.Fprintln(w, "For more help on a specific command:")
	_, _ = fmt.Fprintln(w, "    vrooli <command> --help")
	_, _ = fmt.Fprintln(w)
	_, _ = fmt.Fprintln(w, "Documentation: docs/")
}
