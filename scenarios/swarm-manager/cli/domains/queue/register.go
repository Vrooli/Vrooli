package queue

import (
	"swarm-manager/cli/internal/support"

	"github.com/vrooli/cli-core/cliapp"
)

func Register(deps support.Dependencies) cliapp.SubcommandGroup {
	return cliapp.SubcommandGroup{
		Name:        "queue",
		Description: "Execution queue operations",
		Subcommands: []cliapp.Command{
			support.APICommand("list", "List queue items", deps.QueueList),
			support.APICommand("create", "Create a queue item (--kind KIND [--data JSON])", deps.QueueCreate),
			support.APICommand("delete", "Delete a queue item (--id ID)", deps.QueueDelete),
		},
	}
}
