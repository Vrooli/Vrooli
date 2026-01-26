// Package search provides CLI commands for searching skills.
//
// DOC: docs/reference/cli-commands.md#search
package search

import (
	"encoding/json"
	"flag"
	"fmt"
	"net/url"
	"os"
	"strings"

	"github.com/vrooli/cli-core/cliapp"

	"prompt-manager/cli/internal/appctx"
)

// SearchResult represents a search result item
type SearchResult struct {
	ID          string   `json:"id"`
	Name        string   `json:"name"`
	Description string   `json:"description"`
	Content     string   `json:"content,omitempty"`
	Folder      string   `json:"folder"`
	Tags        []string `json:"tags"`
	Modes       []string `json:"modes"`
	Score       float64  `json:"score,omitempty"`
	Highlight   string   `json:"highlight,omitempty"`
}

// SearchResponse wraps search results with metadata
type SearchResponse struct {
	Results []SearchResult `json:"results"`
	Total   int            `json:"total"`
	Query   string         `json:"query"`
}

// Commands returns the search command group.
func Commands(ctx appctx.Context) cliapp.CommandGroup {
	return cliapp.CommandGroup{
		Title: "Search",
		Commands: []cliapp.Command{
			{
				Name:        "search",
				Aliases:     []string{"find", "q"},
				NeedsAPI:    true,
				Description: "Search skills by content, name, or tags",
				Run: func(args []string) error {
					return cmdSearch(ctx, args)
				},
			},
		},
	}
}

func cmdSearch(ctx appctx.Context, args []string) error {
	fs := flag.NewFlagSet("search", flag.ContinueOnError)
	tag := fs.String("tag", "", "Filter by tag")
	folder := fs.String("folder", "", "Filter by folder (core|local|drafts)")
	jsonOut := fs.Bool("json", false, "Output as JSON")
	if err := fs.Parse(args); err != nil {
		return err
	}

	if fs.NArg() < 1 && *tag == "" && *folder == "" {
		return fmt.Errorf("usage: search <query> [--tag=...] [--folder=...] [--json]")
	}

	query := ""
	if fs.NArg() > 0 {
		query = strings.Join(fs.Args(), " ")
	}

	// Build query parameters
	params := url.Values{}
	if query != "" {
		params.Set("q", query)
	}
	if *tag != "" {
		params.Set("tag", *tag)
	}
	if *folder != "" {
		params.Set("folder", *folder)
	}

	var resp SearchResponse
	if err := ctx.GetWithQuery("/search/skills", params, &resp); err != nil {
		return fmt.Errorf("search failed: %w", err)
	}

	if *jsonOut {
		enc := json.NewEncoder(os.Stdout)
		enc.SetIndent("", "  ")
		return enc.Encode(resp)
	}

	if resp.Total == 0 {
		if query != "" {
			fmt.Printf("No skills found matching: %s\n", query)
		} else {
			fmt.Println("No skills found")
		}
		return nil
	}

	fmt.Printf("Search Results (%d found):\n", resp.Total)
	for _, r := range resp.Results {
		tags := ""
		if len(r.Tags) > 0 {
			tags = " [" + strings.Join(r.Tags, ", ") + "]"
		}
		score := ""
		if r.Score > 0 {
			score = fmt.Sprintf(" (%.1f)", r.Score)
		}
		fmt.Printf("  %s - %s%s%s [%s]\n", r.Name, r.Folder, score, tags, r.ID)
		if r.Highlight != "" {
			fmt.Printf("    → %s\n", truncate(r.Highlight, 80))
		}
	}
	return nil
}

// truncate shortens a string to maxLen characters.
func truncate(s string, maxLen int) string {
	s = strings.ReplaceAll(s, "\n", " ")
	s = strings.TrimSpace(s)
	if len(s) <= maxLen {
		return s
	}
	return s[:maxLen-3] + "..."
}
