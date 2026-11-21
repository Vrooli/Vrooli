// Package runs is the CLI's runs-domain command surface. Mirrors the API's
// Connect-RPC RunsService: inspect runs, block-once wait (no polling), abort,
// and follow live output. The manifest (cli/manifest.json) is the single source
// of truth for the command shape; handlers.go binds the RPCs.
package runs

import (
	"fmt"

	"github.com/vrooli/cli-core/cliapp"
)

// GroupName is the manifest group name this package owns.
const GroupName = "runs"

// Register builds the runs subcommand group from the embedded manifest and
// wires Connect-RPC bindings to handlers in handlers.go. ReportRunEvent is
// node-facing and intentionally absent (omitted in the manifest).
func Register(core *cliapp.ScenarioApp, manifest []byte) (cliapp.SubcommandGroup, error) {
	h := newHandlers(core)
	bindings := map[string]func(cliapp.RunContext) error{
		"RunsService.GetRun":          h.get,
		"RunsService.ListRuns":        h.list,
		"RunsService.WaitRun":         h.wait,
		"RunsService.AbortRun":        h.abort,
		"RunsService.StreamRunEvents": h.follow,
	}
	group, err := cliapp.LoadFromManifest(manifest, GroupName, bindings)
	if err != nil {
		return cliapp.SubcommandGroup{}, fmt.Errorf("runs: load from manifest: %w", err)
	}
	return group, nil
}
