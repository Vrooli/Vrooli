package overview

import (
	"fmt"
	"swarm-manager/internal/backlog"
	"swarm-manager/internal/initiatives"
	"testing"
)

// mockBacklogLister implements BacklogLister for testing.
type mockBacklogLister struct {
	items []backlog.BacklogItem
	err   error
}

func (m *mockBacklogLister) LoadAll(_ []backlog.BacklogKind) ([]backlog.BacklogItem, error) {
	if m.err != nil {
		return nil, m.err
	}
	return m.items, nil
}

// mockInitiativeLister implements InitiativeLister for testing.
type mockInitiativeLister struct {
	items []initiatives.InitiativeWithRollup
	err   error
}

func (m *mockInitiativeLister) List() ([]initiatives.InitiativeWithRollup, error) {
	if m.err != nil {
		return nil, m.err
	}
	return m.items, nil
}

func TestGetOverview_EmptyBacklog(t *testing.T) {
	svc := NewService(
		&mockBacklogLister{items: []backlog.BacklogItem{}},
		&mockInitiativeLister{items: []initiatives.InitiativeWithRollup{}},
	)

	resp, err := svc.GetOverview()
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if len(resp.Items) != 0 {
		t.Errorf("expected 0 items, got %d", len(resp.Items))
	}
	if len(resp.Initiatives) != 0 {
		t.Errorf("expected 0 initiatives, got %d", len(resp.Initiatives))
	}
	if resp.Summary.TotalItems != 0 {
		t.Errorf("expected total_items=0, got %d", resp.Summary.TotalItems)
	}
	if resp.Summary.ActiveInitiatives != 0 {
		t.Errorf("expected active_initiatives=0, got %d", resp.Summary.ActiveInitiatives)
	}
	if len(resp.DependencyGraph.Edges) != 0 {
		t.Errorf("expected 0 edges, got %d", len(resp.DependencyGraph.Edges))
	}
	if len(resp.DependencyGraph.Unblocked) != 0 {
		t.Errorf("expected 0 unblocked, got %d", len(resp.DependencyGraph.Unblocked))
	}
	if len(resp.DependencyGraph.Blocked) != 0 {
		t.Errorf("expected 0 blocked, got %d", len(resp.DependencyGraph.Blocked))
	}
}

func TestGetOverview_SummaryCounts(t *testing.T) {
	items := []backlog.BacklogItem{
		{Name: "a", Kind: backlog.KindIdea, Status: backlog.StatusBacklog, Priority: 5, Tags: []string{}},
		{Name: "b", Kind: backlog.KindIdea, Status: backlog.StatusBacklog, Priority: 3, Tags: []string{}},
		{Name: "c", Kind: backlog.KindFix, Status: backlog.StatusReady, Priority: 7, Tags: []string{}},
		{Name: "d", Kind: backlog.KindExecute, Status: backlog.StatusInProgress, Priority: 2, Tags: []string{}},
		{Name: "e", Kind: backlog.KindExecute, Status: backlog.StatusCompleted, Priority: 1, Tags: []string{}},
	}
	inits := []initiatives.InitiativeWithRollup{
		{Initiative: initiatives.Initiative{Name: "i1", Status: "active"}},
		{Initiative: initiatives.Initiative{Name: "i2", Status: "completed"}},
		{Initiative: initiatives.Initiative{Name: "i3", Status: "active"}},
	}

	svc := NewService(
		&mockBacklogLister{items: items},
		&mockInitiativeLister{items: inits},
	)

	resp, err := svc.GetOverview()
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if resp.Summary.TotalItems != 5 {
		t.Errorf("expected total_items=5, got %d", resp.Summary.TotalItems)
	}
	if resp.Summary.ActiveInitiatives != 2 {
		t.Errorf("expected active_initiatives=2, got %d", resp.Summary.ActiveInitiatives)
	}

	// Verify status counts.
	wantStatus := map[string]int{
		"backlog":     2,
		"ready":       1,
		"in_progress": 1,
		"completed":   1,
	}
	for status, want := range wantStatus {
		if got := resp.Summary.ItemsByStatus[status]; got != want {
			t.Errorf("items_by_status[%s]: want %d, got %d", status, want, got)
		}
	}

	// Verify kind counts.
	wantKind := map[string]int{
		"idea":    2,
		"fix":     1,
		"execute": 2,
	}
	for kind, want := range wantKind {
		if got := resp.Summary.ItemsByKind[kind]; got != want {
			t.Errorf("items_by_kind[%s]: want %d, got %d", kind, want, got)
		}
	}
}

