package analysis

import (
	"fmt"
	"os"
	"strconv"
	"strings"

	"scenario-completeness-scoring/cli/internal/support"

	"github.com/vrooli/cli-core/cliapp"
	"github.com/vrooli/cli-core/cliutil"
)

// Register builds the `analysis` subcommand group for cross-scenario tooling:
// aggregate trends, comparisons, and the component registry.
func Register(core *cliapp.ScenarioApp) cliapp.SubcommandGroup {
	return cliapp.SubcommandGroup{
		Name:        "analysis",
		Description: "Cross-scenario trends, comparisons, and component metadata",
		NeedsAPI:    true,
		Subcommands: []cliapp.Command{
			{Name: "trends", Description: "Show aggregated trend analysis across scenarios", Run: func(args []string) error { return runTrends(core, args) }},
			{Name: "compare", Description: "Compare two or more scenarios (requires --body-file or --scenarios)", Run: func(args []string) error { return runCompare(core, args) }},
			{Name: "components", Description: "List analysis components registered on the server", Run: func(args []string) error { return runComponents(core, args) }},
		},
	}
}

func runTrends(core *cliapp.ScenarioApp, args []string) error {
	fs := support.NewFlagSet("analysis trends")
	limit := fs.Int("limit", 0, "Limit number of scenarios (0 = server default)")
	source := fs.String("source", "", "Filter by source")
	jsonOutput := cliutil.JSONFlag(fs)
	if err := support.ParseFlags(fs, args); err != nil {
		return err
	}

	params := map[string]string{"source": *source}
	if *limit > 0 {
		params["limit"] = strconv.Itoa(*limit)
	}
	body, err := core.Get("/trends", support.BuildQuery(params))
	if err != nil {
		return err
	}

	report := cliapp.ListReport{
		Summary:        []string{"Aggregated trend analysis"},
		ResultsHeading: "Trends payload",
		Results:        support.JSONLines(body),
		RetrievalHints: []string{
			fmt.Sprintf("%s analysis compare --scenarios a,b", support.CLIName),
			fmt.Sprintf("%s score list", support.CLIName),
		},
	}
	if *jsonOutput {
		return cliapp.PrintReportJSON(os.Stdout, report)
	}
	return cliapp.RenderListReport(os.Stdout, report)
}

func runCompare(core *cliapp.ScenarioApp, args []string) error {
	fs := support.NewFlagSet("analysis compare")
	bodyFile := fs.String("body-file", "", "Path to a JSON file describing the comparison payload")
	scenarios := fs.String("scenarios", "", "Comma-separated list of scenarios to compare")
	jsonOutput := cliutil.JSONFlag(fs)
	if err := support.ParseFlags(fs, args); err != nil {
		return err
	}

	var payload interface{}
	switch {
	case strings.TrimSpace(*bodyFile) != "":
		raw, err := support.ReadJSONFile(*bodyFile, true)
		if err != nil {
			return err
		}
		payload = raw
	case strings.TrimSpace(*scenarios) != "":
		parts := make([]string, 0)
		for _, p := range strings.Split(*scenarios, ",") {
			if p = strings.TrimSpace(p); p != "" {
				parts = append(parts, p)
			}
		}
		if len(parts) < 2 {
			return fmt.Errorf("--scenarios requires at least two comma-separated scenarios")
		}
		payload = map[string]interface{}{"scenarios": parts}
	default:
		return fmt.Errorf("analysis compare requires --scenarios a,b,c or --body-file")
	}

	body, err := core.Request("POST", "/compare", nil, payload)
	if err != nil {
		return err
	}

	report := cliapp.ListReport{
		Summary:        []string{"Comparison result"},
		ResultsHeading: "Comparison payload",
		Results:        support.JSONLines(body),
		RetrievalHints: []string{
			fmt.Sprintf("%s analysis trends", support.CLIName),
			fmt.Sprintf("%s score list", support.CLIName),
		},
	}
	if *jsonOutput {
		return cliapp.PrintReportJSON(os.Stdout, report)
	}
	return cliapp.RenderListReport(os.Stdout, report)
}

func runComponents(core *cliapp.ScenarioApp, args []string) error {
	fs := support.NewFlagSet("analysis components")
	jsonOutput := cliutil.JSONFlag(fs)
	if err := support.ParseFlags(fs, args); err != nil {
		return err
	}

	body, err := core.Get("/analysis/components", nil)
	if err != nil {
		return err
	}

	report := cliapp.ListReport{
		Summary:        []string{"Analysis components registered on the server"},
		ResultsHeading: "Components",
		Results:        support.JSONLines(body),
		RetrievalHints: []string{
			fmt.Sprintf("%s analysis trends", support.CLIName),
		},
	}
	if *jsonOutput {
		return cliapp.PrintReportJSON(os.Stdout, report)
	}
	return cliapp.RenderListReport(os.Stdout, report)
}
