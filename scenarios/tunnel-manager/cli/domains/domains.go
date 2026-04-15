package domains

import (
	"tunnel-manager/cli/domains/audit"
	"tunnel-manager/cli/domains/health"
	"tunnel-manager/cli/domains/metrics"
	"tunnel-manager/cli/domains/probes"
	"tunnel-manager/cli/domains/recovery"
	"tunnel-manager/cli/domains/routes"
	"tunnel-manager/cli/internal/support"

	"github.com/vrooli/cli-core/cliapp"
)

func CommandGroups(deps support.Dependencies) []cliapp.CommandGroup {
	return []cliapp.CommandGroup{
		health.CommandGroup(deps),
	}
}

func SubcommandGroups(deps support.Dependencies) []cliapp.SubcommandGroup {
	return []cliapp.SubcommandGroup{
		health.SubcommandGroup(deps),
		routes.Register(deps),
		probes.Register(deps),
		audit.Register(deps),
		metrics.Register(deps),
		recovery.Register(deps),
	}
}
