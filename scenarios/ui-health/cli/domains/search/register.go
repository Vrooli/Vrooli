// Package search is the CLI's search-domain command surface. Mirrors
// the API's Connect-RPC SearchService.
package search

import (
	"fmt"

	"github.com/vrooli/cli-core/cliapp"
)

// GroupName is the manifest group this package owns.
const GroupName = "search"

// Register builds the search subcommand group from the embedded manifest
// and wires Connect-RPC bindings to handlers in handlers.go.
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
	// Allow `cli-health search "<query>"` as shorthand for `cli-health search
	// query "<query>"`. The query subcommand is the dominant verb here and
	// agent prompts/docs are easier to author against the bare form.
	group.DefaultSubcommand = "query"
	return group, nil
}
