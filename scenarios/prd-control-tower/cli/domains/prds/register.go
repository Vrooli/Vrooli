package prds

import (
	"prd-control-tower/cli/internal/support"

	"github.com/vrooli/cli-core/cliapp"
)

func Register(deps support.Dependencies) cliapp.CommandGroup {
	return cliapp.CommandGroup{
		Title: "PRDs",
		Commands: []cliapp.Command{
			support.Command("prd", "PRD management (generate, validate, fix)", deps.PRD),
		},
	}
}
