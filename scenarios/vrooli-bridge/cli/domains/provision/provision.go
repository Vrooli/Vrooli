// Package provision is the CLI's provision-domain command surface. Mirrors the
// API's Connect-RPC ProvisionService: sync a node to a revision (privileged),
// inspect ops, block-once wait (no polling), and read a node's version. The
// manifest (cli/manifest.json) is the single source of truth for the command
// shape; handlers.go binds the RPCs.
package provision

import (
	"fmt"

	"github.com/vrooli/cli-core/cliapp"
)

// GroupName is the manifest group name this package owns.
const GroupName = "provision"

// Register builds the provision subcommand group from the embedded manifest and
// wires Connect-RPC bindings to handlers in handlers.go. ReportProvisionEvent is
// node-facing and intentionally absent (omitted in the manifest).
func Register(core *cliapp.ScenarioApp, manifest []byte) (cliapp.SubcommandGroup, error) {
	h := newHandlers(core)
	bindings := map[string]func(cliapp.RunContext) error{
		"ProvisionService.SyncToRevision":      h.sync,
		"ProvisionService.GetProvisioningOp":   h.get,
		"ProvisionService.ListProvisioningOps": h.list,
		"ProvisionService.WaitProvisioningOp":  h.wait,
		"ProvisionService.GetNodeVersion":      h.version,
	}
	group, err := cliapp.LoadFromManifest(manifest, GroupName, bindings)
	if err != nil {
		return cliapp.SubcommandGroup{}, fmt.Errorf("provision: load from manifest: %w", err)
	}
	return group, nil
}
