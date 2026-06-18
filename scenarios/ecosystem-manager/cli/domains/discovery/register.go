// Package discovery is the CLI's discovery-domain command surface. It mirrors
// the API's Connect-RPC DiscoveryService and the UI's api/discovery.ts client,
// consuming the generated Connect client over the shared cli-core HTTP client.
//
// The command surface (resources / scenarios + flags + governance) is declared
// in cli/manifest.json — the single source of truth — and loaded here via
// cliapp.LoadFromManifest. This package only wires the two proto-bound
// subcommands to their handlers in handlers.go.
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
		"DiscoveryService.ListResources": h.resources,
		"DiscoveryService.ListScenarios": h.scenarios,
	}
	group, err := cliapp.LoadFromManifest(manifest, GroupName, bindings)
	if err != nil {
		return cliapp.SubcommandGroup{}, fmt.Errorf("discovery: load from manifest: %w", err)
	}
	return group, nil
}
