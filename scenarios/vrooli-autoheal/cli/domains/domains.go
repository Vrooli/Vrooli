package domains

import (
	"vrooli-autoheal/cli/domains/checks"
	"vrooli-autoheal/cli/domains/config"
	"vrooli-autoheal/cli/domains/health"
	"vrooli-autoheal/cli/domains/host"
	"vrooli-autoheal/cli/domains/incidents"
	"vrooli-autoheal/cli/domains/monitoring"
	"vrooli-autoheal/cli/domains/timeline"
	"vrooli-autoheal/cli/internal/support"

	"github.com/vrooli/cli-core/cliapp"
)

func CommandGroups(core *cliapp.ScenarioApp, deps support.Dependencies) []cliapp.CommandGroup {
	return []cliapp.CommandGroup{
		health.Register(core, deps),
		checks.LegacyRegister(core, deps),
		timeline.Commands(core),
	}
}

func SubcommandGroups(core *cliapp.ScenarioApp, _ support.Dependencies) []cliapp.SubcommandGroup {
	return []cliapp.SubcommandGroup{
		checks.Register(core),
		config.Register(core),
		host.Register(core),
		incidents.Register(core),
		monitoring.Register(core),
		timeline.Register(core),
	}
}
