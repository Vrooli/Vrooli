package domains

import (
	"github.com/vrooli/cli-core/cliapp"
)

// ManifestCommandGroups and ManifestSubcommandGroups are used by the app
// constructor. They keep manifest parsing in one place while preserving the
// cli-core callback shape required during ScenarioApp construction.
func ManifestCommandGroups(core *cliapp.ScenarioApp, manifest []byte) ([]cliapp.CommandGroup, []cliapp.SubcommandGroup, error) {
	return LoadManifestGroups(core, manifest)
}
