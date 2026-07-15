package domains

import (
	"react-component-library/cli/domains/adoptions"
	"react-component-library/cli/domains/components"
	"react-component-library/cli/domains/preview"
	"react-component-library/cli/domains/versions"
	"react-component-library/cli/domains/workflows"

	"github.com/vrooli/cli-core/cliapp"
)

func CommandGroups(core *cliapp.ScenarioApp) []cliapp.CommandGroup {
	_ = core
	return nil
}

// SubcommandGroups aggregates hierarchical command groups from domain packages.
// Each domain package's Register(core, manifest) builds its SubcommandGroup
// from the embedded cli/manifest.json via cliapp.LoadFromManifest.
func SubcommandGroups(core *cliapp.ScenarioApp, manifestBytes []byte) ([]cliapp.SubcommandGroup, error) {
	registrars := []func(*cliapp.ScenarioApp, []byte) (cliapp.SubcommandGroup, error){
		adoptions.Register,
		components.Register,
		preview.Register,
		versions.Register,
		workflows.Register,
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
