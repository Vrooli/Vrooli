// Package reindex is the CLI's reindex-domain command surface. It binds to the
// SHARED, token-gated search control plane (search-hub.v1.control.
// SearchControlService) — the same RPCs search-hub's sweep drives — exposing the
// operator-facing reindex verbs (run/status/cancel) over it.
package reindex

import (
	"fmt"

	"github.com/vrooli/cli-core/cliapp"
)

// GroupName is the manifest group this package owns.
const GroupName = "reindex"

// Register builds the reindex subcommand group from the embedded manifest
// and wires Connect-RPC bindings to handlers in handlers.go.
func Register(core *cliapp.ScenarioApp, manifest []byte) (cliapp.SubcommandGroup, error) {
	h := newHandlers(core)
	bindings := map[string]func(cliapp.RunContext) error{
		"SearchControlService.Reindex":       h.run,
		"SearchControlService.ReindexStatus": h.status,
		"SearchControlService.ReindexCancel": h.cancel,
	}
	group, err := cliapp.LoadFromManifest(manifest, GroupName, bindings)
	if err != nil {
		return cliapp.SubcommandGroup{}, fmt.Errorf("reindex: load from manifest: %w", err)
	}
	return group, nil
}
