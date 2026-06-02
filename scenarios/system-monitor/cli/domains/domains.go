package domains

import (
	"system-monitor/cli/domains/investigations"
	"system-monitor/cli/domains/maintenance"
	"system-monitor/cli/domains/metrics"
	"system-monitor/cli/domains/overview"
	"system-monitor/cli/domains/reports"
	"system-monitor/cli/domains/settings"

	"github.com/vrooli/cli-core/cliapp"
)

func CommandGroups(core *cliapp.ScenarioApp) []cliapp.CommandGroup {
	return []cliapp.CommandGroup{
		overview.Register(core),
	}
}

func SubcommandGroups(core *cliapp.ScenarioApp) []cliapp.SubcommandGroup {
	return []cliapp.SubcommandGroup{
		metrics.Register(core),
		investigations.Register(core),
		reports.Register(core),
		settings.Register(core),
		maintenance.Register(core),
	}
}
