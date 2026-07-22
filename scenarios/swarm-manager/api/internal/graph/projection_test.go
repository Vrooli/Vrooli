package graph

import (
	"context"
	"testing"

	"swarm-manager/internal/backlog"
)

func ptrStr(s string) *string { return &s }

// --- Mock implementations ---

type mockBacklogLister struct {
	items []backlog.BacklogItem
	err   error
}

func (m *mockBacklogLister) LoadAll(_ []backlog.BacklogKind) ([]backlog.BacklogItem, error) {
	return m.items, m.err
}

type mockGoalLister struct {
	goals []GoalEntry
	err   error
}

func (m *mockGoalLister) List() ([]GoalEntry, error) {
	return m.goals, m.err
}

type mockCaptureLister struct {
	caps []CaptureEntry
	err  error
}

func (m *mockCaptureLister) ListCaptures() ([]CaptureEntry, error) {
	return m.caps, m.err
}

type mockScenarioLister struct {
	scens []ScenarioEntry
	err   error
}

func (m *mockScenarioLister) List(_ context.Context) ([]ScenarioEntry, error) {
	return m.scens, m.err
}

func assertEdgeEndpointsPresent(t *testing.T, resp GraphResponse) {
	t.Helper()

	nodeIDs := make(map[string]struct{}, len(resp.Nodes))
	for _, node := range resp.Nodes {
		nodeIDs[node.ID] = struct{}{}
	}

	for _, edge := range resp.Edges {
		if _, ok := nodeIDs[edge.Source]; !ok {
			t.Fatalf("edge %q references missing source node %q", edge.ID, edge.Source)
		}
		if _, ok := nodeIDs[edge.Target]; !ok {
			t.Fatalf("edge %q references missing target node %q", edge.ID, edge.Target)
		}
	}
}

// --- Tests ---

