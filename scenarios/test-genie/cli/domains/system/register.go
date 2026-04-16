package system

import (
	"github.com/vrooli/cli-core/cliapp"

	"test-genie/cli/internal/deps"
	"test-genie/cli/status"
)

// Register returns the system command group.
func Register(runtime deps.Runtime) cliapp.CommandGroup {
	return cliapp.CommandGroup{
		Title: "System",
		Commands: []cliapp.Command{
			{
				Name:        "status",
				NeedsAPI:    true,
				Description: "Check Test Genie health",
				Usage:       status.UsageLine,
				HelpText:    status.HelpText(),
				Run:         func(args []string) error { return status.Run(runtime.Status) },
			},
		},
	}
}
