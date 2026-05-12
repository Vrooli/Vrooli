package domains

import (
	"flow-verifier/cli/domains/artifacts"
	"flow-verifier/cli/domains/flows"
	"flow-verifier/cli/domains/runs"
	"flow-verifier/cli/domains/scenarios"
	"flow-verifier/cli/domains/settings"
	"flow-verifier/cli/domains/verify"

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
// per domain so adding a new one is mechanical. The runs domain is the
// canonical reference for a SQLite-backed CRUD-ish surface; copy its
// shape (cli/domains/runs/) when adding a real feature.
//
// This is the CLI side of the domain-module pattern; the API side uses
// the same one-liner-per-domain shape via server.New(deps, modules...).
// See docs/concepts/ARCHITECTURE.md "Domain modules" for the canonical
// pattern when adding your scenario's next domain.
//
// For API-backed commands:
//   - set NeedsAPI: true so stale-check + --auto-start preflight works
//   - declare flags and positionals with cliapp.ArgSchema
//   - implement RunCtx handlers and read values from cliapp.RunContext
//   - use generated Connect clients for proto-typed operations
//   - use cliapp.UploadFile only for documented multipart REST exceptions
//   - render proto responses with cliapp.RenderProtoList or RenderProtoMutation
func SubcommandGroups(core *cliapp.ScenarioApp) []cliapp.SubcommandGroup {
	return []cliapp.SubcommandGroup{
		artifacts.Register(core),
		flows.Register(core),
		runs.Register(core),
		scenarios.Register(core),
		settings.Register(core),
		verify.Register(core),
	}
}
