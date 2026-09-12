package proposals

import (
	"github.com/vrooli/cli-core/cliapp"
	"swarm-manager/cli/internal/support"
)

// Register exposes the operator decision surface for agent-authored mutation
// proposals. Goal and backlog workflows both land here, so this is the group an
// operator uses to accept, partially accept, or bounce agent work.
func Register(deps support.Dependencies) cliapp.SubcommandGroup {
	return cliapp.SubcommandGroup{Name: "proposals", Description: "Decide agent-authored mutation proposals", Subcommands: []cliapp.Command{
		support.APICommand("list", "List proposals [--target-type TYPE] [--target-ref REF] [--status S] [--pending] [--json]", deps.ProposalsList),
		support.APICommand("get", "Show a proposal and its mutations (--id ID) [--session ID] [--json]", deps.ProposalsGet),
		support.APICommand("decide", "Apply a proposal (--id ID) [--accept m1,m2] [--note TEXT] [--json]", deps.ProposalsDecide),
		support.APICommand("accept-keep", "Accept a keep-as-is recommendation (--id ID) [--note TEXT] [--json]", deps.ProposalsAcceptKeep),
		support.APICommand("revise", "Send a proposal back for revision (--id ID) [--note TEXT] [--json]", deps.ProposalsRevise),
	}}
}
