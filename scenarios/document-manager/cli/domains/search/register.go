package search

import (
	"fmt"
	"os"
	"strconv"

	"document-manager/cli/internal/support"

	"github.com/vrooli/cli-core/cliapp"
	"github.com/vrooli/cli-core/cliutil"
)

// Register exposes `document-manager search` as a flat command wrapping
// POST /api/search (vector similarity search).
func Register(core *cliapp.ScenarioApp) cliapp.CommandGroup {
	return cliapp.CommandGroup{
		Title: "Search",
		Commands: []cliapp.Command{
			{
				Name:        "search",
				Description: "Run a vector similarity search against indexed documents",
				NeedsAPI:    true,
				Run:         func(args []string) error { return run(core, args) },
			},
		},
	}
}

func run(core *cliapp.ScenarioApp, args []string) error {
	fs := support.NewFlagSet("search")
	query := fs.String("query", "", "Search query text")
	limit := fs.Int("limit", 10, "Maximum number of results (1-100)")
	bodyFile := fs.String("body-file", "", "Optional JSON request body (overrides --query/--limit); use '-' for stdin")
	jsonOutput := cliutil.JSONFlag(fs)
	if err := support.ParseFlags(fs, args); err != nil {
		return err
	}

	if *query == "" && fs.NArg() >= 1 {
		*query = fs.Arg(0)
	}

	var payload interface{}
	if *bodyFile != "" {
		raw, err := support.ReadJSONFile(*bodyFile, true)
		if err != nil {
			return err
		}
		payload = raw
	} else {
		if *query == "" {
			return fmt.Errorf("usage: search --query <text> [--limit N] | --body-file <path>")
		}
		payload = map[string]interface{}{
			"query": *query,
			"limit": *limit,
		}
	}

	body, err := core.Request("POST", "/search", nil, payload)
	if err != nil {
		return err
	}
	var resp support.SearchResponse
	if err := support.Decode(body, &resp); err != nil {
		return err
	}

	summary := []string{
		fmt.Sprintf("Query: %s", resp.Query),
		fmt.Sprintf("Results: %d", resp.Total),
	}
	report := cliapp.ListReport{
		Summary:        summary,
		ResultsHeading: "Matches",
		Results:        resultRows(resp.Results),
		RetrievalHints: []string{
			fmt.Sprintf("%s search --query <text> --limit 25", support.CLIName),
			fmt.Sprintf("%s index --body-file docs.json", support.CLIName),
		},
	}
	if *jsonOutput {
		return cliapp.PrintReportJSON(os.Stdout, report)
	}
	return cliapp.RenderListReport(os.Stdout, report)
}

func resultRows(results []support.SearchResult) []string {
	if len(results) == 0 {
		return []string{"(no results)"}
	}
	rows := make([]string, 0, len(results))
	for _, r := range results {
		score := strconv.FormatFloat(r.Score, 'f', 4, 64)
		snippet := r.Content
		if len(snippet) > 120 {
			snippet = snippet[:117] + "..."
		}
		rows = append(rows, fmt.Sprintf("%s | score=%s | app=%s | %s",
			support.ShortID(r.ID), score, r.ApplicationName, snippet))
	}
	return rows
}
