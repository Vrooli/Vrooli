package search

import (
	"fmt"
	"os"
	"strings"

	"notes/cli/internal/support"

	"github.com/vrooli/cli-core/cliapp"
	"github.com/vrooli/cli-core/cliutil"
)

// Register builds the `search` subcommand group for /api/search.
func Register(core *cliapp.ScenarioApp) cliapp.SubcommandGroup {
	return cliapp.SubcommandGroup{
		Name:        "search",
		Description: "Search notes by text",
		NeedsAPI:    true,
		Subcommands: []cliapp.Command{
			{Name: "text", Aliases: []string{"find"}, Description: "Text search across notes", Run: func(args []string) error { return runText(core, args) }},
		},
	}
}

func runText(core *cliapp.ScenarioApp, args []string) error {
	fs := support.NewFlagSet("search text")
	query := fs.String("query", "", "Search query (required)")
	userID := fs.String("user-id", "", "Owner user ID")
	limit := fs.Int("limit", 0, "Maximum number of results")
	bodyFile := fs.String("body-file", "", "Path to JSON file with the full search payload (overrides flags)")
	jsonOutput := cliutil.JSONFlag(fs)
	if err := support.ParseFlags(fs, args); err != nil {
		return err
	}

	var payload interface{}
	if strings.TrimSpace(*bodyFile) != "" {
		raw, err := support.ReadJSONFile(*bodyFile, true)
		if err != nil {
			return err
		}
		payload = raw
	} else {
		if strings.TrimSpace(*query) == "" {
			return fmt.Errorf("--query is required (or use --body-file)")
		}
		body := map[string]interface{}{
			"query": *query,
		}
		if strings.TrimSpace(*userID) != "" {
			body["user_id"] = *userID
		}
		if *limit > 0 {
			body["limit"] = *limit
		}
		payload = body
	}

	respBody, err := core.Request("POST", "/search", nil, payload)
	if err != nil {
		return err
	}

	var resp support.SearchResponse
	if err := support.Decode(respBody, &resp); err != nil {
		return err
	}

	summary := []string{fmt.Sprintf("Results: %d for query %q", resp.Count, resp.Query)}

	report := cliapp.ListReport{
		Summary:        summary,
		ResultsHeading: "Matches",
		Results:        textRows(resp.Results),
		RetrievalHints: []string{
			fmt.Sprintf("%s note get <note-id>", support.CLIName),
		},
	}
	if *jsonOutput {
		return cliapp.PrintReportJSON(os.Stdout, report)
	}
	return cliapp.RenderListReport(os.Stdout, report)
}

func textRows(notes []support.Note) []string {
	if len(notes) == 0 {
		return []string{"No results"}
	}
	rows := make([]string, 0, len(notes))
	for _, n := range notes {
		rows = append(rows, fmt.Sprintf("%s | %s | updated=%s",
			support.ShortID(n.ID), n.Title, support.FormatTimeValue(n.UpdatedAt)))
	}
	return rows
}
