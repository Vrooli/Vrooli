package main

import (
	"fmt"
	"io"
	"sort"
)

type commandDescriptor struct {
	Name        string
	Group       string
	Summary     string
	Hidden      bool
	Handler     appCommandHandler
	Suggestable bool
}

type appCommandHandler func(app *App, ctx *commandContext, args []string) error

var topLevelCommandTable = []commandDescriptor{
	{Name: "setup", Group: "Lifecycle Commands", Summary: "Initialize the development environment", Handler: (*App).runTopLevelSetupCommand, Suggestable: true},
	{Name: "develop", Group: "Lifecycle Commands", Summary: "Start development servers", Handler: (*App).runTopLevelDevelopCommand, Suggestable: true},
	{Name: "build", Group: "Lifecycle Commands", Summary: "Build the project", Handler: (*App).runTopLevelBuildCommand, Suggestable: true},
	{Name: "deploy", Group: "Lifecycle Commands", Summary: "Deploy to production", Handler: (*App).runTopLevelDeployCommand, Suggestable: true},
	{Name: "clean", Group: "Lifecycle Commands", Summary: "Clean build artifacts", Handler: (*App).runTopLevelCleanCommand, Suggestable: true},
	{Name: "status", Group: "Lifecycle Commands", Summary: "Show system health and status overview", Handler: (*App).runTopLevelStatusCommand, Suggestable: true},
	{Name: "stop", Group: "Lifecycle Commands", Summary: "Stop all or specific components", Handler: (*App).runTopLevelStopCommand, Suggestable: true},
	{Name: "backup", Group: "Lifecycle Commands", Summary: "Run the project backup lifecycle when defined", Handler: (*App).runTopLevelBackupCommand, Suggestable: true},
	{Name: "restore", Group: "Lifecycle Commands", Summary: "Run the project restore lifecycle when defined", Handler: (*App).runTopLevelRestoreCommand, Suggestable: true},
	{Name: "info", Group: "Context Commands", Summary: "Show consolidated project briefing", Handler: (*App).runInfoCommand, Suggestable: true},
	{Name: "scenario", Group: "Scenario Management", Summary: "Manage scenarios from their source locations", Handler: (*App).runScenarioCommand, Suggestable: true},
	{Name: "resource", Group: "Resource Management", Summary: "Manage local resources and dependency services", Handler: (*App).runTopLevelResourceCommand, Suggestable: true},
	{Name: "cleanup", Group: "Maintenance Commands", Summary: "Clean up orphans and stale locks", Handler: (*App).runTopLevelCleanupCommand, Suggestable: true},
	{Name: "doctor", Group: "Maintenance Commands", Summary: "Run environment and tool diagnostics", Handler: (*App).runTopLevelDoctorCommand, Suggestable: true},
	{Name: "orphans", Group: "Maintenance Commands", Summary: "Inspect or clean orphaned Vrooli processes", Handler: (*App).runTopLevelOrphansCommand, Suggestable: true},
	{Name: "locks", Group: "Maintenance Commands", Summary: "Inspect or clean stale port lock files", Handler: (*App).runTopLevelLocksCommand, Suggestable: true},
	{Name: "diagnose-port", Group: "Maintenance Commands", Summary: "Diagnose port conflicts and stale lock ownership", Handler: (*App).runTopLevelDiagnosePortCommand, Suggestable: true},
}

var scenarioCommandTable = []commandDescriptor{
	{Name: "list", Group: "Read-only Commands", Summary: "List discovered scenarios", Handler: (*App).runScenarioListCommand, Suggestable: true},
	{Name: "info", Group: "Read-only Commands", Summary: "Show scenario metadata and runtime summary", Handler: (*App).runScenarioInfoCommand, Suggestable: true},
	{Name: "status", Group: "Read-only Commands", Summary: "Show scenario runtime status", Handler: (*App).runScenarioStatusCommand, Suggestable: true},
	{Name: "run", Group: "Lifecycle and Utility Commands", Summary: "Run a scenario directly (alias of start)", Handler: (*App).runScenarioRunCommand, Suggestable: true},
	{Name: "start", Group: "Lifecycle and Utility Commands", Summary: "Start a scenario", Handler: (*App).runScenarioStartCommand, Suggestable: true},
	{Name: "start-all", Group: "Lifecycle and Utility Commands", Summary: "Start all available scenarios", Handler: (*App).runScenarioStartAllCommand, Suggestable: true},
	{Name: "setup", Group: "Lifecycle and Utility Commands", Summary: "Run the setup lifecycle", Handler: (*App).runScenarioSetupCommand, Suggestable: true},
	{Name: "restart", Group: "Lifecycle and Utility Commands", Summary: "Restart a scenario", Handler: (*App).runScenarioRestartCommand, Suggestable: true},
	{Name: "stop", Group: "Lifecycle and Utility Commands", Summary: "Stop a running scenario", Handler: (*App).runScenarioStopCommand, Suggestable: true},
	{Name: "stop-all", Group: "Lifecycle and Utility Commands", Summary: "Stop all running scenarios", Handler: (*App).runScenarioStopAllCommand, Suggestable: true},
	{Name: "test", Group: "Lifecycle and Utility Commands", Summary: "Run scenario tests", Handler: (*App).runScenarioTestCommand, Suggestable: true},
	{Name: "logs", Group: "Lifecycle and Utility Commands", Summary: "View logs for a scenario", Handler: (*App).runScenarioLogsCommand, Suggestable: true},
	{Name: "open", Group: "Lifecycle and Utility Commands", Summary: "Open a scenario in the browser", Handler: (*App).runScenarioOpenCommand, Suggestable: true},
	{Name: "port", Group: "Lifecycle and Utility Commands", Summary: "Show running port assignments", Handler: (*App).runScenarioPortCommand, Suggestable: true},
	{Name: "ui-smoke", Group: "Lifecycle and Utility Commands", Summary: "Run the Browserless UI smoke harness", Handler: (*App).runScenarioUISmokeCommand, Suggestable: true},
	{Name: "requirements", Group: "Lifecycle and Utility Commands", Summary: "Manage scenario requirements", Handler: (*App).runScenarioRequirementsCommand, Suggestable: true},
	{Name: "template", Group: "Lifecycle and Utility Commands", Summary: "Manage scenario templates", Handler: (*App).runScenarioTemplateCommand, Suggestable: true},
	{Name: "generate", Group: "Lifecycle and Utility Commands", Summary: "Scaffold a scenario from a template", Handler: (*App).runScenarioGenerateCommand, Suggestable: true},
	{Name: "completeness", Group: "Lifecycle and Utility Commands", Summary: "Calculate a completeness score", Handler: (*App).runScenarioCompletenessCommand, Suggestable: true},
	{Name: "heal-from-sandbox", Group: "Lifecycle and Utility Commands", Summary: "Relaunch sandbox-rooted scenario processes", Handler: (*App).runScenarioHealFromSandboxCommand, Suggestable: true},
}

var topLevelCommands = buildTopLevelCommandMap(topLevelCommandTable)
var scenarioCommands = buildScenarioCommandMap(scenarioCommandTable)

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
