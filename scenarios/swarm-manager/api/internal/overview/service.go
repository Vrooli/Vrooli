// Package overview provides a unified situational-awareness endpoint that
// aggregates backlog items, goals, and dependency graph information
// into a single response. This powers the CLI "overview" command and can
// serve as a dashboard data source.
package overview

import (
	"fmt"
	"sort"

	"swarm-manager/internal/backlog"
	"swarm-manager/internal/depgraph"
	"swarm-manager/internal/execution"
	"swarm-manager/internal/goals"
)

// BacklogLister loads backlog items from the store.
type BacklogLister interface {
	LoadAll(kinds []backlog.BacklogKind) ([]backlog.BacklogItem, error)
}

// GoalLister lists goals with their derived scope.
type GoalLister interface {
	List() ([]goals.GoalWithScope, error)
}

// GovernanceProvider returns governance status from the execution service.
type GovernanceProvider interface {
	GovernanceStatus() (*execution.GovernanceStatusResponse, error)
}

// Service assembles overview data from backlog and goal sources.
type Service struct {
	backlog    BacklogLister
	goals      GoalLister
	governance GovernanceProvider
}

// NewService creates an overview Service backed by the given data sources.
func NewService(bl BacklogLister, gl GoalLister) *Service {
	return &Service{
		backlog: bl,
		goals:   gl,
	}
}

// SetGovernanceProvider injects a governance provider for overview responses.
func (s *Service) SetGovernanceProvider(gp GovernanceProvider) {
	s.governance = gp
}

// OverviewResponse is the top-level payload returned by GetOverview.
type OverviewResponse struct {
	Items           []backlog.BacklogItem               `json:"items"`
	Goals           []goals.GoalWithScope               `json:"goals"`
	DependencyGraph DependencyGraph                     `json:"dependency_graph"`
	Summary         OverviewSummary                     `json:"summary"`
	Consistency     ConsistencyReport                   `json:"consistency"`
	Governance      *execution.GovernanceStatusResponse `json:"governance,omitempty"`
}

// DependencyGraph captures the edges and reachability status of backlog items.
type DependencyGraph struct {
	Edges     [][2]string `json:"edges"`     // [from, to] dependency pairs
	Unblocked []string    `json:"unblocked"` // items with all deps satisfied
	Blocked   []string    `json:"blocked"`   // items waiting on incomplete deps
}

// OverviewSummary provides aggregate counts for quick triage.
type OverviewSummary struct {
	TotalItems    int            `json:"total_items"`
	ItemsByStatus map[string]int `json:"items_by_status"`
	ItemsByKind   map[string]int `json:"items_by_kind"`
	ActiveGoals   int            `json:"active_goals"`
}

// GetOverview loads all backlog items and goals, builds the dependency
// graph, and computes summary statistics.
func (s *Service) GetOverview() (*OverviewResponse, error) {
	// Load all backlog items (nil kinds = all kinds).
	items, err := s.backlog.LoadAll(nil)
	if err != nil {
		return nil, fmt.Errorf("load backlog items: %w", err)
	}

	// Load goals with their derived scope.
	goalList, err := s.goals.List()
	if err != nil {
		return nil, fmt.Errorf("load goals: %w", err)
	}

	// Build dependency graph and compute blocked/unblocked sets.
	depGraph := buildDependencyGraph(items)

	// Build summary statistics.
	summary := buildSummary(items, goalList)

	consistency := ConsistencyReport{
		GoalScopeSuggestions: computeGoalScopeSuggestions(items, goalList),
	}

	resp := &OverviewResponse{
		Items:           items,
		Goals:           goalList,
		DependencyGraph: depGraph,
		Summary:         summary,
		Consistency:     consistency,
	}

	if s.governance != nil {
		if govStatus, govErr := s.governance.GovernanceStatus(); govErr == nil {
			resp.Governance = govStatus
		}
	}

	return resp, nil
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

// buildSummary computes aggregate counts across items and goals.
func buildSummary(items []backlog.BacklogItem, goalList []goals.GoalWithScope) OverviewSummary {
	byStatus := make(map[string]int)
	byKind := make(map[string]int)
	for _, item := range items {
		byStatus[string(item.Status)]++
		byKind[string(item.Kind)]++
	}

	activeCount := 0
	for _, goal := range goalList {
		if goal.Goal.Status == goals.StatusActive {
			activeCount++
		}
	}

	return OverviewSummary{
		TotalItems:    len(items),
		ItemsByStatus: byStatus,
		ItemsByKind:   byKind,
		ActiveGoals:   activeCount,
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
