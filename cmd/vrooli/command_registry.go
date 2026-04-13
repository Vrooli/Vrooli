package main

import (
	"fmt"
	"io"
	"sort"
)

type commandDescriptor struct {
	Name              string
	Group             string
	Summary           string
	Hidden            bool
	Handler           appCommandHandler
	Suggestable       bool
	RequiresRoot      bool
	CanRunWithoutRoot func(args []string) bool
}

type appCommandHandler func(app *App, ctx *commandContext, args []string) error

var topLevelCommandTable = []commandDescriptor{
	{Name: "setup", Group: "Lifecycle Commands", Summary: "Initialize the development environment", Handler: runTopLevelSetupCommand, Suggestable: true, RequiresRoot: true, CanRunWithoutRoot: helpOnlyWithoutRoot},
	{Name: "develop", Group: "Lifecycle Commands", Summary: "Start development servers", Handler: runTopLevelDevelopCommand, Suggestable: true, RequiresRoot: true, CanRunWithoutRoot: helpOnlyWithoutRoot},
	{Name: "build", Group: "Lifecycle Commands", Summary: "Build the project", Handler: runTopLevelBuildCommand, Suggestable: true, RequiresRoot: true, CanRunWithoutRoot: helpOnlyWithoutRoot},
	{Name: "deploy", Group: "Lifecycle Commands", Summary: "Deploy to production", Handler: runTopLevelDeployCommand, Suggestable: true, RequiresRoot: true, CanRunWithoutRoot: helpOnlyWithoutRoot},
	{Name: "clean", Group: "Lifecycle Commands", Summary: "Clean build artifacts", Handler: runTopLevelCleanCommand, Suggestable: true, RequiresRoot: true, CanRunWithoutRoot: helpOnlyWithoutRoot},
	{Name: "status", Group: "Lifecycle Commands", Summary: "Show system health and status overview", Handler: bindGlobalCommand(parseTopLevelStatusRequest, runTopLevelStatusRequest, renderTopLevelStatusResponse), Suggestable: true, RequiresRoot: true, CanRunWithoutRoot: helpOnlyWithoutRoot},
	{Name: "stop", Group: "Lifecycle Commands", Summary: "Stop all or specific components", Handler: bindGlobalCommand(parseTopLevelStopRequest, runTopLevelStopRequest, renderTopLevelStopResponse), Suggestable: true, RequiresRoot: true, CanRunWithoutRoot: helpOnlyWithoutRoot},
	{Name: "backup", Group: "Lifecycle Commands", Summary: "Run the project backup lifecycle when defined", Handler: runTopLevelBackupCommand, Suggestable: true, RequiresRoot: true, CanRunWithoutRoot: helpOnlyWithoutRoot},
	{Name: "restore", Group: "Lifecycle Commands", Summary: "Run the project restore lifecycle when defined", Handler: runTopLevelRestoreCommand, Suggestable: true, RequiresRoot: true, CanRunWithoutRoot: helpOnlyWithoutRoot},
	{Name: "info", Group: "Context Commands", Summary: "Show consolidated project briefing", Handler: runInfoTopLevelCommand, Suggestable: true, RequiresRoot: true, CanRunWithoutRoot: helpOnlyWithoutRoot},
	{Name: "scenario", Group: "Scenario Management", Summary: "Manage scenarios from their source locations", Handler: runScenarioRootCommand, Suggestable: true, RequiresRoot: true, CanRunWithoutRoot: scenarioCanRunWithoutRoot},
	{Name: "package", Group: "Package Governance", Summary: "Manage governed shared packages", Handler: runPackageRootCommand, Suggestable: true, RequiresRoot: true, CanRunWithoutRoot: listOrHelpWithoutRoot},
	{Name: "resource", Group: "Resource Management", Summary: "Manage local resources and dependency services", Handler: runTopLevelResourceCommandWithApp, Suggestable: true, RequiresRoot: true, CanRunWithoutRoot: listOrHelpWithoutRoot},
	{Name: "cleanup", Group: "Maintenance Commands", Summary: "Clean up orphans and stale locks", Handler: runTopLevelCleanupCommand, Suggestable: true, RequiresRoot: true, CanRunWithoutRoot: listOrHelpWithoutRoot},
	{Name: "doctor", Group: "Maintenance Commands", Summary: "Run environment and tool diagnostics", Handler: bindGlobalCommand(parseTopLevelDoctorRequest, runTopLevelDoctorRequest, renderTopLevelDoctorResponse), Suggestable: true, RequiresRoot: true, CanRunWithoutRoot: helpOnlyWithoutRoot},
	{Name: "orphans", Group: "Maintenance Commands", Summary: "Inspect or clean orphaned Vrooli processes", Handler: bindGlobalCommand(parseTopLevelOrphansRequest, runTopLevelOrphansRequest, renderTopLevelOrphansResponse), Suggestable: true, RequiresRoot: true, CanRunWithoutRoot: helpOnlyWithoutRoot},
	{Name: "locks", Group: "Maintenance Commands", Summary: "Inspect or clean stale port lock files", Handler: bindGlobalCommand(parseTopLevelLocksRequest, runTopLevelLocksRequest, renderTopLevelLocksResponse), Suggestable: true, RequiresRoot: true, CanRunWithoutRoot: helpOnlyWithoutRoot},
	{Name: "diagnose-port", Group: "Maintenance Commands", Summary: "Diagnose port conflicts and stale lock ownership", Handler: bindGlobalCommand(parseTopLevelDiagnosePortRequest, runTopLevelDiagnosePortRequest, renderTopLevelDiagnosePortResponse), Suggestable: true, RequiresRoot: true, CanRunWithoutRoot: helpOnlyWithoutRoot},
	{Name: "contract", Group: "Maintenance Commands", Summary: "Inspect and validate the repository contract", Handler: runContractCommandWithApp, Suggestable: true, RequiresRoot: false},
	{Name: "lifecycle", Group: "Maintenance Commands", Summary: "Internal lifecycle command plumbing", Hidden: true, Handler: runTopLevelLifecycleCommandWithApp, Suggestable: false, RequiresRoot: true, CanRunWithoutRoot: helpOnlyWithoutRoot},
}

