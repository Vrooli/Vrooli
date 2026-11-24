// Package health is the CLI's health_status-domain command surface,
// mirroring vrooli.audio_tools.v1.health_status.HealthStatusService.
//
// The command surface is declared in cli/manifest.json — the single
// source of truth. Register loads the "health" group and wires each
// binding to a handler in handlers.go. `show --refresh` bypasses the
// registry cache (the handler issues RefreshProviderHealth when the
// flag is set), so RefreshProviderHealth is omitted in the manifest
// rather than bound to its own command. `watch` consumes
// StreamProviderHealth.
package health

import (
	"github.com/vrooli/cli-core/cliapp"

	"audio-tools/cli/internal/climanifest"
)

// GroupName is the manifest group name this package owns.
const GroupName = "health"

// Register builds the health subcommand group from the embedded manifest
// and wires Connect-RPC bindings to handlers.
func Register(core *cliapp.ScenarioApp, manifest []byte) (cliapp.SubcommandGroup, error) {
	h := newHandlers(core)
	show := h.showPrimitive()
	watch := cliapp.Streaming(h.watch)
	bindings := map[string]func(cliapp.RunContext) error{
		"HealthStatusService.GetProviderHealth":    show.Run,
		"HealthStatusService.StreamProviderHealth": watch.Run,
	}
	group, err := climanifest.LoadGroup(manifest, GroupName, bindings)
	if err != nil {
		return cliapp.SubcommandGroup{}, err
	}
	for i := range group.Subcommands {
		switch group.Subcommands[i].Name {
		case "show":
			group.Subcommands[i] = group.Subcommands[i].WithPrimitive(show)
		case "watch":
			group.Subcommands[i] = group.Subcommands[i].WithPrimitive(watch)
		}
	}
	return group, nil
}
