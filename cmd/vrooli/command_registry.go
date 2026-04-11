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
	Handler     topLevelCommandHandler
	Suggestable bool
}

type scenarioCommandDescriptor struct {
	Name        string
	Group       string
	Summary     string
	Hidden      bool
	Handler     topLevelCommandHandler
	Suggestable bool
}

var topLevelCommandTable = []commandDescriptor{
	{Name: "setup", Group: "Lifecycle Commands", Summary: "Initialize the development environment", Handler: runTopLevelSetupCommand, Suggestable: true},
	{Name: "develop", Group: "Lifecycle Commands", Summary: "Start development servers", Handler: runTopLevelDevelopCommand, Suggestable: true},
	{Name: "build", Group: "Lifecycle Commands", Summary: "Build the project", Handler: runTopLevelBuildCommand, Suggestable: true},
	{Name: "deploy", Group: "Lifecycle Commands", Summary: "Deploy to production", Handler: runTopLevelDeployCommand, Suggestable: true},
	{Name: "clean", Group: "Lifecycle Commands", Summary: "Clean build artifacts", Handler: runTopLevelCleanupCommand, Suggestable: true},
	{Name: "status", Group: "Lifecycle Commands", Summary: "Show system health and status overview", Handler: runTopLevelStatusCommand, Suggestable: true},
	{Name: "stop", Group: "Lifecycle Commands", Summary: "Stop all or specific components", Handler: runTopLevelStopCommand, Suggestable: true},
	{Name: "backup", Group: "Lifecycle Commands", Summary: "Run the project backup lifecycle when defined", Handler: runTopLevelBackupCommand, Suggestable: true},
	{Name: "restore", Group: "Lifecycle Commands", Summary: "Run the project restore lifecycle when defined", Handler: runTopLevelRestoreCommand, Suggestable: true},
	{Name: "info", Group: "Context Commands", Summary: "Show consolidated project briefing", Handler: runInfoCommand, Suggestable: true},
	{Name: "scenario", Group: "Scenario Management", Summary: "Manage scenarios from their source locations", Handler: runScenarioCommand, Suggestable: true},
	{Name: "resource", Group: "Resource Management", Summary: "Manage local resources and dependency services", Handler: runTopLevelResourceCommand, Suggestable: true},
	{Name: "cleanup", Group: "Maintenance Commands", Summary: "Clean up orphans and stale locks", Handler: runTopLevelCleanupCommand, Suggestable: true},
	{Name: "doctor", Group: "Maintenance Commands", Summary: "Run environment and tool diagnostics", Handler: runTopLevelDoctorCommand, Suggestable: true},
	{Name: "orphans", Group: "Maintenance Commands", Summary: "Inspect or clean orphaned Vrooli processes", Handler: runTopLevelOrphansCommand, Suggestable: true},
	{Name: "locks", Group: "Maintenance Commands", Summary: "Inspect or clean stale port lock files", Handler: runTopLevelLocksCommand, Suggestable: true},
	{Name: "diagnose-port", Group: "Maintenance Commands", Summary: "Diagnose port conflicts and stale lock ownership", Handler: runTopLevelDiagnosePortCommand, Suggestable: true},
}