var scenarioCommandTable = []commandDescriptor{
	{Name: "list", Group: "Read-only Commands", Summary: "List discovered scenarios", Handler: bindGlobalCommand(parseScenarioListRequest, runScenarioListRequest, renderScenarioListResponse), Suggestable: true, RequiresRoot: true, CanRunWithoutRoot: helpOnlyWithoutRoot},
	{Name: "info", Group: "Read-only Commands", Summary: "Show scenario metadata and runtime summary", Handler: bindGlobalCommand(parseScenarioInfoRequest, runScenarioInfoRequest, renderScenarioInfoResponse), Suggestable: true, RequiresRoot: true, CanRunWithoutRoot: helpOnlyWithoutRoot},
	{Name: "status", Group: "Read-only Commands", Summary: "Show scenario runtime status", Handler: bindGlobalCommand(parseScenarioStatusRequest, runScenarioStatusRequest, renderScenarioStatusResponse), Suggestable: true, RequiresRoot: true, CanRunWithoutRoot: helpOnlyWithoutRoot},
	{Name: "run", Group: "Lifecycle and Utility Commands", Summary: "Run a scenario directly (alias of start)", Handler: runScenarioRunCommand, Suggestable: true, RequiresRoot: true, CanRunWithoutRoot: helpOnlyWithoutRoot},
	{Name: "start", Group: "Lifecycle and Utility Commands", Summary: "Start a scenario", Handler: bindGlobalCommand(parseScenarioStartRequest, runScenarioStartRequest, renderScenarioLifecycleResponse), Suggestable: true, RequiresRoot: true, CanRunWithoutRoot: helpOnlyWithoutRoot},
	{Name: "start-all", Group: "Lifecycle and Utility Commands", Summary: "Start all available scenarios", Handler: runScenarioStartAllCommandWithApp, Suggestable: true, RequiresRoot: true, CanRunWithoutRoot: helpOnlyWithoutRoot},
	{Name: "setup", Group: "Lifecycle and Utility Commands", Summary: "Run the setup lifecycle", Handler: runScenarioSetupCommandWithApp, Suggestable: true, RequiresRoot: true, CanRunWithoutRoot: helpOnlyWithoutRoot},
	{Name: "restart", Group: "Lifecycle and Utility Commands", Summary: "Restart a scenario", Handler: bindGlobalCommand(parseScenarioRestartRequest, runScenarioRestartRequest, renderScenarioLifecycleResponse), Suggestable: true, RequiresRoot: true, CanRunWithoutRoot: helpOnlyWithoutRoot},
	{Name: "stop", Group: "Lifecycle and Utility Commands", Summary: "Stop a running scenario", Handler: bindGlobalCommand(parseScenarioStopRequest, runScenarioStopRequest, renderScenarioLifecycleResponse), Suggestable: true, RequiresRoot: true, CanRunWithoutRoot: helpOnlyWithoutRoot},
	{Name: "stop-all", Group: "Lifecycle and Utility Commands", Summary: "Stop all running scenarios", Handler: runScenarioStopAllCommandWithApp, Suggestable: true, RequiresRoot: true, CanRunWithoutRoot: helpOnlyWithoutRoot},
	{Name: "test", Group: "Lifecycle and Utility Commands", Summary: "Run scenario tests", Handler: runScenarioTestCommandWithApp, Suggestable: true, RequiresRoot: true, CanRunWithoutRoot: helpOnlyWithoutRoot},
	{Name: "logs", Group: "Lifecycle and Utility Commands", Summary: "View logs for a scenario", Handler: runScenarioLogsCommandWithApp, Suggestable: true, RequiresRoot: true, CanRunWithoutRoot: helpOnlyWithoutRoot},
	{Name: "open", Group: "Lifecycle and Utility Commands", Summary: "Open a scenario in the browser", Handler: runScenarioOpenCommandWithApp, Suggestable: true, RequiresRoot: true, CanRunWithoutRoot: helpOnlyWithoutRoot},
	{Name: "port", Group: "Lifecycle and Utility Commands", Summary: "Show running port assignments", Handler: runScenarioPortCommandWithApp, Suggestable: true, RequiresRoot: true, CanRunWithoutRoot: helpOnlyWithoutRoot},
	{Name: "ui-smoke", Group: "Lifecycle and Utility Commands", Summary: "Run the Browserless UI smoke harness", Handler: runScenarioUISmokeCommandWithApp, Suggestable: true, RequiresRoot: true, CanRunWithoutRoot: helpOnlyWithoutRoot},
	{Name: "requirements", Group: "Lifecycle and Utility Commands", Summary: "Manage scenario requirements", Handler: runScenarioRequirementsCommandWithApp, Suggestable: true, RequiresRoot: true, CanRunWithoutRoot: helpOnlyWithoutRoot},
	{Name: "template", Group: "Lifecycle and Utility Commands", Summary: "Manage scenario templates", Handler: runScenarioTemplateCommandWithApp, Suggestable: true, RequiresRoot: true, CanRunWithoutRoot: helpOnlyWithoutRoot},
	{Name: "generate", Group: "Lifecycle and Utility Commands", Summary: "Scaffold a scenario from a template", Handler: runScenarioGenerateCommandWithApp, Suggestable: true, RequiresRoot: true, CanRunWithoutRoot: helpOnlyWithoutRoot},
	{Name: "completeness", Group: "Lifecycle and Utility Commands", Summary: "Calculate a completeness score", Handler: runScenarioCompletenessCommandWithApp, Suggestable: true, RequiresRoot: true, CanRunWithoutRoot: helpOnlyWithoutRoot},
	{Name: "heal-from-sandbox", Group: "Lifecycle and Utility Commands", Summary: "Relaunch sandbox-rooted scenario processes", Handler: runScenarioHealFromSandboxCommandWithApp, Suggestable: true, RequiresRoot: true, CanRunWithoutRoot: helpOnlyWithoutRoot},
}

