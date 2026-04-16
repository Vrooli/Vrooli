package scans

import (
	"fmt"
	"os"

	"accessibility-compliance-hub/cli/internal/support"

	"github.com/vrooli/cli-core/cliapp"
	"github.com/vrooli/cli-core/cliutil"
)

// Register builds the `scan` subcommand group wrapping /api/scans. The API
// currently exposes read-only listing; future write verbs should be added here
// as thin wrappers once the backend supports them.
func Register(core *cliapp.ScenarioApp) cliapp.SubcommandGroup {
	return cliapp.SubcommandGroup{
		Name:        "scan",
		Description: "List accessibility scans",
		NeedsAPI:    true,
		Subcommands: []cliapp.Command{
			{Name: "list", Aliases: []string{"ls"}, Description: "List accessibility scans", Run: func(args []string) error { return runList(core, args) }},
		},
	}
}

func runList(core *cliapp.ScenarioApp, args []string) error {
	fs := support.NewFlagSet("scan list")
	jsonOutput := cliutil.JSONFlag(fs)
	if err := support.ParseFlags(fs, args); err != nil {
		return err
	}

	body, err := core.Get("/scans", nil)
	if err != nil {
		return err
	}
	var scans []support.Scan
	if err := support.Decode(body, &scans); err != nil {
		return err
	}

	report := cliapp.ListReport{
		Summary:        []string{fmt.Sprintf("Scans: %d", len(scans))},
		ResultsHeading: "Scans",
		Results:        scanRows(scans),
		RetrievalHints: []string{
			fmt.Sprintf("%s violation list", support.CLIName),
			fmt.Sprintf("%s report list", support.CLIName),
		},
	}
	if *jsonOutput {
		return cliapp.PrintReportJSON(os.Stdout, report)
	}
	return cliapp.RenderListReport(os.Stdout, report)
}

func scanRows(scans []support.Scan) []string {
	if len(scans) == 0 {
		return []string{"(no scans recorded)"}
	}
	rows := make([]string, 0, len(scans))
	for _, s := range scans {
		rows = append(rows, fmt.Sprintf("%s (%s) | %s | violations=%d | created=%s",
			support.ShortID(s.ID), s.Status, s.URL, s.Violations, support.FormatTimeValue(s.Created)))
	}
	return rows
}
