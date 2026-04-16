package domains

import (
	"tech-tree-designer/cli/domains/catalog"
	"tech-tree-designer/cli/domains/graph"
	"tech-tree-designer/cli/domains/milestones"
	"tech-tree-designer/cli/domains/overview"
	"tech-tree-designer/cli/domains/progress"
	"tech-tree-designer/cli/domains/sectors"
	"tech-tree-designer/cli/domains/stages"
	"tech-tree-designer/cli/domains/trees"
	"tech-tree-designer/cli/internal/support"

	"github.com/vrooli/cli-core/cliapp"
)

func CommandGroups(deps support.Dependencies) []cliapp.CommandGroup {
	return []cliapp.CommandGroup{
		overview.Register(deps),
	}
}

func SubcommandGroups(deps support.Dependencies) []cliapp.SubcommandGroup {
	return []cliapp.SubcommandGroup{
		trees.Register(deps),
		sectors.Register(deps),
		stages.Register(deps),
		progress.Register(deps),
		milestones.Register(deps),
		graph.Register(deps),
		catalog.Register(deps),
	}
}
