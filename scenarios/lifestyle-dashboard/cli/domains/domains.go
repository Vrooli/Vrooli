package domains

import (
	domaincmds "lifestyle-dashboard/cli/domains/domain"
	"lifestyle-dashboard/cli/domains/events"
	"lifestyle-dashboard/cli/domains/stats"

	"github.com/vrooli/cli-core/cliapp"
)

func CommandGroups(core *cliapp.ScenarioApp) []cliapp.CommandGroup {
	_ = core
	return nil
}

func SubcommandGroups(core *cliapp.ScenarioApp) []cliapp.SubcommandGroup {
	return []cliapp.SubcommandGroup{
		events.Register(core),
		domaincmds.Register(core),
		stats.Register(core),
	}
}
