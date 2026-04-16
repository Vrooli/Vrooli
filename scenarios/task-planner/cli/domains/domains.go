package domains

import (
	"task-planner/cli/domains/apps"
	"task-planner/cli/domains/tasks"

	"github.com/vrooli/cli-core/cliapp"
)

// CommandGroups aggregates flat command groups. Task Planner currently has no
// single-verb root commands beyond the built-ins; leaving this as the extension
// point for future additions.
func CommandGroups(_ *cliapp.ScenarioApp) []cliapp.CommandGroup {
	return []cliapp.CommandGroup{}
}

// SubcommandGroups aggregates hierarchical command groups from domain packages.
func SubcommandGroups(core *cliapp.ScenarioApp) []cliapp.SubcommandGroup {
	return []cliapp.SubcommandGroup{
		tasks.Register(core),
		apps.Register(core),
	}
}
