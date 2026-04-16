package domains

import (
	"chart-generator/cli/domains/chart"
	"chart-generator/cli/domains/data"
	"chart-generator/cli/domains/style"
	"chart-generator/cli/domains/template"

	"github.com/vrooli/cli-core/cliapp"
)

// CommandGroups aggregates flat command groups. The chart-generator CLI
// currently has no single-verb domains; the slot is kept for parity with the
// reference layout and future growth.
func CommandGroups(_ *cliapp.ScenarioApp) []cliapp.CommandGroup {
	return nil
}

// SubcommandGroups aggregates hierarchical command groups from domain packages.
func SubcommandGroups(core *cliapp.ScenarioApp) []cliapp.SubcommandGroup {
	return []cliapp.SubcommandGroup{
		chart.Register(core),
		style.Register(core),
		template.Register(core),
		data.Register(core),
	}
}
