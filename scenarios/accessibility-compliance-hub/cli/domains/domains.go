package domains

import (
	"accessibility-compliance-hub/cli/domains/reports"
	"accessibility-compliance-hub/cli/domains/scans"
	"accessibility-compliance-hub/cli/domains/violations"

	"github.com/vrooli/cli-core/cliapp"
)

// CommandGroups aggregates flat command groups. The accessibility-compliance-hub
// CLI currently has no single-verb domains; the slot is kept for parity with
// the reference layout and future growth.
func CommandGroups(_ *cliapp.ScenarioApp) []cliapp.CommandGroup {
	return nil
}

// SubcommandGroups aggregates hierarchical command groups from domain packages.
func SubcommandGroups(core *cliapp.ScenarioApp) []cliapp.SubcommandGroup {
	return []cliapp.SubcommandGroup{
		scans.Register(core),
		violations.Register(core),
		reports.Register(core),
	}
}
