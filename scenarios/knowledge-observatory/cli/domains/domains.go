package domains

import (
	"knowledge-observatory/cli/domains/docs"
	"knowledge-observatory/cli/domains/health"
	"knowledge-observatory/cli/domains/knowledge"
	"knowledge-observatory/cli/internal/support"

	"github.com/vrooli/cli-core/cliapp"
)

func CommandGroups(deps support.Dependencies) []cliapp.CommandGroup {
	return []cliapp.CommandGroup{
		health.Register(deps),
		knowledge.Register(deps),
		docs.Register(deps),
	}
}
