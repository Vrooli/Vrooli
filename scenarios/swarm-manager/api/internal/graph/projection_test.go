package graph

import (
	"context"
	"fmt"
	"testing"

	"swarm-manager/internal/agentmanager"
	"swarm-manager/internal/backlog"
	"swarm-manager/internal/execution"
	"swarm-manager/internal/initiatives"
	"swarm-manager/internal/scenarios"
)

// --- Mock implementations ---

type mockBacklogLister struct {
	items []backlog.BacklogItem
	err   error
}

func (m *mockBacklogLister) LoadAll(_ []backlog.BacklogKind) ([]backlog.BacklogItem, error) {
	return m.items, m.err
}

type mockInitiativeLister struct {
	inits []initiatives.InitiativeWithRollup
	err   error
}

func (m *mockInitiativeLister) List() ([]initiatives.InitiativeWithRollup, error) {
	return m.inits, m.err
}

type mockCaptureLister struct {
	caps []CaptureEntry
	err  error
}

func (m *mockCaptureLister) ListCaptures() ([]CaptureEntry, error) {
	return m.caps, m.err
}

type mockScenarioLister struct {
	scens []scenarios.Scenario
	err   error
}

func (m *mockScenarioLister) LoadAll() ([]scenarios.Scenario, error) {
	return m.scens, m.err
}

type mockExecutionLister struct {
	records []execution.Record
	err     error
}

func (m *mockExecutionLister) List(_ context.Context, _ execution.ListFilters) ([]execution.Record, error) {
	return m.records, m.err
}

type mockRunStateGetter struct {
	available bool
	states    map[string]agentmanager.RunState
	err       error
}

func (m *mockRunStateGetter) IsAvailable(_ context.Context) bool {
	return m.available
}

func (m *mockRunStateGetter) GetRunState(_ context.Context, runID string) (agentmanager.RunState, error) {
	if m.err != nil {
		return agentmanager.RunState{}, m.err
	}
	if s, ok := m.states[runID]; ok {
		return s, nil
	}
	return agentmanager.RunState{}, fmt.Errorf("run %s not found", runID)
}

// --- Tests ---

func TestProjectTopology(t *testing.T) {
	svc := NewProjectionService(ProjectionConfig{
		Backlog: &mockBacklogLister{items: []backlog.BacklogItem{
			{Kind: "execute", Name: "task-a", Title: "Task A", Status: backlog.StatusQueued, Priority: 3, DependsOn: []string{"execute/task-b"}, Initiative: "init-1"},
			{Kind: "execute", Name: "task-b", Title: "Task B", Status: "ready", Priority: 5},
			{Kind: "execute", Name: "task-c", Title: "Done", Status: backlog.StatusCompleted}, // should be excluded
			{Kind: "idea", Name: "archived", Title: "Old", Status: backlog.StatusArchived},    // should be excluded
		}},
		Initiative: &mockInitiativeLister{inits: []initiatives.InitiativeWithRollup{
			{Initiative: initiatives.Initiative{Name: "init-1", Title: "Initiative 1", Status: "active"}},
			{Initiative: initiatives.Initiative{Name: "init-archived", Title: "Archived", Status: "archived"}}, // excluded
		}},
		Capture: &mockCaptureLister{caps: []CaptureEntry{
			{ID: "cap-1", Text: "fix login", Status: "classified", Items: []CaptureClassificationItem{
				{Kind: "execute", Title: "task-a"},  // matches existing item
				{Kind: "fix", Title: "nonexistent"}, // no match, no edge
			}},
			{ID: "cap-2", Text: "empty", Status: "pending"}, // no items, excluded
		}},
		Scenario: &mockScenarioLister{scens: []scenarios.Scenario{
			{Name: "my-app", Status: scenarios.StatusRunning},
		}},
	})

	resp, err := svc.Project(context.Background(), LensTopology)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if resp.Meta.Lens != LensTopology {
		t.Errorf("expected lens topology, got %s", resp.Meta.Lens)
	}

	// Count nodes by type.
	nodeTypes := map[string]int{}
	for _, n := range resp.Nodes {
		nodeTypes[n.Type]++
	}
	if nodeTypes["BacklogItem"] != 2 {
		t.Errorf("expected 2 BacklogItem nodes, got %d", nodeTypes["BacklogItem"])
	}
	if nodeTypes["Initiative"] != 1 {
		t.Errorf("expected 1 Initiative node, got %d", nodeTypes["Initiative"])
	}
	if nodeTypes["Capture"] != 1 {
		t.Errorf("expected 1 Capture node, got %d", nodeTypes["Capture"])
	}
	if nodeTypes["Scenario"] != 1 {
		t.Errorf("expected 1 Scenario node, got %d", nodeTypes["Scenario"])
	}

	// Count edges by type.
	edgeTypes := map[string]int{}
	for _, e := range resp.Edges {
		edgeTypes[e.Type]++
	}
	if edgeTypes["depends_on"] != 1 {
		t.Errorf("expected 1 depends_on edge, got %d", edgeTypes["depends_on"])
	}
	if edgeTypes["member_of"] != 1 {
		t.Errorf("expected 1 member_of edge, got %d", edgeTypes["member_of"])
	}
	if edgeTypes["classified_as"] != 1 {
		t.Errorf("expected 1 classified_as edge, got %d", edgeTypes["classified_as"])
	}
}

