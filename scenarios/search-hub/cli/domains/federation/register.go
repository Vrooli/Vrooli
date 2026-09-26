// Package federation is the CLI's federation-status command surface. It mirrors
// the API's RoutingService.Status Connect-RPC: per-provider reachability plus
// classifier/reranker availability.
//
// The command-line shape is declared in cli/manifest.json (the single source of
// truth) and loaded via cliapp.LoadFromManifestPrimitives; the handler lives in
// handlers.go. The status verb is the group default so `search-hub federation`
// is shorthand for `search-hub federation status`.
//
// Note: the bare top-level `search-hub status` is cli-core's built-in API
// health check (a different concern — this scenario's own self-health), so the
// federation-health surface is namespaced under `federation` to avoid colliding
// with that built-in.
package federation

import (
	"fmt"

	"github.com/vrooli/cli-core/cliapp"
)

// GroupName is the manifest group this package owns.
const GroupName = "federation"

// Register builds the federation subcommand group from the embedded manifest
// and wires the RoutingService.Status binding to the handler in handlers.go.
func Register(core *cliapp.ScenarioApp, manifest []byte) (cliapp.SubcommandGroup, error) {
	h := newHandlers(core)
	bindings := map[string]cliapp.PrimitiveHandler{
		"RoutingService.Status":    cliapp.ProtoList(h.statusCall, h.statusReport),
		"RoutingService.Repromote": cliapp.ProtoMutation(h.repromoteCall, h.repromoteReport),
	}
	group, err := cliapp.LoadFromManifestPrimitives(manifest, GroupName, bindings)
	if err != nil {
		return cliapp.SubcommandGroup{}, fmt.Errorf("federation: load from manifest: %w", err)
	}
	group.DefaultSubcommand = "status"
	return group, nil
}
