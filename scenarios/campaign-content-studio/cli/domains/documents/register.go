package documents

import (
	"fmt"
	"os"
	"strconv"

	"campaign-content-studio/cli/internal/support"

	"github.com/vrooli/cli-core/cliapp"
	"github.com/vrooli/cli-core/cliutil"
)

// Register builds the `document` subcommand group covering list/search.
// The API is the source of truth; this package is a thin wrapper.
func Register(core *cliapp.ScenarioApp) cliapp.SubcommandGroup {
	return cliapp.SubcommandGroup{
		Name:        "document",
		Description: "List and search campaign documents",
		NeedsAPI:    true,
		Subcommands: []cliapp.Command{
			{Name: "list", Aliases: []string{"ls"}, Description: "List documents for a campaign", Run: func(args []string) error { return runList(core, args) }},
			{Name: "search", Description: "Semantic search across campaign documents", Run: func(args []string) error { return runSearch(core, args) }},
		},
	}
}

func runList(core *cliapp.ScenarioApp, args []string) error {
	fs := support.NewFlagSet("document list")
	jsonOutput := cliutil.JSONFlag(fs)
	if err := support.ParseFlags(fs, args); err != nil {
		return err
	}
	if fs.NArg() < 1 {
		return fmt.Errorf("usage: document list <campaign-id>")
	}
	campaignID := fs.Arg(0)

	body, err := core.Get("/campaigns/"+campaignID+"/documents", nil)
	if err != nil {
		return err
	}
	var docs []support.Document
	if err := support.Decode(body, &docs); err != nil {
		return err
	}

	report := cliapp.ListReport{
		Summary:        []string{fmt.Sprintf("Documents in campaign %s: %d", support.ShortID(campaignID), len(docs))},
		ResultsHeading: "Documents",
		Results:        documentRows(docs),
		RetrievalHints: []string{
			fmt.Sprintf("%s document search %s <query>", support.CLIName, campaignID),
		},
	}
	if *jsonOutput {
		return cliapp.PrintReportJSON(os.Stdout, report)
	}
	return cliapp.RenderListReport(os.Stdout, report)
}

func runSearch(core *cliapp.ScenarioApp, args []string) error {
	fs := support.NewFlagSet("document search")
	limit := fs.Int("limit", 10, "Maximum number of results")
	bodyFile := fs.String("body-file", "", "Path to JSON file with full request body (overrides positional query)")
	jsonOutput := cliutil.JSONFlag(fs)
	if err := support.ParseFlags(fs, args); err != nil {
		return err
	}
	if fs.NArg() < 1 {
		return fmt.Errorf("usage: document search <campaign-id> <query> [--limit N] | document search <campaign-id> --body-file <path>")
	}
	campaignID := fs.Arg(0)

	var payload interface{}
	if raw, err := support.ReadJSONFile(*bodyFile, false); err != nil {
		return err
	} else if raw != nil {
		payload = raw
	} else {
		if fs.NArg() < 2 {
			return fmt.Errorf("usage: document search <campaign-id> <query> [--limit N]")
		}
		payload = map[string]interface{}{
			"query": fs.Arg(1),
			"limit": *limit,
		}
	}

	body, err := core.Request("POST", "/campaigns/"+campaignID+"/search", nil, payload)
	if err != nil {
		return err
	}

	var result map[string]interface{}
	if err := support.Decode(body, &result); err != nil {
		result = nil
	}

	report := cliapp.ListReport{
		Summary:        []string{fmt.Sprintf("Search results for campaign %s (limit=%s)", support.ShortID(campaignID), strconv.Itoa(*limit))},
		ResultsHeading: "Search response",
		Results:        support.MapRows(result),
		RetrievalHints: []string{
			fmt.Sprintf("%s document list %s", support.CLIName, campaignID),
		},
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
		line := fmt.Sprintf("%s (%s)", d.Filename, support.ShortID(d.ID))
		if d.ContentType != "" {
			line += " | " + d.ContentType
		}
		if !d.UploadDate.IsZero() {
			line += " | uploaded=" + support.FormatTimeValue(d.UploadDate)
		}
		rows = append(rows, line)
	}
	return rows
}
