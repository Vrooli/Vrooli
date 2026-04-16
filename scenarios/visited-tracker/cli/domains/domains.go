package domains

import (
	"visited-tracker/cli/domains/analytics"
	"visited-tracker/cli/domains/campaigns"
	"visited-tracker/cli/domains/data"
	"visited-tracker/cli/domains/files"
	"visited-tracker/cli/domains/tracking"

	"github.com/vrooli/cli-core/cliapp"
)

type State struct {
	CampaignID *string
}

// CommandGroups aggregates flat command groups. The root /health probe is
// served by cli-core's built-in `status` command, so no status/health/system
// domain is registered here.
func CommandGroups(core *cliapp.ScenarioApp, state State) []cliapp.CommandGroup {
	return []cliapp.CommandGroup{
		tracking.Register(core, state.CampaignID),
		analytics.Register(core, state.CampaignID),
		data.Register(core, state.CampaignID),
	}
}

func SubcommandGroups(core *cliapp.ScenarioApp, state State) []cliapp.SubcommandGroup {
	return []cliapp.SubcommandGroup{
		campaigns.Register(core, state.CampaignID),
		files.Register(core, state.CampaignID),
	}
}
