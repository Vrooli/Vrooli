package suites

import (
	"test-genie/cli/execute"
	"test-genie/cli/internal/deps"

	"github.com/vrooli/cli-core/cliapp"
)

// Register returns the suite-oriented command group.
func Register(runtime deps.Runtime) cliapp.CommandGroup {
	return cliapp.CommandGroup{
		Title: "Suites",
		Commands: []cliapp.Command{
			// execute is a durable_run command: it owns a server-owned start ->
			// follow/wait/reattach lifecycle (execute.RunDurable, built on the cli-core
			// durable_run primitive), so human/--json/--jsonl share one run path and
			// output mode selects only the renderer. It declares the durable_run
			// exception and carries matching cli-core evidence via WithLegacyPrimitive,
			// so the special-case shape is proven, not merely asserted.
			cliapp.Command{
				Name:        "execute",
				NeedsAPI:    true,
				Description: "Execute a suite for a scenario",
				Usage:       execute.UsageLine,
				HelpText:    execute.HelpText(),
				Architecture: cliapp.CommandArchitecture{
					Exception:       cliapp.ExceptionDurableRun,
					ExceptionReason: "owns a server-owned durable run (start -> follow/wait/reattach); output mode is renderer-only",
				},
			}.WithLegacyPrimitive(cliapp.DurableRunLegacy(func(args []string) error { return execute.Run(runtime.Execute, args) })),
		},
	}
}
