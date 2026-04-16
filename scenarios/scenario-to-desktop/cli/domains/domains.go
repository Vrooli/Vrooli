package domains

import (
	"scenario-to-desktop/cli/domains/bundle"
	"scenario-to-desktop/cli/domains/deploytarget"
	"scenario-to-desktop/cli/domains/pipeline"
	"scenario-to-desktop/cli/domains/signing"
	"scenario-to-desktop/cli/domains/system"
	"scenario-to-desktop/cli/domains/telemetry"
	"scenario-to-desktop/cli/internal/support"

	"github.com/vrooli/cli-core/cliapp"
)

func CommandGroups(deps support.Dependencies) []cliapp.CommandGroup {
	return system.CommandGroups(deps)
}

func SubcommandGroups(deps support.Dependencies) []cliapp.SubcommandGroup {
	return []cliapp.SubcommandGroup{
		pipeline.Register(deps),
		bundle.Register(deps),
		telemetry.Register(deps),
		signing.Register(deps),
		deploytarget.Register(deps),
		system.WineRegister(deps),
	}
}
