package domains

import (
	"infrastructure-manager/cli/domains/condition"
	"infrastructure-manager/cli/domains/coverage"
	"infrastructure-manager/cli/domains/focus"
	"infrastructure-manager/cli/domains/ladder"
	"infrastructure-manager/cli/domains/portability"

	"github.com/vrooli/api-core/spacecli"
	"github.com/vrooli/api-core/spacedoc"
	"github.com/vrooli/cli-core/cliapp"
)

// CommandGroups aggregates flat command groups from domain packages.
//
// This scenario is the reliability instrument, and it is also the interim
// space owner for the two projections whose control layer has no scenario to
// hold them: `capacity` (`vrooli capacity`) and `commissioning` (`vrooli
// setup`). Registering `spacecli` here is what makes those two denominators
// readable through the same typed verb every other owner exposes, rather than
// only as a file path this scenario happens to know. The other nine
// projections are registered by their own owners and are deliberately refused
// here — an instrument that could serve another layer's space could also
// change it.
func CommandGroups(core *cliapp.ScenarioApp) []cliapp.CommandGroup {
	_ = core
	return []cliapp.CommandGroup{
		spacecli.CommandGroup(spacecli.Config{Owner: "infrastructure-manager", Projections: []spacedoc.Projection{
			spacedoc.ProjectionCapacity,
			spacedoc.ProjectionCommissioning,
		}}),
	}
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
	conditionGroup, err := condition.Register(core, manifest)
	if err != nil {
		return nil, err
	}
	groups = append(groups, conditionGroup)
	focusGroup, err := focus.Register(core, manifest)
	if err != nil {
		return nil, err
	}
	groups = append(groups, focusGroup)
	coverageGroup, err := coverage.Register(core, manifest)
	if err != nil {
		return nil, err
	}
	groups = append(groups, coverageGroup)
	portabilityGroup, err := portability.Register(core, manifest)
	if err != nil {
		return nil, err
	}
	groups = append(groups, portabilityGroup)
	ladderGroup, err := ladder.Register(core, manifest)
	if err != nil {
		return nil, err
	}
	groups = append(groups, ladderGroup)
	return groups, nil
}
