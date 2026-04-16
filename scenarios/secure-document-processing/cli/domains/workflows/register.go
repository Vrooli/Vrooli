package workflows

import (
	"fmt"
	"os"

	"secure-document-processing/cli/internal/support"

	"github.com/vrooli/cli-core/cliapp"
	"github.com/vrooli/cli-core/cliutil"
)

// Register builds the `workflow` subcommand group over GET /api/workflows.
// The API currently exposes a single list endpoint; workflow create/run/delete
// are deferred until the API adds those routes.
func Register(core *cliapp.ScenarioApp) cliapp.SubcommandGroup {
	return cliapp.SubcommandGroup{
		Name:        "workflow",
		Description: "List available processing workflows",
		NeedsAPI:    true,
		Subcommands: []cliapp.Command{
			{Name: "list", Aliases: []string{"ls"}, Description: "List workflows", Run: func(args []string) error { return runList(core, args) }},
		},
	}
}

func runList(core *cliapp.ScenarioApp, args []string) error {
	fs := support.NewFlagSet("workflow list")
	jsonOutput := cliutil.JSONFlag(fs)
	if err := support.ParseFlags(fs, args); err != nil {
		return err
	}

	body, err := core.Get("/workflows", nil)
	if err != nil {
		return err
	}
	var wfs []support.Workflow
	if err := support.Decode(body, &wfs); err != nil {
		return err
	}

	report := cliapp.ListReport{
		Summary:        []string{fmt.Sprintf("Workflows: %d", len(wfs))},
		ResultsHeading: "Workflows",
		Results:        workflowRows(wfs),
		RetrievalHints: []string{fmt.Sprintf("%s workflow list --json", support.CLIName)},
	}
	if *jsonOutput {
		return cliapp.PrintReportJSON(os.Stdout, report)
	}
	return cliapp.RenderListReport(os.Stdout, report)
}

func workflowRows(wfs []support.Workflow) []string {
	if len(wfs) == 0 {
		return []string{"No workflows found"}
	}
	rows := make([]string, 0, len(wfs))
	for _, w := range wfs {
		row := fmt.Sprintf("%s | %s", support.ShortID(w.ID), w.Name)
		if w.Type != "" {
			row += fmt.Sprintf(" | type=%s", w.Type)
		}
		if w.Description != "" {
			row += fmt.Sprintf(" | %s", w.Description)
		}
		rows = append(rows, row)
	}
	return rows
}
