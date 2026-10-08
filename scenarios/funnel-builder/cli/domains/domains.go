package domains

import (
	"funnel-builder/cli/domains/funnels"
	"funnel-builder/cli/domains/projects"
	"funnel-builder/cli/domains/templates"

	"github.com/vrooli/cli-core/cliapp"
)

func CommandGroups(core *cliapp.ScenarioApp) []cliapp.CommandGroup {
	_ = core
	return nil
}

func SubcommandGroups(core *cliapp.ScenarioApp) []cliapp.SubcommandGroup {
	return []cliapp.SubcommandGroup{
		funnels.Register(core),
		projects.Register(core),
		templates.Register(core),
	}
}
