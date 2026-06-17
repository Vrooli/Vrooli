// Package devices is the CLI's devices-domain command surface. It mirrors the
// DevicesService Connect-RPC service: the owner's view of their trust group
// (list/get/pair/approve/rename/revoke) plus the two NEW-device join paths
// (redeem a pairing code, or request approval).
//
// The package follows the canonical domain shape (mirrored by
// cli/domains/transfer): a
// Register(core, manifest) that builds a cliapp.SubcommandGroup from
// cli/manifest.json via cliapp.LoadFromManifest, plus one handler per
// Connect-RPC subcommand in handlers.go. The manifest is the single source of
// truth for the command-line shape (governance, flags, positionals, RPC
// bindings); handlers carry only the call + render logic.
//
// Auth: owner-authed RPCs (list/get/pair/approve/rename/revoke) ride the owner
// JWT that cli-core attaches from the configured token source. The two join
// RPCs (redeem/request) are deliberately open — a device joining the trust
// group has no owner token yet — and `redeem`/`request` print the one-time hub
// device token the caller then exports as $DEVICE_SYNC_HUB_DEVICE_TOKEN for
// subsequent `transfer` commands.
package devices

import (
	"fmt"

	"github.com/vrooli/cli-core/cliapp"
)

// GroupName is the manifest group name this package owns. Exported so the
// package's tests can call cliapp.RequireProtoServiceCoverage against the same
// manifest the runtime loads.
const GroupName = "devices"

// Register builds the devices subcommand group from the embedded manifest and
// wires Connect-RPC bindings to handlers in handlers.go.
func Register(core *cliapp.ScenarioApp, manifest []byte) (cliapp.SubcommandGroup, error) {
	h := newHandlers(core)
	bindings := map[string]func(cliapp.RunContext) error{
		"DevicesService.ListDevices":       h.list,
		"DevicesService.GetDevice":         h.get,
		"DevicesService.IssuePairingCode":  h.pair,
		"DevicesService.RedeemPairingCode": h.redeem,
		"DevicesService.RequestPairing":    h.request,
		"DevicesService.ApprovePairing":    h.approve,
		"DevicesService.RenameDevice":      h.rename,
		"DevicesService.RevokeDevice":      h.revoke,
	}
	group, err := cliapp.LoadFromManifest(manifest, GroupName, bindings)
	if err != nil {
		return cliapp.SubcommandGroup{}, fmt.Errorf("devices: load from manifest: %w", err)
	}
	return group, nil
}
