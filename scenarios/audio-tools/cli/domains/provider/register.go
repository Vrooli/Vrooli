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
	"time"

	"github.com/vrooli/cli-core/cliapp"

	"audio-tools/cli/internal/climanifest"
)

// GroupName is the manifest group name this package owns.
const GroupName = "provider"

// Register builds the provider subcommand group from the embedded
// manifest and wires Connect-RPC bindings to handlers.
func Register(core *cliapp.ScenarioApp, manifest []byte) (cliapp.SubcommandGroup, error) {
	return register(core, manifest, nil)
}

// RegisterWithClock is the runtime composition seam for the timestamp shown
// by provider list. Tests retain Register/newHandlers with a deterministic
// epoch clock; the application root supplies the process clock.
func RegisterWithClock(core *cliapp.ScenarioApp, manifest []byte, now func() time.Time) (cliapp.SubcommandGroup, error) {
	return register(core, manifest, now)
}

func register(core *cliapp.ScenarioApp, manifest []byte, now func() time.Time) (cliapp.SubcommandGroup, error) {
	if now == nil {
		now = func() time.Time { return time.Unix(0, 0) }
	}
	h := newHandlersWithClock(core, now)
	primitives := map[string]cliapp.PrimitiveHandler{
		"ProviderLifecycleService.ListLocalProviders": h.listPrimitive(),
		"ProviderLifecycleService.StartProvider":      h.startPrimitive(),
		"ProviderLifecycleService.StopProvider":       h.stopPrimitive(),
		"ProviderLifecycleService.RestartProvider":    h.restartPrimitive(),
		"ProviderLifecycleService.PullModel":          h.pullModelPrimitive(),
		"ProviderLifecycleService.GetProviderLogs":    cliapp.Streaming(h.logs),
	}
	bindings := map[string]func(cliapp.RunContext) error{
		"ProviderLifecycleService.ListLocalProviders": primitives["ProviderLifecycleService.ListLocalProviders"].Run,
		"ProviderLifecycleService.StartProvider":      primitives["ProviderLifecycleService.StartProvider"].Run,
		"ProviderLifecycleService.StopProvider":       primitives["ProviderLifecycleService.StopProvider"].Run,
		"ProviderLifecycleService.RestartProvider":    primitives["ProviderLifecycleService.RestartProvider"].Run,
		"ProviderLifecycleService.PullModel":          primitives["ProviderLifecycleService.PullModel"].Run,
		"ProviderLifecycleService.GetProviderLogs":    primitives["ProviderLifecycleService.GetProviderLogs"].Run,
	}
	group, err := climanifest.LoadGroup(manifest, GroupName, bindings)
	if err != nil {
		return cliapp.SubcommandGroup{}, err
	}
	for i := range group.Subcommands {
		switch group.Subcommands[i].Name {
		case "list", "start", "stop", "restart", "pull-model", "logs":
			method := map[string]string{"list": "ListLocalProviders", "start": "StartProvider", "stop": "StopProvider", "restart": "RestartProvider", "pull-model": "PullModel", "logs": "GetProviderLogs"}[group.Subcommands[i].Name]
			group.Subcommands[i] = group.Subcommands[i].WithPrimitive(primitives["ProviderLifecycleService."+method])
		}
	}
	return group, nil
}
