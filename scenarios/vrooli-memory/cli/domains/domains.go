package domains

import (
	"vrooli-memory/cli/domains/facets"
	"vrooli-memory/cli/domains/forest"
	"vrooli-memory/cli/domains/harness"
	"vrooli-memory/cli/domains/journal"
	"vrooli-memory/cli/domains/recall"
	"vrooli-memory/cli/domains/rules"
	"vrooli-memory/cli/domains/scopes"

	"github.com/vrooli/cli-core/cliapp"
)

// CommandGroups aggregates flat command groups from domain packages.
//
// Keep app.go focused on CLI metadata and cli-core wiring. As the scenario
// grows, add domains like domains/tasks or domains/projects and append their
// registrations here. For greenfield scenarios, domain packages are the
// default architecture; do not treat flat command files as the long-term plan.
func CommandGroups(core *cliapp.ScenarioApp) []cliapp.CommandGroup {
	commands := append([]cliapp.Command{}, journal.Commands(core)...)
	commands = append(commands, recall.Commands(core)...)
	commands = append(commands, harness.Commands(core)...)
	commands = append(commands, forest.Commands(core)...)
	return []cliapp.CommandGroup{{Title: "Memory", Commands: commands}}
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
// See docs/concepts/ARCHITECTURE.md "Domain modules" for the canonical
// pattern when swapping the example domain for your scenario's first
// domain.
//
// For API-backed commands the manifest carries the declarative surface
// (governance, flags, positionals, RPC binding). Handlers stay in
// handlers.go and are wired via the bindings map; refer to
// templates/scenarios/react-vite/docs/internal/SEAMS.md (manifest ↔
// handlers bindings seam) for the contract.
func SubcommandGroups(core *cliapp.ScenarioApp, manifest []byte) ([]cliapp.SubcommandGroup, error) {
	groups := []cliapp.SubcommandGroup{}
	journalGroup, err := journal.Register(core, manifest)
	if err != nil {
		return nil, err
	}
	harnessGroup, err := harness.Register(core, manifest)
	if err != nil {
		return nil, err
	}
	recallGroup, err := recall.Register(core, manifest)
	if err != nil {
		return nil, err
	}
	forestGroup, err := forest.Register(core, manifest)
	if err != nil {
		return nil, err
	}
	facetsGroup, err := facets.Register(core, manifest)
	if err != nil {
		return nil, err
	}
	rulesGroup, err := rules.Register(core, manifest)
	if err != nil {
		return nil, err
	}
	scopesGroup, err := scopes.Register(core, manifest)
	if err != nil {
		return nil, err
	}
	groups = append(groups, journalGroup, harnessGroup, recallGroup, forestGroup, rulesGroup, facetsGroup, scopesGroup)
	return groups, nil
}
