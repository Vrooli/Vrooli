package domains

import (
	"agent-manager/cli/domains/declarations"
	"agent-manager/cli/domains/events"
	"agent-manager/cli/domains/findings"
	"agent-manager/cli/domains/health"
	"agent-manager/cli/domains/maintenance"
	"agent-manager/cli/domains/measures"
	"agent-manager/cli/domains/ops"
	"agent-manager/cli/domains/policy"
	"agent-manager/cli/domains/profiles"
	"agent-manager/cli/domains/runners"
	"agent-manager/cli/domains/runs"
	"agent-manager/cli/domains/settings"
	"agent-manager/cli/domains/tasks"
	"agent-manager/cli/domains/workflows"
	"agent-manager/cli/internal/support"

	"github.com/vrooli/cli-core/cliapp"
)

func CommandGroups(deps support.Dependencies) []cliapp.CommandGroup {
	return []cliapp.CommandGroup{
		profiles.Register(deps),
		declarations.Register(deps),
		workflows.Register(deps),
		tasks.Register(deps),
		runs.Register(deps),
		runners.Register(deps),
		policy.Register(deps),
		settings.Register(deps),
		maintenance.Register(deps),
		ops.Register(deps),
		health.Register(deps),
		events.Register(deps),
		findings.Register(deps),
	}
}

func SubcommandGroups(deps support.Dependencies) []cliapp.SubcommandGroup {
	return []cliapp.SubcommandGroup{runs.SubcommandGroup(deps), measures.Register()}
}
