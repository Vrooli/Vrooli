// Package dispatch is the CLI's dispatch-domain command surface. Mirrors the
// API's Connect-RPC DispatchService and is the operator entry point for running
// an allowlisted typed job on a fleet node. The manifest (cli/manifest.json) is
// the single source of truth for the command shape; handlers.go binds the RPC.
package dispatch

import (
	"fmt"

	"github.com/vrooli/cli-core/cliapp"
)

// GroupName is the manifest group name this package owns.
const GroupName = "dispatch"

// Register builds the dispatch subcommand group from the embedded manifest and
// wires Connect-RPC bindings to handlers in handlers.go.
func Register(core *cliapp.ScenarioApp, manifest []byte) (cliapp.SubcommandGroup, error) {
	h := newHandlers(core)
	bindings := map[string]func(cliapp.RunContext) error{
		"DispatchService.DispatchJob": h.job,
	}
	group, err := cliapp.LoadFromManifest(manifest, GroupName, bindings)
	if err != nil {
		return cliapp.SubcommandGroup{}, fmt.Errorf("dispatch: load from manifest: %w", err)
	}
	return group, nil
}
