// Package discover provides CLI commands for unified skill discovery.
//
// DOC: docs/reference/cli-commands.md#discover
package discover

import (
	"encoding/json"
	"flag"
	"fmt"
	"net/url"
	"os"
	"strings"

	"prompt-manager/cli/internal/appctx"

	"github.com/vrooli/cli-core/cliapp"
	"github.com/vrooli/cli-core/cliutil"
)

// DiscoverRequest is the request body for the discover endpoint.
type DiscoverRequest struct {
	Queries    []string `json:"queries"`
	Complexity string   `json:"complexity,omitempty"`
	Limit      int      `json:"limit,omitempty"`
	Type       string   `json:"type,omitempty"`
}

// DiscoverResult represents a single discovery result.
type DiscoverResult struct {
	Type         string   `json:"type,omitempty"`
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
	Status       string   `json:"status,omitempty"`
	Owner        string   `json:"owner,omitempty"`
	ShowCommand  string   `json:"showCommand,omitempty"`
	RunCommand   string   `json:"runCommand,omitempty"`
}

// DiscoverResponse wraps discovery results.
type DiscoverResponse struct {
	Results                []DiscoverResult `json:"results"`
	Total                  int              `json:"total"`
	Query                  string           `json:"query"`
	Method                 string           `json:"method"`
	TotalContentChars      int              `json:"totalContentChars"`
	ReadCommand            string           `json:"readCommand"`
	ShowCommand            string           `json:"showCommand,omitempty"`
	RunCommand             string           `json:"runCommand,omitempty"`
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
			{
				Name:        "discovery-gaps",
				Aliases:     []string{"gaps"},
				NeedsAPI:    true,
				Description: "Show clustered unmet-capability queries (discovery misses) within a window",
				Run: func(args []string) error {
					return cmdDiscoveryGaps(ctx, args)
				},
			},
		},
	}
}

// DiscoveryGapCluster is one clustered group of missed queries.
type DiscoveryGapCluster struct {
	Query    string   `json:"query"`
	Count    int      `json:"count"`
	LastSeen string   `json:"lastSeen"`
	Types    []string `json:"types,omitempty"`
	Examples []string `json:"examples,omitempty"`
}

// DiscoveryGapsResponse wraps the clustered discovery-gap results.
type DiscoveryGapsResponse struct {
	Clusters []DiscoveryGapCluster `json:"clusters"`
	Total    int                   `json:"total"`
	Since    string                `json:"since"`
}

func cmdDiscoveryGaps(ctx appctx.Context, args []string) error {
	fs := flag.NewFlagSet("discovery-gaps", flag.ContinueOnError)
	since := fs.String("since", "7d", "Window to report (e.g. 7d, 24h, 30m)")
	resultType := fs.String("type", "", "Filter by missed type (skill|action|all)")
	limit := fs.Int("limit", 20, "Maximum number of clusters to show")
	jsonOut := cliutil.JSONFlag(fs)
	if err := cliutil.ParseInterspersed(fs, args); err != nil {
		return err
	}
	if *resultType != "" && !validDiscoverType(*resultType) {
		return fmt.Errorf("--type must be one of: skill, action, all")
	}

	query := url.Values{}
	query.Set("since", *since)
	if *resultType != "" {
		query.Set("type", *resultType)
	}

	var resp DiscoveryGapsResponse
	if err := ctx.GetWithQuery("/discovery-gaps", query, &resp); err != nil {
		return fmt.Errorf("discovery-gaps failed: %w", err)
	}

	if *jsonOut {
		enc := json.NewEncoder(os.Stdout)
		enc.SetIndent("", "  ")
		return enc.Encode(resp)
	}

	if resp.Total == 0 {
		fmt.Printf("No discovery gaps in the last %s — discovery is finding what agents look for.\n", *since)
		return nil
	}

	shown := resp.Clusters
	if *limit > 0 && len(shown) > *limit {
		shown = shown[:*limit]
	}
	fmt.Printf("Top unmet-capability queries (last %s, %d cluster(s)):\n\n", *since, resp.Total)
	fmt.Printf("  %-5s  %-7s  %s\n", "Count", "Types", "Query")
	fmt.Printf("  %-5s  %-7s  %s\n", "-----", "-------", "-----------------------------------------")
	for _, cluster := range shown {
		types := strings.Join(cluster.Types, ",")
		if types == "" {
			types = "-"
		}
		fmt.Printf("  %-5d  %-7s  %s\n", cluster.Count, types, cluster.Query)
	}
	fmt.Println()
	fmt.Println("These are queries that returned nothing useful. Each is a candidate for:")
	fmt.Println("  - a new action over an existing CLI command, or")
	fmt.Println("  - a capability-gap / cli-backlog when no command covers it yet.")
	return nil
}

