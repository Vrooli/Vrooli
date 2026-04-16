package domains

import (
	"time-tools/cli/domains/event"
	"time-tools/cli/domains/schedule"
	timeops "time-tools/cli/domains/time"

	"github.com/vrooli/cli-core/cliapp"
)

// CommandGroups aggregates flat command groups. Time Tools has no single-verb
// domains — every surface is subcommand-rich — so this returns nothing.
func CommandGroups(_ *cliapp.ScenarioApp) []cliapp.CommandGroup {
	return nil
}

// SubcommandGroups aggregates hierarchical command groups from domain packages.
func SubcommandGroups(core *cliapp.ScenarioApp) []cliapp.SubcommandGroup {
	return []cliapp.SubcommandGroup{
		timeops.Register(core),
		schedule.Register(core),
		event.Register(core),
	}
}
