package domains

import (
	"ai-model-orchestra-controller/cli/domains/ai"

	"github.com/vrooli/cli-core/cliapp"
)

// CommandGroups aggregates flat command groups. No flat groups are registered
// for this scenario; the built-in `status` command from cli-core covers root
// health and all scenario-specific operations live under the `ai` subcommand
// group.
func CommandGroups(_ *cliapp.ScenarioApp) []cliapp.CommandGroup {
	return nil
}

// SubcommandGroups aggregates hierarchical command groups from domain packages.
func SubcommandGroups(core *cliapp.ScenarioApp) []cliapp.SubcommandGroup {
	return []cliapp.SubcommandGroup{
		ai.Register(core),
	}
}
