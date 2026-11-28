package domains

import (
	"react-component-library/cli/domains/activity"
	"react-component-library/cli/domains/adoptions"
	"react-component-library/cli/domains/asset"
	"react-component-library/cli/domains/catalog"
	"react-component-library/cli/domains/components"
	"react-component-library/cli/domains/health"
	"react-component-library/cli/domains/page"
	"react-component-library/cli/domains/preview"
	"react-component-library/cli/domains/review"
	"react-component-library/cli/domains/search"
	"react-component-library/cli/domains/sketch"
	"react-component-library/cli/domains/versions"
	"react-component-library/cli/domains/workflows"
	"react-component-library/cli/domains/workspaces"

	"github.com/vrooli/cli-core/cliapp"
)

// SubcommandGroups aggregates hierarchical command groups from domain packages.
// Each domain package's Register(core, manifest) builds its SubcommandGroup
// from the embedded cli/manifest.json via cliapp.LoadFromManifest.
func SubcommandGroups(core *cliapp.ScenarioApp, manifestBytes []byte) ([]cliapp.SubcommandGroup, error) {
	registrars := []func(*cliapp.ScenarioApp, []byte) (cliapp.SubcommandGroup, error){
		activity.Register,
		adoptions.Register,
		asset.Register,
		components.Register,
		preview.Register,
		page.Register,
		review.Register,
		search.Register,
		sketch.Register,
		versions.Register,
		workflows.Register,
		catalog.Register,
		health.Register,
		workspaces.Register,
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
