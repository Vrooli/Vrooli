package domains

import (
	"app-monitor/cli/domains/apps"
	"app-monitor/cli/domains/diagnostics"
	"app-monitor/cli/domains/docker"
	"app-monitor/cli/domains/lighthouse"
	"app-monitor/cli/domains/presets"
	"app-monitor/cli/domains/resources"
	"app-monitor/cli/domains/rules"
	"app-monitor/cli/domains/system"
	"app-monitor/cli/domains/tools"

	"github.com/vrooli/cli-core/cliapp"
)

// CommandGroups aggregates flat command groups. Single-verb domains like `rules`
// live here so the invocation stays `app-monitor rules ...` instead of
// `app-monitor rules list ...`.
func CommandGroups(core *cliapp.ScenarioApp) []cliapp.CommandGroup {
	return []cliapp.CommandGroup{
		rules.Register(core),
	}
}

// SubcommandGroups aggregates hierarchical command groups from domain packages.
func SubcommandGroups(core *cliapp.ScenarioApp) []cliapp.SubcommandGroup {
	return []cliapp.SubcommandGroup{
		apps.Register(core),
		diagnostics.Register(core),
		system.Register(core),
		docker.Register(core),
		lighthouse.Register(core),
		presets.Register(core),
		resources.Register(core),
		tools.Register(core),
	}
}
