package domains

import (
	"performance-health/cli/domains/analysis"
	"performance-health/cli/domains/audit"
	"performance-health/cli/domains/benchmark"
	"performance-health/cli/domains/budget"
	"performance-health/cli/domains/fleet"
	"performance-health/cli/domains/lighthouse"
	"performance-health/cli/domains/readiness"
	"performance-health/cli/domains/startup"
	"performance-health/cli/domains/sweep"
	"performance-health/cli/domains/trend"

	"github.com/vrooli/cli-core/cliapp"
)

// registrars is every domain's Register function, in stable (alphabetical)
// order. Adding a domain = one line here.
var registrars = []func(*cliapp.ScenarioApp, []byte) (cliapp.SubcommandGroup, error){
	analysis.Register,
	audit.Register,
	benchmark.Register,
	budget.Register,
	fleet.Register,
	lighthouse.Register,
	readiness.Register,
	startup.Register,
	sweep.Register,
	trend.Register,
}

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
	groups := make([]cliapp.SubcommandGroup, 0, len(registrars))
	for _, register := range registrars {
		group, err := register(core, manifest)
		if err != nil {
			return nil, err
		}
		groups = append(groups, group)
	}
	return groups, nil
}
