package domains

import "github.com/vrooli/cli-core/cliapp"

// CommandGroups aggregates flat command groups from domain packages.
//
// Keep app.go focused on CLI metadata and cli-core wiring. As the scenario
// grows, add domains like domains/tasks or domains/projects and append their
// registrations here.
func CommandGroups(core *cliapp.ScenarioApp) []cliapp.CommandGroup {
	_ = core
	return nil
}

// SubcommandGroups aggregates hierarchical command groups from domain packages.
//
// Prefer domain packages as the default growth path:
//
//	cli/domains/tasks/register.go
//	cli/domains/projects/register.go
//
// For API-backed commands:
//   - set NeedsAPI: true so stale-check + --auto-start preflight works
//   - call core.Get(...) / core.Request(...) for versioned /api/v1 routes
//   - use cliapp.RenderOperationalReport / RenderListReport /
//     RenderMutationReport for default human output contracts
func SubcommandGroups(core *cliapp.ScenarioApp) []cliapp.SubcommandGroup {
	_ = core
	return nil
}
