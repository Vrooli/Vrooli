package main

import (
	"encoding/json"
	"flag"
	"fmt"
	"sort"
	"strings"

	"github.com/vrooli/cli-core/cliutil"
)

// OverviewResponse mirrors the API overview endpoint payload.
type OverviewResponse struct {
	Items           []BacklogItem       `json:"items"`
	Goals           []OverviewGoal      `json:"goals"`
	DependencyGraph OverviewDepGraph    `json:"dependency_graph"`
	Summary         OverviewSummary     `json:"summary"`
	Consistency     OverviewConsistency `json:"consistency"`
	Governance      *GovernanceStatus   `json:"governance,omitempty"`
}

// OverviewConsistency surfaces drift signals the Portfolio Manager should raise
// but never auto-apply.
type OverviewConsistency struct {
	GoalScopeSuggestions []GoalScopeSuggestion `json:"goal_scope_suggestions"`
}

type GoalScopeSuggestion struct {
	Goal   string `json:"goal"`
	Reason string `json:"reason"`
}

// LaneStatus mirrors the per-lane utilization slice in the API governance
// status. Lane names are "investigate", "execute", "review", "reconcile".
type LaneStatus struct {
	Lane     string `json:"lane"`
	Active   int    `json:"active"`
	Capacity int    `json:"capacity"`
	Queue    int    `json:"queue"`
}

// GovernanceStatus mirrors the API governance status.
type GovernanceStatus struct {
	Lanes               []LaneStatus `json:"lanes"`
	ActiveExecutions    int          `json:"active_executions"`
	QueueDepth          int          `json:"queue_depth"`
	MaxQueueDepth       int          `json:"max_queue_depth"`
	CircuitBrokenItems  []string     `json:"circuit_broken_items"`
	EstimatedQueuedCost float64      `json:"estimated_queued_cost"`
}

// OverviewGoal pairs a goal with its derived closure rollup.
type OverviewGoal struct {
	Goal struct {
		Name     string `json:"name"`
		Title    string `json:"title"`
		Status   string `json:"status"`
		Priority int    `json:"priority"`
	} `json:"goal"`
	Scope struct {
		Total          int      `json:"total"`
		CompletedCount int      `json:"completed_count"`
		Ready          []string `json:"ready"`
		Blocked        []string `json:"blocked"`
	} `json:"scope"`
}

// OverviewDepGraph captures dependency edges and blocked/unblocked sets.
type OverviewDepGraph struct {
	Edges     [][2]string `json:"edges"`
	Unblocked []string    `json:"unblocked"`
	Blocked   []string    `json:"blocked"`
}

// OverviewSummary provides aggregate counts.
type OverviewSummary struct {
	TotalItems    int            `json:"total_items"`
	ItemsByStatus map[string]int `json:"items_by_status"`
	ItemsByKind   map[string]int `json:"items_by_kind"`
	ActiveGoals   int            `json:"active_goals"`
}

func (a *App) cmdOverview(args []string) error {
	fs := flag.NewFlagSet("overview", flag.ContinueOnError)
	formatFlag := fs.String("format", "markdown", "Output format: json or markdown")
	jsonOut := cliutil.JSONFlag(fs)
	if err := cliutil.ParseInterspersed(fs, args); err != nil {
		return err
	}

	body, err := a.core.Get("/overview", nil)
	if err != nil {
		return err
	}

	// --json is a shortcut for --format json
	if *jsonOut || *formatFlag == "json" {
		cliutil.PrintJSON(body)
		return nil
	}

	var resp OverviewResponse
	if err := json.Unmarshal(body, &resp); err != nil {
		return fmt.Errorf("failed to parse overview response: %w", err)
	}

	printOverviewMarkdown(resp)
	return nil
}

func printOverviewMarkdown(resp OverviewResponse) {
	fmt.Println("## Swarm Manager Overview")
	fmt.Println()

	printOverviewGovernance(resp.Governance)
	printOverviewSummary(resp.Summary)
	printOverviewGoals(resp.Goals)
	printOverviewUnblocked(resp)
	printOverviewBlocked(resp.DependencyGraph)
	printOverviewEdges(resp.DependencyGraph)
	printOverviewScopeSuggestions(resp.Consistency.GoalScopeSuggestions)

	// Next steps.
	printCommandListSection("Next Steps", []string{
		cliCommand("backlog", "list"),
		cliCommand("goals", "list"),
	})
}

