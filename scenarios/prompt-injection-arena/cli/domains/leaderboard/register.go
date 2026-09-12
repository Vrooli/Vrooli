package leaderboard

import (
	"fmt"
	"os"
	"strconv"

	"prompt-injection-arena/cli/internal/support"

	"github.com/vrooli/cli-core/cliapp"
	"github.com/vrooli/cli-core/cliutil"
)

// Register builds the `leaderboard` subcommand group with agents and injections
// rankings. Both call GET /api/v1/leaderboards/<type>.
func Register(core *cliapp.ScenarioApp) cliapp.SubcommandGroup {
	return cliapp.SubcommandGroup{
		Name:        "leaderboard",
		Description: "Rank robustness of agents and effectiveness of injections",
		NeedsAPI:    true,
		Subcommands: []cliapp.Command{
			{Name: "agents", Description: "Agent robustness leaderboard", Run: func(args []string) error { return runBoard(core, args, "agents") }},
			{Name: "injections", Description: "Injection effectiveness leaderboard", Run: func(args []string) error { return runBoard(core, args, "injections") }},
		},
	}
}

func runBoard(core *cliapp.ScenarioApp, args []string, kind string) error {
	fs := support.NewFlagSet("leaderboard " + kind)
	limit := fs.Int("limit", 20, "Maximum rows to return")
	jsonOutput := cliutil.JSONFlag(fs)
	if err := support.ParseFlags(fs, args); err != nil {
		return err
	}

	query := support.BuildQuery(map[string]string{"limit": strconv.Itoa(*limit)})
	body, err := core.Get("/leaderboards/"+kind, query)
	if err != nil {
		return err
	}

	var resp support.LeaderboardResponse
	if err := support.Decode(body, &resp); err != nil {
		return err
	}

	report := cliapp.ListReport{
		Summary: []string{
			fmt.Sprintf("%s leaderboard: %d entries", kind, resp.TotalEntries),
			fmt.Sprintf("Updated: %s", support.FormatTimeValue(resp.UpdatedAt)),
		},
		ResultsHeading: "Standings",
		Results:        boardRows(resp.Leaderboard, kind),
		RetrievalHints: []string{
			fmt.Sprintf("%s leaderboard %s --limit 50", support.CLIName, kind),
		},
	}
	if *jsonOutput {
		return cliapp.PrintReportJSON(os.Stdout, report)
	}
	return cliapp.RenderListReport(os.Stdout, report)
}

func boardRows(entries []support.LeaderboardEntry, kind string) []string {
	if len(entries) == 0 {
		return []string{"(no entries)"}
	}
	rows := make([]string, 0, len(entries))
	for i, e := range entries {
		rank := i + 1
		if r, ok := e.AdditionalInfo["rank"].(float64); ok {
			rank = int(r)
		}
		extra := ""
		switch kind {
		case "agents":
			if m, ok := e.AdditionalInfo["model_name"].(string); ok {
				extra = " | model=" + m
			}
		case "injections":
			if c, ok := e.AdditionalInfo["category"].(string); ok {
				extra = " | category=" + c
			}
		}
		rows = append(rows, fmt.Sprintf("#%d %s (%s) | score=%.2f | tests=%d passed=%d (%.1f%%)%s",
			rank, e.Name, support.ShortID(e.ID), e.Score, e.TestsRun, e.TestsPassed, e.PassPercentage, extra))
	}
	return rows
}
