package domains

import (
	"ecosystem-manager/cli/internal/appctx"
	"ecosystem-manager/cli/logs"
	"ecosystem-manager/cli/queue"
	"ecosystem-manager/cli/steer"
	"ecosystem-manager/cli/tasks"

	"github.com/vrooli/cli-core/cliapp"
)

// CommandGroups aggregates the scenario's domain command groups.
func CommandGroups(ctx appctx.Context) []cliapp.CommandGroup {
	return []cliapp.CommandGroup{
		tasks.Commands(ctx)[0],
		steer.Commands(ctx),
		queue.Commands(ctx),
		logs.Commands(),
	}
}
