package domains

import (
	"algorithm-library/cli/domains/algorithm"
	"algorithm-library/cli/domains/categories"
	"algorithm-library/cli/domains/contribution"
	"algorithm-library/cli/domains/performance"
	"algorithm-library/cli/domains/problem"
	"algorithm-library/cli/domains/stats"

	"github.com/vrooli/cli-core/cliapp"
)

// CommandGroups aggregates flat command groups. Single-verb domains live here
// so the invocation stays `algorithm-library <verb>` instead of
// `algorithm-library <verb> list`.
func CommandGroups(core *cliapp.ScenarioApp) []cliapp.CommandGroup {
	return []cliapp.CommandGroup{
		categories.Register(core),
		stats.Register(core),
	}
}

// SubcommandGroups aggregates hierarchical command groups from domain packages.
func SubcommandGroups(core *cliapp.ScenarioApp) []cliapp.SubcommandGroup {
	return []cliapp.SubcommandGroup{
		algorithm.Register(core),
		contribution.Register(core),
		performance.Register(core),
		problem.Register(core),
	}
}
