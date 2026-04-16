package domains

import (
	"workspace-sandbox/cli/domains/changes"
	"workspace-sandbox/cli/domains/health"
	"workspace-sandbox/cli/domains/maintenance"
	"workspace-sandbox/cli/domains/process"
	"workspace-sandbox/cli/domains/provenance"
	"workspace-sandbox/cli/domains/sandbox"
	"workspace-sandbox/cli/internal/support"

	"github.com/vrooli/cli-core/cliapp"
)

func CommandGroups(deps support.Dependencies) []cliapp.CommandGroup {
	return []cliapp.CommandGroup{
		health.Register(deps),
	}
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
