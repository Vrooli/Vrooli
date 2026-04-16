package domains

import (
	"prd-control-tower/cli/domains/drafts"
	"prd-control-tower/cli/domains/prds"
	"prd-control-tower/cli/domains/requirements"
	"prd-control-tower/cli/internal/support"

	"github.com/vrooli/cli-core/cliapp"
)

func CommandGroups(deps support.Dependencies) []cliapp.CommandGroup {
	return []cliapp.CommandGroup{
		drafts.Register(deps),
		prds.Register(deps),
		requirements.Register(deps),
	}
}
