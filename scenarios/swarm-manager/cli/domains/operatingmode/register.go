package operatingmode

import (
	"swarm-manager/cli/internal/support"

	"github.com/vrooli/cli-core/cliapp"
)

func Register(deps support.Dependencies) cliapp.SubcommandGroup {
	return cliapp.SubcommandGroup{
		Name:        "operating-mode",
		Description: "Operating-mode catalog management",
		Subcommands: []cliapp.Command{
			support.APICommand("list", "List registered operating modes with usage counts [--json]", deps.OperatingModeList),
			support.APICommand("get", "Get a single mode (--mode MODE) including linked initiatives [--json]", deps.OperatingModeGet),
			support.APICommand("set", "Edit a mode's label/description (--mode MODE [--label LABEL] [--description TEXT | --clear-description]) [--json]", deps.OperatingModeSet),
		},
	}
}
