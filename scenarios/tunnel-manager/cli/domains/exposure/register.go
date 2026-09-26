// Package exposure is the CLI's exposure-domain command surface. Mirrors
// the API's Connect-RPC ExposureService and the UI's exposure feature.
//
// Follows the canonical domain shape: Register(core, manifest) returns a
// cliapp.SubcommandGroup built from cli/manifest.json via
// cliapp.LoadFromManifest, plus one handler per Connect-RPC subcommand.
package exposure

import (
	"fmt"

	"github.com/vrooli/cli-core/cliapp"
)

// GroupName is the manifest group name this package owns.
const GroupName = "exposure"

// Register builds the exposure subcommand group from the embedded manifest
// and wires Connect-RPC bindings to handlers in handlers.go.
func Register(core *cliapp.ScenarioApp, manifest []byte) (cliapp.SubcommandGroup, error) {
	h := newHandlers(core)
	bindings := map[string]cliapp.PrimitiveHandler{
		"ExposureService.Expose":        cliapp.ProtoMutation(h.exposeCall, h.exposeReport),
		"ExposureService.ExtendLease":   cliapp.ProtoMutation(h.extendCall, h.extendReport),
		"ExposureService.RevokeLease":   cliapp.ProtoMutation(h.revokeCall, h.revokeReport),
		"ExposureService.Unexpose":      cliapp.ProtoMutation(h.unexposeCall, h.unexposeReport),
		"ExposureService.ListLeases":    cliapp.ProtoList(h.leasesCall, h.leasesReport),
		"ExposureService.ListExposures": cliapp.ProtoList(h.listCall, h.listReport),
		"ExposureService.IsExposed":     cliapp.ProtoList(h.checkCall, h.checkReport),
		"ExposureService.Reconcile":     cliapp.ProtoMutation(h.reconcileCall, h.reconcileReport),
	}
	group, err := cliapp.LoadFromManifestPrimitives(manifest, GroupName, bindings)
	if err != nil {
		return cliapp.SubcommandGroup{}, fmt.Errorf("exposure: load from manifest: %w", err)
	}
	return group, nil
}
