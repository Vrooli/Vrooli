package domains

import (
	"elo-swipe/cli/domains/comparisons"
	"elo-swipe/cli/domains/lists"
	"elo-swipe/cli/domains/rankings"
	"elo-swipe/cli/domains/swipe"
	"elo-swipe/cli/internal/client"

	"github.com/vrooli/cli-core/cliapp"
)

func CommandGroups(core *cliapp.ScenarioApp) []cliapp.CommandGroup {
	_ = core
	return nil
}

func SubcommandGroups(core *cliapp.ScenarioApp) []cliapp.SubcommandGroup {
	api := client.New(core)
	return []cliapp.SubcommandGroup{
		lists.Register(api),
		comparisons.Register(api),
		rankings.Register(api),
		swipe.Register(api),
	}
}
