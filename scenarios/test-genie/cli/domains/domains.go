package domains

import (
	"github.com/vrooli/cli-core/cliapp"

	"test-genie/cli/domains/local"
	"test-genie/cli/domains/suites"
	"test-genie/cli/domains/system"
	"test-genie/cli/internal/deps"
)

// CommandGroups aggregates the scenario's domain registrations.
func CommandGroups(runtime deps.Runtime) []cliapp.CommandGroup {
	return []cliapp.CommandGroup{
		suites.Register(runtime),
		local.Register(runtime),
		system.Register(runtime),
	}
}
