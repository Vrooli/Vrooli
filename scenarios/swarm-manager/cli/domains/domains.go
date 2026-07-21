package domains

import (
	"swarm-manager/cli/domains/agentmanager"
	"swarm-manager/cli/domains/aisearch"
	"swarm-manager/cli/domains/autofiler"
	"swarm-manager/cli/domains/backlog"
	"swarm-manager/cli/domains/captures"
	"swarm-manager/cli/domains/evidence"
	"swarm-manager/cli/domains/execution"
	"swarm-manager/cli/domains/health"
	"swarm-manager/cli/domains/initiatives"
	"swarm-manager/cli/domains/migration"
	"swarm-manager/cli/domains/operations"
	"swarm-manager/cli/domains/portfolio"
	"swarm-manager/cli/domains/prompts"
	"swarm-manager/cli/domains/queue"
	"swarm-manager/cli/domains/records"
	"swarm-manager/cli/domains/related"
	"swarm-manager/cli/domains/scenarios"
	"swarm-manager/cli/domains/sessions"
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
		evidence.Register(deps),
		prompts.Register(deps),
		initiatives.Register(deps),
		captures.Register(deps),
		records.Register(deps),
		related.Register(deps),
		agentmanager.Register(deps),
		operations.Register(deps),
		portfolio.Register(deps),
		sessions.Register(deps),
		stats.Register(deps),
		aisearch.Register(deps),
		autofiler.Register(deps),
	}
}
