// Package discovery is the CLI's discovery-domain command surface. Mirrors the
// API's Connect-RPC DiscoveryService. Operators run `data-backup-manager
// discovery targets` / `... discovery destinations` to see onboarding
// suggestions, and `... discovery dismiss --id <id>` to hide one.
//
// There is intentionally NO "accept" command: accepting a suggestion is just
// `targets register ...` / `destinations create ...` with the values the
// suggestion prints, which keeps a single source of truth and reuses their
// validation (separate-root rule, encryption-on).
//
// The manifest (cli/manifest.json) is the single source of truth for the
// command shape; this package only wires bindings to handlers in handlers.go.
package discovery

import (
	"fmt"

	"github.com/vrooli/cli-core/cliapp"
)

// GroupName is the manifest group this package owns.
const GroupName = "discovery"

// Register builds the discovery subcommand group from the embedded manifest and
// wires Connect-RPC bindings to handlers.
func Register(core *cliapp.ScenarioApp, manifest []byte) (cliapp.SubcommandGroup, error) {
	h := newHandlers(core)
	bindings := map[string]func(cliapp.RunContext) error{
		"DiscoveryService.ListTargetSuggestions":      h.targets,
		"DiscoveryService.ListDestinationSuggestions": h.destinations,
		"DiscoveryService.DismissSuggestion":          h.dismiss,
	}
	group, err := cliapp.LoadFromManifest(manifest, GroupName, bindings)
	if err != nil {
		return cliapp.SubcommandGroup{}, fmt.Errorf("discovery: load from manifest: %w", err)
	}
	return group, nil
}
