// Package gate is the CLI's gate-domain command surface. Mirrors the API's
// Connect-RPC GateService: run a cross-OS deployment gate (validate a scenario
// natively on one node per target OS and aggregate one verdict), then inspect /
// block on it. The manifest (cli/manifest.json) is the single source of truth
// for the command shape; handlers.go binds the RPCs.
package gate

import (
	"fmt"

	"github.com/vrooli/cli-core/cliapp"
)

// GroupName is the manifest group name this package owns.
const GroupName = "gate"

// Register builds the gate subcommand group from the embedded manifest and wires
// Connect-RPC bindings to handlers in handlers.go.
func Register(core *cliapp.ScenarioApp, manifest []byte) (cliapp.SubcommandGroup, error) {
	h := newHandlers(core)
	bindings := map[string]func(cliapp.RunContext) error{
		"GateService.RunGate":   h.run,
		"GateService.GetGate":   h.get,
		"GateService.WaitGate":  h.wait,
		"GateService.ListGates": h.list,
	}
	group, err := cliapp.LoadFromManifest(manifest, GroupName, bindings)
	if err != nil {
		return cliapp.SubcommandGroup{}, fmt.Errorf("gate: load from manifest: %w", err)
	}
	return group, nil
}
