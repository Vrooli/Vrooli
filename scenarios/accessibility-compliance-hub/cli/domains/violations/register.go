package violations

import (
	"fmt"
	"os"

	"accessibility-compliance-hub/cli/internal/support"

	"github.com/vrooli/cli-core/cliapp"
	"github.com/vrooli/cli-core/cliutil"
)

// Register builds the `violation` subcommand group wrapping /api/violations.
func Register(core *cliapp.ScenarioApp) cliapp.SubcommandGroup {
	return cliapp.SubcommandGroup{
		Name:        "violation",
		Description: "List accessibility violations",
		NeedsAPI:    true,
		Subcommands: []cliapp.Command{
			{Name: "list", Aliases: []string{"ls"}, Description: "List accessibility violations", Run: func(args []string) error { return runList(core, args) }},
		},
	}
}

func runList(core *cliapp.ScenarioApp, args []string) error {
	fs := support.NewFlagSet("violation list")
	jsonOutput := cliutil.JSONFlag(fs)
	if err := support.ParseFlags(fs, args); err != nil {
		return err
	}

	body, err := core.Get("/violations", nil)
	if err != nil {
		return err
	}
	var items []support.Violation
	if err := support.Decode(body, &items); err != nil {
		return err
	}

	report := cliapp.ListReport{
		Summary:        []string{fmt.Sprintf("Violations: %d", len(items))},
		ResultsHeading: "Violations",
		Results:        violationRows(items),
		RetrievalHints: []string{
			fmt.Sprintf("%s scan list", support.CLIName),
			fmt.Sprintf("%s report list", support.CLIName),
		},
	}
	if *jsonOutput {
		return cliapp.PrintReportJSON(os.Stdout, report)
	}
	return cliapp.RenderListReport(os.Stdout, report)
}

func violationRows(items []support.Violation) []string {
	if len(items) == 0 {
		return []string{"(no violations recorded)"}
	}
	rows := make([]string, 0, len(items))
	for _, v := range items {
		rows = append(rows, fmt.Sprintf("%s [%s] %s | element=%s | %s",
			support.ShortID(v.ID), v.Severity, v.Type, v.Element, v.Description))
	}
	return rows
}
