package domains

import (
	"visited-tracker/cli/domains/analytics"
	"visited-tracker/cli/domains/campaigns"
	"visited-tracker/cli/domains/data"
	"visited-tracker/cli/domains/files"
	"visited-tracker/cli/domains/health"
	"visited-tracker/cli/domains/tracking"

	"github.com/vrooli/cli-core/cliapp"
)

type State struct {
	CampaignID *string
}

func CommandGroups(core *cliapp.ScenarioApp, state State) []cliapp.CommandGroup {
	return []cliapp.CommandGroup{
		health.Register(core, state.CampaignID),
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
