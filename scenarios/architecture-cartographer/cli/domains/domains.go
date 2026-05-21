package domains

import (
	"github.com/vrooli/cli-core/cliapp"
)

// CommandGroups aggregates flat command groups from domain packages.
//
// Keep app.go focused on CLI metadata and cli-core wiring. As the scenario
// grows, add domains like domains/conflicts or domains/graph and append
// their registrations here. For greenfield scenarios, domain packages are
// the default architecture; do not treat flat command files as the long-term
// plan.
func CommandGroups(core *cliapp.ScenarioApp) []cliapp.CommandGroup {
	_ = core
	return nil
}

// SubcommandGroups aggregates hierarchical command groups from domain packages.
//
// Each domain package owns a Register(core, manifest) function returning a
// SubcommandGroup built from the scenario's cli/manifest.json. The aggregator
// passes the embedded manifest bytes through unchanged; per-domain Register
// implementations call cliapp.LoadFromManifest with the relevant group name.
//
// This is the CLI side of the domain-module pattern; the API side uses
// the same one-liner-per-domain shape via server.New(deps, modules...).
//
// The architecture-cartographer scenario will register its product
// domains (graph, manifest, conflicts, signals, apply, analytics) here
// as each phase of the implementation plan lands; until then the slice
// is empty.
func SubcommandGroups(core *cliapp.ScenarioApp, manifest []byte) ([]cliapp.SubcommandGroup, error) {
	_ = core
	_ = manifest
	return []cliapp.SubcommandGroup{}, nil
}
