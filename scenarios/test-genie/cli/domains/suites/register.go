package suites

import (
	"test-genie/cli/execute"
	"test-genie/cli/generate"
	"test-genie/cli/internal/deps"

	"github.com/vrooli/cli-core/cliapp"
)

// Register returns the suite-oriented command group.
func Register(runtime deps.Runtime) cliapp.CommandGroup {
	return cliapp.CommandGroup{
		Title: "Suites",
		Commands: []cliapp.Command{
			{
				Name:        "generate",
				NeedsAPI:    true,
				Description: "Queue suite generation",
				Usage:       generate.UsageLine,
				HelpText:    generate.HelpText(),
				Run:         func(args []string) error { return generate.Run(runtime.Generate, args) },
			},
			{
				Name:        "execute",
				NeedsAPI:    true,
				Description: "Execute a suite for a scenario",
				Usage:       execute.UsageLine,
				HelpText:    execute.HelpText(),
				Run:         func(args []string) error { return execute.Run(runtime.Execute, args) },
			},
		},
	}
}
