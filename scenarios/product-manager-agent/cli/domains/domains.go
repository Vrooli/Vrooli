package domains

import (
	"product-manager-agent/cli/domains/analyze"
	"product-manager-agent/cli/domains/dashboard"
	"product-manager-agent/cli/domains/features"
	"product-manager-agent/cli/domains/roadmap"
	"product-manager-agent/cli/domains/sprint"

	"github.com/vrooli/cli-core/cliapp"
)

// CommandGroups aggregates flat command groups. Single-verb domains live here so
// the invocation stays `product-manager-agent <verb>` instead of
// `product-manager-agent <verb> <sub>`.
func CommandGroups(core *cliapp.ScenarioApp) []cliapp.CommandGroup {
	return []cliapp.CommandGroup{
		dashboard.Register(core),
	}
}

// SubcommandGroups aggregates hierarchical command groups from domain packages.
func SubcommandGroups(core *cliapp.ScenarioApp) []cliapp.SubcommandGroup {
	return []cliapp.SubcommandGroup{
		features.Register(core),
		roadmap.Register(core),
		sprint.Register(core),
		analyze.Register(core),
	}
}
