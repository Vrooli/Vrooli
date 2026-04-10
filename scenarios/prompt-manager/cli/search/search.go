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
	"time"

	"github.com/vrooli/cli-core/cliapp"
	"github.com/vrooli/cli-core/cliutil"

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

type MatchRange struct {
	Start int `json:"start"`
	End   int `json:"end"`
}

// ContentSearchMatch represents a line-level content search match.
type ContentSearchMatch struct {
	SkillID     string       `json:"skillId"`
	SkillName   string       `json:"skillName"`
	File        string       `json:"file"`
	Folder      string       `json:"folder"`
	LineNumber  int          `json:"lineNumber"`
	Line        string       `json:"line"`
	MatchRanges []MatchRange `json:"matchRanges"`
}

// ContentSearchResponse wraps content search results.
type ContentSearchResponse struct {
	Matches []ContentSearchMatch `json:"matches"`
	Total   int                  `json:"total"`
	Query   string               `json:"query"`
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
	Query       string   `json:"query,omitempty"`
	Queries     []string `json:"queries,omitempty"`
	Limit       int      `json:"limit"`
	Output      string   `json:"output,omitempty"`
	Format      string   `json:"format,omitempty"`
	RenderLimit int      `json:"renderLimit,omitempty"`
}

// AvailabilityStatus represents AI search availability
type AvailabilityStatus struct {
	Available    bool   `json:"available"`
	Ollama       bool   `json:"ollama"`
	Qdrant       bool   `json:"qdrant"`
	IndexedCount int    `json:"indexedCount"`
	Message      string `json:"message,omitempty"`
}

// ReindexResponse represents the response from a reindex operation.
type ReindexResponse struct {
	Indexed int    `json:"indexed"`
	Skipped int    `json:"skipped"`
	Errors  int    `json:"errors"`
	Message string `json:"message"`
}

// ReindexStatus represents the status of a reindex job.
type ReindexStatus struct {
	Running    bool   `json:"running"`
	StartedAt  string `json:"startedAt,omitempty"`
	FinishedAt string `json:"finishedAt,omitempty"`
	Indexed    int    `json:"indexed"`
	Skipped    int    `json:"skipped"`
	Errors     int    `json:"errors"`
	Total      int    `json:"total"`
	Message    string `json:"message,omitempty"`
	Canceled   bool   `json:"canceled,omitempty"`
	Error      string `json:"error,omitempty"`
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
			{
				Name:        "search-status",
				Aliases:     []string{"search status"},
				NeedsAPI:    true,
				Description: "Check AI search availability and indexed skill count",
				Run: func(args []string) error {
					return cmdStatus(ctx, args)
				},
			},
			{
				Name:        "search-reindex",
				Aliases:     []string{"search reindex"},
				NeedsAPI:    true,
				Description: "Start full reindex of skills into vector database",
				Run: func(args []string) error {
					return cmdReindex(ctx, args)
				},
			},
			{
				Name:        "search-reindex-status",
				Aliases:     []string{"search reindex status", "search reindex-status"},
				NeedsAPI:    true,
				Description: "Check progress of ongoing reindex operation",
				Run: func(args []string) error {
					return cmdReindexStatus(ctx, args)
				},
			},
			{
				Name:        "search-reindex-cancel",
				Aliases:     []string{"search reindex cancel", "search reindex-cancel"},
				NeedsAPI:    true,
				Description: "Cancel an active reindex operation",
				Run: func(args []string) error {
					return cmdReindexCancel(ctx, args)
				},
			},
		},
	}
}

