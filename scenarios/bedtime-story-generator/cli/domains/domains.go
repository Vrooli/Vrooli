package domains

import (
	"bedtime-story-generator/cli/domains/stories"
	"bedtime-story-generator/cli/domains/themes"

	"github.com/vrooli/cli-core/cliapp"
)

// CommandGroups aggregates flat command groups. Single-verb domains like
// `themes` live here so the invocation stays
// `bedtime-story-generator themes` instead of
// `bedtime-story-generator themes list`.
func CommandGroups(core *cliapp.ScenarioApp) []cliapp.CommandGroup {
	return []cliapp.CommandGroup{
		themes.Register(core),
	}
}

// SubcommandGroups aggregates hierarchical command groups from domain packages.
func SubcommandGroups(core *cliapp.ScenarioApp) []cliapp.SubcommandGroup {
	return []cliapp.SubcommandGroup{
		stories.Register(core),
	}
}
