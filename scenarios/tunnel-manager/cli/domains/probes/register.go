// Package probes is the CLI's probes-domain command surface. Mirrors the
// API's Connect-RPC ProbesService and the UI's api/probes.ts client.
//
// Follows the canonical domain shape: a Register(core, manifest) returning
// a cliapp.SubcommandGroup built from cli/manifest.json via
// cliapp.LoadFromManifest, plus one handler per Connect-RPC subcommand in
// handlers.go. The manifest is the single source of truth for the
// command-line shape.
package probes

import (
	"fmt"

	"github.com/vrooli/cli-core/cliapp"
)

// GroupName is the manifest group name this package owns.
const GroupName = "probes"

// Register builds the probes subcommand group from the embedded manifest
// and wires Connect-RPC bindings to handlers in handlers.go.
func Register(core *cliapp.ScenarioApp, manifest []byte) (cliapp.SubcommandGroup, error) {
	h := newHandlers(core)
	bindings := map[string]func(cliapp.RunContext) error{
		"ProbesService.RunProbes":  h.run,
		"ProbesService.ListProbes": h.history,
		"ProbesService.Classify":   h.classify,
	}
	group, err := cliapp.LoadFromManifest(manifest, GroupName, bindings)
	if err != nil {
		return cliapp.SubcommandGroup{}, fmt.Errorf("probes: load from manifest: %w", err)
	}
	return group, nil
}
