// Package health is the CLI's health_status-domain command surface,
// mirroring vrooli.audio_tools.v1.health_status.HealthStatusService.
package health

import (
	"github.com/vrooli/cli-core/cliapp"
)

// Register returns the health SubcommandGroup. Default output is the
// human-friendly capability/provider table per
// feedback_cli_default_human_output; --json switches to proto JSON;
// `show --refresh` bypasses the registry cache; `watch` consumes
// StreamProviderHealth.
func Register(core *cliapp.ScenarioApp) cliapp.SubcommandGroup {
	h := newHandlers(core)
	return cliapp.SubcommandGroup{
		Name:        "health",
		Description: "Show per-capability provider health (capability/provider/tier/state table; `show` for one snapshot, `watch` to stream)",
		NeedsAPI:    true,
		Subcommands: []cliapp.Command{
			{
				Name:        "show",
				Description: "Print the cached per-capability provider health (use --refresh to bypass the cache)",
				Args: cliapp.ArgSchema{
					Flags: []cliapp.Flag{
						{Name: "refresh", Bool: true, Description: "Bypass the cache and re-run every checker before printing"},
					},
				},
				RunCtx: h.show,
			},
			{
				Name:        "watch",
				Description: "Stream rollup events; exits on Ctrl-C",
				RunCtx:      h.watch,
			},
		},
	}
}
