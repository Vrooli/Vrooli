package domains

import (
	"vrooli-autoheal/cli/domains/checks"
	"vrooli-autoheal/cli/domains/config"
	"vrooli-autoheal/cli/domains/health"
	"vrooli-autoheal/cli/domains/host"
	"vrooli-autoheal/cli/domains/incidents"
	"vrooli-autoheal/cli/domains/measures"
	"vrooli-autoheal/cli/domains/monitoring"
	"vrooli-autoheal/cli/domains/retention"
	"vrooli-autoheal/cli/domains/storm"
	"vrooli-autoheal/cli/domains/timeline"
	"vrooli-autoheal/cli/internal/support"

	"github.com/vrooli/api-core/spacecli"
	"github.com/vrooli/api-core/spacedoc"
	"github.com/vrooli/cli-core/cliapp"
)

func CommandGroups(core *cliapp.ScenarioApp, deps support.Dependencies) []cliapp.CommandGroup {
	return []cliapp.CommandGroup{
		health.Register(core, deps),
		checks.LegacyRegister(core, deps),
		timeline.Commands(core),
		spacecli.CommandGroup(spacecli.Config{Owner: "vrooli-autoheal", Projections: []spacedoc.Projection{
			spacedoc.ProjectionSupervision,
			spacedoc.ProjectionAvailability,
			spacedoc.ProjectionRecovery,
			// substrate ships a space document like the other three; without
			// this line the document is reachable only by file path, which
			// makes this scenario an owner that publishes nothing.
			spacedoc.ProjectionSubstrate,
		}}),
	}
}

func SubcommandGroups(core *cliapp.ScenarioApp, _ support.Dependencies) []cliapp.SubcommandGroup {
	return []cliapp.SubcommandGroup{
		checks.Register(core),
		config.Register(core),
		host.Register(core),
		incidents.Register(core),
		monitoring.Register(core),
		measures.Register(core),
		retention.Register(core),
		storm.Register(core),
		timeline.Register(core),
	}
}
