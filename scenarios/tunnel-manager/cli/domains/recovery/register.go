// Package recovery is the CLI's recovery-domain command surface. Mirrors
// the API's Connect-RPC RecoveryService and the UI's recovery feature.
//
// Follows the canonical domain shape: Register(core, manifest) returns a
// cliapp.SubcommandGroup built from cli/manifest.json via
// cliapp.LoadFromManifest, plus one handler per Connect-RPC subcommand in
// handlers.go.
package recovery

import (
	"fmt"

	"github.com/vrooli/cli-core/cliapp"
)

// GroupName is the manifest group name this package owns.
const GroupName = "recovery"

// Register builds the recovery subcommand group from the embedded
// manifest and wires Connect-RPC bindings to handlers in handlers.go.
func Register(core *cliapp.ScenarioApp, manifest []byte) (cliapp.SubcommandGroup, error) {
	h := newHandlers(core)
	bindings := map[string]func(cliapp.RunContext) error{
		"RecoveryService.GetState":   h.state,
		"RecoveryService.ListEvents": h.events,
		"RecoveryService.Recover":    h.recover,
	}
	group, err := cliapp.LoadFromManifest(manifest, GroupName, bindings)
	if err != nil {
		return cliapp.SubcommandGroup{}, fmt.Errorf("recovery: load from manifest: %w", err)
	}
	return group, nil
}
