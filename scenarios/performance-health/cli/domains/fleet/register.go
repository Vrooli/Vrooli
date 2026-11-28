// Package fleet is the CLI's fleet-domain command surface. It mirrors the API's
// FleetService: deterministic, structured performance-offender queries.
package fleet

import (
	"fmt"

	"github.com/vrooli/cli-core/cliapp"
)

// GroupName is the manifest group name this package owns.
const GroupName = "fleet"

// Register builds the fleet subcommand group from the embedded manifest.
func Register(core *cliapp.ScenarioApp, manifest []byte) (cliapp.SubcommandGroup, error) {
	h := newHandlers(core)
	bindings := map[string]func(cliapp.RunContext) error{
		"FleetService.ScanFleet": h.scan,
	}
	group, err := cliapp.LoadFromManifest(manifest, GroupName, bindings)
	if err != nil {
		return cliapp.SubcommandGroup{}, fmt.Errorf("fleet: load from manifest: %w", err)
	}
	return group, nil
}
