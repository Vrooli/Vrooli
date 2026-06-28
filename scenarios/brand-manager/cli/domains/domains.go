package domains

import (
	"brand-manager/cli/domains/apply"
	"brand-manager/cli/domains/assets"
	"brand-manager/cli/domains/assignments"
	"brand-manager/cli/domains/brands"
	"brand-manager/cli/domains/design"
	"brand-manager/cli/domains/discovery"
	"brand-manager/cli/domains/generation"

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
	brandsGroup, err := brands.Register(core, manifest)
	if err != nil {
		return nil, err
	}
	groups = append(groups, brandsGroup)
	assignmentsGroup, err := assignments.Register(core, manifest)
	if err != nil {
		return nil, err
	}
	groups = append(groups, assignmentsGroup)
	assetsGroup, err := assets.Register(core, manifest)
	if err != nil {
		return nil, err
	}
	groups = append(groups, assetsGroup)
	generationGroup, err := generation.Register(core, manifest)
	if err != nil {
		return nil, err
	}
	groups = append(groups, generationGroup)
	applyGroup, err := apply.Register(core, manifest)
	if err != nil {
		return nil, err
	}
	groups = append(groups, applyGroup)
	discoveryGroup, err := discovery.Register(core, manifest)
	if err != nil {
		return nil, err
	}
	groups = append(groups, discoveryGroup)
	designGroup, err := design.Register(core, manifest)
	if err != nil {
		return nil, err
	}
	groups = append(groups, designGroup)
	return groups, nil
}
