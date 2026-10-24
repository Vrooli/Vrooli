package domains

import (
	"device-studio/cli/domains/creation"
	"device-studio/cli/domains/devices"
	"device-studio/cli/domains/library"
	"device-studio/cli/domains/media"
	"device-studio/cli/domains/motions"
	"device-studio/cli/domains/physicalsetups"
	"device-studio/cli/domains/projects"
	"device-studio/cli/domains/renders"
	"device-studio/cli/domains/scenes"
	"device-studio/cli/domains/shots"

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
	projectsGroup, err := projects.Register(core, manifest)
	if err != nil {
		return nil, err
	}
	groups = append(groups, projectsGroup)
	mediaGroup, err := media.Register(core, manifest)
	if err != nil {
		return nil, err
	}
	groups = append(groups, mediaGroup)
	shotsGroup, err := shots.Register(core, manifest)
	if err != nil {
		return nil, err
	}
	groups = append(groups, shotsGroup)
	rendersGroup, err := renders.Register(core, manifest)
	if err != nil {
		return nil, err
	}
	groups = append(groups, rendersGroup)
	for _, register := range []func(*cliapp.ScenarioApp, []byte) (cliapp.SubcommandGroup, error){devices.Register, scenes.Register, motions.Register, physicalsetups.Register, library.Register, creation.Register} {
		group, err := register(core, manifest)
		if err != nil {
			return nil, err
		}
		groups = append(groups, group)
	}
	return groups, nil
}
