package domains

import (
	"scenario-to-desktop/cli/domains/build"
	"scenario-to-desktop/cli/domains/bundle"
	"scenario-to-desktop/cli/domains/deploytarget"
	"scenario-to-desktop/cli/domains/docs"
	"scenario-to-desktop/cli/domains/evidence"
	"scenario-to-desktop/cli/domains/pipeline"
	"scenario-to-desktop/cli/domains/preflight"
	"scenario-to-desktop/cli/domains/signing"
	"scenario-to-desktop/cli/domains/state"
	"scenario-to-desktop/cli/domains/system"
	"scenario-to-desktop/cli/domains/tasks"
	"scenario-to-desktop/cli/domains/telemetry"
	"scenario-to-desktop/cli/internal/support"

	"github.com/vrooli/cli-core/cliapp"
)

func CommandGroups(deps support.Dependencies) []cliapp.CommandGroup {
	return system.CommandGroups(deps)
}

func SubcommandGroups(deps support.Dependencies) []cliapp.SubcommandGroup {
	return []cliapp.SubcommandGroup{
		build.Register(deps),
		pipeline.Register(deps),
		preflight.Register(deps),
		bundle.Register(deps),
		docs.Register(deps),
		deploytarget.Register(deps),
		evidence.Register(deps),
		signing.Register(deps),
		state.Register(deps),
		tasks.Register(deps),
		telemetry.Register(deps),
		system.WineRegister(deps),
	}
}
