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
				Description: "Rank reduction opportunities by expected net production lines",
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
	limit := fs.Int("limit", 5, "Maximum number of reduction opportunities to return")
	minLines := fs.Int("min-lines", 0, "Minimum estimated net reduction")
	format := fs.String("format", "detailed", "Output format: detailed, paths, json")
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
	}
	if *minLines > 0 {
		query["min_lines"] = strconv.Itoa(*minLines)
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
			for _, location := range rec.Locations {
				fmt.Fprintln(os.Stdout, location.Path)
			}
		}
		return nil
	case "json":
		return cliapp.PrintReportJSON(os.Stdout, response)
	}

	report := cliapp.ListReport{
		Summary: []string{
			fmt.Sprintf("Scenario: %s", response.Scenario),
			fmt.Sprintf("Recommendations: %d", response.Count),
			"Ranking: expected net production lines removed",
		},
		ResultsHeading: "Candidates",
		Results:        recommendationRows(response.Recommendations),
		RetrievalHints: []string{fmt.Sprintf("%s recommend-refactors %s --json", cliName, response.Scenario)},
	}
	report.RetrievalHints = append(report.RetrievalHints, response.Warnings...)
	if *jsonOutput {
		return cliapp.PrintReportJSON(os.Stdout, response)
	}
	return cliapp.RenderListReport(os.Stdout, report)
}

func recommendationRows(recommendations []support.Recommendation) []string {
	if len(recommendations) == 0 {
		return []string{"No refactor candidates found"}
	}
	rows := make([]string, 0, len(recommendations))
	for _, rec := range recommendations {
		expected := "uncalibrated"
		if rec.ExpectedNetLines != nil {
			expected = fmt.Sprintf("%.1f expected net lines", *rec.ExpectedNetLines)
		}
		precision := "unknown precision"
		if rec.Calibration.Precision != nil {
			if rec.Calibration.ScopeFrameID != "" {
				precision = fmt.Sprintf("measured target precision %.0f%% (%d groups)", 100**rec.Calibration.Precision, rec.Calibration.ScopeCases)
			} else {
				precision = fmt.Sprintf("historical precision %.0f%% (%d labelled cases)", 100**rec.Calibration.Precision, rec.Calibration.HistoricalCases)
			}
		}
		line := fmt.Sprintf("%s | %s | %s | estimated net %d | %s", rec.Title, expected, precision, rec.NetLines, rec.DetectorID)
		rows = append(rows, line)
	}

	return rows
}
