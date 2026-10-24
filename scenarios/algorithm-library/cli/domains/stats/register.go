package stats

import (
	"fmt"
	"os"
	"sort"

	"algorithm-library/cli/internal/support"

	"github.com/vrooli/cli-core/cliapp"
	"github.com/vrooli/cli-core/cliutil"
)

// Register exposes `algorithm-library stats` as a flat command returning the
// library-wide counts and language distribution.
func Register(core *cliapp.ScenarioApp) cliapp.CommandGroup {
	return cliapp.CommandGroup{
		Title: "Stats",
		Commands: []cliapp.Command{
			{
				Name:        "stats",
				Description: "Show library-wide statistics and language distribution",
				NeedsAPI:    true,
				Run:         func(args []string) error { return runStats(core, args) },
			},
		},
	}
}

func runStats(core *cliapp.ScenarioApp, args []string) error {
	fs := support.NewFlagSet("stats")
	jsonOutput := cliutil.JSONFlag(fs)
	if err := support.ParseFlags(fs, args); err != nil {
		return err
	}

	body, err := core.Get("/algorithms/stats", nil)
	if err != nil {
		return err
	}

	var resp support.StatsResponse
	if err := support.Decode(body, &resp); err != nil {
		return err
	}

	summary := []string{
		fmt.Sprintf("Total algorithms: %d", resp.Statistics.TotalAlgorithms),
		fmt.Sprintf("Total implementations: %d", resp.Statistics.TotalImplementations),
		fmt.Sprintf("Total test cases: %d", resp.Statistics.TotalTestCases),
		fmt.Sprintf("Validated implementations: %d", resp.Statistics.ValidatedCount),
	}

	languages := make([]string, 0, len(resp.Languages))
	for name := range resp.Languages {
		languages = append(languages, name)
	}
	sort.Strings(languages)

	results := make([]string, 0, len(languages))
	for _, name := range languages {
		results = append(results, fmt.Sprintf("%s: %d implementations", name, resp.Languages[name]))
	}
	if len(results) == 0 {
		results = []string{"(no language data available)"}
	}

	report := cliapp.ListReport{
		Summary:        summary,
		ResultsHeading: "Language distribution",
		Results:        results,
		RetrievalHints: []string{
			fmt.Sprintf("%s categories", support.CLIName),
			fmt.Sprintf("%s algorithm search <query>", support.CLIName),
		},
	}
	if *jsonOutput {
		return cliapp.PrintReportJSON(os.Stdout, report)
	}
	return cliapp.RenderListReport(os.Stdout, report)
}
