package contractcli

import (
	"fmt"
	"io"
	"strings"

	contractapp "github.com/vrooli/vrooli/internal/app/contract"
	"github.com/vrooli/vrooli/internal/cli/clipolicy"
	"github.com/vrooli/vrooli/internal/cli/commandtree"
)

type (
	NoArgsRequest struct{}
)

func ParseValidateRequest(args []string) (NoArgsRequest, error) {
	for _, arg := range args {
		switch arg {
		case "--help", "-h":
			return NoArgsRequest{}, clipolicy.CommandHelpOnly(RenderValidateHelpText())
		default:
			return NoArgsRequest{}, clipolicy.UnknownOptionError("contract validate", arg)
		}
	}
	return NoArgsRequest{}, nil
}

func ParseShowRequest(args []string) (NoArgsRequest, error) {
	for _, arg := range args {
		switch arg {
		case "--help", "-h":
			return NoArgsRequest{}, clipolicy.CommandHelpOnly(RenderShowHelpText())
		default:
			return NoArgsRequest{}, clipolicy.UnknownOptionError("contract show", arg)
		}
	}
	return NoArgsRequest{}, nil
}

func ParseResolveScenarioRequest(args []string) (contractapp.ResolveScenarioRequest, error) {
	if len(args) == 0 {
		return contractapp.ResolveScenarioRequest{}, fmt.Errorf("contract resolve scenario requires a scenario name")
	}
	scenarioName := strings.TrimSpace(args[0])
	if scenarioName == "" {
		return contractapp.ResolveScenarioRequest{}, fmt.Errorf("contract resolve scenario requires a scenario name")
	}
	req := contractapp.ResolveScenarioRequest{ScenarioName: scenarioName}
	for i := 1; i < len(args); i++ {
		switch args[i] {
		case "--help", "-h":
			return contractapp.ResolveScenarioRequest{}, clipolicy.CommandHelpOnly(RenderResolveScenarioHelpText())
		case "--file":
			if i+1 >= len(args) {
				return contractapp.ResolveScenarioRequest{}, clipolicy.UsageErrorf("contract resolve scenario", "missing value for --file")
			}
			req.FileKey = strings.TrimSpace(args[i+1])
			i++
		default:
			return contractapp.ResolveScenarioRequest{}, clipolicy.UnknownOptionError("contract resolve scenario", args[i])
		}
	}
	return req, nil
}

func ParseMatchGlobRequest(args []string) (contractapp.MatchGlobRequest, error) {
	for _, arg := range args {
		if arg == "--help" || arg == "-h" {
			return contractapp.MatchGlobRequest{}, clipolicy.CommandHelpOnly(RenderMatchGlobHelpText())
		}
	}
	if len(args) != 2 {
		return contractapp.MatchGlobRequest{}, clipolicy.UsageErrorf("contract match-glob", "usage: vrooli contract match-glob <pattern> <path>")
	}
	return contractapp.MatchGlobRequest{Pattern: args[0], Path: args[1]}, nil
}

func RenderCommandHelp(w io.Writer) {
	_, _ = fmt.Fprintln(w, "vrooli contract - Inspect and validate the repository contract")
	_, _ = fmt.Fprintln(w)
	_, _ = fmt.Fprintln(w, "Usage:")
	_, _ = fmt.Fprintln(w, "  vrooli contract <subcommand> [options]")
	_, _ = fmt.Fprintln(w)
	commandtree.RenderGroups(w, commandtree.VisibleEntries(CommandSpecs(), ""))
}

func RenderResolveHelp(w io.Writer) {
	_, _ = fmt.Fprintln(w, RenderResolveHelpText())
}

func RenderValidateHelp(w io.Writer) {
	_, _ = fmt.Fprintln(w, RenderValidateHelpText())
}

func RenderShowHelp(w io.Writer) {
	_, _ = fmt.Fprintln(w, RenderShowHelpText())
}

func RenderResolveScenarioHelp(w io.Writer) {
	_, _ = fmt.Fprintln(w, RenderResolveScenarioHelpText())
}

func RenderMatchGlobHelp(w io.Writer) {
	_, _ = fmt.Fprintln(w, RenderMatchGlobHelpText())
}

func RenderValidateHelpText() string {
	return ValidateHelpText
}

func RenderShowHelpText() string {
	return ShowHelpText
}

func RenderResolveHelpText() string {
	return ResolveHelpText
}

func RenderResolveScenarioHelpText() string {
	return ResolveScenarioHelpText
}

func RenderMatchGlobHelpText() string {
	return MatchGlobHelpText
}
