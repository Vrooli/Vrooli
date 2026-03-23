// Package overview provides a unified situational-awareness endpoint that
// aggregates backlog items, initiatives, and dependency graph information
// into a single response. This powers the CLI "overview" command and can
// serve as a dashboard data source.
package overview

import (
	"fmt"
	"sort"

	"swarm-manager/internal/backlog"
	"swarm-manager/internal/depgraph"
	"swarm-manager/internal/initiatives"
)

// BacklogLister loads backlog items from the store.
type BacklogLister interface {
	LoadAll(kinds []backlog.BacklogKind) ([]backlog.BacklogItem, error)
}

// InitiativeLister lists initiatives with computed rollup status.
type InitiativeLister interface {
	List() ([]initiatives.InitiativeWithRollup, error)
}

// Service assembles overview data from backlog and initiative sources.
type Service struct {
	backlog     BacklogLister
	initiatives InitiativeLister
}

// NewService creates an overview Service backed by the given data sources.
func NewService(bl BacklogLister, il InitiativeLister) *Service {
	return &Service{
		backlog:     bl,
		initiatives: il,
	}
}

// OverviewResponse is the top-level payload returned by GetOverview.
type OverviewResponse struct {
	Items           []backlog.BacklogItem              `json:"items"`
	Initiatives     []initiatives.InitiativeWithRollup `json:"initiatives"`
	DependencyGraph DependencyGraph                    `json:"dependency_graph"`
	Summary         OverviewSummary                    `json:"summary"`
}

// DependencyGraph captures the edges and reachability status of backlog items.
type DependencyGraph struct {
	Edges     [][2]string `json:"edges"`     // [from, to] dependency pairs
	Unblocked []string    `json:"unblocked"` // items with all deps satisfied
	Blocked   []string    `json:"blocked"`   // items waiting on incomplete deps
}

// OverviewSummary provides aggregate counts for quick triage.
type OverviewSummary struct {
	TotalItems        int            `json:"total_items"`
	ItemsByStatus     map[string]int `json:"items_by_status"`
	ItemsByKind       map[string]int `json:"items_by_kind"`
	ActiveInitiatives int            `json:"active_initiatives"`
}

// GetOverview loads all backlog items and initiatives, builds the dependency
// graph, and computes summary statistics.
func (s *Service) GetOverview() (*OverviewResponse, error) {
	// Load all backlog items (nil kinds = all kinds).
	items, err := s.backlog.LoadAll(nil)
	if err != nil {
		return nil, fmt.Errorf("load backlog items: %w", err)
	}

	// Load initiatives with rollup.
	inits, err := s.initiatives.List()
	if err != nil {
		return nil, fmt.Errorf("load initiatives: %w", err)
	}

	// Build dependency graph and compute blocked/unblocked sets.
	depGraph := buildDependencyGraph(items)

	// Build summary statistics.
	summary := buildSummary(items, inits)

	return &OverviewResponse{
		Items:           items,
		Initiatives:     inits,
		DependencyGraph: depGraph,
		Summary:         summary,
	}, nil
}

// itemKey returns the canonical "kind/name" identifier for a backlog item.
func itemKey(item backlog.BacklogItem) string {
	return string(item.Kind) + "/" + item.Name
}

// buildDependencyGraph constructs the graph from item depends_on fields and
// uses the depgraph package to determine blocked/unblocked items.
func buildDependencyGraph(items []backlog.BacklogItem) DependencyGraph {
	g := depgraph.New()

	// Register every item as a node with its dependencies.
	for _, item := range items {
		g.AddNode(itemKey(item), item.DependsOn)
	}

	// Build completed set for unblocked/blocked computation.
	completed := make(map[string]bool, len(items))
	for _, item := range items {
		if item.Status == backlog.StatusCompleted {
			completed[itemKey(item)] = true
		}
	}

	edges := g.Edges()
	if edges == nil {
		edges = [][2]string{}
	}
	unblocked := g.UnblockedItems(completed)
	if unblocked == nil {
		unblocked = []string{}
	}
	blocked := g.BlockedItems(completed)
	if blocked == nil {
		blocked = []string{}
	}

	return DependencyGraph{
		Edges:     edges,
		Unblocked: unblocked,
		Blocked:   blocked,
	}
}

// buildSummary computes aggregate counts across items and initiatives.
func buildSummary(items []backlog.BacklogItem, inits []initiatives.InitiativeWithRollup) OverviewSummary {
	byStatus := make(map[string]int)
	byKind := make(map[string]int)
	for _, item := range items {
		byStatus[string(item.Status)]++
		byKind[string(item.Kind)]++
	}

	activeCount := 0
	for _, init := range inits {
		if init.Initiative.Status == "active" {
			activeCount++
		}
	}

	return OverviewSummary{
		TotalItems:        len(items),
		ItemsByStatus:     byStatus,
		ItemsByKind:       byKind,
		ActiveInitiatives: activeCount,
	}
}

// SortedMapKeys returns the keys of a map[string]int in sorted order.
// Exported for use by the CLI markdown formatter.
func SortedMapKeys(m map[string]int) []string {
	keys := make([]string, 0, len(m))
	for k := range m {
		keys = append(keys, k)
	}
	sort.Strings(keys)
	return keys
}
