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
			support.APICommand("brief", "Get the bounded operating-mode authoring brief [--mode MODE] [--json]", deps.OperatingModeBrief),
			support.APICommand("set", "Edit a mode's label/description (--mode MODE [--label LABEL] [--description TEXT | --clear-description]) [--json]", deps.OperatingModeSet),
			support.APICommand("scaffold", "Scaffold a new mode folder from the template (--id MODE [--label LABEL] [--description TEXT] [--force]) [--json]", deps.OperatingModeScaffold),
			support.APICommand("validate", "Validate a mode from disk (--mode MODE) [--json]", deps.OperatingModeValidate),
			support.APICommand("simulate", "Simulate a mode's phase walk (--mode MODE [--preset ID] [--registered]) [--json]", deps.OperatingModeSimulate),
			support.APICommand("start", "Start a round of a plan-target mode directly on its target (--mode MODE --target REF [--phase PHASE] [--note MSG] [--override]) [--json]", deps.OperatingModeStart),
		},
	}
}
