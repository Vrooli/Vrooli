// Package diagnose hosts the `audio-tools diagnose ...` subtree.
package diagnose

import (
	"github.com/vrooli/cli-core/cliapp"
)

func Register(core *cliapp.ScenarioApp) cliapp.SubcommandGroup {
	h := newHandlers(core)
	return cliapp.SubcommandGroup{
		Name:        "diagnose",
		Description: "Probe provider availability and routing",
		NeedsAPI:    true,
		Subcommands: []cliapp.Command{
			{
				Name:        "providers",
				Description: "Print the per-tier provider-availability matrix",
				RunCtx:      h.providers,
			},
		},
	}
}
