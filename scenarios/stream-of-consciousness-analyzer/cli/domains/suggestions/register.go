package suggestions

import (
	"fmt"
	"os"

	"stream-of-consciousness-analyzer/cli/internal/support"

	"github.com/vrooli/cli-core/cliapp"
)

const cliName = "stream-of-consciousness-analyzer"

type response struct {
	Suggestions []struct {
		Label      string  `json:"label"`
		Confidence float64 `json:"confidence"`
	} `json:"suggestions"`
	Provider string `json:"provider"`
}

func Register(core *cliapp.ScenarioApp) cliapp.SubcommandGroup {
	return cliapp.SubcommandGroup{
		Name:        "suggestion",
		Description: "Generate model-assisted suggestions",
		Subcommands: []cliapp.Command{
			{Name: "generate", NeedsAPI: true, Description: "Generate suggestions for a scheme", Run: func(args []string) error { return runGenerate(core, args) }},
		},
	}
}

func runGenerate(core *cliapp.ScenarioApp, args []string) error {
	fs, jsonOut, err := support.ParseFlags("suggestion generate", args)
	if err != nil {
		return err
	}
	if err := support.RequireArg(fs, "suggestion generate <scheme-id> [--json]"); err != nil {
		return err
	}

	body, err := core.Request("POST", "/schemes/"+fs.Arg(0)+"/suggestions", nil, nil)
	if err != nil {
		return err
	}

	var parsed response
	if err := support.Unmarshal(body, &parsed); err != nil {
		return err
	}

	report := cliapp.ListReport{
		Summary: []string{
			"Suggestions generated",
			"Provider: " + parsed.Provider,
		},
		ResultsHeading: "Suggestions",
		Results:        renderSuggestions(parsed),
		RetrievalHints: []string{
			cliName + " scheme export " + fs.Arg(0),
			cliName + " provider list",
		},
	}
	if *jsonOut {
		return cliapp.PrintReportJSON(os.Stdout, report)
	}
	return cliapp.RenderListReport(os.Stdout, report)
}

func renderSuggestions(parsed response) []string {
	lines := make([]string, 0, len(parsed.Suggestions))
	for _, item := range parsed.Suggestions {
		lines = append(lines, fmt.Sprintf("%s (confidence %.2f)", item.Label, item.Confidence))
	}
	return lines
}
