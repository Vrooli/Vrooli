package ops

import (
	"agent-manager/cli/internal/support"

	"github.com/vrooli/cli-core/cliapp"
)

func Register(deps support.Dependencies) cliapp.CommandGroup {
	return cliapp.CommandGroup{
		Title: "Operational Stats",
		Commands: []cliapp.Command{
			support.Command("ops", "Inspect typed-event operational stats", deps.Ops),
		},
	}
}
