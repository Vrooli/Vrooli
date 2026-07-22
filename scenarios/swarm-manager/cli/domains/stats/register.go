package stats

import (
	"swarm-manager/cli/internal/support"

	"github.com/vrooli/cli-core/cliapp"
)

func Register(deps support.Dependencies) cliapp.SubcommandGroup {
	return cliapp.SubcommandGroup{
		Name:        "stats",
		Description: "Event-driven analytics and metrics",
		Subcommands: []cliapp.Command{
			support.APICommand("summary", "Full stats dashboard", deps.StatsSummary),
			support.APICommand("throughput", "Throughput metrics", deps.StatsThroughput),
			support.APICommand("blocking", "Blocking analysis", deps.StatsBlocking),
			support.APICommand("milestones", "Milestone health", deps.StatsMilestones),
			support.APICommand("agent", "Agent efficiency metrics", deps.StatsAgent),
			support.APICommand("sessions", "Native agent session metrics", deps.StatsSessions),
			support.APICommand("sandbox-adoption", "Sandbox-default rollout adoption breakdown (scrapes agent-manager /metrics)", deps.StatsSandboxAdoption),
		},
	}
}
