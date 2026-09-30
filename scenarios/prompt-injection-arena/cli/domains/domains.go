package domains

import (
	"prompt-injection-arena/cli/domains/admin"
	"prompt-injection-arena/cli/domains/export"
	"prompt-injection-arena/cli/domains/injections"
	"prompt-injection-arena/cli/domains/leaderboard"
	"prompt-injection-arena/cli/domains/security"
	"prompt-injection-arena/cli/domains/statistics"
	"prompt-injection-arena/cli/domains/tournaments"
	"prompt-injection-arena/cli/domains/vector"

	"github.com/vrooli/cli-core/cliapp"
)

// CommandGroups aggregates flat command groups. `statistics` is a single
// read-only endpoint so it is exposed as a top-level verb rather than nested.
func CommandGroups(core *cliapp.ScenarioApp) []cliapp.CommandGroup {
	return []cliapp.CommandGroup{
		statistics.Register(core),
	}
}

// SubcommandGroups aggregates hierarchical command groups from domain packages.
func SubcommandGroups(core *cliapp.ScenarioApp) []cliapp.SubcommandGroup {
	return []cliapp.SubcommandGroup{
		injections.Register(core),
		leaderboard.Register(core),
		security.Register(core),
		vector.Register(core),
		tournaments.Register(core),
		export.Register(core),
		admin.Register(core),
	}
}
