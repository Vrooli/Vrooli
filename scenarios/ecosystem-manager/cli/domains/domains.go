package domains

import (
	"ecosystem-manager/cli/domains/discovery"
	"ecosystem-manager/cli/internal/appctx"
	"ecosystem-manager/cli/logs"
	"ecosystem-manager/cli/queue"
	"ecosystem-manager/cli/steer"
	"ecosystem-manager/cli/tasks"

	"github.com/vrooli/cli-core/cliapp"
)

// CommandGroups aggregates the scenario's domain command groups. core is
// threaded through for Connect-RPC-migrated domains (discovery) whose handlers
// build a generated Connect client; REST domains use the appctx wrapper.
func CommandGroups(ctx appctx.Context, core *cliapp.ScenarioApp) []cliapp.CommandGroup {
	return []cliapp.CommandGroup{
		tasks.Commands(ctx)[0],
		steer.Commands(ctx),
		queue.Commands(ctx),
		discovery.Commands(core),
		logs.Commands(),
	}
}
