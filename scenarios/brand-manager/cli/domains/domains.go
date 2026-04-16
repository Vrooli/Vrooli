package domains

import (
	"brand-manager/cli/domains/assignments"
	"brand-manager/cli/domains/brands"
	"brand-manager/cli/domains/operations"

	"github.com/vrooli/cli-core/cliapp"
)

func CommandGroups(core *cliapp.ScenarioApp) []cliapp.CommandGroup {
	return []cliapp.CommandGroup{
		brands.Register(core),
		assignments.Register(core),
		operations.Register(core),
	}
}

func SubcommandGroups(core *cliapp.ScenarioApp) []cliapp.SubcommandGroup {
	_ = core
	return nil
}
