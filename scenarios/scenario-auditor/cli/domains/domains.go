package domains

import (
	"scenario-auditor/cli/domains/fixes"
	"scenario-auditor/cli/domains/rules"
	"scenario-auditor/cli/domains/scenarios"
	"scenario-auditor/cli/domains/security"
	"scenario-auditor/cli/domains/standards"
	"scenario-auditor/cli/domains/system"

	"github.com/vrooli/cli-core/cliapp"
)

func CommandGroups(core *cliapp.ScenarioApp) []cliapp.CommandGroup {
	_ = core
	return nil
}

func SubcommandGroups(core *cliapp.ScenarioApp) []cliapp.SubcommandGroup {
	return []cliapp.SubcommandGroup{
		system.Register(core),
		scenarios.Register(core),
		rules.Register(core),
		standards.Register(core),
		security.Register(core),
		fixes.Register(core),
	}
}
