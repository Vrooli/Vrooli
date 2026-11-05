// Package pairing is the CLI's pairing-domain command surface (`pair …`),
// mirroring the API's Connect-RPC PairingService. The owner commands (issue/
// approve/list) require an owner token; the node-facing commands (redeem/
// request) are what a node's bootstrap installer runs to join the fleet.
package pairing

import (
	"fmt"

	"github.com/vrooli/cli-core/cliapp"
)

// GroupName is the manifest group name this package owns.
const GroupName = "pair"

// Register builds the pair subcommand group from the embedded manifest and
// wires Connect-RPC bindings to handlers in handlers.go.
func Register(core *cliapp.ScenarioApp, manifest []byte) (cliapp.SubcommandGroup, error) {
	h := newHandlers(core)
	bindings := map[string]func(cliapp.RunContext) error{
		"PairingService.IssuePairingCode":    h.issue,
		"PairingService.RedeemPairingCode":   h.redeem,
		"PairingService.RequestPairing":      h.request,
		"PairingService.ApprovePairing":      h.approve,
		"PairingService.ListPairingRequests": h.list,
	}
	group, err := cliapp.LoadFromManifest(manifest, GroupName, bindings)
	if err != nil {
		return cliapp.SubcommandGroup{}, fmt.Errorf("pair: load from manifest: %w", err)
	}
	return group, nil
}
