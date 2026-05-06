package scenariocli

import "github.com/vrooli/vrooli/internal/cli/commandtree"

func TestHelpText() string {
	spec := commandSpec(CommandTest)
	spec.Args = commandtree.ArgSchema{
		Positionals: []commandtree.PositionalArg{
			{Name: "scenario name", Required: true},
			{Name: "phase"},
		},
		Options: []commandtree.OptionArg{
			{Name: "--path", ValueName: "path", Description: "Run tests from a custom scenario path"},
			{Name: "--allow-skip-missing-runtime", Description: "Allow a missing runtime to skip execution"},
			{Name: "--manage-runtime", Description: "Start and stop runtime dependencies as part of the test run"},
		},
	}
	spec.Help.Description = "Run scenario tests. Supported selectors include structure, dependencies, unit, integration, business, performance, all, and e2e."
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

func TemplateCommandHelpText() string {
	return commandtree.RenderHelpText(commandtree.Help{
		Title:        "Scenario Template Commands",
		Usage:        "vrooli scenario template <subcommand> [options]",
		DefaultGroup: "Scenario Templates",
	}, templateCommandSpecs())
}

func DesignCommandHelpText() string {
	return commandtree.RenderHelpText(commandtree.Help{
		Title:        "Scenario Design Commands",
		Usage:        "vrooli scenario design <subcommand> [options]",
		DefaultGroup: "Scenario Design",
	}, designCommandSpecs())
}

func TemplateGenerateHelpText() string {
	return commandtree.HelpText("", "vrooli scenario generate", "Scaffold a scenario from a template.", commandtree.Help{
		Usage: "vrooli scenario generate <template> --id <slug> --display-name <name> --description <text> [options]",
	}, templateGenerateArgSchema())
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
