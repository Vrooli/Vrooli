package domains

import (
	"scalable-app-cookbook/cli/domains/implementations"
	"scalable-app-cookbook/cli/domains/patterns"
	"scalable-app-cookbook/cli/domains/recipes"

	"github.com/vrooli/cli-core/cliapp"
)

// CommandGroups aggregates flat command groups. Single-verb domains like
// `implementations` live here so the invocation stays
// `scalable-app-cookbook implementations ...` instead of
// `scalable-app-cookbook implementations list ...`.
func CommandGroups(core *cliapp.ScenarioApp) []cliapp.CommandGroup {
	return []cliapp.CommandGroup{
		implementations.Register(core),
	}
}

// SubcommandGroups aggregates hierarchical command groups from domain packages.
func SubcommandGroups(core *cliapp.ScenarioApp) []cliapp.SubcommandGroup {
	return []cliapp.SubcommandGroup{
		patterns.Register(core),
		recipes.Register(core),
	}
}
