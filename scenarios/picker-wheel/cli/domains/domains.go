package domains

import (
	"picker-wheel/cli/domains/history"
	"picker-wheel/cli/domains/spin"
	"picker-wheel/cli/domains/wheel"

	"github.com/vrooli/cli-core/cliapp"
)

// CommandGroups aggregates flat command groups. Single-verb surfaces (spin,
// history) live here so invocations stay `picker-wheel spin ...` and
// `picker-wheel history` instead of nested `picker-wheel spin run ...`.
func CommandGroups(core *cliapp.ScenarioApp) []cliapp.CommandGroup {
	return []cliapp.CommandGroup{
		spin.Register(core),
		history.Register(core),
	}
}

// SubcommandGroups aggregates hierarchical command groups from domain packages.
func SubcommandGroups(core *cliapp.ScenarioApp) []cliapp.SubcommandGroup {
	return []cliapp.SubcommandGroup{
		wheel.Register(core),
	}
}