func cmdDiscover(ctx appctx.Context, args []string) error {
	fs := flag.NewFlagSet("discover", flag.ContinueOnError)
	complexity := fs.String("complexity", "moderate", "Task complexity (minor|moderate|major|architectural)")
	limit := fs.Int("limit", 10, "Maximum number of results")
	resultType := fs.String("type", "skill", "Result type (skill|action|all)")
	jsonOut := cliutil.JSONFlag(fs)
	if err := cliutil.ParseInterspersed(fs, args); err != nil {
		return err
	}

	if fs.NArg() < 1 {
		return fmt.Errorf("usage: discover \"query1\" \"query2\" [--complexity moderate] [--limit 10] [--type skill|action|all] [--json]")
	}
	if !validDiscoverType(*resultType) {
		return fmt.Errorf("--type must be one of: skill, action, all")
	}

	queries := fs.Args()

	req := DiscoverRequest{
		Queries:    queries,
		Complexity: *complexity,
		Limit:      *limit,
	}
	if *resultType != "skill" {
		req.Type = *resultType
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

	label := discoverLabel(*resultType)
	if resp.Total == 0 {
		fmt.Printf("No %s found for: %s\n", label, strings.Join(queries, ", "))
		return nil
	}

	// Header
	fmt.Printf("Found %d %s (~%s chars combined, %s budget):\n\n",
		resp.Total, label, formatNumber(resp.TotalContentChars), resp.Complexity)

	// Table
	if *resultType == "skill" {
		fmt.Printf("  %-3s  %-6s  %-7s  %-30s  %s\n", "#", "Score", "Source", "ID", "Chars")
		fmt.Printf("  %-3s  %-6s  %-7s  %-30s  %s\n", "---", "------", "-------", "------------------------------", "-----")
	} else {
		fmt.Printf("  %-3s  %-6s  %-7s  %-7s  %-30s  %s\n", "#", "Score", "Type", "Source", "ID", "Chars")
		fmt.Printf("  %-3s  %-6s  %-7s  %-7s  %-30s  %s\n", "---", "------", "-------", "-------", "------------------------------", "-----")
	}

	for i, r := range resp.Results {
		scoreStr := fmt.Sprintf("%.2f", r.Score)
		charsStr := formatNumber(r.ContentChars)
		id := r.ID
		if len(id) > 30 {
			id = id[:27] + "..."
		}
		if *resultType == "skill" {
			fmt.Printf("  %-3d  %-6s  %-7s  %-30s  %s\n", i+1, scoreStr, r.Source, id, charsStr)
		} else {
			rowType := r.Type
			if rowType == "" {
				rowType = "skill"
			}
			fmt.Printf("  %-3d  %-6s  %-7s  %-7s  %-30s  %s\n", i+1, scoreStr, rowType, r.Source, id, charsStr)
		}
	}

	// Read command
	if resp.ReadCommand != "" {
		fmt.Printf("\n  %s\n", resp.ReadCommand)
	}
	if resp.ShowCommand != "" {
		fmt.Printf("\n  %s\n", resp.ShowCommand)
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

func validDiscoverType(value string) bool {
	switch strings.ToLower(strings.TrimSpace(value)) {
	case "skill", "action", "all":
		return true
	default:
		return false
	}
}

func discoverLabel(value string) string {
	switch strings.ToLower(strings.TrimSpace(value)) {
	case "action":
		return "actions"
	case "all":
		return "results"
	default:
		return "skills"
	}
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
