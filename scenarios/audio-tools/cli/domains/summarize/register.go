// Package summarize is the CLI's summarize-domain command surface,
// mirroring vrooli.audio_tools.v1.summarize.SummarizeService.
//
// The command surface is declared in cli/manifest.json — the single
// source of truth. Register loads the "summarize" group and wires each
// binding to a handler in handlers.go.
package summarize

import (
	"github.com/vrooli/cli-core/cliapp"

	"audio-tools/cli/internal/climanifest"
)

// GroupName is the manifest group name this package owns.
const GroupName = "summarize"

// Register builds the summarize subcommand group from the embedded
// manifest and wires Connect-RPC bindings to handlers.
func Register(core *cliapp.ScenarioApp, manifest []byte) (cliapp.SubcommandGroup, error) {
	h := newHandlers(core)
	bindings := map[string]func(cliapp.RunContext) error{
		"SummarizeService.Summarize": h.text,
	}
	return climanifest.LoadGroup(manifest, GroupName, bindings)
}
