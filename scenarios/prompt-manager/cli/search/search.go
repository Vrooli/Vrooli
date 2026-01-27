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
	Results     []AISearchResult `json:"results,omitempty"`
	Combined    string           `json:"combined,omitempty"`
	SkillCount  int              `json:"skillCount,omitempty"`
	TotalTokens int              `json:"totalTokens,omitempty"`
	Format      string           `json:"format,omitempty"`
	Total       int              `json:"total"`
	Query       string           `json:"query"`
	Method      string           `json:"method"`
	Output      string           `json:"output,omitempty"`
}

// AISearchRequest represents the request body for AI search
type AISearchRequest struct {
	Query       string `json:"query"`
	Limit       int    `json:"limit"`
	Output      string `json:"output,omitempty"`
	Format      string `json:"format,omitempty"`
	RenderLimit int    `json:"renderLimit,omitempty"`
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
	args = reorderFlagArgs(args, map[string]bool{
		"tag":          true,
		"folder":       true,
		"limit":        true,
		"output":       true,
		"format":       true,
		"render-limit": true,
	})
	fs := flag.NewFlagSet("search", flag.ContinueOnError)
	tag := fs.String("tag", "", "Filter by tag")
	folder := fs.String("folder", "", "Filter by folder (core|local|drafts)")
	textOnly := fs.Bool("text", false, "Force text-only search (skip AI)")
	limit := fs.Int("limit", 5, "Maximum number of results")
	output := fs.String("output", "results", "Output mode (results|combined|both)")
	format := fs.String("format", "xml", "Combined output format (xml|markdown|json)")
	renderLimit := fs.Int("render-limit", 0, "Limit number of skills rendered in combined output")
	jsonOut := fs.Bool("json", false, "Output as JSON")
	if err := fs.Parse(args); err != nil {
		return err
	}

	if fs.NArg() < 1 && *tag == "" && *folder == "" {
		return fmt.Errorf("usage: search <query> [--text] [--limit=N] [--output=results|combined|both] [--format=xml|markdown|json] [--render-limit=N] [--tag=...] [--folder=...] [--json]")
	}

	query := ""
	if fs.NArg() > 0 {
		query = strings.Join(fs.Args(), " ")
	}

	if query == "" && outputIncludesCombined(*output) {
		return fmt.Errorf("combined output requires a query")
	}

	// For filters-only queries (no text query), use text search
	if query == "" || (*textOnly && !outputIncludesCombined(*output)) {
		return textSearch(ctx, query, *tag, *folder, *jsonOut)
	}

	// Try AI search first
	return aiSearch(ctx, query, *limit, strings.ToLower(strings.TrimSpace(*output)), strings.ToLower(strings.TrimSpace(*format)), *renderLimit, *jsonOut)
}

// aiSearch performs AI-powered semantic search.
func aiSearch(ctx appctx.Context, query string, limit int, output, format string, renderLimit int, jsonOut bool) error {
	req := AISearchRequest{
		Query:       query,
		Limit:       limit,
		Output:      output,
		Format:      format,
		RenderLimit: renderLimit,
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

	if outputIncludesCombined(output) && resp.Combined != "" {
		fmt.Print(resp.Combined)
		if output == "both" {
			fmt.Print("\n\n")
		} else {
			return nil
		}
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

func outputIncludesCombined(output string) bool {
	switch strings.ToLower(strings.TrimSpace(output)) {
	case "combined", "both":
		return true
	default:
		return false
	}
}

func reorderFlagArgs(args []string, flagsWithValues map[string]bool) []string {
	if len(args) == 0 {
		return args
	}
	var flagArgs []string
	var positional []string

	for i := 0; i < len(args); i++ {
		arg := args[i]
		if strings.HasPrefix(arg, "-") {
			flagArgs = append(flagArgs, arg)
			name := strings.TrimLeft(arg, "-")
			if eq := strings.IndexRune(name, '='); eq != -1 {
				name = name[:eq]
			}
			if flagsWithValues[name] && !strings.Contains(arg, "=") && i+1 < len(args) {
				flagArgs = append(flagArgs, args[i+1])
				i++
			}
			continue
		}
		positional = append(positional, arg)
	}

	return append(flagArgs, positional...)
}
