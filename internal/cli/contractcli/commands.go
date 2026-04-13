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
		{Name: string(CommandValidate), Summary: "Validate repo contract configuration and live drift", Group: "Repository Contract", Handler: CommandValidate},
		{Name: string(CommandShow), Summary: "Show the effective repository contract", Group: "Repository Contract", Handler: CommandShow},
		{Name: string(CommandResolve), Summary: "Resolve contract-derived paths", Group: "Repository Contract", Handler: CommandResolve},
		{Name: string(CommandMatchGlob), Summary: "Test a contract glob against a path", Group: "Repository Contract", Handler: CommandMatchGlob},
	}
}

func ResolveCommandSpecs() []commandtree.Spec[CommandID] {
	return []commandtree.Spec[CommandID]{
		{Name: string(CommandResolveScenario), Summary: "Resolve contract paths for a scenario", Group: "Repository Contract", Handler: CommandResolveScenario},
	}
}