var (
	topLevelCommands           = buildTopLevelCommandMap(topLevelCommandTable)
	scenarioCommands           = buildScenarioCommandMap(scenarioCommandTable)
	topLevelCommandDescriptors = buildCommandDescriptorMap(topLevelCommandTable)
	scenarioCommandDescriptors = buildCommandDescriptorMap(scenarioCommandTable)
)

func buildTopLevelCommandMap(descriptors []commandDescriptor) map[string]appCommandHandler {
	commands := make(map[string]appCommandHandler, len(descriptors))
	for _, descriptor := range descriptors {
		commands[descriptor.Name] = descriptor.Handler
	}
	return commands
}

func buildScenarioCommandMap(descriptors []commandDescriptor) map[string]appCommandHandler {
	commands := make(map[string]appCommandHandler, len(descriptors))
	for _, descriptor := range descriptors {
		commands[descriptor.Name] = descriptor.Handler
	}
	return commands
}

func buildCommandDescriptorMap(descriptors []commandDescriptor) map[string]commandDescriptor {
	items := make(map[string]commandDescriptor, len(descriptors))
	for _, descriptor := range descriptors {
		items[descriptor.Name] = descriptor
	}
	return items
}

func helpOnlyWithoutRoot(args []string) bool {
	return wantsCommandHelp(args)
}

