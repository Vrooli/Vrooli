package domains

import (
	"vrooli-events/cli/domains/events"
	"vrooli-events/cli/domains/store"

	"github.com/vrooli/cli-core/cliapp"
)

func CommandGroups(core *cliapp.ScenarioApp) []cliapp.CommandGroup {
	return []cliapp.CommandGroup{
		events.Register(core),
		store.Register(core),
	}
}
