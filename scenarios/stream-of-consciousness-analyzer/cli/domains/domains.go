package domains

import (
	"stream-of-consciousness-analyzer/cli/domains/edges"
	"stream-of-consciousness-analyzer/cli/domains/information"
	"stream-of-consciousness-analyzer/cli/domains/providers"
	"stream-of-consciousness-analyzer/cli/domains/schemes"
	"stream-of-consciousness-analyzer/cli/domains/suggestions"
	"stream-of-consciousness-analyzer/cli/domains/thoughts"

	"github.com/vrooli/cli-core/cliapp"
)

func CommandGroups(core *cliapp.ScenarioApp) []cliapp.CommandGroup {
	_ = core
	return nil
}

func SubcommandGroups(core *cliapp.ScenarioApp) []cliapp.SubcommandGroup {
	return []cliapp.SubcommandGroup{
		schemes.Register(core),
		thoughts.Register(core),
		edges.Register(core),
		information.Register(core),
		providers.Register(core),
		suggestions.Register(core),
	}
}
