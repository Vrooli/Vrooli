// Package targets is the CLI's targets-domain command surface. Mirrors the
// API's Connect-RPC TargetsService. Scenarios call `data-backup-manager
// targets register ...` at their lifecycle to self-register backup sources;
// operators use list/get/deregister to inspect and manage the catalog.
//
// The manifest (cli/manifest.json) is the single source of truth for the
// command shape (flags, positionals, governance, RPC bindings); this package
// only wires bindings to handlers in handlers.go.
package targets

import (
	"fmt"

	"github.com/vrooli/cli-core/cliapp"
)

// GroupName is the manifest group this package owns.
const GroupName = "targets"

// Register builds the targets subcommand group from the embedded manifest and
// wires Connect-RPC bindings to handlers.
func Register(core *cliapp.ScenarioApp, manifest []byte) (cliapp.SubcommandGroup, error) {
	h := newHandlers(core)
	bindings := map[string]func(cliapp.RunContext) error{
		"TargetsService.RegisterTarget":   h.register,
		"TargetsService.DeregisterTarget": h.deregister,
		"TargetsService.GetTarget":        h.get,
		"TargetsService.ListTargets":      h.list,
	}
	group, err := cliapp.LoadFromManifest(manifest, GroupName, bindings)
	if err != nil {
		return cliapp.SubcommandGroup{}, fmt.Errorf("targets: load from manifest: %w", err)
	}
	return group, nil
}
