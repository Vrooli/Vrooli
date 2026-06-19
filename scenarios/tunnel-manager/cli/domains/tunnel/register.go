// Package tunnel is the CLI's tunnel-domain command surface. Mirrors the
// API's Connect-RPC TunnelService and the UI's metrics feature client.
//
// Follows the canonical domain shape: a Register(core, manifest) returning a
// cliapp.SubcommandGroup built from cli/manifest.json via
// cliapp.LoadFromManifest, plus one handler per Connect-RPC subcommand in
// handlers.go. The manifest is the single source of truth for the command-line
// shape.
package tunnel

import (
	"fmt"

	"github.com/vrooli/cli-core/cliapp"
)

// GroupName is the manifest group name this package owns.
const GroupName = "tunnel"

// Register builds the tunnel subcommand group from the embedded manifest and
// wires Connect-RPC bindings to handlers in handlers.go.
func Register(core *cliapp.ScenarioApp, manifest []byte) (cliapp.SubcommandGroup, error) {
	h := newHandlers(core)
	bindings := map[string]func(cliapp.RunContext) error{
		"TunnelService.GetStatus":   h.status,
		"TunnelService.ListMetrics": h.metrics,
		"TunnelService.Scrape":      h.scrape,
	}
	group, err := cliapp.LoadFromManifest(manifest, GroupName, bindings)
	if err != nil {
		return cliapp.SubcommandGroup{}, fmt.Errorf("tunnel: load from manifest: %w", err)
	}
	return group, nil
}