func printOverviewGovernance(gov *GovernanceStatus) {
	if gov == nil {
		return
	}
	printSection("Governance")
	fmt.Printf("  Active total: %d | Queue: %d/%d\n",
		gov.ActiveExecutions, gov.QueueDepth, gov.MaxQueueDepth)
	// Per-lane breakdown — mirrors the four-lane Operations Center
	// header on the UI.
	for _, lane := range gov.Lanes {
		fmt.Printf("  %-12s %d/%d active", lane.Lane, lane.Active, lane.Capacity)
		if lane.Queue > 0 {
			fmt.Printf(" | queue %d", lane.Queue)
		}
		fmt.Println()
	}
	if gov.EstimatedQueuedCost > 0 {
		fmt.Printf("  Estimated queued cost: $%.2f\n", gov.EstimatedQueuedCost)
	}
	if len(gov.CircuitBrokenItems) > 0 {
		fmt.Printf("  Circuit-broken: %s\n", strings.Join(gov.CircuitBrokenItems, ", "))
	}
	fmt.Println()
}

func printOverviewSummary(summary OverviewSummary) {
	printSection("Summary")
	statusParts := formatMapCounts(summary.ItemsByStatus)
	fmt.Printf("  Total items: %d | By status: %s\n", summary.TotalItems, statusParts)
	kindParts := formatMapCounts(summary.ItemsByKind)
	fmt.Printf("  By kind: %s\n", kindParts)
	fmt.Printf("  Active goals: %d\n", summary.ActiveGoals)
	fmt.Println()
}

func printOverviewGoals(goals []OverviewGoal) {
	if len(goals) == 0 {
		return
	}
	printSection("Goals")
	for _, item := range goals {
		fmt.Printf("  %s -- %d/%d completed -- %s\n", item.Goal.Title, item.Scope.CompletedCount, item.Scope.Total, item.Goal.Status)
		if len(item.Scope.Ready) > 0 {
			fmt.Printf("    Ready: %s\n", strings.Join(item.Scope.Ready, ", "))
		}
	}
	fmt.Println()
}

func printOverviewUnblocked(resp OverviewResponse) {
	if len(resp.DependencyGraph.Unblocked) == 0 {
		return
	}
	printSection("Ready to Execute (unblocked)")
	// Build a lookup for quick access to item details.
	itemMap := make(map[string]BacklogItem, len(resp.Items))
	for _, item := range resp.Items {
		itemMap[item.Kind+"/"+item.Name] = item
	}
	for _, key := range resp.DependencyGraph.Unblocked {
		if item, ok := itemMap[key]; ok {
			fmt.Printf("  - %s -- %s (priority: %d)\n", key, item.Title, item.Priority)
		} else {
			fmt.Printf("  - %s\n", key)
		}
	}
	fmt.Println()
}

func printOverviewBlocked(graph OverviewDepGraph) {
	if len(graph.Blocked) == 0 {
		return
	}
	printSection("Blocked")
	// Build a reverse lookup: which deps block each item.
	blockers := make(map[string][]string)
	for _, edge := range graph.Edges {
		from, to := edge[0], edge[1]
		blockers[from] = append(blockers[from], to)
	}
	for _, key := range graph.Blocked {
		deps := blockers[key]
		sort.Strings(deps)
		fmt.Printf("  - %s -- blocked by: %s\n", key, strings.Join(deps, ", "))
	}
	fmt.Println()
}

func printOverviewEdges(graph OverviewDepGraph) {
	if len(graph.Edges) == 0 {
		return
	}
	printSection("Dependency Graph")
	for _, edge := range graph.Edges {
		fmt.Printf("  %s -> %s\n", edge[0], edge[1])
	}
	fmt.Println()
}

func printOverviewScopeSuggestions(suggestions []GoalScopeSuggestion) {
	if len(suggestions) == 0 {
		return
	}
	printSection("Goal Scope Suggestions")
	for _, s := range suggestions {
		fmt.Printf("  - %s: %s\n", s.Goal, s.Reason)
	}
	fmt.Println()
}

// formatMapCounts renders a map as "key1(N) key2(M) ..." in sorted key order.
func formatMapCounts(m map[string]int) string {
	keys := sortedKeys(m)
	parts := make([]string, 0, len(keys))
	for _, k := range keys {
		parts = append(parts, fmt.Sprintf("%s(%d)", k, m[k]))
	}
	return strings.Join(parts, " ")
}

// sortedKeys returns map keys in sorted order.
func sortedKeys(m map[string]int) []string {
	keys := make([]string, 0, len(m))
	for k := range m {
		keys = append(keys, k)
	}
	sort.Strings(keys)
	return keys
}
