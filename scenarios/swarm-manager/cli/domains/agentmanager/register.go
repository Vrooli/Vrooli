package agentmanager

import (
	"swarm-manager/cli/internal/support"

	"github.com/vrooli/cli-core/cliapp"
)

func Register(deps support.Dependencies) cliapp.SubcommandGroup {
	return cliapp.SubcommandGroup{
		Name:        "agent-manager",
		Description: "Agent-manager integration",
		Subcommands: []cliapp.Command{
			support.APICommand("status", "Get agent-manager availability and profile status", deps.AgentManagerStatus),
			support.APICommand("run-get", "Get run status (--id ID) [--json]", deps.AgentManagerRunGet),
			support.APICommand("run-stop", "Stop a run (--id ID) [--json]", deps.AgentManagerRunStop),
		},
	}
}
