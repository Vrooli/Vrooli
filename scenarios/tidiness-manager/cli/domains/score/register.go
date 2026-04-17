package score

import (
	"fmt"
	"os"
	"tidiness-manager/cli/internal/support"

	"github.com/vrooli/cli-core/cliapp"
	"github.com/vrooli/cli-core/cliutil"
)

const cliName = "tidiness-manager"

func Register(core *cliapp.ScenarioApp) cliapp.CommandGroup {
	return cliapp.CommandGroup{
		Title: "Quality",
		Commands: []cliapp.Command{
			{
				Name:        "score",
				NeedsAPI:    true,
				Description: "Show the aggregate tidiness score for a scenario",
				Run: func(args []string) error {
					return run(core, args)
				},
			},
		},
	}
}

func run(core *cliapp.ScenarioApp, args []string) error {
	fs := support.NewFlagSet("score")
	jsonOutput := cliutil.JSONFlag(fs)
	if err := support.ParseFlags(fs, args); err != nil {
		return err
	}
	if fs.NArg() < 1 {
		return fmt.Errorf("usage: score <scenario> [--json]")
	}
	scenario := fs.Arg(0)

	body, err := core.Get("/scenarios/"+scenario+"/tidiness", nil)
	if err != nil {
		return err
	}

	var response support.TidinessScoreResponse
	if err := support.Decode(body, &response); err != nil {
		return err
	}

	report := cliapp.ListReport{
		Summary: []string{
			fmt.Sprintf("Scenario: %s", response.Scenario),
			fmt.Sprintf("Tidiness score: %.1f/100", response.Score),
			fmt.Sprintf("Open violations: %d", response.Violations),
		},
		ResultsHeading: "Breakdown",
		Results:        scoreRows(response),
		RetrievalHints: []string{fmt.Sprintf("%s issues %s --limit 20", cliName, response.Scenario)},
	}
	if *jsonOutput {
		return cliapp.PrintReportJSON(os.Stdout, report)
	}
	return cliapp.RenderListReport(os.Stdout, report)
}

func scoreRows(response support.TidinessScoreResponse) []string {
	rows := []string{}
	if response.Breakdown != nil {
		rows = append(rows,
			fmt.Sprintf("Lint: %.1f", response.Breakdown.LintScore),
			fmt.Sprintf("Type safety: %.1f", response.Breakdown.TypeSafetyScore),
			fmt.Sprintf("Complexity: %.1f", response.Breakdown.ComplexityScore),
			fmt.Sprintf("File length: %.1f", response.Breakdown.FileLengthScore),
			fmt.Sprintf("Test coverage: %.1f", response.Breakdown.TestCoverageScore),
			fmt.Sprintf("Tech debt: %.1f", response.Breakdown.TechDebtScore),
			fmt.Sprintf("Comments: %.1f", response.Breakdown.CommentsScore),
			fmt.Sprintf("Duplication: %.1f", response.Breakdown.DuplicationScore),
		)
	}
	if response.Metrics != nil {
		rows = append(rows,
			fmt.Sprintf("Files: %d", response.Metrics.TotalFiles),
			fmt.Sprintf("Lines: %d", response.Metrics.TotalLines),
			fmt.Sprintf("Avg complexity: %.1f", response.Metrics.AvgComplexity),
			fmt.Sprintf("Duplication: %.1f%%", response.Metrics.DuplicationPct),
		)
	}
	if len(rows) == 0 {
		rows = append(rows, "No score breakdown available")
	}
	return rows
}
