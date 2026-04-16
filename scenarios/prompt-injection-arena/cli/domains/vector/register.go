package vector

import (
	"fmt"
	"os"

	"prompt-injection-arena/cli/internal/support"

	"github.com/vrooli/cli-core/cliapp"
	"github.com/vrooli/cli-core/cliutil"
)

// Register builds the `vector` subcommand group wrapping the Qdrant-backed
// /api/v1/vector/* endpoints.
func Register(core *cliapp.ScenarioApp) cliapp.SubcommandGroup {
	return cliapp.SubcommandGroup{
		Name:        "vector",
		Description: "Vector search and indexing for injection techniques",
		NeedsAPI:    true,
		Subcommands: []cliapp.Command{
			{Name: "search", Description: "Semantic search over injection techniques", Run: func(args []string) error { return runSearch(core, args) }},
			{Name: "index", Description: "Index an injection technique by ID into Qdrant", Run: func(args []string) error { return runIndex(core, args) }},
		},
	}
}

func runSearch(core *cliapp.ScenarioApp, args []string) error {
	fs := support.NewFlagSet("vector search")
	query := fs.String("query", "", "Query text")
	limit := fs.Int("limit", 10, "Maximum hits to return")
	bodyFile := fs.String("body-file", "", "Path to a JSON file with the full request body (overrides other flags)")
	jsonOutput := cliutil.JSONFlag(fs)
	if err := support.ParseFlags(fs, args); err != nil {
		return err
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
			return fmt.Errorf("usage: vector search --query TEXT [--limit N] [--body-file PATH]")
		}
		payload = map[string]interface{}{
			"query": *query,
			"limit": *limit,
		}
	}

	body, err := core.Request("POST", "/vector/search", nil, payload)
	if err != nil {
		return err
	}

	var resp support.VectorSearchResponse
	if err := support.Decode(body, &resp); err != nil {
		return err
	}

	report := cliapp.ListReport{
		Summary:        []string{fmt.Sprintf("Query: %q | hits: %d", resp.Query, resp.Count)},
		ResultsHeading: "Search results",
		Results:        searchRows(resp.Results),
		RetrievalHints: []string{
			fmt.Sprintf("%s injections similar --query \"...\"", support.CLIName),
		},
	}
	if *jsonOutput {
		return cliapp.PrintReportJSON(os.Stdout, report)
	}
	return cliapp.RenderListReport(os.Stdout, report)
}

func runIndex(core *cliapp.ScenarioApp, args []string) error {
	fs := support.NewFlagSet("vector index")
	injectionID := fs.String("injection-id", "", "Injection technique ID (required)")
	bodyFile := fs.String("body-file", "", "Path to a JSON file with the full request body (overrides other flags)")
	jsonOutput := cliutil.JSONFlag(fs)
	if err := support.ParseFlags(fs, args); err != nil {
		return err
	}

	var payload interface{}
	if *bodyFile != "" {
		raw, err := support.ReadJSONFile(*bodyFile, true)
		if err != nil {
			return err
		}
		payload = raw
	} else {
		if *injectionID == "" {
			return fmt.Errorf("usage: vector index --injection-id UUID [--body-file PATH]")
		}
		payload = map[string]interface{}{
			"injection_id": *injectionID,
		}
	}

	body, err := core.Request("POST", "/vector/index", nil, payload)
	if err != nil {
		return err
	}
	message := support.EnvelopeMessage(body)
	if message == "" {
		message = "Injection indexed"
	}

	report := cliapp.MutationReport{
		Result:  []string{message},
		Changes: []string{fmt.Sprintf("Indexed injection: %s", *injectionID)},
		NextCommand: []string{
			fmt.Sprintf("%s vector search --query \"...\"", support.CLIName),
		},
	}
	if *jsonOutput {
		return cliapp.PrintReportJSON(os.Stdout, report)
	}
	return cliapp.RenderMutationReport(os.Stdout, report)
}

func searchRows(items []support.VectorSearchResult) []string {
	if len(items) == 0 {
		return []string{"(no matches)"}
	}
	rows := make([]string, 0, len(items))
	for _, r := range items {
		name := r.Name
		if name == "" {
			if v, ok := r.Payload["name"].(string); ok {
				name = v
			}
		}
		cat := r.Category
		if cat == "" {
			if v, ok := r.Payload["category"].(string); ok {
				cat = v
			}
		}
		rows = append(rows, fmt.Sprintf("%s (%s) | %s | score=%.3f",
			name, support.ShortID(r.ID), cat, r.Score))
	}
	return rows
}
