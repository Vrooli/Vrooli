package statistics

import (
	"fmt"
	"os"
	"sort"

	"prompt-injection-arena/cli/internal/support"

	"github.com/vrooli/cli-core/cliapp"
	"github.com/vrooli/cli-core/cliutil"
)

// Register exposes `statistics` as a flat, top-level command since the API
// surface is a single GET /api/v1/statistics with no subcommands.
func Register(core *cliapp.ScenarioApp) cliapp.CommandGroup {
	return cliapp.CommandGroup{
		Title: "Statistics",
		Commands: []cliapp.Command{
			{
				Name:        "statistics",
				Aliases:     []string{"stats"},
				Description: "Show aggregate arena statistics",
				NeedsAPI:    true,
				Run:         func(args []string) error { return runStats(core, args) },
			},
		},
	}
}

func runStats(core *cliapp.ScenarioApp, args []string) error {
	fs := support.NewFlagSet("statistics")
	jsonOutput := cliutil.JSONFlag(fs)
	if err := support.ParseFlags(fs, args); err != nil {
		return err
	}

	body, err := core.Get("/statistics", nil)
	if err != nil {
		return err
	}

	var resp support.StatisticsResponse
	if err := support.Decode(body, &resp); err != nil {
		return err
	}

	summary := []string{
		fmt.Sprintf("Updated: %s", support.FormatTimeValue(resp.UpdatedAt)),
		fmt.Sprintf("Injections: %d", resp.Totals["injections"]),
		fmt.Sprintf("Agents: %d", resp.Totals["agents"]),
		fmt.Sprintf("Tests (total): %d", resp.Totals["tests"]),
		fmt.Sprintf("Tests (last 24h): %d", resp.RecentActivity["tests_last_24h"]),
	}

	results := []string{"=== Injection categories ==="}
	results = append(results, countMapRows(resp.InjectionCategories)...)
	results = append(results, "=== Agent models ===")
	results = append(results, countMapRows(resp.AgentModels)...)

	report := cliapp.ListReport{
		Summary:        summary,
		ResultsHeading: "Breakdown",
		Results:        results,
		RetrievalHints: []string{
			fmt.Sprintf("%s leaderboard agents", support.CLIName),
			fmt.Sprintf("%s leaderboard injections", support.CLIName),
		},
	}
	if *jsonOutput {
		return cliapp.PrintReportJSON(os.Stdout, report)
	}
	return cliapp.RenderListReport(os.Stdout, report)
}

func countMapRows(counts map[string]int) []string {
	if len(counts) == 0 {
		return []string{"(no data)"}
	}
	keys := make([]string, 0, len(counts))
	for k := range counts {
		keys = append(keys, k)
	}
	sort.Strings(keys)
	rows := make([]string, 0, len(keys))
	for _, k := range keys {
		rows = append(rows, fmt.Sprintf("%s: %d", k, counts[k]))
	}
	return rows
}
