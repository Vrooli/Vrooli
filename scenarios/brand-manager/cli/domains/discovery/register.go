// Package discovery is the CLI's discovery-domain command surface. Mirrors the
// API's Connect-RPC DiscoveryService and the UI's api/discovery.ts client.
//
// The manifest (cli/manifest.json) carries the declarative surface (governance,
// flags, RPC bindings) and is the SINGLE source of truth for the command-line
// shape; handlers in handlers.go are wired via the bindings map.
package discovery

import (
	"fmt"

	"github.com/vrooli/cli-core/cliapp"
)

// GroupName is the manifest group name this package owns.
const GroupName = "discovery"

// Register builds the discovery subcommand group from the embedded manifest and
// wires Connect-RPC bindings to handlers in handlers.go.
func Register(core *cliapp.ScenarioApp, manifest []byte) (cliapp.SubcommandGroup, error) {
	h := newHandlers(core)
	bindings := map[string]func(cliapp.RunContext) error{
		"DiscoveryService.DiscoverScenario": h.scan,
		"DiscoveryService.ImportBrand":      h.importBrand,
	}
	group, err := cliapp.LoadFromManifest(manifest, GroupName, bindings)
	if err != nil {
		return cliapp.SubcommandGroup{}, fmt.Errorf("discovery: load from manifest: %w", err)
	}
	return group, nil
}
