package health

import (
	"swarm-manager/cli/internal/support"

	"github.com/vrooli/cli-core/cliapp"
)

func Register(deps support.Dependencies) cliapp.CommandGroup {
	return cliapp.CommandGroup{
		Title: "Health",
		Commands: []cliapp.Command{
			{
				Name:        "status",
				Aliases:     []string{"health"},
				NeedsAPI:    true,
				Description: "Check API health and readiness",
				Run:         deps.Status,
			},
			support.APICommand("overview", "Full backlog situational awareness [--format json|markdown]", deps.Overview),
		},
	}
}
