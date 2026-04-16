package drafts

import (
	"prd-control-tower/cli/internal/support"

	"github.com/vrooli/cli-core/cliapp"
)

func Register(deps support.Dependencies) cliapp.CommandGroup {
	return cliapp.CommandGroup{
		Title: "Drafts",
		Commands: []cliapp.Command{
			support.Command("list-drafts", "List PRD drafts", deps.ListDrafts),
		},
	}
}
