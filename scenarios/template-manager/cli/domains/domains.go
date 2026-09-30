package domains

import (
	"template-manager/cli/domains/debt"
	"template-manager/cli/domains/guidance"
	"template-manager/cli/domains/lifecycle"
	"template-manager/cli/domains/measures"
	"template-manager/cli/domains/monitor"
	"template-manager/cli/domains/registry"
	"template-manager/cli/domains/resourcetemplate"
	"template-manager/cli/domains/runs"

	"github.com/vrooli/cli-core/cliapp"
)

// CommandGroups aggregates flat command groups from domain packages.
//
// Keep app.go focused on CLI metadata and cli-core wiring. As the scenario
// grows, add domains like domains/tasks or domains/projects and append their
// registrations here. For greenfield scenarios, domain packages are the
// default architecture; do not treat flat command files as the long-term plan.
func CommandGroups(core *cliapp.ScenarioApp) []cliapp.CommandGroup {
	return lifecycle.CommandGroups(core)
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
	groups = append(groups, lifecycle.SubcommandGroups(core)...)
	guidanceGroup, err := guidance.Register(core, manifest)
	if err != nil {
		return nil, err
	}
	groups = append(groups, guidanceGroup)
	measuresGroup, err := measures.Register(core, manifest)
	if err != nil {
		return nil, err
	}
	groups = append(groups, measuresGroup)
	monitorGroup, err := monitor.Register(core, manifest)
	if err != nil {
		return nil, err
	}
	groups = append(groups, monitorGroup)
	registryGroup, err := registry.Register(core, manifest)
	if err != nil {
		return nil, err
	}
	groups = append(groups, registryGroup)
	resourceTemplateGroup, err := resourcetemplate.Register(core, manifest)
	if err != nil {
		return nil, err
	}
	groups = append(groups, resourceTemplateGroup)
	runsGroup, err := runs.Register(core, manifest)
	if err != nil {
		return nil, err
	}
	groups = append(groups, runsGroup)
	debtGroup, err := debt.Register(core, manifest)
	if err != nil {
		return nil, err
	}
	groups = append(groups, debtGroup)
	return groups, nil
}
