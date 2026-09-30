package captures

import (
	"swarm-manager/cli/internal/support"

	"github.com/vrooli/cli-core/cliapp"
)

func Register(deps support.Dependencies) cliapp.SubcommandGroup {
	return cliapp.SubcommandGroup{
		Name:        "captures",
		Description: "Quick-capture management",
		Subcommands: []cliapp.Command{
			support.APICommand("list", "List captures [--json]", deps.CapturesList),
			support.APICommand("create", "Create a capture (--text TEXT [--file FILE]...) [--json]", deps.CapturesCreate),
			support.APICommand("get", "Get capture details (--id ID) [--json]", deps.CapturesGet),
			support.APICommand("delete", "Delete a capture (--id ID)", deps.CapturesDelete),
			support.APICommand("classify", "Trigger classification for a capture (--id ID) [--json]", deps.CapturesClassify),
		},
	}
}
