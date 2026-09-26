package review

import (
	"swarm-manager/cli/internal/support"

	"github.com/vrooli/cli-core/cliapp"
)

func Register(deps support.Dependencies) cliapp.SubcommandGroup {
	return cliapp.SubcommandGroup{Name: "review", Description: "Independent review evidence", Subcommands: []cliapp.Command{
		support.APICommand("list", "List review evidence rounds (--kind KIND --name NAME)", deps.ReviewList),
		support.APICommand("verify", "Mark evidence item as verified (--kind KIND --name NAME --round N --evidence-id ID)", deps.ReviewVerify),
		support.APICommand("request", "Request additional evidence (--kind KIND --name NAME --round N --message MSG)", deps.ReviewRequest),
		support.APICommand("trigger", "Manually trigger review agent (--id EXECUTION_ID --kind KIND --name NAME)", deps.ReviewTrigger),
	}}
}
