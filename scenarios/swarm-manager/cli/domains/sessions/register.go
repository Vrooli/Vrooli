package sessions

import (
	"swarm-manager/cli/internal/support"

	"github.com/vrooli/cli-core/cliapp"
)

func Register(deps support.Dependencies) cliapp.SubcommandGroup {
	return cliapp.SubcommandGroup{
		Name:        "sessions",
		Description: "Agent session management",
		Subcommands: []cliapp.Command{
			support.APICommand("list", "List agent sessions [--kind KIND] [--status STATUS] [--active-only] [--limit N] [--json]", deps.SessionsList),
			support.APICommand("get", "Get agent session details (--id ID) [--json]", deps.SessionsGet),
			support.APICommand("create", "Create a draft session (--kind KIND) [--title TEXT] [--starter-job ID] [--target TYPE/REF] [--target-name NAME] [--json]", deps.SessionsCreate),
			support.APICommand("create-batch", "Create bounded draft sessions (--file PATH --actor ACTOR) [--override-reason TEXT] [--json]", deps.SessionsCreateBatch),
			support.APICommand("attach", "Attach typed draft context (--id ID --entity TYPE/REF...) [--json]", deps.SessionsAttach),
			support.APICommand("start", "Start a draft session (--id ID) [--message TEXT] [--json]", deps.SessionsStart),
			support.APICommand("continue", "Continue a running session (--id ID --message TEXT) [--json]", deps.SessionsContinue),
			support.APICommand("complete", "Refresh and complete a terminal session (--id ID) [--json]", deps.SessionsComplete),
			support.APICommand("reap", "Expire running sessions past the configured TTL [--json]", deps.SessionsReap),
			support.APICommand("events", "List session run events (--id ID) [--after-sequence N] [--limit N] [--json]", deps.SessionsEvents),
			support.APICommand("proposal-apply", "Apply accepted proposal mutations (--id ID --proposal ID --accept MUTATION_ID...) [--note TEXT] [--json]", deps.SessionsProposalApply),
			support.APICommand("proposal-revise", "Request proposal revision (--id ID --proposal ID) [--note TEXT] [--json]", deps.SessionsProposalRevise),
			support.APICommand("proposal-wait", "Record an explicit pending-review decision (--id ID --proposal ID --note TEXT) [--json]", deps.SessionsProposalWait),
			support.APICommand("proposal-accept-keep", "Accept a no-change recommendation (--id ID --proposal ID) [--note TEXT] [--json]", deps.SessionsProposalAcceptKeep),
			support.APICommand("startup-brief", "Get draft startup brief (--id ID) [--refresh] [--json]", deps.SessionsStartupBrief),
			support.APICommand("prompt-preview", "Show the prompt a message would send, without sending it (--id ID) [--message TEXT] [--json]", deps.SessionsPromptPreview),
			support.APICommand("delete", "Delete an agent session (--id ID --yes) [--json]", deps.SessionsDelete),
			support.APICommand("disposition", "Disposition a draft session (--id ID --disposition dropped|retained --reason TEXT) [--json]", deps.SessionsDisposition),
		},
	}
}