var scenarioCommandTable = []scenarioCommandDescriptor{
	{Name: "list", Group: "Read-only Commands", Summary: "List discovered scenarios", Handler: wrapScenarioStdout(runScenarioListCommand), Suggestable: true},
	{Name: "info", Group: "Read-only Commands", Summary: "Show scenario metadata and runtime summary", Handler: wrapScenarioStdout(runScenarioInfoCommand), Suggestable: true},
	{Name: "status", Group: "Read-only Commands", Summary: "Show scenario runtime status", Handler: wrapScenarioStdout(runScenarioStatusCommand), Suggestable: true},
	{Name: "run", Group: "Lifecycle and Utility Commands", Summary: "Run a scenario directly (alias of start)", Handler: runScenarioRunCommand, Suggestable: true},
	{Name: "start", Group: "Lifecycle and Utility Commands", Summary: "Start a scenario", Handler: runScenarioStartCommand, Suggestable: true},
	{Name: "start-all", Group: "Lifecycle and Utility Commands", Summary: "Start all available scenarios", Handler: runScenarioStartAllCommand, Suggestable: true},
	{Name: "setup", Group: "Lifecycle and Utility Commands", Summary: "Run the setup lifecycle", Handler: runScenarioSetupCommand, Suggestable: true},
	{Name: "restart", Group: "Lifecycle and Utility Commands", Summary: "Restart a scenario", Handler: runScenarioRestartCommand, Suggestable: true},
	{Name: "stop", Group: "Lifecycle and Utility Commands", Summary: "Stop a running scenario", Handler: runScenarioStopCommand, Suggestable: true},
	{Name: "stop-all", Group: "Lifecycle and Utility Commands", Summary: "Stop all running scenarios", Handler: runScenarioStopAllCommand, Suggestable: true},
	{Name: "test", Group: "Lifecycle and Utility Commands", Summary: "Run scenario tests", Handler: runScenarioTestCommand, Suggestable: true},
	{Name: "logs", Group: "Lifecycle and Utility Commands", Summary: "View logs for a scenario", Handler: runScenarioLogsCommand, Suggestable: true},
	{Name: "open", Group: "Lifecycle and Utility Commands", Summary: "Open a scenario in the browser", Handler: runScenarioOpenCommand, Suggestable: true},
	{Name: "port", Group: "Lifecycle and Utility Commands", Summary: "Show running port assignments", Handler: wrapScenarioStdout(runScenarioPortCommand), Suggestable: true},
	{Name: "ui-smoke", Group: "Lifecycle and Utility Commands", Summary: "Run the Browserless UI smoke harness", Handler: runScenarioUISmokeCommand, Suggestable: true},
	{Name: "requirements", Group: "Lifecycle and Utility Commands", Summary: "Manage scenario requirements", Handler: runScenarioRequirementsCommand, Suggestable: true},
	{Name: "template", Group: "Lifecycle and Utility Commands", Summary: "Manage scenario templates", Handler: runScenarioTemplateCommand, Suggestable: true},
	{Name: "generate", Group: "Lifecycle and Utility Commands", Summary: "Scaffold a scenario from a template", Handler: runScenarioGenerateCommand, Suggestable: true},
	{Name: "completeness", Group: "Lifecycle and Utility Commands", Summary: "Calculate a completeness score", Handler: runScenarioCompletenessCommand, Suggestable: true},
	{Name: "heal-from-sandbox", Group: "Lifecycle and Utility Commands", Summary: "Relaunch sandbox-rooted scenario processes", Handler: runScenarioHealFromSandboxCommand, Suggestable: true},
}

var topLevelCommands = buildTopLevelCommandMap(topLevelCommandTable)
var scenarioCommands = buildScenarioCommandMap(scenarioCommandTable)

func buildTopLevelCommandMap(descriptors []commandDescriptor) map[string]topLevelCommandHandler {
	commands := make(map[string]topLevelCommandHandler, len(descriptors))
	for _, descriptor := range descriptors {
		commands[descriptor.Name] = descriptor.Handler
	}
	return commands
}

func buildScenarioCommandMap(descriptors []scenarioCommandDescriptor) map[string]topLevelCommandHandler {
	commands := make(map[string]topLevelCommandHandler, len(descriptors))
	for _, descriptor := range descriptors {
		commands[descriptor.Name] = descriptor.Handler
	}
	return commands
}

func wrapScenarioStdout(handler func(root string, globals globalOptions, args []string, stdout io.Writer) error) topLevelCommandHandler {
	return func(root string, globals globalOptions, args []string, stdout, stderr io.Writer) error {
		return handler(root, globals, args, stdout)
	}
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

func groupedScenarioCommands() []scenarioCommandDescriptor {
	visible := make([]scenarioCommandDescriptor, 0, len(scenarioCommandTable))
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

func (d scenarioCommandDescriptor) commandName() string    { return d.Name }
func (d scenarioCommandDescriptor) commandGroup() string   { return d.Group }
func (d scenarioCommandDescriptor) commandSummary() string { return d.Summary }
