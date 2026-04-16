package migration

import (
	"swarm-manager/cli/internal/support"

	"github.com/vrooli/cli-core/cliapp"
)

func Register(deps support.Dependencies) cliapp.CommandGroup {
	return cliapp.CommandGroup{
		Title: "Migration",
		Commands: []cliapp.Command{
			{
				Name:        "migrate-workshop",
				Description: "Migrate backlog items from clarify/suggest/enhance to workshop/plan.md [--dry-run] [--root PATH]",
				Run:         deps.MigrateWorkshop,
			},
		},
	}
}
