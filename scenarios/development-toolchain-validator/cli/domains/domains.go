package domains

import (
	"development-toolchain-validator/cli/domains/golden"
	"development-toolchain-validator/cli/domains/notes"

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
// This is the CLI side of the domain-module pattern; the API side uses
// the same one-liner-per-domain shape via server.New(deps, modules...).
// See docs/concepts/ARCHITECTURE.md "Domain modules" for the canonical
// pattern when swapping the notes reference for your scenario's first
// domain.
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
		golden.Register(core),
		notes.Register(core),
	}
}
