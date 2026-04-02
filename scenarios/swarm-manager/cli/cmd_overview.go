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
	Items           []BacklogItem        `json:"items"`
	Initiatives     []OverviewInitiative `json:"initiatives"`
	DependencyGraph OverviewDepGraph     `json:"dependency_graph"`
	Summary         OverviewSummary      `json:"summary"`
	Governance      *GovernanceStatus    `json:"governance,omitempty"`
}

// GovernanceStatus mirrors the API governance status.
type GovernanceStatus struct {
	ActiveExecutions    int      `json:"active_executions"`
	MaxConcurrent       int      `json:"max_concurrent"`
	QueueDepth          int      `json:"queue_depth"`
	MaxQueueDepth       int      `json:"max_queue_depth"`
	CircuitBrokenItems  []string `json:"circuit_broken_items"`
	EstimatedQueuedCost float64  `json:"estimated_queued_cost"`
}

// OverviewInitiative pairs an initiative with its rollup status.
type OverviewInitiative struct {
	Initiative Initiative       `json:"initiative"`
	Rollup     InitiativeRollup `json:"rollup"`
}

// OverviewDepGraph captures dependency edges and blocked/unblocked sets.
type OverviewDepGraph struct {
	Edges     [][2]string `json:"edges"`
	Unblocked []string    `json:"unblocked"`
	Blocked   []string    `json:"blocked"`
}

// OverviewSummary provides aggregate counts.
type OverviewSummary struct {
	TotalItems        int            `json:"total_items"`
	ItemsByStatus     map[string]int `json:"items_by_status"`
	ItemsByKind       map[string]int `json:"items_by_kind"`
	ActiveInitiatives int            `json:"active_initiatives"`
}

func (a *App) cmdOverview(args []string) error {
	fs := flag.NewFlagSet("overview", flag.ContinueOnError)
	formatFlag := fs.String("format", "markdown", "Output format: json or markdown")
	jsonOut := cliutil.JSONFlag(fs)
	if err := cliutil.ParseInterspersed(fs, args); err != nil {
		return err
	}

	body, err := a.getV1("/overview", nil)
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

	// Governance section.
	if resp.Governance != nil {
		gov := resp.Governance
		printSection("Governance")
		fmt.Printf("  Executions: %d/%d active | Queue: %d/%d\n",
			gov.ActiveExecutions, gov.MaxConcurrent, gov.QueueDepth, gov.MaxQueueDepth)
		if gov.EstimatedQueuedCost > 0 {
			fmt.Printf("  Estimated queued cost: $%.2f\n", gov.EstimatedQueuedCost)
		}
		if len(gov.CircuitBrokenItems) > 0 {
			fmt.Printf("  Circuit-broken: %s\n", strings.Join(gov.CircuitBrokenItems, ", "))
		}
		fmt.Println()
	}

	// Summary section.
	printSection("Summary")
	statusParts := formatMapCounts(resp.Summary.ItemsByStatus)
	fmt.Printf("  Total items: %d | By status: %s\n", resp.Summary.TotalItems, statusParts)
	kindParts := formatMapCounts(resp.Summary.ItemsByKind)
	fmt.Printf("  By kind: %s\n", kindParts)
	fmt.Printf("  Active initiatives: %d\n", resp.Summary.ActiveInitiatives)
	fmt.Println()

	// Initiatives section.
	if len(resp.Initiatives) > 0 {
		printSection("Initiatives")
		for _, item := range resp.Initiatives {
			init := item.Initiative
			rollup := item.Rollup
			fmt.Printf("  %s -- %d/%d completed -- %s\n",
				init.Title, rollup.Completed, rollup.Total, init.Status)
			if len(init.Items) > 0 {
				fmt.Printf("    Items: %s\n", strings.Join(init.Items, ", "))
			}
		}
		fmt.Println()
	}

	// Ready to execute (unblocked).
	if len(resp.DependencyGraph.Unblocked) > 0 {
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

	// Blocked items.
	if len(resp.DependencyGraph.Blocked) > 0 {
		printSection("Blocked")
		// Build a reverse lookup: which deps block each item.
		blockers := make(map[string][]string)
		for _, edge := range resp.DependencyGraph.Edges {
			from, to := edge[0], edge[1]
			blockers[from] = append(blockers[from], to)
		}
		for _, key := range resp.DependencyGraph.Blocked {
			deps := blockers[key]
			sort.Strings(deps)
			fmt.Printf("  - %s -- blocked by: %s\n", key, strings.Join(deps, ", "))
		}
		fmt.Println()
	}

	// Dependency graph edges.
	if len(resp.DependencyGraph.Edges) > 0 {
		printSection("Dependency Graph")
		for _, edge := range resp.DependencyGraph.Edges {
			fmt.Printf("  %s -> %s\n", edge[0], edge[1])
		}
		fmt.Println()
	}

	// Next steps.
	printCommandListSection("Next Steps", []string{
		cliCommand("backlog", "list"),
		cliCommand("initiatives", "list"),
	})
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
