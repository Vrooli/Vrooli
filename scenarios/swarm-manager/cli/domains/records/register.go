package records

import (
	"swarm-manager/cli/internal/support"

	"github.com/vrooli/cli-core/cliapp"
)

func Register(deps support.Dependencies) cliapp.SubcommandGroup {
	return cliapp.SubcommandGroup{
		Name:        "records",
		Description: "Narrative artifacts of completed work (recursive-learning write side)",
		Subcommands: []cliapp.Command{
			support.APICommand("list", "List records [--scenario X] [--kind K] [--backlog-ref kind/name] [--include-stubs] [--limit N] [--offset N] [--json]", deps.RecordsList),
			support.APICommand("get", "Get a record (--id ID) [--json]", deps.RecordsGet),
			support.APICommand("create", "Create a record (--kind K --scenario X --trigger '...' [--approach '...'] [--ruled-out '...']... [--commit SHA] [--files PATH]... [--backlog-ref kind/name] [--supersedes ID] [--outcome ...]) [--json]", deps.RecordsCreate),
			support.APICommand("edit", "Fill a stub record's narrative (--id ID --trigger '...' --approach '...' [--ruled-out '...']... [--commit SHA] [--files PATH]... [--outcome ...]) [--json]", deps.RecordsEdit),
			support.APICommand("search", "Semantic search over records ('<query>' [--kind K] [--scenario X] [--limit N]) [--json]", deps.RecordsSearch),
			support.APICommand("supersede", "Mark a record superseded by a successor (--id ID --by SUCCESSOR-ID [--reason '...']) [--json]", deps.RecordsSupersede),
		},
	}
}
