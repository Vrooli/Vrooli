package events

import (
	"agent-manager/cli/internal/support"

	"github.com/vrooli/cli-core/cliapp"
)

func Register(deps support.Dependencies) cliapp.CommandGroup {
	return cliapp.CommandGroup{
		Title: "Events",
		Commands: []cliapp.Command{
			support.Command("events", "Query typed-operational event log", deps.Events),
		},
	}
}
