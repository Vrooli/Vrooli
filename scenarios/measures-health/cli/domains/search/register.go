// Package search is the CLI's measures-index command surface. It mirrors the
// API's Connect-RPC SearchService: `measures-health search query <question>`
// matches an analytical question to a measure and returns the computed answer
// (the same federated provider search-hub routes analytical questions to), and
// `measures-health search status` reports index/backend availability.
//
// The manifest (cli/manifest.json, group "search") is the single source of truth
// for the command-line shape (flags, positionals, governance, RPC binding);
// handlers live in handlers.go and are wired via the bindings map.
package search

import (
	"fmt"

	"github.com/vrooli/cli-core/cliapp"
)

// GroupName is the manifest group name this package owns.
const GroupName = "search"

// Register builds the search subcommand group from the embedded manifest and
// wires its Connect-RPC bindings to handlers.
func Register(core *cliapp.ScenarioApp, manifest []byte) (cliapp.SubcommandGroup, error) {
	h := newHandlers(core)
	bindings := map[string]func(cliapp.RunContext) error{
		"SearchService.Search": h.query,
		"SearchService.Status": h.status,
	}
	group, err := cliapp.LoadFromManifest(manifest, GroupName, bindings)
	if err != nil {
		return cliapp.SubcommandGroup{}, fmt.Errorf("search: load from manifest: %w", err)
	}
	return group, nil
}
