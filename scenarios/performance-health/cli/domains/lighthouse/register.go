// Package lighthouse is the CLI's lighthouse-domain command surface. It mirrors
// the API's LighthouseService: score a scenario's UI with Lighthouse.
package lighthouse

import (
	"fmt"

	"github.com/vrooli/cli-core/cliapp"
)

// GroupName is the manifest group name this package owns.
const GroupName = "lighthouse"

// Register builds the lighthouse subcommand group from the embedded manifest.
func Register(core *cliapp.ScenarioApp, manifest []byte) (cliapp.SubcommandGroup, error) {
	h := newHandlers(core)
	bindings := map[string]func(cliapp.RunContext) error{
		"LighthouseService.RunLighthouse": h.run,
	}
	group, err := cliapp.LoadFromManifest(manifest, GroupName, bindings)
	if err != nil {
		return cliapp.SubcommandGroup{}, fmt.Errorf("lighthouse: load from manifest: %w", err)
	}
	return group, nil
}
