// Package fleet is the CLI's fleet-domain command surface. Mirrors the API's
// Connect-RPC FleetService: roll the whole fleet (or a subset) to a target
// revision and inspect rollouts. The manifest (cli/manifest.json) is the single
// source of truth for the command shape; handlers.go binds the RPCs.
package fleet

import (
	"fmt"

	"github.com/vrooli/cli-core/cliapp"
)

// GroupName is the manifest group name this package owns.
const GroupName = "fleet"

// Register builds the fleet subcommand group from the embedded manifest and
// wires Connect-RPC bindings to handlers in handlers.go.
func Register(core *cliapp.ScenarioApp, manifest []byte) (cliapp.SubcommandGroup, error) {
	h := newHandlers(core)
	bindings := map[string]func(cliapp.RunContext) error{
		"FleetService.RollFleet":    h.roll,
		"FleetService.GetRollout":   h.get,
		"FleetService.ListRollouts": h.list,
	}
	group, err := cliapp.LoadFromManifest(manifest, GroupName, bindings)
	if err != nil {
		return cliapp.SubcommandGroup{}, fmt.Errorf("fleet: load from manifest: %w", err)
	}
	return group, nil
}