func listOrHelpWithoutRoot(args []string) bool {
	return len(args) == 0 || wantsCommandHelp(args)
}

func scenarioCanRunWithoutRoot(args []string) bool {
	if len(args) == 0 || wantsCommandHelp(args) {
		return true
	}
	descriptor, ok := scenarioCommandDescriptors[args[0]]
	if !ok {
		return true
	}
	if !descriptor.RequiresRoot {
		return true
	}
	if descriptor.CanRunWithoutRoot == nil {
		return false
	}
	return descriptor.CanRunWithoutRoot(args[1:])
}

func topLevelCommandNames() []string {
	names := make([]string, 0, len(topLevelCommandTable))
	for _, descriptor := range topLevelCommandTable {
		if descriptor.Suggestable {
			names = append(names, descriptor.Name)
		}
	}
	sort.Strings(names)
	return names
}

func scenarioCommandNames() []string {
	names := make([]string, 0, len(scenarioCommandTable))
	for _, descriptor := range scenarioCommandTable {
		if descriptor.Suggestable {
			names = append(names, descriptor.Name)
		}
	}
	sort.Strings(names)
	return names
}

func groupedTopLevelCommands() []commandDescriptor {
	visible := make([]commandDescriptor, 0, len(topLevelCommandTable))
	for _, descriptor := range topLevelCommandTable {
		if !descriptor.Hidden {
			visible = append(visible, descriptor)
		}
	}
	return visible
}

func groupedScenarioCommands() []commandDescriptor {
	visible := make([]commandDescriptor, 0, len(scenarioCommandTable))
	for _, descriptor := range scenarioCommandTable {
		if !descriptor.Hidden {
			visible = append(visible, descriptor)
		}
	}
	return visible
}

func renderCommandGroups[T interface {
	commandName() string
	commandGroup() string
	commandSummary() string
}](w io.Writer, entries []T) {
	currentGroup := ""
	for _, entry := range entries {
		group := entry.commandGroup()
		if group != currentGroup {
			if currentGroup != "" {
				_, _ = fmt.Fprintln(w)
			}
			_, _ = fmt.Fprintln(w, group+":")
			currentGroup = group
		}
		_, _ = fmt.Fprintf(w, "    %-18s %s\n", entry.commandName(), entry.commandSummary())
	}
}

func (d commandDescriptor) commandName() string    { return d.Name }
func (d commandDescriptor) commandGroup() string   { return d.Group }
func (d commandDescriptor) commandSummary() string { return d.Summary }
