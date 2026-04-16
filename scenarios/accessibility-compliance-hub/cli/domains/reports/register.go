package reports

import (
	"fmt"
	"os"

	"accessibility-compliance-hub/cli/internal/support"

	"github.com/vrooli/cli-core/cliapp"
	"github.com/vrooli/cli-core/cliutil"
)

// Register builds the `report` subcommand group wrapping /api/reports.
func Register(core *cliapp.ScenarioApp) cliapp.SubcommandGroup {
	return cliapp.SubcommandGroup{
		Name:        "report",
		Description: "List compliance reports",
		NeedsAPI:    true,
		Subcommands: []cliapp.Command{
			{Name: "list", Aliases: []string{"ls"}, Description: "List compliance reports", Run: func(args []string) error { return runList(core, args) }},
		},
	}
}

func runList(core *cliapp.ScenarioApp, args []string) error {
	fs := support.NewFlagSet("report list")
	jsonOutput := cliutil.JSONFlag(fs)
	if err := support.ParseFlags(fs, args); err != nil {
		return err
	}

	body, err := core.Get("/reports", nil)
	if err != nil {
		return err
	}
	var items []support.Report
	if err := support.Decode(body, &items); err != nil {
		return err
	}

	report := cliapp.ListReport{
		Summary:        []string{fmt.Sprintf("Reports: %d", len(items))},
		ResultsHeading: "Reports",
		Results:        reportRows(items),
		RetrievalHints: []string{
			fmt.Sprintf("%s scan list", support.CLIName),
			fmt.Sprintf("%s violation list", support.CLIName),
		},
	}
	if *jsonOutput {
		return cliapp.PrintReportJSON(os.Stdout, report)
	}
	return cliapp.RenderListReport(os.Stdout, report)
}

func reportRows(items []support.Report) []string {
	if len(items) == 0 {
		return []string{"(no reports generated)"}
	}
	rows := make([]string, 0, len(items))
	for _, r := range items {
		rows = append(rows, fmt.Sprintf("%s | scan=%s | score=%.1f | issues=%d | %s | date=%s",
			support.ShortID(r.ID), support.ShortID(r.ScanID), r.Score, len(r.Issues), r.Title, support.FormatTimeValue(r.Date)))
	}
	return rows
}
