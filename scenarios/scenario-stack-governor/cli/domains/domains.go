package domains

import (
	"scenario-stack-governor/cli/domains/fix"
	"scenario-stack-governor/cli/domains/rules"
	"scenario-stack-governor/cli/domains/run"
	"scenario-stack-governor/cli/domains/scenarios"

	"github.com/vrooli/cli-core/cliapp"
)

func CommandGroups(core *cliapp.ScenarioApp) []cliapp.CommandGroup {
	return []cliapp.CommandGroup{
		run.Register(core),
		fix.Register(core),
	}
}

func SubcommandGroups(core *cliapp.ScenarioApp) []cliapp.SubcommandGroup {
	return []cliapp.SubcommandGroup{
		rules.Register(core),
		scenarios.Register(core),
	}
}
