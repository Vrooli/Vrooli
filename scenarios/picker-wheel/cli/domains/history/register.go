package history

import (
	"fmt"
	"os"

	"picker-wheel/cli/internal/support"

	"github.com/vrooli/cli-core/cliapp"
	"github.com/vrooli/cli-core/cliutil"
)

// Register exposes `picker-wheel history` as a flat command since history is a
// single, read-only surface in the API (`GET /api/history`). The server owns
// filtering/ordering; the CLI is a thin renderer.
func Register(core *cliapp.ScenarioApp) cliapp.CommandGroup {
	return cliapp.CommandGroup{
		Title: "History",
		Commands: []cliapp.Command{
			{
				Name:        "history",
				Description: "List recent spin history",
				NeedsAPI:    true,
				Run:         func(args []string) error { return runHistory(core, args) },
			},
		},
	}
}

func runHistory(core *cliapp.ScenarioApp, args []string) error {
	fs := support.NewFlagSet("history")
	jsonOutput := cliutil.JSONFlag(fs)
	if err := support.ParseFlags(fs, args); err != nil {
		return err
	}

	body, err := core.Get("/history", nil)
	if err != nil {
		return err
	}
	var entries []support.SpinResult
	if err := support.Decode(body, &entries); err != nil {
		return err
	}

	report := cliapp.ListReport{
		Summary:        []string{fmt.Sprintf("Spin history entries: %d", len(entries))},
		ResultsHeading: "History",
		Results:        historyRows(entries),
		RetrievalHints: []string{
			fmt.Sprintf("%s spin --body-file payload.json", support.CLIName),
			fmt.Sprintf("%s wheel list", support.CLIName),
		},
	}
	if *jsonOutput {
		return cliapp.PrintReportJSON(os.Stdout, report)
	}
	return cliapp.RenderListReport(os.Stdout, report)
}

func historyRows(entries []support.SpinResult) []string {
	if len(entries) == 0 {
		return []string{"(no history available)"}
	}
	rows := make([]string, 0, len(entries))
	for _, e := range entries {
		wheel := e.WheelID
		if wheel == "" {
			wheel = "-"
		}
		rows = append(rows, fmt.Sprintf("%s | wheel=%s | result=%s",
			support.FormatTimeValue(e.Timestamp), wheel, e.Result))
	}
	return rows
}
