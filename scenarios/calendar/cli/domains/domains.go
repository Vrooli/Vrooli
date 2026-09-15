package domains

import (
	"calendar/cli/domains/events"
	"calendar/cli/domains/schedule"
	"calendar/cli/domains/sync"

	"github.com/vrooli/cli-core/cliapp"
)

// CommandGroups aggregates flat command groups. The calendar CLI currently has
// no single-verb domains, but the slot is kept for parity with the reference
// layout and future growth.
func CommandGroups(_ *cliapp.ScenarioApp) []cliapp.CommandGroup {
	return nil
}

// SubcommandGroups aggregates hierarchical command groups from domain packages.
func SubcommandGroups(core *cliapp.ScenarioApp) []cliapp.SubcommandGroup {
	return []cliapp.SubcommandGroup{
		events.Register(core),
		schedule.Register(core),
		sync.Register(core),
	}
}
