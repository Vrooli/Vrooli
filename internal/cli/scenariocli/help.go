package scenariocli

import "github.com/vrooli/vrooli/internal/cli/commandtree"

func TestHelpText() string {
	spec := commandSpec(CommandTest)
	spec.Args = commandtree.ArgSchema{
		Positionals: []commandtree.PositionalArg{
			{Name: "scenario name", Required: true},
			{Name: "test-genie args"},
		},
		Options: []commandtree.OptionArg{
			{Name: "<test-genie args>", Description: "Arguments after the scenario name pass through to `test-genie execute`"},
		},
	}
	spec.Help.Description = "Run scenario tests as a direct alias for `test-genie --auto-start execute <scenario> ...`.\n\n" +
		"Vrooli does not run a lifecycle test phase or provide wrapper-specific fallback behavior here. Test Genie owns phases, presets, JSON/JSONL output, foreground/background policy, and the exit code.\n\n" +
		"The run is owned by the test-genie SERVER, so this command is cancel-survivable: the run id and a\n" +
		"re-attach command are printed up front, and the run keeps going if your shell/tool times out. Do NOT\n" +
		"poll with repeated checks — just re-attach with the printed `wait` command, which blocks once and exits\n" +
		"with the run's real code (124 on timeout).\n\n" +
		"Run-handle subcommands (proxied to `test-genie runs …`, the durable per-scenario run history):\n" +
		"  wait   <scenario> <run-id> [--timeout <seconds>] [--json]  Block until terminal; exit with the run's code\n" +
		"  status <scenario> <run-id> [--json]                        Live snapshot + recommended next-check backoff\n" +
		"  follow <scenario> <run-id>                                 Stream the run's events to completion\n" +
		"  logs   <scenario> <run-id>                                 Alias for follow (replays a finished run)\n" +
		"  abort  <scenario> <run-id> [--json]                        Cancel a running run (→ aborted)"
	return commandtree.SpecHelpText("", "vrooli scenario test", spec)
}

func RequirementsHelpText() string {
	return commandtree.RenderHelpText(commandtree.Help{
		Title:        "Scenario Requirements Commands",
		Usage:        "vrooli scenario requirements <subcommand> [options]",
		DefaultGroup: "Scenario Requirements",
	}, requirementsCommandSpecs())
}

func RequirementsSnapshotHelpText() string {
	return commandtree.HelpText("", "vrooli scenario requirements snapshot", "Show the latest requirements sync snapshot.", commandtree.Help{}, commandtree.ArgSchema{
		Positionals: []commandtree.PositionalArg{{Name: "scenario name", Required: true}},
	})
}

func requirementsCommandSpecs() []commandtree.Spec[string] {
	return []commandtree.Spec[string]{
		{
			Name:    "report",
			Summary: "Generate requirement coverage summary",
			Group:   "Scenario Requirements",
			Args: commandtree.ArgSchema{
				Positionals: []commandtree.PositionalArg{{Name: "scenario name", Required: true}},
			},
			Handler: "report",
		},
		{
			Name:    "validate",
			Summary: "Validate requirement files",
			Group:   "Scenario Requirements",
			Args: commandtree.ArgSchema{
				Positionals: []commandtree.PositionalArg{{Name: "scenario name", Required: true}},
				Options:     []commandtree.OptionArg{{Name: "--quiet", Description: "Suppress non-error output"}},
			},
			Handler: "validate",
		},
		{
			Name:    "sync",
			Summary: "Sync requirement statuses from local evidence",
			Group:   "Scenario Requirements",
			Args: commandtree.ArgSchema{
				Positionals: []commandtree.PositionalArg{{Name: "scenario name", Required: true}},
			},
			Handler: "sync",
		},
		{
			Name:    "manual-log",
			Summary: "Record manual validation evidence",
			Group:   "Scenario Requirements",
			Args: commandtree.ArgSchema{
				Positionals: []commandtree.PositionalArg{
					{Name: "scenario name", Required: true},
					{Name: "requirement", Required: true},
				},
				Options: []commandtree.OptionArg{
					{Name: "--status", ValueName: "value", Description: "Validation status"},
					{Name: "--notes", ValueName: "text", Description: "Validation notes"},
				},
			},
			Handler: "manual-log",
		},
		{
			Name:    "snapshot",
			Summary: "Show latest requirements sync snapshot",
			Group:   "Scenario Requirements",
			Args: commandtree.ArgSchema{
				Positionals: []commandtree.PositionalArg{{Name: "scenario name", Required: true}},
			},
			Handler: "snapshot",
		},
		{
			Name:    "lint-prd",
			Summary: "Check PRD to requirements mapping",
			Group:   "Scenario Requirements",
			Args: commandtree.ArgSchema{
				Positionals: []commandtree.PositionalArg{{Name: "scenario name", Required: true}},
				Options:     []commandtree.OptionArg{commandtree.JSONOption()},
			},
			Handler: "lint-prd",
		},
		{
			Name:    "phase",
			Summary: "Inspect validations for a single phase",
			Group:   "Scenario Requirements",
			Args: commandtree.ArgSchema{
				Positionals: []commandtree.PositionalArg{{Name: "scenario name", Required: true}},
				Options:     []commandtree.OptionArg{{Name: "--phase", ValueName: "phase", Description: "Requirement phase to inspect"}},
			},
			Handler: "phase",
		},
		{
			Name:    "init",
			Summary: "Scaffold a requirements registry",
			Group:   "Scenario Requirements",
			Args: commandtree.ArgSchema{
				Positionals: []commandtree.PositionalArg{{Name: "scenario name", Required: true}},
			},
			Handler: "init",
		},
	}
}
