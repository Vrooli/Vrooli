package scenarios

import (
	"swarm-manager/cli/internal/support"

	"github.com/vrooli/cli-core/cliapp"
)

func Register(deps support.Dependencies) cliapp.SubcommandGroup {
	return cliapp.SubcommandGroup{
		Name:        "scenarios",
		Description: "Scenario catalog and lifecycle",
		Subcommands: []cliapp.Command{
			support.APICommand("list", "List scenarios [--search, --status, --tags, --sort, --order]", deps.ScenariosList),
			support.APICommand("get", "Get scenario details (--name NAME)", deps.ScenariosGet),
			support.APICommand("update", "Update scenario metadata (--name NAME --data JSON)", deps.ScenariosUpdate),
			support.APICommand("delete", "Delete a scenario (--name NAME [--archive])", deps.ScenariosDelete),
			support.APICommand("files", "List scenario files (--name NAME)", deps.ScenariosFiles),
			support.APICommand("spec-sync-archive", "Queue spec-sync-archive execution (--name NAME)", deps.ScenariosSpecSync),
			support.APICommand("start", "Start a scenario (--name NAME)", deps.ScenariosStart),
			support.APICommand("stop", "Stop a scenario (--name NAME)", deps.ScenariosStop),
			support.APICommand("restart", "Restart a scenario (--name NAME)", deps.ScenariosRestart),
			support.APICommand("review-queue", "Get prioritized scenario review queue [--limit N]", deps.ScenariosReviewQueue),
		},
	}
}
