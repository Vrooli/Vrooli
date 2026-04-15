package main

import "github.com/vrooli/cli-core/cliapp"

// registerCommandGroups is the main extension point for custom scenario
// behavior. Keep operational commands in cli-core and add domain command groups
// here or in dedicated cmd_*.go files.
func (a *App) registerCommandGroups(core *cliapp.ScenarioApp) []cliapp.CommandGroup {
	_ = core

	// Add scenario-specific command groups here. For API-backed commands:
	//   - set NeedsAPI: true so stale-check + --auto-start preflight works
	//   - call core.Get(...) / core.Request(...) for versioned /api/v1 routes
	//   - keep one domain per file for a screaming-architecture layout
	return nil
}

// registerSubcommandGroups is the preferred extension point once a CLI grows
// beyond a few flat commands. Put each domain in its own cmd_<domain>.go file.
func (a *App) registerSubcommandGroups(core *cliapp.ScenarioApp) []cliapp.SubcommandGroup {
	_ = core
	return nil
}
