package fleet

import (
	"fmt"

	"github.com/vrooli/cli-core/cliapp"
)

// GroupName is the manifest group this domain binds to.
const GroupName = "fleet"

// Register wires the `fleet` group's manifest commands to their handlers.
func Register(core *cliapp.ScenarioApp, manifest []byte) (cliapp.SubcommandGroup, error) {
	h := newHandlers(core)
	group, err := cliapp.LoadFromManifest(manifest, GroupName, map[string]func(cliapp.RunContext) error{
		"FleetService.ScanFleet": h.scan,
	})
	if err != nil {
		return cliapp.SubcommandGroup{}, fmt.Errorf("fleet: load from manifest: %w", err)
	}
	return group, nil
}