func cmdSearch(ctx appctx.Context, args []string) error {
	args = reorderFlagArgs(args, map[string]bool{
		"tag":            true,
		"folder":         true,
		"limit":          true,
		"output":         true,
		"format":         true,
		"render-limit":   true,
		"content":        false,
		"case-sensitive": false,
		"whole-word":     false,
		"regex":          false,
	})
	fs := flag.NewFlagSet("search", flag.ContinueOnError)
	tag := fs.String("tag", "", "Filter by tag")
	folder := fs.String("folder", "", "Filter by folder (core|local|drafts)")
	contentOnly := fs.Bool("content", false, "Search within skill contents (line-level matches)")
	textOnly := fs.Bool("text", false, "Force text-only search (skip AI)")
	caseSensitive := fs.Bool("case-sensitive", false, "Case-sensitive content search")
	wholeWord := fs.Bool("whole-word", false, "Whole word matching for content search")
	regex := fs.Bool("regex", false, "Treat query as regex for content search")
	limit := fs.Int("limit", 5, "Maximum number of results")
	output := fs.String("output", "results", "Output mode (results|combined|both)")
	format := fs.String("format", "xml", "Combined output format (xml|markdown|json)")
	renderLimit := fs.Int("render-limit", 0, "Limit number of skills rendered in combined output")
	jsonOut := fs.Bool("json", false, "Output as JSON")
	if err := cliutil.ParseInterspersed(fs, args); err != nil {
		return err
	}

	if fs.NArg() < 1 && *tag == "" && *folder == "" {
		return fmt.Errorf("usage: search <query> [--content] [--text] [--case-sensitive] [--whole-word] [--regex] [--limit=N] [--output=results|combined|both] [--format=xml|markdown|json] [--render-limit=N] [--tag=...] [--folder=...] [--json]")
	}

	allArgs := fs.Args()
	query := ""
	if len(allArgs) > 0 {
		query = strings.Join(allArgs, " ")
	}

	if query == "" && outputIncludesCombined(*output) {
		return fmt.Errorf("combined output requires a query")
	}

	if *contentOnly {
		if query == "" {
			return fmt.Errorf("content search requires a query")
		}
		return contentSearch(ctx, query, *tag, *folder, *limit, *caseSensitive, *wholeWord, *regex, *jsonOut)
	}

	// For filters-only queries (no text query), use text search
	if query == "" || (*textOnly && !outputIncludesCombined(*output)) {
		return textSearch(ctx, query, *tag, *folder, *jsonOut)
	}

	// For AI search: if multiple positional args, use multi-query (each searched independently)
	if len(allArgs) > 1 {
		return aiSearchMulti(ctx, allArgs, *limit, strings.ToLower(strings.TrimSpace(*output)), strings.ToLower(strings.TrimSpace(*format)), *renderLimit, *jsonOut)
	}

	// Single query AI search
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

// aiSearchMulti performs AI search with multiple independent queries, merging results.
func aiSearchMulti(ctx appctx.Context, queries []string, limit int, output, format string, renderLimit int, jsonOut bool) error {
	req := AISearchRequest{
		Queries:     queries,
		Limit:       limit,
		Output:      output,
		Format:      format,
		RenderLimit: renderLimit,
	}

	var resp AISearchResponse
	if err := ctx.Post("/search/ai", req, &resp); err != nil {
		fmt.Fprintln(os.Stderr, "(AI search unavailable, using text search)")
		return textSearch(ctx, strings.Join(queries, " "), "", "", jsonOut)
	}

	if jsonOut {
		enc := json.NewEncoder(os.Stdout)
		enc.SetIndent("", "  ")
		return enc.Encode(resp)
	}

	methodLabel := "AI multi-query"
	if resp.Method == "text" {
		methodLabel = "text (AI unavailable)"
	}

	if resp.Total == 0 {
		fmt.Printf("No skills found matching: %s (%s search)\n", strings.Join(queries, ", "), methodLabel)
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

	fmt.Printf("Search Results (%d found, %s across %d queries):\n", resp.Total, methodLabel, len(queries))
	for _, r := range resp.Results {
		tags := ""
		if len(r.Tags) > 0 {
			tags = " [" + strings.Join(r.Tags, ", ") + "]"
		}
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

func contentSearch(ctx appctx.Context, query, tag, folder string, limit int, caseSensitive, wholeWord, regex, jsonOut bool) error {
	params := url.Values{}
	params.Set("q", query)
	if tag != "" {
		params.Set("tag", tag)
	}
	if folder != "" {
		params.Set("folder", folder)
	}
	if limit > 0 {
		params.Set("limit", fmt.Sprintf("%d", limit))
	}
	if caseSensitive {
		params.Set("caseSensitive", "true")
	}
	if wholeWord {
		params.Set("wholeWord", "true")
	}
	if regex {
		params.Set("regex", "true")
	}

	var resp ContentSearchResponse
	if err := ctx.GetWithQuery("/search/skills/content", params, &resp); err != nil {
		return fmt.Errorf("content search failed: %w", err)
	}

	if jsonOut {
		enc := json.NewEncoder(os.Stdout)
		enc.SetIndent("", "  ")
		return enc.Encode(resp)
	}

	if resp.Total == 0 {
		fmt.Printf("No content matches found for: %s\n", query)
		return nil
	}

	grouped := groupContentMatches(resp.Matches)
	fmt.Printf("Content Matches (%d found):\n", resp.Total)
	for _, group := range grouped {
		fmt.Printf("  %s (%d matches)\n", group.File, len(group.Matches))
		for _, match := range group.Matches {
			line := highlightMatchLine(match.Line, match.MatchRanges, 120)
			fmt.Printf("    %d: %s\n", match.LineNumber, line)
		}
	}

	return nil
}

type contentMatchGroup struct {
	File    string
	Matches []ContentSearchMatch
}

func groupContentMatches(matches []ContentSearchMatch) []contentMatchGroup {
	order := make([]string, 0)
	byFile := make(map[string][]ContentSearchMatch)
	for _, match := range matches {
		if _, ok := byFile[match.File]; !ok {
			order = append(order, match.File)
		}
		byFile[match.File] = append(byFile[match.File], match)
	}

	grouped := make([]contentMatchGroup, 0, len(order))
	for _, file := range order {
		grouped = append(grouped, contentMatchGroup{
			File:    file,
			Matches: byFile[file],
		})
	}

	return grouped
}

func highlightMatchLine(line string, ranges []MatchRange, maxLen int) string {
	if len(line) <= maxLen {
		return applyRanges(line, ranges)
	}
	if len(ranges) == 0 {
		return truncate(line, maxLen)
	}

	start := ranges[0].Start - maxLen/3
	if start < 0 {
		start = 0
	}
	end := start + maxLen
	if end > len(line) {
		end = len(line)
		start = end - maxLen
		if start < 0 {
			start = 0
		}
	}

	window := line[start:end]
	windowRanges := make([]MatchRange, 0, len(ranges))
	for _, r := range ranges {
		if r.End <= start || r.Start >= end {
			continue
		}
		windowRanges = append(windowRanges, MatchRange{
			Start: max(0, r.Start-start),
			End:   min(end-start, r.End-start),
		})
	}

	prefix := ""
	suffix := ""
	if start > 0 {
		prefix = "..."
	}
	if end < len(line) {
		suffix = "..."
	}

	return prefix + applyRanges(window, windowRanges) + suffix
}

func applyRanges(line string, ranges []MatchRange) string {
	if len(ranges) == 0 {
		return line
	}

	var builder strings.Builder
	cursor := 0
	for _, r := range ranges {
		if r.Start < cursor {
			continue
		}
		if r.Start > len(line) {
			break
		}
		builder.WriteString(line[cursor:r.Start])
		builder.WriteString("[")
		builder.WriteString(line[r.Start:r.End])
		builder.WriteString("]")
		cursor = r.End
	}
	if cursor < len(line) {
		builder.WriteString(line[cursor:])
	}
	return builder.String()
}

func max(a, b int) int {
	if a > b {
		return a
	}
	return b
}

func min(a, b int) int {
	if a < b {
		return a
	}
	return b
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

// cmdStatus checks AI search availability and displays status.
func cmdStatus(ctx appctx.Context, args []string) error {
	fs := flag.NewFlagSet("search-status", flag.ContinueOnError)
	jsonOut := fs.Bool("json", false, "Output as JSON")
	fs.Bool("j", false, "Output as JSON (shorthand)")
	if err := cliutil.ParseInterspersed(fs, args); err != nil {
		return err
	}

	// Check if -j was used
	fs.Visit(func(f *flag.Flag) {
		if f.Name == "j" {
			*jsonOut = true
		}
	})

	var status AvailabilityStatus
	if err := ctx.Get("/search/ai/status", &status); err != nil {
		return fmt.Errorf("failed to get AI search status: %w", err)
	}

	if *jsonOut {
		enc := json.NewEncoder(os.Stdout)
		enc.SetIndent("", "  ")
		return enc.Encode(status)
	}

	fmt.Println("AI Search Status")
	fmt.Println("----------------")
	fmt.Printf("Available:      %s\n", boolToYesNo(status.Available))
	fmt.Printf("Ollama:         %s\n", boolToConnected(status.Ollama))
	fmt.Printf("Qdrant:         %s\n", boolToConnected(status.Qdrant))
	fmt.Printf("Indexed:        %d (skills, agents, teams)\n", status.IndexedCount)
	if status.Message != "" {
		fmt.Printf("Message:        %s\n", status.Message)
	}
	return nil
}

// cmdReindex starts a full reindex of skills into the vector database.
func cmdReindex(ctx appctx.Context, args []string) error {
	fs := flag.NewFlagSet("search-reindex", flag.ContinueOnError)
	wait := fs.Bool("wait", false, "Wait for reindex to complete")
	jsonOut := fs.Bool("json", false, "Output as JSON")
	fs.Bool("j", false, "Output as JSON (shorthand)")
	if err := cliutil.ParseInterspersed(fs, args); err != nil {
		return err
	}

	// Check if -j was used
	fs.Visit(func(f *flag.Flag) {
		if f.Name == "j" {
			*jsonOut = true
		}
	})

	var resp ReindexStatus
	if err := ctx.Post("/search/ai/reindex", nil, &resp); err != nil {
		return fmt.Errorf("failed to start reindex: %w", err)
	}

	// If not running after our request, it means it was already running
	if !resp.Running && resp.Message == "" {
		resp.Message = "Reindex already in progress"
	}

	if *jsonOut && !*wait {
		enc := json.NewEncoder(os.Stdout)
		enc.SetIndent("", "  ")
		return enc.Encode(resp)
	}

	if !*wait {
		if resp.Message != "" {
			fmt.Println(resp.Message)
		} else {
			fmt.Println("Reindex started")
		}
		fmt.Println("Use 'search reindex-status' to monitor progress")
		return nil
	}

	// Wait mode: poll until completion
	fmt.Println("Reindexing skills...")
	startTime := time.Now()

	for {
		var status ReindexStatus
		if err := ctx.Get("/search/ai/reindex/status", &status); err != nil {
			return fmt.Errorf("failed to get reindex status: %w", err)
		}

		if !status.Running {
			// Reindex complete
			if *jsonOut {
				enc := json.NewEncoder(os.Stdout)
				enc.SetIndent("", "  ")
				return enc.Encode(status)
			}

			duration := time.Since(startTime)
			fmt.Println()
			if status.Canceled {
				fmt.Println("Reindex canceled")
			} else if status.Error != "" {
				fmt.Printf("Reindex failed: %s\n", status.Error)
			} else {
				fmt.Println("Reindex complete")
			}
			fmt.Printf("  Indexed:  %d\n", status.Indexed)
			fmt.Printf("  Skipped:  %d\n", status.Skipped)
			fmt.Printf("  Errors:   %d\n", status.Errors)
			fmt.Printf("  Duration: %.1fs\n", duration.Seconds())
			return nil
		}

		// Print progress
		progress := status.Indexed + status.Skipped + status.Errors
		fmt.Printf("Progress: %d/%d indexed, %d skipped, %d errors\n",
			status.Indexed, status.Total, status.Skipped, status.Errors)

		// Don't print progress again if we're almost done
		if progress >= status.Total {
			time.Sleep(500 * time.Millisecond)
		} else {
			time.Sleep(2 * time.Second)
		}
	}
}

// cmdReindexStatus checks the progress of an ongoing reindex operation.
func cmdReindexStatus(ctx appctx.Context, args []string) error {
	fs := flag.NewFlagSet("search-reindex-status", flag.ContinueOnError)
	jsonOut := fs.Bool("json", false, "Output as JSON")
	fs.Bool("j", false, "Output as JSON (shorthand)")
	if err := cliutil.ParseInterspersed(fs, args); err != nil {
		return err
	}

	// Check if -j was used
	fs.Visit(func(f *flag.Flag) {
		if f.Name == "j" {
			*jsonOut = true
		}
	})

	var status ReindexStatus
	if err := ctx.Get("/search/ai/reindex/status", &status); err != nil {
		return fmt.Errorf("failed to get reindex status: %w", err)
	}

	if *jsonOut {
		enc := json.NewEncoder(os.Stdout)
		enc.SetIndent("", "  ")
		return enc.Encode(status)
	}

	fmt.Println("Reindex Status")
	fmt.Println("--------------")
	fmt.Printf("Running:    %s\n", boolToYesNo(status.Running))

	if status.StartedAt != "" {
		fmt.Printf("Started:    %s\n", status.StartedAt)
	}
	if status.FinishedAt != "" {
		fmt.Printf("Finished:   %s\n", status.FinishedAt)
	}

	if status.Total > 0 {
		progress := status.Indexed + status.Skipped + status.Errors
		percent := 0
		if status.Total > 0 {
			percent = (progress * 100) / status.Total
		}
		fmt.Printf("Progress:   %d/%d (%d%%)\n", progress, status.Total, percent)
	}

	fmt.Printf("  Indexed:  %d\n", status.Indexed)
	fmt.Printf("  Skipped:  %d\n", status.Skipped)
	fmt.Printf("  Errors:   %d\n", status.Errors)

	if status.Canceled {
		fmt.Println("  (canceled)")
	}
	if status.Error != "" {
		fmt.Printf("Error:      %s\n", status.Error)
	}
	if status.Message != "" {
		fmt.Printf("Message:    %s\n", status.Message)
	}

	return nil
}

// cmdReindexCancel cancels an active reindex operation.
func cmdReindexCancel(ctx appctx.Context, args []string) error {
	fs := flag.NewFlagSet("search-reindex-cancel", flag.ContinueOnError)
	jsonOut := fs.Bool("json", false, "Output as JSON")
	fs.Bool("j", false, "Output as JSON (shorthand)")
	if err := cliutil.ParseInterspersed(fs, args); err != nil {
		return err
	}

	// Check if -j was used
	fs.Visit(func(f *flag.Flag) {
		if f.Name == "j" {
			*jsonOut = true
		}
	})

	var status ReindexStatus
	if err := ctx.Post("/search/ai/reindex/cancel", nil, &status); err != nil {
		return fmt.Errorf("failed to cancel reindex: %w", err)
	}

	if *jsonOut {
		enc := json.NewEncoder(os.Stdout)
		enc.SetIndent("", "  ")
		return enc.Encode(status)
	}

	if status.Canceled {
		fmt.Println("Reindex canceled")
	} else if !status.Running {
		fmt.Println("No reindex operation running")
	} else {
		fmt.Println("Cancel request sent")
	}

	return nil
}

// boolToYesNo converts a boolean to "yes" or "no".
func boolToYesNo(b bool) string {
	if b {
		return "yes"
	}
	return "no"
}

// boolToConnected converts a boolean to "connected" or "disconnected".
func boolToConnected(b bool) string {
	if b {
		return "connected"
	}
	return "disconnected"
}