func TestProjectTopology(t *testing.T) {
	svc := NewProjectionService(ProjectionConfig{
		Backlog: &mockBacklogLister{items: []backlog.BacklogItem{
			{Kind: "execute", Name: "task-a", Title: "Task A", Status: backlog.StatusQueued, Priority: 3, DependsOn: []string{"execute/task-b"}, Milestone: "init-1", AcceptanceAllow: []string{"scenarios/my-app"}},
			{Kind: "execute", Name: "task-b", Title: "Task B", Status: "ready", Priority: 5},
			{Kind: "execute", Name: "task-c", Title: "Done", Status: backlog.StatusCompleted},                                           // should be excluded
			{Kind: "idea", Name: "archived", Title: "Old", Status: backlog.StatusCompleted, ArchivedAt: ptrStr("2026-01-01T00:00:00Z")}, // should be excluded
		}},
		Goal: &mockGoalLister{goals: []GoalEntry{
			{Name: "init-1", Title: "Milestone 1", Status: "active", Items: []string{"execute/task-a", "execute/task-c"}},
			{Name: "init-archived", Title: "Archived", Status: "completed", ArchivedAt: ptrStr("2026-01-01T00:00:00Z")}, // excluded
		}},
		Capture: &mockCaptureLister{caps: []CaptureEntry{
			{ID: "cap-1", Text: "fix login", Status: "classified", Items: []CaptureClassificationItem{
				{Kind: "execute", Title: "task-a"},  // matches existing item
				{Kind: "fix", Title: "nonexistent"}, // no match, no edge
			}},
			{ID: "cap-2", Text: "empty", Status: "pending"}, // no items, excluded
		}},
		Scenario: &mockScenarioLister{scens: []ScenarioEntry{
			{Name: "my-app", Status: "running"},
		}},
	})

	resp, err := svc.Project(context.Background(), ProjectionParams{Lens: LensTopology})
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
	if nodeTypes["Goal"] != 1 {
		t.Errorf("expected 1 Goal node, got %d", nodeTypes["Goal"])
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

	assertEdgeEndpointsPresent(t, resp)
}

func TestProjectTopology_TargetsEdges(t *testing.T) {
	svc := NewProjectionService(ProjectionConfig{
		Backlog: &mockBacklogLister{items: []backlog.BacklogItem{
			{Kind: "execute", Name: "deploy-app", Title: "Deploy", Status: "ready", AcceptanceAllow: []string{"scenarios/my-app/**"}},
			{Kind: "execute", Name: "no-match", Title: "Other", Status: "ready", AcceptanceAllow: []string{"scenarios/other/**"}},
		}},
		Scenario: &mockScenarioLister{scens: []ScenarioEntry{
			{Name: "my-app", Status: "running"},
		}},
	})

	resp, err := svc.Project(context.Background(), ProjectionParams{Lens: LensTopology})
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

func TestMemberOfEdges(t *testing.T) {
	svc := NewProjectionService(ProjectionConfig{
		Backlog: &mockBacklogLister{items: []backlog.BacklogItem{
			{Kind: "execute", Name: "task-1", Title: "T1", Status: "ready", Milestone: "my-init"},
			{Kind: "fix", Name: "bug-1", Title: "B1", Status: "ready"}, // no milestone
		}},
		Goal: &mockGoalLister{goals: []GoalEntry{
			{Name: "my-init", Title: "Init", Status: "active"},
		}},
	})

	resp, err := svc.Project(context.Background(), ProjectionParams{Lens: LensTopology})
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
			if e.Target != "goal/my-init" {
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

	resp, err := svc.Project(context.Background(), ProjectionParams{Lens: LensTopology})
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

func TestTopologyGoalRollup(t *testing.T) {
	svc := NewProjectionService(ProjectionConfig{
		Backlog: &mockBacklogLister{items: []backlog.BacklogItem{
			{Kind: "execute", Name: "done", Status: backlog.StatusCompleted},
			{Kind: "execute", Name: "doing", Status: backlog.StatusInProgress},
			{Kind: "execute", Name: "broken", Status: backlog.StatusFailed},
			{Kind: "execute", Name: "todo", Status: backlog.StatusReady},
		}},
		Goal: &mockGoalLister{goals: []GoalEntry{
			{
				Name:   "init-1",
				Title:  "Init 1",
				Status: "active",
				Items: []string{
					"execute/done",
					"execute/doing",
					"execute/broken",
					"execute/todo",
					"execute/missing",
				},
			},
		}},
	})

	resp, err := svc.Project(context.Background(), ProjectionParams{Lens: LensTopology})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	var found bool
	for _, n := range resp.Nodes {
		if n.Type != "Goal" {
			continue
		}
		found = true
		data, ok := n.Data.(GraphGoalNodeData)
		if !ok {
			t.Fatalf("expected GraphGoalNodeData, got %T", n.Data)
		}
		if data.Rollup.Total != 5 {
			t.Errorf("expected total=5, got %v", data.Rollup.Total)
		}
		if data.Rollup.Completed != 1 {
			t.Errorf("expected completed=1, got %v", data.Rollup.Completed)
		}
		if data.Rollup.InProgress != 1 {
			t.Errorf("expected in_progress=1, got %v", data.Rollup.InProgress)
		}
		if data.Rollup.Failed != 1 {
			t.Errorf("expected failed=1, got %v", data.Rollup.Failed)
		}
		if data.Rollup.Pending != 2 {
			t.Errorf("expected pending=2, got %v", data.Rollup.Pending)
		}
	}
	if !found {
		t.Error("expected to find a Goal node")
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

// TestComputeMilestoneRollup_NewStatuses asserts that in_review and
// review_pending items count as InProgress (they are still in flight from the
// milestone's perspective), while needs_followup counts as Failed (it is a
// terminal state that needs more work).
func TestComputeGoalRollup_NewStatuses(t *testing.T) {
	items := []string{
		"execute/a",
		"execute/b",
		"execute/c",
		"execute/d",
		"execute/e",
	}
	itemByKey := map[string]backlog.BacklogItem{
		"execute/a": {Name: "a", Status: backlog.StatusInReview},
		"execute/b": {Name: "b", Status: backlog.StatusReviewPending},
		"execute/c": {Name: "c", Status: backlog.StatusNeedsFollowup},
		"execute/d": {Name: "d", Status: backlog.StatusCompleted},
		"execute/e": {Name: "e", Status: backlog.StatusInProgress},
	}
	got := computeGoalRollup(items, itemByKey)

	if got.Total != 5 {
		t.Errorf("Total = %d, want 5", got.Total)
	}
	if got.Completed != 1 {
		t.Errorf("Completed = %d, want 1 (only the terminal-completed item)", got.Completed)
	}
	if got.Failed != 1 {
		t.Errorf("Failed = %d, want 1 (needs_followup counts as Failed)", got.Failed)
	}
	if got.InProgress != 3 {
		t.Errorf("InProgress = %d, want 3 (in_review + review_pending + in_progress)", got.InProgress)
	}
	if got.Pending != 0 {
		t.Errorf("Pending = %d, want 0", got.Pending)
	}
}
