package domains

import (
	"system-monitor/cli/domains/capacity"
	"system-monitor/cli/domains/investigations"
	"system-monitor/cli/domains/maintenance"
	"system-monitor/cli/domains/metrics"
	"system-monitor/cli/domains/overview"
	"system-monitor/cli/domains/reports"
	"system-monitor/cli/domains/settings"

	"github.com/vrooli/api-core/spacecli"
	"github.com/vrooli/api-core/spacedoc"
	"github.com/vrooli/cli-core/cliapp"
)

func CommandGroups(core *cliapp.ScenarioApp) []cliapp.CommandGroup {
	return []cliapp.CommandGroup{
		overview.Register(core),
		spacecli.CommandGroup(spacecli.Config{Owner: "system-monitor", Projection: spacedoc.ProjectionAttribution}),
	}
}

func SubcommandGroups(core *cliapp.ScenarioApp) []cliapp.SubcommandGroup {
	return []cliapp.SubcommandGroup{
		metrics.Register(core),
		investigations.Register(core),
		reports.Register(core),
		settings.Register(core),
		maintenance.Register(core),
		capacity.Register(core),
	}
}
