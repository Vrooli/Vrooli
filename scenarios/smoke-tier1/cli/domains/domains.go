package domains

import (
	"smoke-tier1/cli/domains/notes"

	"github.com/vrooli/cli-core/cliapp"
)

// CommandGroups aggregates flat command groups from domain packages.
//
// Keep app.go focused on CLI metadata and cli-core wiring. As the scenario
// grows, add domains like domains/tasks or domains/projects and append their
// registrations here. For greenfield scenarios, domain packages are the
// default architecture; do not treat flat command files as the long-term plan.
func CommandGroups(core *cliapp.ScenarioApp) []cliapp.CommandGroup {
	_ = core
	return nil
}

// SubcommandGroups aggregates hierarchical command groups from domain packages.
//
// Each domain package owns a Register(core *cliapp.ScenarioApp) function
// returning a SubcommandGroup; this aggregator is intentionally a one-liner
// per domain so adding a new one is mechanical. The notes domain is the
// canonical CRUD reference — copy its shape (cli/domains/notes/) when
// adding a real feature.
//
// For API-backed commands:
//   - set NeedsAPI: true so stale-check + --auto-start preflight works
//   - call core.Get(...) / core.Request(...) for versioned /api/v1 routes
//   - use cliapp.RenderOperationalReport / RenderListReport /
//     RenderMutationReport for default human output contracts
//   - use cliapp.PrintReportJSON(...) when a --json mode should mirror the
//     same structured report
func SubcommandGroups(core *cliapp.ScenarioApp) []cliapp.SubcommandGroup {
	return []cliapp.SubcommandGroup{
		notes.Register(core),
	}
}
