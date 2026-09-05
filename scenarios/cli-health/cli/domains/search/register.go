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
	bindings := map[string]cliapp.PrimitiveHandler{
		// query is declared architecture.primitive "proto_list" in the manifest and
		// built with the matching cli-core primitive, so its operation runs outside
		// any output-format branch. The primitive carries proof of proto_list that
		// LoadFromManifestPrimitives reconciles against the manifest declaration.
		"SearchService.Search": cliapp.ProtoList(h.searchCall, h.searchReport),
		// status is declared architecture.primitive "operational" — backend
		// availability rendered through the Status -> Triage -> Next Steps contract.
		"SearchService.Status": cliapp.ProtoOperational(h.statusCall, h.statusReport),
	}
	group, err := cliapp.LoadFromManifestPrimitives(manifest, GroupName, bindings)
	if err != nil {
		return cliapp.SubcommandGroup{}, fmt.Errorf("search: load from manifest: %w", err)
	}
	// Allow `cli-health search "<query>"` as shorthand for `cli-health search
	// query "<query>"`. The query subcommand is the dominant verb here and
	// agent prompts/docs are easier to author against the bare form.
	group.DefaultSubcommand = "query"
	return group, nil
}
