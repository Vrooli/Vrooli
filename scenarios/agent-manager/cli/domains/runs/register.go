package runs

import (
	"agent-manager/cli/internal/support"

	"github.com/vrooli/cli-core/cliapp"
)

func Register(deps support.Dependencies) cliapp.CommandGroup {
	return cliapp.CommandGroup{
		Title: "Runs",
		Commands: []cliapp.Command{
			support.Command("run", "Manage run executions", deps.Run),
			support.Command("scenario-smoke", "Run the live tracking/provenance smoke check", deps.ScenarioSmoke),
		},
	}
}
