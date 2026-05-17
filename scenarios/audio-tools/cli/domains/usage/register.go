// Package usage hosts the `audio-tools usage ...` subtree, mirroring
// vrooli.audio_tools.v1.usage.UsageService.
package usage

import (
	"github.com/vrooli/cli-core/cliapp"
)

func Register(core *cliapp.ScenarioApp) cliapp.SubcommandGroup {
	h := newHandlers(core)
	return cliapp.SubcommandGroup{
		Name:        "usage",
		Description: "Inspect usage rows recorded by the provider chains",
		NeedsAPI:    true,
		Subcommands: []cliapp.Command{
			{
				Name:        "list",
				Description: "List recent usage rows",
				RunCtx:      h.list,
			},
			{
				Name:        "summary",
				Description: "Show usage summary for the last 24h",
				RunCtx:      h.summary,
			},
		},
	}
}
