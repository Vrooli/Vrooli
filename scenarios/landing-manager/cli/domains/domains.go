package domains

import (
	"landing-manager/cli/domains/analytics"
	"landing-manager/cli/domains/customize"
	"landing-manager/cli/domains/generate"
	"landing-manager/cli/domains/lifecycle"
	"landing-manager/cli/domains/personas"
	"landing-manager/cli/domains/preview"
	"landing-manager/cli/domains/templates"

	"github.com/vrooli/cli-core/cliapp"
)

// CommandGroups aggregates flat command groups. Single-verb surfaces live here so the
// invocation stays `landing-manager <verb>` instead of `landing-manager <verb> run`.
func CommandGroups(core *cliapp.ScenarioApp) []cliapp.CommandGroup {
	return []cliapp.CommandGroup{
		generate.Register(core),
		customize.Register(core),
		preview.Register(core),
	}
}

// SubcommandGroups aggregates hierarchical command groups from domain packages.
func SubcommandGroups(core *cliapp.ScenarioApp) []cliapp.SubcommandGroup {
	return []cliapp.SubcommandGroup{
		templates.Register(core),
		personas.Register(core),
		analytics.Register(core),
		lifecycle.Register(core),
	}
}
