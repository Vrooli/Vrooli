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
func SubcommandGroups(core *cliapp.ScenarioApp, manifestBytes []byte) ([]cliapp.SubcommandGroup, error) {
	registrars := []func(*cliapp.ScenarioApp, []byte) (cliapp.SubcommandGroup, error){
		artifacts.Register,
		flows.Register,
		runs.Register,
		scenarios.Register,
		settings.Register,
		verify.Register,
	}
	groups := make([]cliapp.SubcommandGroup, 0, len(registrars))
	for _, r := range registrars {
		g, err := r(core, manifestBytes)
		if err != nil {
			return nil, err
		}
		groups = append(groups, g)
	}
	return groups, nil
}
