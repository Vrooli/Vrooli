// Package sweep is the CLI's sweep-domain command surface. It mirrors the API's
// SweepService: run the out-of-band per-flow capture cadence for a scenario.
package sweep

import (
	"fmt"

	"github.com/vrooli/cli-core/cliapp"
)

// GroupName is the manifest group name this package owns.
const GroupName = "sweep"

// Register builds the sweep subcommand group from the embedded manifest.
func Register(core *cliapp.ScenarioApp, manifest []byte) (cliapp.SubcommandGroup, error) {
	h := newHandlers(core)
	bindings := map[string]func(cliapp.RunContext) error{
		"SweepService.RunSweep": h.run,
	}
	group, err := cliapp.LoadFromManifest(manifest, GroupName, bindings)
	if err != nil {
		return cliapp.SubcommandGroup{}, fmt.Errorf("sweep: load from manifest: %w", err)
	}
	return group, nil
}
