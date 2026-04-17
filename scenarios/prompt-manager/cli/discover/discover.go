// Package discover provides CLI commands for unified skill discovery.
//
// DOC: docs/reference/cli-commands.md#discover
package discover

import (
	"encoding/json"
	"flag"
	"fmt"
	"os"
	"prompt-manager/cli/internal/appctx"
	"strings"

	"github.com/vrooli/cli-core/cliapp"
	"github.com/vrooli/cli-core/cliutil"
)

// DiscoverRequest is the request body for the discover endpoint.
type DiscoverRequest struct {
	Queries    []string `json:"queries"`
	Complexity string   `json:"complexity,omitempty"`
	Limit      int      `json:"limit,omitempty"`
}

// DiscoverResult represents a single discovery result.
type DiscoverResult struct {
	ID           string   `json:"id"`
	Name         string   `json:"name"`
	Description  string   `json:"description,omitempty"`
	Tags         []string `json:"tags,omitempty"`
	Modes        []string `json:"modes,omitempty"`
	Score        float64  `json:"score"`
	ScorePercent int      `json:"scorePercent"`
	Source       string   `json:"source"`
	TopicDepth   *int     `json:"topicDepth,omitempty"`
	TopicID      string   `json:"topicId,omitempty"`
	ContentChars int      `json:"contentChars"`
}

// DiscoverResponse wraps discovery results.
type DiscoverResponse struct {
	Results                []DiscoverResult `json:"results"`
	Total                  int              `json:"total"`
	Query                  string           `json:"query"`
	Method                 string           `json:"method"`
	TotalContentChars      int              `json:"totalContentChars"`
	ReadCommand            string           `json:"readCommand"`
	BudgetChars            int              `json:"budgetChars,omitempty"`
	BudgetStatus           string           `json:"budgetStatus,omitempty"`
	RecommendedReadCommand string           `json:"recommendedReadCommand,omitempty"`
	Complexity             string           `json:"complexity,omitempty"`
}

// Commands returns the discover command group.
func Commands(ctx appctx.Context) cliapp.CommandGroup {
	return cliapp.CommandGroup{
		Title: "Discovery",
		Commands: []cliapp.Command{
			{
				Name:        "discover",
				Aliases:     []string{"disc"},
				NeedsAPI:    true,
				Description: "Discover relevant skills via topic + AI search with budget awareness",
				Run: func(args []string) error {
					return cmdDiscover(ctx, args)
				},
			},
		},
	}
}

func cmdDiscover(ctx appctx.Context, args []string) error {
	fs := flag.NewFlagSet("discover", flag.ContinueOnError)
	complexity := fs.String("complexity", "moderate", "Task complexity (minor|moderate|major|architectural)")
	limit := fs.Int("limit", 10, "Maximum number of results")
	jsonOut := cliutil.JSONFlag(fs)
	if err := cliutil.ParseInterspersed(fs, args); err != nil {
		return err
	}

	if fs.NArg() < 1 {
		return fmt.Errorf("usage: discover \"query1\" \"query2\" [--complexity moderate] [--limit 10] [--json]")
	}

	queries := fs.Args()

	req := DiscoverRequest{
		Queries:    queries,
		Complexity: *complexity,
		Limit:      *limit,
	}

	var resp DiscoverResponse
	if err := ctx.Post("/discover", req, &resp); err != nil {
		return fmt.Errorf("discover failed: %w", err)
	}

	if *jsonOut {
		enc := json.NewEncoder(os.Stdout)
		enc.SetIndent("", "  ")
		return enc.Encode(resp)
	}

	if resp.Total == 0 {
		fmt.Printf("No skills found for: %s\n", strings.Join(queries, ", "))
		return nil
	}

	// Header
	fmt.Printf("Found %d skills (~%s chars combined, %s budget):\n\n",
		resp.Total, formatNumber(resp.TotalContentChars), resp.Complexity)

	// Table
	fmt.Printf("  %-3s  %-6s  %-7s  %-30s  %s\n", "#", "Score", "Source", "ID", "Chars")
	fmt.Printf("  %-3s  %-6s  %-7s  %-30s  %s\n", "---", "------", "-------", "------------------------------", "-----")

	for i, r := range resp.Results {
		scoreStr := fmt.Sprintf("%.2f", r.Score)
		charsStr := formatNumber(r.ContentChars)
		id := r.ID
		if len(id) > 30 {
			id = id[:27] + "..."
		}
		fmt.Printf("  %-3d  %-6s  %-7s  %-30s  %s\n", i+1, scoreStr, r.Source, id, charsStr)
	}

	// Read command
	if resp.ReadCommand != "" {
		fmt.Printf("\n  %s\n", resp.ReadCommand)
	}

	// Budget status
	switch resp.BudgetStatus {
	case "under", "at":
		fmt.Printf("\n  Within budget (%s / %s chars). Safe to include all in plan.\n",
			formatNumber(resp.TotalContentChars), formatNumber(resp.BudgetChars))
	case "over":
		fmt.Printf("\n  Over budget (%s / %s chars).\n",
			formatNumber(resp.TotalContentChars), formatNumber(resp.BudgetChars))
		if resp.RecommendedReadCommand != "" {
			fmt.Printf("  Recommended (trimmed): %s\n", resp.RecommendedReadCommand)
		}
	}

	return nil
}

func formatNumber(n int) string {
	if n < 1000 {
		return fmt.Sprintf("%d", n)
	}
	if n < 1000000 {
		return fmt.Sprintf("%d,%03d", n/1000, n%1000)
	}
	return fmt.Sprintf("%d,%03d,%03d", n/1000000, (n/1000)%1000, n%1000)
}