func TestProjectTopology_TargetsEdges(t *testing.T) {
	svc := NewProjectionService(ProjectionConfig{
		Backlog: &mockBacklogLister{items: []backlog.BacklogItem{
			{Kind: "execute", Name: "deploy-app", Title: "Deploy", Status: "ready", AcceptanceAllow: []string{"scenarios/my-app/**"}},
			{Kind: "execute", Name: "no-match", Title: "Other", Status: "ready", AcceptanceAllow: []string{"scenarios/other/**"}},
		}},
		Scenario: &mockScenarioLister{scens: []scenarios.Scenario{
			{Name: "my-app", Status: scenarios.StatusRunning},
		}},
	})

	resp, err := svc.Project(context.Background(), LensTopology)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	targetsCount := 0
	for _, e := range resp.Edges {
		if e.Type == "targets" {
			targetsCount++
			if e.Source != "backlog-item/execute/deploy-app" {
				t.Errorf("expected targets source to be deploy-app, got %s", e.Source)
			}
			if e.Target != "scenario/my-app" {
				t.Errorf("expected targets target to be scenario/my-app, got %s", e.Target)
			}
		}
	}
	if targetsCount != 1 {
		t.Errorf("expected 1 targets edge, got %d", targetsCount)
	}
}

func TestProjectFlow(t *testing.T) {
	svc := NewProjectionService(ProjectionConfig{
		Backlog: &mockBacklogLister{items: []backlog.BacklogItem{
			{Kind: "execute", Name: "active-task", Title: "Active", Status: backlog.StatusInProgress},
			{Kind: "execute", Name: "done-task", Title: "Done", Status: backlog.StatusCompleted}, // excluded
		}},
		Execution: &mockExecutionLister{records: []execution.Record{
			{ExecutionID: "exec-1", BacklogKind: "execute", BacklogName: "active-task", Status: execution.StatusRunning, Mode: "manual", RunID: "run-1"},
			{ExecutionID: "exec-2", BacklogKind: "execute", BacklogName: "active-task", Status: execution.StatusNeedsFixup, Mode: "manual", ParentExecutionID: "exec-1"},
			{ExecutionID: "exec-3", BacklogKind: "execute", BacklogName: "done-task", Status: execution.StatusCompleted, Mode: "manual"}, // excluded
		}},
	})

	resp, err := svc.Project(context.Background(), LensFlow)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	nodeTypes := map[string]int{}
	for _, n := range resp.Nodes {
		nodeTypes[n.Type]++
	}
	if nodeTypes["BacklogItem"] != 1 {
		t.Errorf("expected 1 BacklogItem node, got %d", nodeTypes["BacklogItem"])
	}
	if nodeTypes["ExecutionRecord"] != 2 {
		t.Errorf("expected 2 ExecutionRecord nodes, got %d", nodeTypes["ExecutionRecord"])
	}

	edgeTypes := map[string]int{}
	for _, e := range resp.Edges {
		edgeTypes[e.Type]++
	}
	if edgeTypes["executes"] != 2 {
		t.Errorf("expected 2 executes edges, got %d", edgeTypes["executes"])
	}
	if edgeTypes["follow_up"] != 1 {
		t.Errorf("expected 1 follow_up edge, got %d", edgeTypes["follow_up"])
	}
}

