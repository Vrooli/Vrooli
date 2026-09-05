// Package nodes is the CLI's registry-domain command surface. Mirrors the
// API's Connect-RPC NodeRegistryService and the UI's api/nodes.ts client.
//
// New domain packages copy this shape: a Register(core, manifest) returning a
// cliapp.SubcommandGroup built from cli/manifest.json via
// cliapp.LoadFromManifest, plus one handler per Connect-RPC subcommand in
// handlers.go. The manifest is the SINGLE source of truth for the command-line
// shape (governance, flags, positionals, RPC bindings).
package nodes

import (
	"fmt"

	"github.com/vrooli/cli-core/cliapp"
)

// GroupName is the manifest group name this package owns.
const GroupName = "nodes"

// Register builds the nodes subcommand group from the embedded manifest and
// wires Connect-RPC bindings to handlers in handlers.go.
func Register(core *cliapp.ScenarioApp, manifest []byte) (cliapp.SubcommandGroup, error) {
	h := newHandlers(core)
	bindings := map[string]func(cliapp.RunContext) error{
		"NodeRegistryService.RegisterNode":     h.register,
		"NodeRegistryService.ListNodes":        h.list,
		"NodeRegistryService.GetNode":          h.get,
		"NodeRegistryService.GetNodeReadiness": h.doctor,
		"NodeRegistryService.UpdateNode":       h.update,
		"NodeRegistryService.RevokeNode":       h.revoke,
		"NodeRegistryService.RemoveNode":       h.remove,
	}
	group, err := cliapp.LoadFromManifest(manifest, GroupName, bindings)
	if err != nil {
		return cliapp.SubcommandGroup{}, fmt.Errorf("nodes: load from manifest: %w", err)
	}
	return group, nil
}
