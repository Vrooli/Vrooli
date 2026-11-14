// Package query is the CLI's query-domain command surface. It mirrors the
// API's RoutingService.Query Connect-RPC: the federated search entry point that
// fans a query out across registered providers and returns results grouped by
// provider.
//
// Like the providers domain, the command-line shape is declared in
// cli/manifest.json (the single source of truth) and loaded via
// cliapp.LoadFromManifest; the handler lives in handlers.go. The bare form
// `search-hub query "<text>"` is wired as the group default so the dominant
// verb needs no redundant subcommand token.
package query

import (
	"fmt"

	"github.com/vrooli/cli-core/cliapp"
)

// GroupName is the manifest group this package owns. Exported so tests can call
// RequireProtoServiceCoverage against the same manifest the runtime loads.
const GroupName = "query"

// Register builds the query subcommand group from the embedded manifest and
// wires the RoutingService.Query binding to the handler in handlers.go.
func Register(core *cliapp.ScenarioApp, manifest []byte) (cliapp.SubcommandGroup, error) {
	h := newHandlers(core)
	bindings := map[string]cliapp.PrimitiveHandler{
		"RoutingService.Query": cliapp.ProtoList(h.queryCall, h.queryReport),
	}
	group, err := cliapp.LoadFromManifestPrimitives(manifest, GroupName, bindings)
	if err != nil {
		return cliapp.SubcommandGroup{}, fmt.Errorf("query: load from manifest: %w", err)
	}
	// Allow `search-hub query "<text>"` as shorthand for
	// `search-hub query query "<text>"`. Query is the dominant (only) verb of
	// this group; agent prompts and docs read cleaner against the bare form.
	group.DefaultSubcommand = "query"
	return group, nil
}
