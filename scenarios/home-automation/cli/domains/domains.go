package domains

import (
	"home-automation/cli/domains/automations"
	"home-automation/cli/domains/contexts"
	"home-automation/cli/domains/devices"
	"home-automation/cli/domains/homeassistant"
	"home-automation/cli/domains/profiles"

	"github.com/vrooli/cli-core/cliapp"
)

func CommandGroups(core *cliapp.ScenarioApp) []cliapp.CommandGroup {
	_ = core
	return nil
}

func SubcommandGroups(core *cliapp.ScenarioApp) []cliapp.SubcommandGroup {
	return []cliapp.SubcommandGroup{
		devices.Register(core),
		automations.Register(core),
		contexts.Register(core),
		profiles.Register(core),
		homeassistant.Register(core),
	}
}
