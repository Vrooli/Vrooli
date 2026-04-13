package contractcli

import (
	"fmt"
	"io"
	"strings"

	"github.com/vrooli/vrooli/internal/cli/commandtree"
)

type (
	helpOnlyError struct {
		text string
	}
	NoArgsRequest          struct{}
	ResolveScenarioRequest struct {
		ScenarioName string
		FileKey      string
	}
	MatchGlobRequest struct {
		Pattern string
		Path    string
	}
)

func (e helpOnlyError) Error() string    { return e.text }
func (e helpOnlyError) HelpText() string { return e.text }

func commandHelpOnly(text string) error {
	return helpOnlyError{text: text}
}

func ParseValidateRequest(args []string) (NoArgsRequest, error) {
	for _, arg := range args {
		switch arg {
		case "--help", "-h":
			return NoArgsRequest{}, commandHelpOnly(RenderValidateHelpText())
		default:
			return NoArgsRequest{}, fmt.Errorf("unknown option for contract validate: %s", arg)
		}
	}
	return NoArgsRequest{}, nil
}

func ParseShowRequest(args []string) (NoArgsRequest, error) {
	for _, arg := range args {
		switch arg {
		case "--help", "-h":
			return NoArgsRequest{}, commandHelpOnly(RenderShowHelpText())
		default:
			return NoArgsRequest{}, fmt.Errorf("unknown option for contract show: %s", arg)
		}
	}
	return NoArgsRequest{}, nil
}

func ParseResolveScenarioRequest(args []string) (ResolveScenarioRequest, error) {
	if len(args) == 0 {
		return ResolveScenarioRequest{}, fmt.Errorf("contract resolve scenario requires a scenario name")
	}
	scenarioName := strings.TrimSpace(args[0])
	if scenarioName == "" {
		return ResolveScenarioRequest{}, fmt.Errorf("contract resolve scenario requires a scenario name")
	}
	req := ResolveScenarioRequest{ScenarioName: scenarioName}
	for i := 1; i < len(args); i++ {
		switch args[i] {
		case "--help", "-h":
			return ResolveScenarioRequest{}, commandHelpOnly(RenderResolveScenarioHelpText())
		case "--file":
			if i+1 >= len(args) {
				return ResolveScenarioRequest{}, fmt.Errorf("missing value for --file")
			}
			req.FileKey = strings.TrimSpace(args[i+1])
			i++
		default:
			return ResolveScenarioRequest{}, fmt.Errorf("unknown option for contract resolve scenario: %s", args[i])
		}
	}
	return req, nil
}

func ParseMatchGlobRequest(args []string) (MatchGlobRequest, error) {
	for _, arg := range args {
		if arg == "--help" || arg == "-h" {
			return MatchGlobRequest{}, commandHelpOnly(RenderMatchGlobHelpText())
		}
	}
	if len(args) != 2 {
		return MatchGlobRequest{}, fmt.Errorf("usage: vrooli contract match-glob <pattern> <path>")
	}
	return MatchGlobRequest{Pattern: args[0], Path: args[1]}, nil
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
