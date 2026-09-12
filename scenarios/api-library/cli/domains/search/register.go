package search

import (
	"fmt"
	"os"
	"strconv"
	"strings"

	"api-library/cli/internal/support"

	"github.com/vrooli/cli-core/cliapp"
	"github.com/vrooli/cli-core/cliutil"
)

// Register exposes `api-library search <query>` as a flat command. The API
// supports both GET and POST on /search; we always POST so complex filters
// round-trip as JSON.
func Register(core *cliapp.ScenarioApp) cliapp.CommandGroup {
	return cliapp.CommandGroup{
		Title: "Discovery",
		Commands: []cliapp.Command{
			{
				Name:        "search",
				Description: "Search APIs by capability or keyword",
				NeedsAPI:    true,
				Run:         func(args []string) error { return runSearch(core, args) },
			},
		},
	}
}

func runSearch(core *cliapp.ScenarioApp, args []string) error {
	fs := support.NewFlagSet("search")
	limit := fs.Int("limit", 10, "Maximum number of results")
	configured := fs.String("configured", "", "Filter by configured status (true/false)")
	configuredOnly := fs.Bool("configured-only", false, "Shortcut for --configured true")
	maxPrice := fs.Float64("max-price", 0, "Maximum price per request (0 means no filter)")
	categoryList := fs.String("category", "", "Filter by category (comma-separated for multiple)")
	bodyFile := fs.String("body-file", "", "Path to JSON file with a pre-built search request body")
	jsonOutput := cliutil.JSONFlag(fs)
	if err := support.ParseFlags(fs, args); err != nil {
		return err
	}

	var body interface{}
	if strings.TrimSpace(*bodyFile) != "" {
		raw, err := support.ReadJSONFile(*bodyFile, true)
		if err != nil {
			return err
		}
		body = raw
	} else {
		if fs.NArg() < 1 {
			return fmt.Errorf("usage: search <query> [--limit N] [--configured true|false] [--configured-only] [--max-price N] [--category a,b] [--body-file PATH]")
		}
		query := strings.Join(fs.Args(), " ")
		payload := map[string]interface{}{
			"query": query,
			"limit": *limit,
		}
		filters := map[string]interface{}{}
		switch {
		case *configuredOnly:
			filters["configured"] = true
		case strings.TrimSpace(*configured) != "":
			parsed, err := strconv.ParseBool(strings.TrimSpace(*configured))
			if err != nil {
				return fmt.Errorf("--configured must be true or false: %w", err)
			}
			filters["configured"] = parsed
		}
		if *maxPrice > 0 {
			filters["max_price"] = *maxPrice
		}
		if cats := splitCSV(*categoryList); len(cats) > 0 {
			filters["categories"] = cats
		}
		if len(filters) > 0 {
			payload["filters"] = filters
		}
		body = payload
	}

	raw, err := core.Request("POST", "/search", nil, body)
	if err != nil {
		return err
	}

	var resp support.SearchResponse
	if err := support.Decode(raw, &resp); err != nil {
		return err
	}

	method := resp.Method
	if method == "" {
		method = "keyword"
	}
	report := cliapp.ListReport{
		Summary: []string{
			fmt.Sprintf("Query: %s", resp.Query),
			fmt.Sprintf("Method: %s", method),
			fmt.Sprintf("Found %d results", resp.Count),
		},
		ResultsHeading: "Matches",
		Results:        searchRows(resp.Results),
		RetrievalHints: []string{
			fmt.Sprintf("%s apis get <id>", support.CLIName),
			fmt.Sprintf("%s search <query> --max-price 0.001", support.CLIName),
		},
	}
	if *jsonOutput {
		return cliapp.PrintReportJSON(os.Stdout, report)
	}
	return cliapp.RenderListReport(os.Stdout, report)
}

func searchRows(results []support.SearchResult) []string {
	if len(results) == 0 {
		return []string{"(no matches)"}
	}
	rows := make([]string, 0, len(results))
	for _, r := range results {
		configured := "no"
		if r.Configured {
			configured = "yes"
		}
		pricing := r.PricingSummary
		if pricing == "" {
			pricing = "n/a"
		}
		rows = append(rows,
			fmt.Sprintf("%s (%s) | provider=%s | category=%s | configured=%s | relevance=%.3f | pricing=%s",
				r.Name, support.ShortID(r.ID), r.Provider, orDash(r.Category), configured, r.RelevanceScore, pricing))
	}
	return rows
}

func splitCSV(value string) []string {
	value = strings.TrimSpace(value)
	if value == "" {
		return nil
	}
	parts := strings.Split(value, ",")
	out := make([]string, 0, len(parts))
	for _, p := range parts {
		p = strings.TrimSpace(p)
		if p != "" {
			out = append(out, p)
		}
	}
	return out
}

func orDash(value string) string {
	value = strings.TrimSpace(value)
	if value == "" {
		return "-"
	}
	return value
}
