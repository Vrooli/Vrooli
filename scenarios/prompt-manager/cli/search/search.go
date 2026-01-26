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

// SearchResult represents a search result item (text search)
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

// SearchResponse wraps search results with metadata (text search)
type SearchResponse struct {
	Results []SearchResult `json:"results"`
	Total   int            `json:"total"`
	Query   string         `json:"query"`
}

// AISearchResult represents an AI search result item
type AISearchResult struct {
	ID           string   `json:"id"`
	Name         string   `json:"name"`
	Description  string   `json:"description,omitempty"`
	Folder       string   `json:"folder"`
	Tags         []string `json:"tags,omitempty"`
	Modes        []string `json:"modes,omitempty"`
	Score        float64  `json:"score"`
	ScorePercent int      `json:"scorePercent"`
}

// AISearchResponse wraps AI search results with metadata
type AISearchResponse struct {
	Results []AISearchResult `json:"results"`
	Total   int              `json:"total"`
	Query   string           `json:"query"`
	Method  string           `json:"method"`
}

// AISearchRequest represents the request body for AI search
type AISearchRequest struct {
	Query string `json:"query"`
	Limit int    `json:"limit"`
}

// AvailabilityStatus represents AI search availability
type AvailabilityStatus struct {
	Available    bool   `json:"available"`
	Ollama       bool   `json:"ollama"`
	Qdrant       bool   `json:"qdrant"`
	IndexedCount int    `json:"indexedCount"`
	Message      string `json:"message,omitempty"`
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
				Description: "Search skills (AI-powered by default, --text for text-only)",
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
	textOnly := fs.Bool("text", false, "Force text-only search (skip AI)")
	limit := fs.Int("limit", 5, "Maximum number of results")
	jsonOut := fs.Bool("json", false, "Output as JSON")
	if err := fs.Parse(args); err != nil {
		return err
	}

	if fs.NArg() < 1 && *tag == "" && *folder == "" {
		return fmt.Errorf("usage: search <query> [--text] [--limit=N] [--tag=...] [--folder=...] [--json]")
	}

	query := ""
	if fs.NArg() > 0 {
		query = strings.Join(fs.Args(), " ")
	}

	// For filters-only queries (no text query), use text search
	if query == "" || *textOnly {
		return textSearch(ctx, query, *tag, *folder, *jsonOut)
	}

	// Try AI search first
	return aiSearch(ctx, query, *limit, *jsonOut)
}

// aiSearch performs AI-powered semantic search.
func aiSearch(ctx appctx.Context, query string, limit int, jsonOut bool) error {
	req := AISearchRequest{
		Query: query,
		Limit: limit,
	}

	var resp AISearchResponse
	if err := ctx.Post("/search/ai", req, &resp); err != nil {
		// Fall back to text search on error
		fmt.Fprintln(os.Stderr, "(AI search unavailable, using text search)")
		return textSearch(ctx, query, "", "", jsonOut)
	}

	if jsonOut {
		enc := json.NewEncoder(os.Stdout)
		enc.SetIndent("", "  ")
		return enc.Encode(resp)
	}

	// Show method used
	methodLabel := "AI"
	if resp.Method == "text" {
		methodLabel = "text (AI unavailable)"
	}

	if resp.Total == 0 {
		fmt.Printf("No skills found matching: %s (%s search)\n", query, methodLabel)
		return nil
	}

	fmt.Printf("Search Results (%d found, %s search):\n", resp.Total, methodLabel)
	for _, r := range resp.Results {
		tags := ""
		if len(r.Tags) > 0 {
			tags = " [" + strings.Join(r.Tags, ", ") + "]"
		}
		// Show score as percentage for AI search
		score := fmt.Sprintf(" (%d%%)", r.ScorePercent)
		fmt.Printf("  %s - %s%s%s [%s]\n", r.Name, r.Folder, score, tags, r.ID)
		if r.Description != "" {
			fmt.Printf("    → %s\n", truncate(r.Description, 80))
		}
	}
	return nil
}

// textSearch performs traditional text search.
func textSearch(ctx appctx.Context, query, tag, folder string, jsonOut bool) error {
	// Build query parameters
	params := url.Values{}
	if query != "" {
		params.Set("q", query)
	}
	if tag != "" {
		params.Set("tag", tag)
	}
	if folder != "" {
		params.Set("folder", folder)
	}

	var resp SearchResponse
	if err := ctx.GetWithQuery("/search/skills", params, &resp); err != nil {
		return fmt.Errorf("search failed: %w", err)
	}

	if jsonOut {
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

	fmt.Printf("Search Results (%d found, text search):\n", resp.Total)
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
