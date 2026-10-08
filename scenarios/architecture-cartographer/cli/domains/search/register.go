// Package search is the CLI's search-domain command surface. Mirrors the API's
// Connect-RPC SearchService (term-agnostic domain-map search).
package search

import (
	"fmt"

	"github.com/vrooli/cli-core/cliapp"
)

// GroupName is the manifest group this package owns.
const GroupName = "search"

// Register builds the search subcommand group from the embedded manifest and
// wires Connect-RPC bindings to handlers in handlers.go.
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
	// Allow `architecture-cartographer search "<query>"` as shorthand for the
	// `query` subcommand — the dominant verb, easier for agents/docs to author.
	group.DefaultSubcommand = "query"
	return group, nil
}
