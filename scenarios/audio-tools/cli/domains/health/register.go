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
	bindings := map[string]func(cliapp.RunContext) error{
		"HealthStatusService.GetProviderHealth":    h.show,
		"HealthStatusService.StreamProviderHealth": h.watch,
	}
	return climanifest.LoadGroup(manifest, GroupName, bindings)
}
