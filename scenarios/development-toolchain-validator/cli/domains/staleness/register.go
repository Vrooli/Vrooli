// Package staleness is the CLI's staleness command surface.
package staleness

import (
	"github.com/vrooli/cli-core/cliapp"
)

// Register returns the `staleness` subcommand group.
func Register(core *cliapp.ScenarioApp) cliapp.SubcommandGroup {
	h := newHandlers(core)
	return cliapp.SubcommandGroup{
		Name:        "staleness",
		Description: "List manifests whose pinned template/skill versions drift from current",
		NeedsAPI:    true,
		Subcommands: []cliapp.Command{
			{
				Name:        "list",
				Description: "List stale (skill, golden) tuples",
				RunCtx:      h.list,
			},
		},
	}
}
