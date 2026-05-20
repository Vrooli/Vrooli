package domains

import (
	"github.com/vrooli/cli-core/cliapp"

	"cli-health/cli/domains/reindex"
	"cli-health/cli/domains/search"
	"cli-health/cli/domains/validate"
)

// CommandGroups aggregates flat command groups from domain packages.
//
// Keep app.go focused on CLI metadata and cli-core wiring. As the scenario
// grows, add domains and append their registrations here.
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
func SubcommandGroups(core *cliapp.ScenarioApp, manifest []byte) ([]cliapp.SubcommandGroup, error) {
	out := make([]cliapp.SubcommandGroup, 0, 3)
	for _, reg := range []func(*cliapp.ScenarioApp, []byte) (cliapp.SubcommandGroup, error){
		validate.Register,
		search.Register,
		reindex.Register,
	} {
		group, err := reg(core, manifest)
		if err != nil {
			return nil, err
		}
		out = append(out, group)
	}
	return out, nil
}
