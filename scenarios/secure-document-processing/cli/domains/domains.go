package domains

import (
	"secure-document-processing/cli/domains/documents"
	"secure-document-processing/cli/domains/jobs"
	"secure-document-processing/cli/domains/workflows"

	"github.com/vrooli/cli-core/cliapp"
)

// CommandGroups aggregates flat command groups. The secure-document-processing
// CLI currently has no single-verb domains; the slot is kept for parity with
// the reference layout and future growth.
func CommandGroups(_ *cliapp.ScenarioApp) []cliapp.CommandGroup {
	return nil
}

// SubcommandGroups aggregates hierarchical command groups from domain packages.
func SubcommandGroups(core *cliapp.ScenarioApp) []cliapp.SubcommandGroup {
	return []cliapp.SubcommandGroup{
		documents.Register(core),
		jobs.Register(core),
		workflows.Register(core),
	}
}
