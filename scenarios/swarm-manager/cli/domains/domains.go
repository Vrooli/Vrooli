package domains

import (
	"swarm-manager/cli/domains/agentmanager"
	"swarm-manager/cli/domains/autofiler"
	"swarm-manager/cli/domains/backlog"
	"swarm-manager/cli/domains/captures"
	"swarm-manager/cli/domains/evidence"
	"swarm-manager/cli/domains/execution"
	"swarm-manager/cli/domains/goals"
	"swarm-manager/cli/domains/health"
	"swarm-manager/cli/domains/measures"
	"swarm-manager/cli/domains/milestones"
	"swarm-manager/cli/domains/operations"
	"swarm-manager/cli/domains/portfolio"
	"swarm-manager/cli/domains/prompts"
	"swarm-manager/cli/domains/proposals"
	"swarm-manager/cli/domains/queue"
	"swarm-manager/cli/domains/records"
	"swarm-manager/cli/domains/review"
	"swarm-manager/cli/domains/scenarios"
	"swarm-manager/cli/domains/search"
	"swarm-manager/cli/domains/sessions"
	"swarm-manager/cli/domains/settings"
	"swarm-manager/cli/domains/stats"
	"swarm-manager/cli/domains/transitions"
	"swarm-manager/cli/internal/support"

	"github.com/vrooli/cli-core/cliapp"
)

func CommandGroups(deps support.Dependencies) []cliapp.CommandGroup {
	return []cliapp.CommandGroup{
		health.Register(deps),
	}
}

func SubcommandGroups(deps support.Dependencies) []cliapp.SubcommandGroup {
	return []cliapp.SubcommandGroup{
		backlog.Register(deps),
		scenarios.Register(deps),
		settings.Register(deps),
		queue.Register(deps),
		execution.Register(deps),
		review.Register(deps),
		evidence.Register(),
		prompts.Register(deps),
		goals.Register(deps),
		milestones.Register(deps),
		proposals.Register(deps),
		captures.Register(deps),
		records.Register(deps),
		agentmanager.Register(deps),
		operations.Register(deps),
		portfolio.Register(deps),
		sessions.Register(deps),
		stats.Register(deps),
		measures.Register(),
		search.Register(deps),
		autofiler.Register(),
		transitions.Register(deps),
	}
}
