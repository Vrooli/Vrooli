package domains

import (
	"scenario-completeness-scoring/cli/domains/config"
	"scenario-completeness-scoring/cli/domains/health"
	"scenario-completeness-scoring/cli/domains/scoring"

	"github.com/vrooli/cli-core/cliapp"
)

func CommandGroups(core *cliapp.ScenarioApp) []cliapp.CommandGroup {
	return []cliapp.CommandGroup{
		health.Register(core),
		scoring.Register(core),
		config.Register(core),
	}
}
