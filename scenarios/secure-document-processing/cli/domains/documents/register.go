package documents

import (
	"fmt"
	"os"

	"secure-document-processing/cli/internal/support"

	"github.com/vrooli/cli-core/cliapp"
	"github.com/vrooli/cli-core/cliutil"
)

// Register builds the `document` subcommand group over GET /api/documents.
// The API currently exposes a single list endpoint; per-document get, create,
// update, and delete are deferred until the API adds those routes.
func Register(core *cliapp.ScenarioApp) cliapp.SubcommandGroup {
	return cliapp.SubcommandGroup{
		Name:        "document",
		Description: "List managed documents",
		NeedsAPI:    true,
		Subcommands: []cliapp.Command{
			{Name: "list", Aliases: []string{"ls"}, Description: "List documents", Run: func(args []string) error { return runList(core, args) }},
		},
	}
}

func runList(core *cliapp.ScenarioApp, args []string) error {
	fs := support.NewFlagSet("document list")
	jsonOutput := cliutil.JSONFlag(fs)
	if err := support.ParseFlags(fs, args); err != nil {
		return err
	}

	body, err := core.Get("/documents", nil)
	if err != nil {
		return err
	}
	var docs []support.Document
	if err := support.Decode(body, &docs); err != nil {
		return err
	}

	report := cliapp.ListReport{
		Summary:        []string{fmt.Sprintf("Documents: %d", len(docs))},
		ResultsHeading: "Documents",
		Results:        documentRows(docs),
		RetrievalHints: []string{fmt.Sprintf("%s document list --json", support.CLIName)},
	}
	if *jsonOutput {
		return cliapp.PrintReportJSON(os.Stdout, report)
	}
	return cliapp.RenderListReport(os.Stdout, report)
}

func documentRows(docs []support.Document) []string {
	if len(docs) == 0 {
		return []string{"No documents found"}
	}
	rows := make([]string, 0, len(docs))
	for _, d := range docs {
		rows = append(rows, fmt.Sprintf("%s | %s | %s | created=%s",
			support.ShortID(d.ID),
			d.Filename,
			d.Status,
			support.FormatTimeValue(d.Created)))
	}
	return rows
}
