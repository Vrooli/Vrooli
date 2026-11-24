// Package usage hosts the `audio-tools usage ...` subtree, mirroring
// vrooli.audio_tools.v1.usage.UsageService.
//
// The command surface is declared in cli/manifest.json — the single
// source of truth. Register loads the "usage" group and wires each
// binding to a handler in handlers.go.
package usage

import (
	"github.com/vrooli/cli-core/cliapp"

	"audio-tools/cli/internal/climanifest"
)

// GroupName is the manifest group name this package owns.
const GroupName = "usage"

// Register builds the usage subcommand group from the embedded manifest
// and wires Connect-RPC bindings to handlers.
func Register(core *cliapp.ScenarioApp, manifest []byte) (cliapp.SubcommandGroup, error) {
	h := newHandlers(core)
	list := h.listPrimitive()
	summary := h.summaryPrimitive()
	bindings := map[string]func(cliapp.RunContext) error{
		"UsageService.ListRecent": list.Run,
		"UsageService.GetSummary": summary.Run,
	}
	group, err := climanifest.LoadGroup(manifest, GroupName, bindings)
	if err != nil {
		return cliapp.SubcommandGroup{}, err
	}
	for i := range group.Subcommands {
		switch group.Subcommands[i].Name {
		case "list":
			group.Subcommands[i] = group.Subcommands[i].WithPrimitive(list)
		case "summary":
			group.Subcommands[i] = group.Subcommands[i].WithPrimitive(summary)
		}
	}
	return group, nil
}
