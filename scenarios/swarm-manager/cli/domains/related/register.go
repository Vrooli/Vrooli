package related

import (
	"github.com/vrooli/cli-core/cliapp"
	"swarm-manager/cli/internal/support"
)

func Register(deps support.Dependencies) cliapp.SubcommandGroup {
	return cliapp.SubcommandGroup{Name: "related", Description: "Discover linked, scoped, and semantically similar work", Subcommands: []cliapp.Command{support.APICommand("backlog", "Related work for a backlog item (<kind> <name>) [--json]", deps.RelatedBacklog), support.APICommand("initiative", "Related work for an initiative (<name>) [--json]", deps.RelatedInitiative)}}
}
