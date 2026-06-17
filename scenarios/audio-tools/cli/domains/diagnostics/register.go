// Package diagnostics is the CLI's diagnostics-domain command surface,
// mirroring vrooli.audio_tools.v1.diagnostics.DiagnosticsService.
//
// The command surface is declared in cli/manifest.json — the single
// source of truth. Register loads the "diagnostics" group and wires each
// binding to a handler in handlers.go.
package diagnostics

import (
	"fmt"

	"github.com/vrooli/cli-core/cliapp"
)

// GroupName is the manifest group name this package owns.
const GroupName = "diagnostics"

// Register builds the diagnostics subcommand group from the embedded
// manifest and wires Connect-RPC bindings to handlers.
func Register(core *cliapp.ScenarioApp, manifest []byte) (cliapp.SubcommandGroup, error) {
	h := newHandlers(core)
	bindings := map[string]func(cliapp.RunContext) error{
		"DiagnosticsService.RunSuite":   h.run,
		"DiagnosticsService.GetLastRun": h.last,
	}
	group, err := cliapp.LoadFromManifest(manifest, GroupName, bindings)
	if err != nil {
		return cliapp.SubcommandGroup{}, fmt.Errorf("diagnostics: load from manifest: %w", err)
	}
	return group, nil
}
