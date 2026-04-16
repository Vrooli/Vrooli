package domains

import (
	"git-control-tower/cli/domains/audit"
	"git-control-tower/cli/domains/branch"
	"git-control-tower/cli/domains/repo"
	"git-control-tower/cli/domains/review"

	"github.com/vrooli/cli-core/cliapp"
)

func CommandGroups(core *cliapp.ScenarioApp) []cliapp.CommandGroup {
	_ = core
	return nil
}

func SubcommandGroups(core *cliapp.ScenarioApp) []cliapp.SubcommandGroup {
	return []cliapp.SubcommandGroup{
		repo.Register(core),
		branch.Register(core),
		review.Register(core),
		audit.Register(core),
	}
}
