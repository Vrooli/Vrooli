// Package deps is the CLI surface for the fleet dependency & vulnerability
// intelligence index. It mirrors the API's DependencyService.
package deps

import (
	"fmt"

	"github.com/vrooli/cli-core/cliapp"
)

// GroupName is the manifest group this package owns.
const GroupName = "deps"

// Register builds the deps subcommand group from the embedded manifest.
func Register(core *cliapp.ScenarioApp, manifest []byte) (cliapp.SubcommandGroup, error) {
	h := newHandlers(core)
	bindings := map[string]func(cliapp.RunContext) error{
		"DependencyService.Search": h.search,
		"DependencyService.Status": h.status,
	}
	group, err := cliapp.LoadFromManifest(manifest, GroupName, bindings)
	if err != nil {
		return cliapp.SubcommandGroup{}, fmt.Errorf("deps: load from manifest: %w", err)
	}
	return group, nil
}
