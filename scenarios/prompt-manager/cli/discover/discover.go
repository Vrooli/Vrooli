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
				Usage:       `prompt-manager discover "<query>" ["<query>" ...] [options]`,
				HelpText: `Arguments:
  <query>                One or more discovery queries (quote multi-word queries)

Options:
  --complexity <value>   Task complexity: minor|moderate|major|architectural (default: moderate)
  --limit <n>            Maximum number of results (default: 10)
  --type <value>         Result type: skill|action|all (default: skill)
  --json                 Emit JSON output instead of human format`,
				Run: func(args []string) error {
					return cmdDiscover(ctx, args)
				},
			},
			{
				Name:        "discovery-gaps",
				Aliases:     []string{"gaps"},
				NeedsAPI:    true,
				Description: "Show clustered unmet-capability queries (discovery misses) within a window",
				Usage:       "prompt-manager discovery-gaps [options]",
				HelpText: `Options:
  --since <window>       Window to report, e.g. 7d, 24h, 30m (default: 7d)
  --type <value>         Filter by missed type: skill|action|all (default: all)
  --limit <n>            Maximum number of clusters to show (default: 20)
  --json                 Emit JSON output instead of human format`,
				Run: func(args []string) error {
					return cmdDiscoveryGaps(ctx, args)
				},
			},
			{
				Name:        "skill-usage",
				Aliases:     []string{"usage"},
				NeedsAPI:    true,
				Description: "Show per-skill demand: how often discovery returned a skill vs how often an agent read it",
				Usage:       "prompt-manager skill-usage [options]",
				HelpText: `Options:
  --since <window>       Window to report, e.g. 7d, 24h, 30m (default: 7d)
  --limit <n>            Maximum number of rows to show (default: 20)
  --json                 Emit JSON output instead of human format`,
				Run: func(args []string) error {
					return cmdSkillUsage(ctx, args)
				},
			},
			{
				Name:        "discovery-metrics",
				Aliases:     []string{"disc-metrics"},
				NeedsAPI:    true,
				Description: "Show aggregate discovery telemetry (call volume, returned-count distribution, budget/clipping rates) within a window",
				Usage:       "prompt-manager discovery-metrics [options]",
				HelpText: `Options:
  --since <window>       Window to report, e.g. 7d, 24h, 30m (default: 7d)
  --type <value>         Filter by call type: skill|action|all (default: all)
  --json                 Emit JSON output instead of human format`,
				Run: func(args []string) error {
					return cmdDiscoveryMetrics(ctx, args)
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

// DistributionStats summarizes a numeric sample (e.g. returned-count per call).
type DistributionStats struct {
	Count  int     `json:"count"`
	Min    float64 `json:"min"`
	P10    float64 `json:"p10"`
	Median float64 `json:"median"`
	P90    float64 `json:"p90"`
	Max    float64 `json:"max"`
	Mean   float64 `json:"mean"`
}

// ComplexityMetric is the per-tier breakdown of the headline numbers.
type ComplexityMetric struct {
	CallCount      int     `json:"callCount"`
	OverBudgetRate float64 `json:"overBudgetRate"`
	MedianReturned float64 `json:"medianReturned"`
}

// BudgetHogSkill is a large skill that pressures the budget.
type BudgetHogSkill struct {
	ID                  string `json:"id"`
	MaxChars            int    `json:"maxChars"`
	Occurrences         int    `json:"occurrences"`
	OverBudgetSightings int    `json:"overBudgetSightings"`
}

// DiscoveryMetricsResponse mirrors the API's aggregate discovery report.
type DiscoveryMetricsResponse struct {
	Since             string                      `json:"since"`
	CallCount         int                         `json:"callCount"`
	ReturnedCount     DistributionStats           `json:"returnedCount"`
	BudgetedCallCount int                         `json:"budgetedCallCount"`
	OverBudgetRate    float64                     `json:"overBudgetRate"`
	NearThresholdRate float64                     `json:"nearThresholdRate"`
	ProbedCallCount   int                         `json:"probedCallCount"`
	ThresholdClipRate float64                     `json:"thresholdClipRate"`
	ClippedPerProbe   DistributionStats           `json:"clippedPerProbe"`
	PerComplexity     map[string]ComplexityMetric `json:"perComplexity"`
	BudgetHogs        []BudgetHogSkill            `json:"budgetHogs"`
}

func cmdDiscoveryMetrics(ctx appctx.Context, args []string) error {
	fs := flag.NewFlagSet("discovery-metrics", flag.ContinueOnError)
	since := fs.String("since", "7d", "Window to report (e.g. 7d, 24h, 30m)")
	resultType := fs.String("type", "", "Filter by call type (skill|action|all)")
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

	var resp DiscoveryMetricsResponse
	if err := ctx.GetWithQuery("/discovery-metrics", query, &resp); err != nil {
		return fmt.Errorf("discovery-metrics failed: %w", err)
	}

	if *jsonOut {
		enc := json.NewEncoder(os.Stdout)
		enc.SetIndent("", "  ")
		return enc.Encode(resp)
	}

	if resp.CallCount == 0 {
		fmt.Printf("No discovery calls recorded in the last %s.\n", *since)
		return nil
	}

	fmt.Printf("Discovery metrics (last %s, %d call(s)):\n\n", *since, resp.CallCount)
	fmt.Printf("  Returned per call:   median %.0f  (p10 %.0f, p90 %.0f, min %.0f, max %.0f)\n",
		resp.ReturnedCount.Median, resp.ReturnedCount.P10, resp.ReturnedCount.P90,
		resp.ReturnedCount.Min, resp.ReturnedCount.Max)
	fmt.Printf("  Over-budget rate:    %.0f%% of %d budgeted call(s)\n",
		resp.OverBudgetRate*100, resp.BudgetedCallCount)
	fmt.Printf("  Near-threshold rate: %.0f%% (calls whose lowest result sits on the score floor)\n",
		resp.NearThresholdRate*100)
	if resp.ProbedCallCount > 0 {
		fmt.Printf("  Clipping (probed):   %.0f%% of %d probed call(s) clipped >=1 result (median %.0f clipped)\n",
			resp.ThresholdClipRate*100, resp.ProbedCallCount, resp.ClippedPerProbe.Median)
	} else {
		fmt.Println("  Clipping (probed):   no probe samples (set DISCOVERY_PROBE_SAMPLE to enable)")
	}

	if len(resp.PerComplexity) > 0 {
		fmt.Println()
		fmt.Printf("  %-15s  %-6s  %-12s  %s\n", "Complexity", "Calls", "Over-budget", "Median ret.")
		for _, tier := range []string{"minor", "moderate", "major", "architectural"} {
			m, ok := resp.PerComplexity[tier]
			if !ok {
				continue
			}
			fmt.Printf("  %-15s  %-6d  %-12s  %.0f\n",
				tier, m.CallCount, fmt.Sprintf("%.0f%%", m.OverBudgetRate*100), m.MedianReturned)
		}
	}

	if len(resp.BudgetHogs) > 0 {
		fmt.Println()
		fmt.Println("  Largest skills seen (budget hogs):")
		fmt.Printf("  %-32s  %-8s  %-6s  %s\n", "ID", "MaxChars", "Seen", "OverBudgetSeen")
		for _, h := range resp.BudgetHogs {
			id := h.ID
			if len(id) > 32 {
				id = id[:29] + "..."
			}
			fmt.Printf("  %-32s  %-8d  %-6d  %d\n", id, h.MaxChars, h.Occurrences, h.OverBudgetSightings)
		}
	}

	fmt.Println()
	fmt.Println("These metrics drive evidence-based tuning: a high near-threshold or clip rate")
	fmt.Println("suggests lowering AI_SEARCH_THRESHOLD; a high over-budget rate suggests raising")
	fmt.Println("the affected complexity budget.")
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

// SkillUsageRow mirrors one row of GET /skill-usage.
type SkillUsageRow struct {
	SkillID           string         `json:"skillId"`
	Returned          int            `json:"returned"`
	Reads             int            `json:"reads"`
	DemandReads       int            `json:"demandReads"`
	ViaDiscovery      int            `json:"viaDiscovery"`
	ReadsByCallerKind map[string]int `json:"readsByCallerKind,omitempty"`
	ConversionRate    *float64       `json:"conversionRate,omitempty"`
	LastReadAt        string         `json:"lastReadAt,omitempty"`
}

// SkillUsageResponse wraps the per-skill demand aggregation.
type SkillUsageResponse struct {
	Since  string          `json:"since"`
	Unread []string        `json:"unread,omitempty"`
	Rows   []SkillUsageRow `json:"rows"`
}

func cmdSkillUsage(ctx appctx.Context, args []string) error {
	fs := flag.NewFlagSet("skill-usage", flag.ContinueOnError)
	since := fs.String("since", "7d", "Window to report (e.g. 7d, 24h, 30m)")
	limit := fs.Int("limit", 20, "Maximum number of rows to show")
	jsonOut := cliutil.JSONFlag(fs)
	if err := cliutil.ParseInterspersed(fs, args); err != nil {
		return err
	}

	query := url.Values{}
	query.Set("since", *since)

	var resp SkillUsageResponse
	if err := ctx.GetWithQuery("/skill-usage", query, &resp); err != nil {
		return fmt.Errorf("skill-usage failed: %w", err)
	}

	if *jsonOut {
		enc := json.NewEncoder(os.Stdout)
		enc.SetIndent("", "  ")
		return enc.Encode(resp)
	}

	if len(resp.Rows) == 0 {
		fmt.Printf("No skill reads or discovery results recorded in the last %s.\n", *since)
		return nil
	}

	fmt.Printf("Skill demand (last %s, %d skill(s) seen):\n\n", *since, len(resp.Rows))
	fmt.Printf("  %-42s %8s %8s %8s %8s  %s\n", "SKILL", "RETURNED", "READS", "DEMAND", "VIA-DISC", "CONV")
	shown := 0
	for _, row := range resp.Rows {
		if shown >= *limit {
			break
		}
		conv := "-"
		if row.ConversionRate != nil {
			conv = fmt.Sprintf("%.2f", *row.ConversionRate)
		}
		fmt.Printf("  %-42s %8d %8d %8d %8d  %s\n", truncateID(row.SkillID, 42), row.Returned, row.Reads, row.DemandReads, row.ViaDiscovery, conv)
		shown++
	}
	if len(resp.Rows) > shown {
		fmt.Printf("\n  ... %d more (use --limit)\n", len(resp.Rows)-shown)
	}

	if len(resp.Unread) > 0 {
		fmt.Printf("\nReturned but never read (%d) — search-precision suspects, most-offered first:\n", len(resp.Unread))
		for i, id := range resp.Unread {
			if i >= 10 {
				fmt.Printf("  ... %d more\n", len(resp.Unread)-i)
				break
			}
			fmt.Printf("  %s\n", id)
		}
	}

	fmt.Printf("\nDEMAND counts agent-member reads only; audit and operator reads are excluded\nso that visiting a skill does not raise its own demand rank.\n")
	return nil
}

func truncateID(id string, max int) string {
	if len(id) <= max {
		return id
	}
	return id[:max-3] + "..."
}
