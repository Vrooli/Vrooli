package requirements

import (
	"prd-control-tower/cli/internal/support"

	"github.com/vrooli/cli-core/cliapp"
)

func Register(deps support.Dependencies) cliapp.CommandGroup {
	return cliapp.CommandGroup{
		Title: "Requirements",
		Commands: []cliapp.Command{
			support.Command("requirements", "Requirements management (generate, validate)", deps.Requirements),
		},
	}
}
