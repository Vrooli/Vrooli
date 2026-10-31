package recommendations

import (
	"fmt"
	"os"
	"strconv"

	"tidiness-manager/cli/internal/support"

	"github.com/vrooli/cli-core/cliapp"
	"github.com/vrooli/cli-core/cliutil"
)

const cliName = "tidiness-manager"

func Register(core *cliapp.ScenarioApp) cliapp.CommandGroup {
	return cliapp.CommandGroup{
		Title: "Recommendations",
		Commands: []cliapp.Command{
			{
				Name:        "recommend-refactors",
				NeedsAPI:    true,
				Description: "Get prioritized refactor recommendations",
				Run: func(args []string) error {
					return run(core, args)
				},
			},
		},
	}
}

func run(core *cliapp.ScenarioApp, args []string) error {
	fs := support.NewFlagSet("recommend-refactors")
	scenario := fs.String("scenario", "", "Scenario name")
	limit := fs.Int("limit", 5, "Maximum number of files to return")
	sortBy := fs.String("sort-by", "priority", "Sort by priority, staleness, complexity, length, or duplication")
	minLines := fs.Int("min-lines", 0, "Minimum line count to include")
	maxVisits := fs.Int("max-visits", 0, "Maximum visit count to include")
	format := fs.String("format", "detailed", "Output format: detailed, paths, json")
	autoScan := fs.Bool("auto-scan", true, "Auto-seed file metrics when missing")
	jsonOutput := cliutil.JSONFlag(fs)
	if err := support.ParseFlags(fs, args); err != nil {
		return err
	}
	if *scenario == "" && fs.NArg() > 0 {
		*scenario = fs.Arg(0)
	}
	if *scenario == "" {
		return fmt.Errorf("scenario is required")
	}

	query := map[string]string{
		"scenario": *scenario,
		"limit":    strconv.Itoa(*limit),
		"sort_by":  *sortBy,
	}
	if *minLines > 0 {
		query["min_lines"] = strconv.Itoa(*minLines)
	}
	if *maxVisits > 0 {
		query["max_visits"] = strconv.Itoa(*maxVisits)
	}
	if *autoScan {
		query["auto_scan"] = "true"
	}

	body, err := core.Get("/agent/refactor-recommendations", support.BuildQuery(query))
	if err != nil {
		return err
	}

	var response support.RecommendationsResponse
	if err := support.Decode(body, &response); err != nil {
		return err
	}

	switch *format {
	case "paths":
		for _, rec := range response.Recommendations {
			fmt.Fprintln(os.Stdout, rec.FilePath)
		}
		return nil
	case "json":
		return cliapp.PrintReportJSON(os.Stdout, response)
	}

	report := cliapp.ListReport{
		Summary: []string{
			fmt.Sprintf("Scenario: %s", response.Scenario),
			fmt.Sprintf("Recommendations: %d", response.Count),
			fmt.Sprintf("Sort order: %s", *sortBy),
		},
		ResultsHeading: "Candidates",
		Results:        recommendationRows(response.Recommendations),
		RetrievalHints: []string{
			fmt.Sprintf("%s visit <file-path> --scenario %s --note \"...\"", cliName, response.Scenario),
			fmt.Sprintf("%s exclude <file-path> --scenario %s --reason \"...\"", cliName, response.Scenario),
		},
	}
	if response.Warning != "" {
		report.RetrievalHints = append(report.RetrievalHints, response.Warning)
	}
	if *jsonOutput {
		return cliapp.PrintReportJSON(os.Stdout, report)
	}
	return cliapp.RenderListReport(os.Stdout, report)
}

func recommendationRows(recommendations []support.Recommendation) []string {
	if len(recommendations) == 0 {
		return []string{"No refactor candidates found"}
	}
	rows := make([]string, 0, len(recommendations))
	for _, rec := range recommendations {
		line := fmt.Sprintf("%s | priority %.0f | visits %d | %d lines", rec.FilePath, rec.RefactorPriority, rec.VisitCount, rec.LineCount)
		if rec.ComplexityMax != nil {
			line += fmt.Sprintf(" | max complexity %d", *rec.ComplexityMax)
		}
		rows = append(rows, line)
	}
	return rows
}