func TestProjectOperations(t *testing.T) {
	svc := NewProjectionService(ProjectionConfig{
		Backlog: &mockBacklogLister{items: []backlog.BacklogItem{
			{Kind: "execute", Name: "task-a", Title: "A", Status: backlog.StatusInProgress, AcceptanceAllow: []string{"scenarios/my-app/**"}},
		}},
		Scenario: &mockScenarioLister{scens: []scenarios.Scenario{
			{Name: "my-app", Status: scenarios.StatusRunning},
		}},
		Execution: &mockExecutionLister{records: []execution.Record{
			{ExecutionID: "exec-1", BacklogKind: "execute", BacklogName: "task-a", Status: execution.StatusRunning, Mode: "manual", RunID: "run-1"},
		}},
		RunState: &mockRunStateGetter{
			available: true,
			states: map[string]agentmanager.RunState{
				"run-1": {RunID: "run-1", TaskID: "task-1", Status: "running"},
			},
		},
	})

	resp, err := svc.Project(context.Background(), LensOperations)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	nodeTypes := map[string]int{}
	for _, n := range resp.Nodes {
		nodeTypes[n.Type]++
	}
	if nodeTypes["Scenario"] != 1 {
		t.Errorf("expected 1 Scenario node, got %d", nodeTypes["Scenario"])
	}
	if nodeTypes["ExecutionRecord"] != 1 {
		t.Errorf("expected 1 ExecutionRecord node, got %d", nodeTypes["ExecutionRecord"])
	}
	if nodeTypes["Run"] != 1 {
		t.Errorf("expected 1 Run node, got %d", nodeTypes["Run"])
	}

	edgeTypes := map[string]int{}
	for _, e := range resp.Edges {
		edgeTypes[e.Type]++
	}
	if edgeTypes["spawned_run"] != 1 {
		t.Errorf("expected 1 spawned_run edge, got %d", edgeTypes["spawned_run"])
	}
	if edgeTypes["targets"] != 1 {
		t.Errorf("expected 1 targets edge, got %d", edgeTypes["targets"])
	}

	if resp.Meta.AgentManagerAvailable == nil || !*resp.Meta.AgentManagerAvailable {
		t.Error("expected agent_manager_available to be true")
	}
}

func TestProjectOperations_AgentUnavailable(t *testing.T) {
	svc := NewProjectionService(ProjectionConfig{
		Backlog: &mockBacklogLister{items: nil},
		Scenario: &mockScenarioLister{scens: []scenarios.Scenario{
			{Name: "my-app", Status: scenarios.StatusRunning},
		}},
		Execution: &mockExecutionLister{records: []execution.Record{
			{ExecutionID: "exec-1", BacklogKind: "execute", BacklogName: "task-a", Status: execution.StatusRunning, RunID: "run-1"},
		}},
		RunState: &mockRunStateGetter{available: false},
	})

	resp, err := svc.Project(context.Background(), LensOperations)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	// Run node should be absent.
	for _, n := range resp.Nodes {
		if n.Type == "Run" {
			t.Error("expected no Run nodes when agent-manager unavailable")
		}
	}
	if resp.Meta.AgentManagerAvailable == nil || *resp.Meta.AgentManagerAvailable {
		t.Error("expected agent_manager_available to be false")
	}
}

