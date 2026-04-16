package docs

import (
	"fmt"
	"os"

	"scenario-to-mcp/cli/internal/support"

	"github.com/vrooli/cli-core/cliapp"
	"github.com/vrooli/cli-core/cliutil"
)

// Register builds the `docs` subcommand group for documentation discovery.
// The API is the source of truth for the documentation index; this package is
// a thin wrapper that formats responses through the standard output contracts.
func Register(core *cliapp.ScenarioApp) cliapp.SubcommandGroup {
	return cliapp.SubcommandGroup{
		Name:        "docs",
		Description: "List and read scenario documentation",
		NeedsAPI:    true,
		Subcommands: []cliapp.Command{
			{Name: "list", Aliases: []string{"ls"}, Description: "List available documentation", Run: func(args []string) error { return runList(core, args) }},
			{Name: "get", Aliases: []string{"show"}, Description: "Read a documentation entry by id", Run: func(args []string) error { return runGet(core, args) }},
		},
	}
}

func runList(core *cliapp.ScenarioApp, args []string) error {
	fs := support.NewFlagSet("docs list")
	jsonOutput := cliutil.JSONFlag(fs)
	if err := support.ParseFlags(fs, args); err != nil {
		return err
	}

	body, err := core.Get("/docs", nil)
	if err != nil {
		return err
	}
	var resp support.DocListResponse
	if err := support.Decode(body, &resp); err != nil {
		return err
	}

	report := cliapp.ListReport{
		Summary:        []string{fmt.Sprintf("Docs: %d", resp.Count)},
		ResultsHeading: "Documents",
		Results:        docRows(resp.Docs),
		RetrievalHints: []string{
			fmt.Sprintf("%s docs get <id>", support.CLIName),
		},
	}
	if *jsonOutput {
		return cliapp.PrintReportJSON(os.Stdout, report)
	}
	return cliapp.RenderListReport(os.Stdout, report)
}

func runGet(core *cliapp.ScenarioApp, args []string) error {
	fs := support.NewFlagSet("docs get")
	jsonOutput := cliutil.JSONFlag(fs)
	if err := support.ParseFlags(fs, args); err != nil {
		return err
	}
	if fs.NArg() < 1 {
		return fmt.Errorf("usage: docs get <doc-id>")
	}
	id := fs.Arg(0)

	query := support.BuildQuery(map[string]string{"id": id})
	body, err := core.Get("/docs/content", query)
	if err != nil {
		return err
	}
	var doc support.DocContent
	if err := support.Decode(body, &doc); err != nil {
		return err
	}

	results := []string{
		fmt.Sprintf("ID: %s", doc.ID),
		fmt.Sprintf("Title: %s", doc.Title),
	}
	if doc.Category != "" {
		results = append(results, fmt.Sprintf("Category: %s", doc.Category))
	}
	if doc.Summary != "" {
		results = append(results, fmt.Sprintf("Summary: %s", doc.Summary))
	}
	results = append(results, fmt.Sprintf("Path: %s", doc.RelativePath))
	if doc.Source != "" {
		results = append(results, fmt.Sprintf("Source: %s", doc.Source))
	}
	results = append(results, fmt.Sprintf("Last modified: %s", support.FormatTimeValue(doc.LastModified)))
	results = append(results, fmt.Sprintf("Size: %d bytes", doc.Size))
	results = append(results, "--- Content ---")
	results = append(results, doc.Content)

	report := cliapp.ListReport{
		Summary:        []string{fmt.Sprintf("Doc: %s", doc.Title)},
		ResultsHeading: "Document",
		Results:        results,
		RetrievalHints: []string{
			fmt.Sprintf("%s docs list", support.CLIName),
		},
	}
	if *jsonOutput {
		return cliapp.PrintReportJSON(os.Stdout, report)
	}
	return cliapp.RenderListReport(os.Stdout, report)
}

func docRows(docs []support.DocMetadata) []string {
	if len(docs) == 0 {
		return []string{"No docs available"}
	}
	rows := make([]string, 0, len(docs))
	for _, d := range docs {
		category := d.Category
		if category == "" {
			category = "-"
		}
		rows = append(rows, fmt.Sprintf("%s | category=%s | path=%s | id=%s",
			d.Title, category, d.RelativePath, support.ShortID(d.ID)))
	}
	return rows
}
