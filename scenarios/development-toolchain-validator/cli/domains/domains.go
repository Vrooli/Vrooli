package domains

import (
	"development-toolchain-validator/cli/domains/connections"
	"development-toolchain-validator/cli/domains/references"

	"github.com/vrooli/cli-core/cliapp"
)

func CommandGroups(core *cliapp.ScenarioApp) []cliapp.CommandGroup {
	_ = core
	return nil
}

func SubcommandGroups(core *cliapp.ScenarioApp) []cliapp.SubcommandGroup {
	return []cliapp.SubcommandGroup{
		references.Register(core),
		connections.Register(core),
	}
}
