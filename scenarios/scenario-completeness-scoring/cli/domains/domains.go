package domains

import (
	"scenario-completeness-scoring/cli/domains/analysis"
	"scenario-completeness-scoring/cli/domains/config"
	"scenario-completeness-scoring/cli/domains/monitoring"
	"scenario-completeness-scoring/cli/domains/scores"

	"github.com/vrooli/cli-core/cliapp"
)

// CommandGroups aggregates flat command groups. This CLI has no flat commands
// today; the built-in `status` command from cli-core already covers the root
// `/health` probe.
func CommandGroups(_ *cliapp.ScenarioApp) []cliapp.CommandGroup {
	return nil
}

// SubcommandGroups aggregates hierarchical command groups from domain packages.
func SubcommandGroups(core *cliapp.ScenarioApp) []cliapp.SubcommandGroup {
	return []cliapp.SubcommandGroup{
		scores.Register(core),
		config.Register(core),
		monitoring.Register(core),
		analysis.Register(core),
	}
}