func TestMemberOfEdges(t *testing.T) {
	svc := NewProjectionService(ProjectionConfig{
		Backlog: &mockBacklogLister{items: []backlog.BacklogItem{
			{Kind: "execute", Name: "task-1", Title: "T1", Status: "ready", Initiative: "my-init"},
			{Kind: "fix", Name: "bug-1", Title: "B1", Status: "ready"}, // no initiative
		}},
		Initiative: &mockInitiativeLister{inits: []initiatives.InitiativeWithRollup{
			{Initiative: initiatives.Initiative{Name: "my-init", Title: "Init", Status: "active"}},
		}},
	})

	resp, err := svc.Project(context.Background(), LensTopology)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	memberOfCount := 0
	for _, e := range resp.Edges {
		if e.Type == "member_of" {
			memberOfCount++
			if e.Source != "backlog-item/execute/task-1" {
				t.Errorf("expected member_of source to be task-1, got %s", e.Source)
			}
			if e.Target != "initiative/my-init" {
				t.Errorf("expected member_of target to be my-init, got %s", e.Target)
			}
		}
	}
	if memberOfCount != 1 {
		t.Errorf("expected 1 member_of edge, got %d", memberOfCount)
	}
}

func TestClassifiedAsEdges(t *testing.T) {
	svc := NewProjectionService(ProjectionConfig{
		Backlog: &mockBacklogLister{items: []backlog.BacklogItem{
			{Kind: "execute", Name: "deploy", Title: "deploy", Status: "ready"},
		}},
		Capture: &mockCaptureLister{caps: []CaptureEntry{
			{ID: "cap-1", Text: "deploy", Status: "classified", Items: []CaptureClassificationItem{
				{Kind: "execute", Title: "deploy"},  // matches
				{Kind: "fix", Title: "no-such-bug"}, // no match
			}},
		}},
	})

	resp, err := svc.Project(context.Background(), LensTopology)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	classifiedCount := 0
	for _, e := range resp.Edges {
		if e.Type == "classified_as" {
			classifiedCount++
		}
	}
	if classifiedCount != 1 {
		t.Errorf("expected 1 classified_as edge, got %d", classifiedCount)
	}
}

func TestTopologyInitiativeRollup(t *testing.T) {
	svc := NewProjectionService(ProjectionConfig{
		Backlog: &mockBacklogLister{items: nil},
		Initiative: &mockInitiativeLister{inits: []initiatives.InitiativeWithRollup{
			{
				Initiative: initiatives.Initiative{Name: "init-1", Title: "Init 1", Status: "active"},
				Rollup:     initiatives.RollupStatus{Total: 5, Completed: 2, InProgress: 1, Failed: 1, Pending: 1},
			},
		}},
	})

	resp, err := svc.Project(context.Background(), LensTopology)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	var found bool
	for _, n := range resp.Nodes {
		if n.Type != "Initiative" {
			continue
		}
		found = true
		data, ok := n.Data.(map[string]any)
		if !ok {
			t.Fatal("expected Data to be map[string]any")
		}
		rollup, ok := data["rollup"].(map[string]any)
		if !ok {
			t.Fatal("expected rollup to be map[string]any")
		}
		if rollup["total"] != 5 {
			t.Errorf("expected total=5, got %v", rollup["total"])
		}
		if rollup["completed"] != 2 {
			t.Errorf("expected completed=2, got %v", rollup["completed"])
		}
		if rollup["in_progress"] != 1 {
			t.Errorf("expected in_progress=1, got %v", rollup["in_progress"])
		}
		if rollup["failed"] != 1 {
			t.Errorf("expected failed=1, got %v", rollup["failed"])
		}
		if rollup["pending"] != 1 {
			t.Errorf("expected pending=1, got %v", rollup["pending"])
		}
	}
	if !found {
		t.Error("expected to find an Initiative node")
	}
}

func TestMatchesAcceptancePattern(t *testing.T) {
	tests := []struct {
		pattern  string
		scenario string
		want     bool
	}{
		{"scenarios/my-app/**", "my-app", true},
		{"scenarios/my-app/*", "my-app", true},
		{"scenarios/my-app", "my-app", true},
		{"scenarios/other/**", "my-app", false},
		{"my-app", "my-app", true},
	}
	for _, tc := range tests {
		got := matchesAcceptancePattern(tc.pattern, tc.scenario)
		if got != tc.want {
			t.Errorf("matchesAcceptancePattern(%q, %q) = %v, want %v", tc.pattern, tc.scenario, got, tc.want)
		}
	}
}
