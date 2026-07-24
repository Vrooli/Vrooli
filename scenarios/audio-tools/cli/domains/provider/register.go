// Package provider is the CLI's provider_lifecycle-domain command
// surface, mirroring
// vrooli.audio_tools.v1.provider_lifecycle.ProviderLifecycleService.
//
// The command surface (name/positionals/flags/governance and the
// Service.Method binding) is declared in cli/manifest.json — the single
// source of truth. Register loads the "provider" group and wires each
// binding to a handler in handlers.go.
//
// `--dry-run` is provided globally by cli-core (cliapp.GlobalFlags);
// when set, every mutating subcommand emits the X-Dry-Run: true header
// automatically through cliapp.NewConnectHTTPClient's request pipeline.
package provider

import (
	"github.com/vrooli/cli-core/cliapp"

	"audio-tools/cli/internal/climanifest"
)

// GroupName is the manifest group name this package owns.
const GroupName = "provider"

// Register builds the provider subcommand group from the embedded
// manifest and wires Connect-RPC bindings to handlers.
func Register(core *cliapp.ScenarioApp, manifest []byte) (cliapp.SubcommandGroup, error) {
	h := newHandlers(core)
	bindings := map[string]func(cliapp.RunContext) error{
		"ProviderLifecycleService.ListLocalProviders": h.list,
		"ProviderLifecycleService.StartProvider":      h.start,
		"ProviderLifecycleService.StopProvider":       h.stop,
		"ProviderLifecycleService.RestartProvider":    h.restart,
		"ProviderLifecycleService.PullModel":          h.pullModel,
		"ProviderLifecycleService.GetProviderLogs":    h.logs,
	}
	return climanifest.LoadGroup(manifest, GroupName, bindings)
}
