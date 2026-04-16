package health

import (
	"swarm-manager/cli/internal/support"

	"github.com/vrooli/cli-core/cliapp"
)

// Register exposes scenario-specific operational commands. Root /health is
// served by cli-core's built-in `status` command, so no status/health wrapper
// lives here.
func Register(deps support.Dependencies) cliapp.CommandGroup {
	return cliapp.CommandGroup{
		Title: "Overview",
		Commands: []cliapp.Command{
			support.APICommand("overview", "Full backlog situational awareness [--format json|markdown]", deps.Overview),
		},
	}
}
