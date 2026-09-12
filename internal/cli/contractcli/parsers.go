package contractcli

import (
	"io"
	"strings"

	contractapp "github.com/vrooli/vrooli/internal/app/contract"
	"github.com/vrooli/vrooli/internal/cli/commandtree"
)

type (
	NoArgsRequest struct{}
)

func ParseValidateRequest(args []string) (NoArgsRequest, error) {
	if _, err := commandtree.ParseArgs("contract validate", commandHelpText(CommandValidate), commandSpec(CommandValidate).Args, args); err != nil {
		return NoArgsRequest{}, err
	}
	return NoArgsRequest{}, nil
}

func ParseShowRequest(args []string) (NoArgsRequest, error) {
	if _, err := commandtree.ParseArgs("contract show", commandHelpText(CommandShow), commandSpec(CommandShow).Args, args); err != nil {
		return NoArgsRequest{}, err
	}
	return NoArgsRequest{}, nil
}

func ParseResolveScenarioRequest(args []string) (contractapp.ResolveScenarioRequest, error) {
	spec := commandSpec(CommandResolveScenario)
	parsed, err := commandtree.ParseArgs("contract resolve scenario", commandHelpText(CommandResolveScenario), spec.Args, args)
	if err != nil {
		return contractapp.ResolveScenarioRequest{}, err
	}
	return contractapp.ResolveScenarioRequest{
		ScenarioName: strings.TrimSpace(parsed.Positionals[0]),
		FileKey:      strings.TrimSpace(parsed.FlagValue("--file")),
	}, nil
}

func ParseMatchGlobRequest(args []string) (contractapp.MatchGlobRequest, error) {
	spec := commandSpec(CommandMatchGlob)
	parsed, err := commandtree.ParseArgs("contract match-glob", commandHelpText(CommandMatchGlob), spec.Args, args)
	if err != nil {
		return contractapp.MatchGlobRequest{}, err
	}
	return contractapp.MatchGlobRequest{Pattern: parsed.Positionals[0], Path: parsed.Positionals[1]}, nil
}

func RenderCommandHelp(w io.Writer) {
	commandtree.WriteHelp(w, commandtree.RenderHelpText(commandtree.Help{
		Title:        "vrooli contract - Inspect and validate the repository contract",
		Usage:        "vrooli contract <subcommand> [options]",
		DefaultGroup: "Repository Contract",
	}, CommandSpecs()))
}

func RenderResolveHelp(w io.Writer) {
	commandtree.WriteHelp(w, commandtree.RenderHelpText(commandtree.Help{
		Title:        "vrooli contract resolve - Resolve contract-derived paths",
		Usage:        "vrooli contract resolve <subcommand> [options]",
		DefaultGroup: "Repository Contract",
	}, ResolveCommandSpecs()))
}

func RenderValidateHelp(w io.Writer) {
	commandtree.WriteHelp(w, commandHelpText(CommandValidate))
}

func RenderShowHelp(w io.Writer) {
	commandtree.WriteHelp(w, commandHelpText(CommandShow))
}

func RenderResolveScenarioHelp(w io.Writer) {
	commandtree.WriteHelp(w, commandHelpText(CommandResolveScenario))
}

func RenderMatchGlobHelp(w io.Writer) {
	commandtree.WriteHelp(w, commandHelpText(CommandMatchGlob))
}

func RenderValidateHelpText() string {
	return commandHelpText(CommandValidate)
}

func RenderShowHelpText() string {
	return commandHelpText(CommandShow)
}

func RenderResolveHelpText() string {
	return strings.TrimSuffix(commandtree.RenderHelpText(commandtree.Help{
		Title:        "vrooli contract resolve - Resolve contract-derived paths",
		Usage:        "vrooli contract resolve <subcommand> [options]",
		DefaultGroup: "Repository Contract",
	}, ResolveCommandSpecs()), "\n")
}

func RenderResolveScenarioHelpText() string {
	return commandHelpText(CommandResolveScenario)
}

func RenderMatchGlobHelpText() string {
	return commandHelpText(CommandMatchGlob)
}
