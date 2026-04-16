package domains

import (
	"swarm-manager/cli/domains/agentmanager"
	"swarm-manager/cli/domains/backlog"
	"swarm-manager/cli/domains/captures"
	"swarm-manager/cli/domains/execution"
	"swarm-manager/cli/domains/health"
	"swarm-manager/cli/domains/initiatives"
	"swarm-manager/cli/domains/migration"
	"swarm-manager/cli/domains/prompts"
	"swarm-manager/cli/domains/queue"
	"swarm-manager/cli/domains/scenarios"
	"swarm-manager/cli/domains/settings"
	"swarm-manager/cli/domains/stats"
	"swarm-manager/cli/internal/support"

	"github.com/vrooli/cli-core/cliapp"
)

func CommandGroups(deps support.Dependencies) []cliapp.CommandGroup {
	return []cliapp.CommandGroup{
		health.Register(deps),
		migration.Register(deps),
	}
}

func SubcommandGroups(deps support.Dependencies) []cliapp.SubcommandGroup {
	return []cliapp.SubcommandGroup{
		backlog.Register(deps),
		scenarios.Register(deps),
		settings.Register(deps),
		queue.Register(deps),
		execution.Register(deps),
		prompts.Register(deps),
		initiatives.Register(deps),
		captures.Register(deps),
		agentmanager.Register(deps),
		stats.Register(deps),
	}
}
