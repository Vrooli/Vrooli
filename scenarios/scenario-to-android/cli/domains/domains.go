package domains

import (
	"scenario-to-android/cli/domains/build"
	"scenario-to-android/cli/domains/system"

	"github.com/vrooli/cli-core/cliapp"
)

// CommandGroups aggregates flat command groups. This scenario has none at the
// moment; the hierarchical domains live in SubcommandGroups below.
func CommandGroups(_ *cliapp.ScenarioApp) []cliapp.CommandGroup {
	return nil
}

// SubcommandGroups aggregates hierarchical command groups from domain packages.
func SubcommandGroups(core *cliapp.ScenarioApp) []cliapp.SubcommandGroup {
	return []cliapp.SubcommandGroup{
		build.Register(core),
		system.Register(core),
	}
}
