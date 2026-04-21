package initiatives

import (
	"swarm-manager/cli/internal/support"

	"github.com/vrooli/cli-core/cliapp"
)

func Register(deps support.Dependencies) cliapp.SubcommandGroup {
	return cliapp.SubcommandGroup{
		Name:        "initiatives",
		Description: "Initiative management",
		Subcommands: []cliapp.Command{
			support.APICommand("list", "List initiatives [--json]", deps.InitiativesList),
			support.APICommand("get", "Get initiative details (--name NAME) [--json]", deps.InitiativesGet),
			support.APICommand("context", "Get an initiative with its member items + related initiatives (--name NAME) [--json]", deps.InitiativesContext),
			support.APICommand("create", "Create initiative (--data JSON) [--json]", deps.InitiativesCreate),
			support.APICommand("update", "Update initiative (--name NAME --data JSON) [--json]", deps.InitiativesUpdate),
			support.APICommand("delete", "Delete initiative (--name NAME)", deps.InitiativesDelete),
			support.APICommand("add-items", "Add items to initiative (--name NAME --items kind/name,...) [--json]", deps.InitiativesAddItems),
			support.APICommand("remove-items", "Remove items from initiative (--name NAME --items kind/name,...) [--json]", deps.InitiativesRemove),
			support.APICommand("files", "List files in an initiative (--name NAME) [--json]", deps.InitiativesFiles),
			support.APICommand("file-get", "Get a file from an initiative (--name NAME --path PATH) [--out FILE] [--json]", deps.InitiativesFileGet),
			support.APICommand("file-upload", "Upload a file to an initiative (--name NAME --path PATH) (--stdin|--file|--content)", deps.InitiativesFileUp),
			support.APICommand("file-op", "File operation on initiative (--name NAME --op OP --source PATH) [--dest PATH] [--json]", deps.InitiativesFileOp),
			support.APICommand("search-ai", "Semantic search over initiatives (<query> [--limit N] [--status S,...] [--include-archived] [--json])", deps.InitiativesSearchAI),
		},
	}
}
