package domains

import (
	"vrooli-bridge/cli/domains/projects"

	"github.com/vrooli/cli-core/cliapp"
)

// CommandGroups aggregates flat command groups. Vrooli Bridge has no flat
// commands today; all scenario-specific commands live under `project`.
func CommandGroups(core *cliapp.ScenarioApp) []cliapp.CommandGroup {
	_ = core
	return nil
}

// SubcommandGroups aggregates hierarchical command groups from domain packages.
func SubcommandGroups(core *cliapp.ScenarioApp) []cliapp.SubcommandGroup {
	return []cliapp.SubcommandGroup{
		projects.Register(core),
	}
}
