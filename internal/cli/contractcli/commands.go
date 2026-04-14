package contractcli

import (
	"github.com/vrooli/vrooli/internal/cli/commandtree"
)

type CommandID string

const (
	CommandValidate        CommandID = "validate"
	CommandShow            CommandID = "show"
	CommandResolve         CommandID = "resolve"
	CommandMatchGlob       CommandID = "match-glob"
	CommandResolveScenario CommandID = "scenario"
)

func CommandSpecs() []commandtree.Spec[CommandID] {
	return []commandtree.Spec[CommandID]{
		{
			Name:    string(CommandValidate),
			Summary: "Validate repo contract configuration and live drift",
			Group:   "Repository Contract",
			Help: commandtree.Help{
				Description: "Runs schema validation plus in-process semantic and live drift checks.",
			},
			Args: commandtree.ArgSchema{
				Options: []commandtree.OptionArg{commandtree.JSONOption()},
			},
			Handler: CommandValidate,
		},
		{
			Name:    string(CommandShow),
			Summary: "Show the effective repository contract",
			Group:   "Repository Contract",
			Args: commandtree.ArgSchema{
				Options: []commandtree.OptionArg{commandtree.JSONOption()},
			},
			Handler: CommandShow,
		},
		{Name: string(CommandResolve), Summary: "Resolve contract-derived paths", Group: "Repository Contract", Handler: CommandResolve},
		{
			Name:    string(CommandMatchGlob),
			Summary: "Test a contract glob against a path",
			Group:   "Repository Contract",
			Args: commandtree.ArgSchema{
				Positionals: []commandtree.PositionalArg{
					{Name: "pattern", Required: true},
					{Name: "path", Required: true},
				},
				Options: []commandtree.OptionArg{commandtree.JSONOption()},
			},
			Handler: CommandMatchGlob,
		},
	}
}

func ResolveCommandSpecs() []commandtree.Spec[CommandID] {
	return []commandtree.Spec[CommandID]{
		{
			Name:    string(CommandResolveScenario),
			Summary: "Resolve contract paths for a scenario",
			Group:   "Repository Contract",
			Help: commandtree.Help{
				Description: "Known keys: service, docs, requirements, api, ui, cli, initialization",
			},
			Args: commandtree.ArgSchema{
				Positionals: []commandtree.PositionalArg{{Name: "name", Required: true}},
				Options: []commandtree.OptionArg{
					{Name: "--file", ValueName: "key", Description: "Resolve a single contract file key"},
					commandtree.JSONOption(),
				},
			},
			Handler: CommandResolveScenario,
		},
	}
}

func commandSpec(id CommandID) commandtree.Spec[CommandID] {
	for _, spec := range CommandSpecs() {
		if spec.Handler == id {
			return spec
		}
	}
	for _, spec := range ResolveCommandSpecs() {
		if spec.Handler == id {
			return spec
		}
	}
	panic("unknown contract command spec: " + string(id))
}

func commandHelpText(id CommandID) string {
	spec := commandSpec(id)
	base := "vrooli contract " + spec.Name
	if id == CommandResolveScenario {
		base = "vrooli contract resolve scenario"
	}
	return commandtree.SpecHelpText("", base, spec)
}
