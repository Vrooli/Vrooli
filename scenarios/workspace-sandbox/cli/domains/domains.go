package domains

import (
	"workspace-sandbox/cli/domains/changes"
	"workspace-sandbox/cli/domains/maintenance"
	"workspace-sandbox/cli/domains/process"
	"workspace-sandbox/cli/domains/provenance"
	"workspace-sandbox/cli/domains/sandbox"
	"workspace-sandbox/cli/internal/support"

	"github.com/vrooli/cli-core/cliapp"
)

// CommandGroups aggregates flat command groups. The root /health probe is
// served by cli-core's built-in `status` command, so no status/health/system
// domain is registered here.
func CommandGroups(_ support.Dependencies) []cliapp.CommandGroup {
	return nil
}

func SubcommandGroups(deps support.Dependencies) []cliapp.SubcommandGroup {
	return []cliapp.SubcommandGroup{
		sandbox.Register(deps),
		process.Register(deps),
		changes.Register(deps),
		maintenance.Register(deps),
		provenance.Register(deps),
	}
}
