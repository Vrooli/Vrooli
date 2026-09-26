package domains

import (
	"tidiness-manager/cli/domains/campaigns"
	"tidiness-manager/cli/domains/issues"
	"tidiness-manager/cli/domains/recommendations"
	"tidiness-manager/cli/domains/scan"
	"tidiness-manager/cli/domains/scenarios"
	"tidiness-manager/cli/domains/score"
	"tidiness-manager/cli/domains/tracking"

	"github.com/vrooli/cli-core/cliapp"
)

func CommandGroups(core *cliapp.ScenarioApp) []cliapp.CommandGroup {
	return []cliapp.CommandGroup{
		scan.Register(core),
		recommendations.Register(core),
		score.Register(core),
	}
}

func SubcommandGroups(core *cliapp.ScenarioApp) []cliapp.SubcommandGroup {
	return []cliapp.SubcommandGroup{
		issues.Register(core),
		campaigns.Register(core),
		scenarios.Register(core),
		tracking.Register(),
	}
}