func TestGetOverview_DependencyGraph(t *testing.T) {
	items := []backlog.BacklogItem{
		{Name: "base", Kind: backlog.KindResearch, Status: backlog.StatusCompleted, Priority: 5, Tags: []string{}},
		{Name: "mid", Kind: backlog.KindExecute, Status: backlog.StatusReady, Priority: 3, Tags: []string{}, DependsOn: []string{"research/base"}},
		{Name: "top", Kind: backlog.KindExecute, Status: backlog.StatusBacklog, Priority: 1, Tags: []string{}, DependsOn: []string{"execute/mid"}},
	}

	svc := NewService(
		&mockBacklogLister{items: items},
		&mockInitiativeLister{items: []initiatives.InitiativeWithRollup{}},
	)

	resp, err := svc.GetOverview()
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	// Edges: mid -> base, top -> mid
	if len(resp.DependencyGraph.Edges) != 2 {
		t.Fatalf("expected 2 edges, got %d: %v", len(resp.DependencyGraph.Edges), resp.DependencyGraph.Edges)
	}

	// "mid" depends on completed "base" so it should be unblocked.
	// "base" is completed so it should not appear in unblocked.
	// "top" depends on incomplete "mid" so it should be blocked.
	assertContains(t, resp.DependencyGraph.Unblocked, "execute/mid", "unblocked")
	assertNotContains(t, resp.DependencyGraph.Unblocked, "research/base", "unblocked")
	assertContains(t, resp.DependencyGraph.Blocked, "execute/top", "blocked")
	assertNotContains(t, resp.DependencyGraph.Blocked, "execute/mid", "blocked")
}

func TestGetOverview_NoDependencies(t *testing.T) {
	items := []backlog.BacklogItem{
		{Name: "a", Kind: backlog.KindIdea, Status: backlog.StatusBacklog, Priority: 5, Tags: []string{}},
		{Name: "b", Kind: backlog.KindFix, Status: backlog.StatusReady, Priority: 3, Tags: []string{}},
	}

	svc := NewService(
		&mockBacklogLister{items: items},
		&mockInitiativeLister{items: []initiatives.InitiativeWithRollup{}},
	)

	resp, err := svc.GetOverview()
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if len(resp.DependencyGraph.Edges) != 0 {
		t.Errorf("expected 0 edges, got %d", len(resp.DependencyGraph.Edges))
	}
	// Both items have no deps and are not completed, so both are unblocked.
	if len(resp.DependencyGraph.Unblocked) != 2 {
		t.Errorf("expected 2 unblocked items, got %d: %v", len(resp.DependencyGraph.Unblocked), resp.DependencyGraph.Unblocked)
	}
	if len(resp.DependencyGraph.Blocked) != 0 {
		t.Errorf("expected 0 blocked items, got %d", len(resp.DependencyGraph.Blocked))
	}
}

func TestGetOverview_BacklogError(t *testing.T) {
	svc := NewService(
		&mockBacklogLister{err: fmt.Errorf("disk error")},
		&mockInitiativeLister{items: []initiatives.InitiativeWithRollup{}},
	)

	_, err := svc.GetOverview()
	if err == nil {
		t.Fatal("expected error, got nil")
	}
}

func TestGetOverview_InitiativesError(t *testing.T) {
	svc := NewService(
		&mockBacklogLister{items: []backlog.BacklogItem{}},
		&mockInitiativeLister{err: fmt.Errorf("disk error")},
	)

	_, err := svc.GetOverview()
	if err == nil {
		t.Fatal("expected error, got nil")
	}
}

func TestSortedMapKeys(t *testing.T) {
	m := map[string]int{"cherry": 1, "apple": 2, "banana": 3}
	keys := SortedMapKeys(m)
	want := []string{"apple", "banana", "cherry"}
	if len(keys) != len(want) {
		t.Fatalf("expected %d keys, got %d", len(want), len(keys))
	}
	for i, k := range keys {
		if k != want[i] {
			t.Errorf("key[%d]: want %q, got %q", i, want[i], k)
		}
	}
}

func assertContains(t *testing.T, slice []string, value, label string) {
	t.Helper()
	for _, s := range slice {
		if s == value {
			return
		}
	}
	t.Errorf("%s should contain %q, got %v", label, value, slice)
}

func assertNotContains(t *testing.T, slice []string, value, label string) {
	t.Helper()
	for _, s := range slice {
		if s == value {
			t.Errorf("%s should not contain %q, got %v", label, value, slice)
			return
		}
	}
}

func TestGetOverview_MissingInitiativeGraceful(t *testing.T) {
	// Items reference an initiative that doesn't appear in the initiative list.
	items := []backlog.BacklogItem{
		{Name: "orphan-a", Kind: backlog.KindIdea, Status: backlog.StatusBacklog, Priority: 3, Tags: []string{}, Initiative: "deleted-init"},
		{Name: "orphan-b", Kind: backlog.KindFix, Status: backlog.StatusReady, Priority: 2, Tags: []string{}, Initiative: "deleted-init"},
	}

	svc := NewService(
		&mockBacklogLister{items: items},
		&mockInitiativeLister{items: []initiatives.InitiativeWithRollup{}}, // no initiatives returned
	)

	resp, err := svc.GetOverview()
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	// Items should still be present.
	if len(resp.Items) != 2 {
		t.Errorf("expected 2 items, got %d", len(resp.Items))
	}

	// Initiatives section should be empty (deleted initiative not returned).
	if len(resp.Initiatives) != 0 {
		t.Errorf("expected 0 initiatives, got %d", len(resp.Initiatives))
	}

	// Summary should still count items correctly.
	if resp.Summary.TotalItems != 2 {
		t.Errorf("expected total_items=2, got %d", resp.Summary.TotalItems)
	}
}
