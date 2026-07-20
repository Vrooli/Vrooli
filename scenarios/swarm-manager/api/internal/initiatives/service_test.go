package initiatives

import (
	"context"
	"errors"
	"fmt"
	"strings"
	"testing"

	"swarm-manager/internal/backlog"
)

func strPtr(value string) *string {
	return &value
}

func slicePtr(values []string) *[]string {
	return &values
}

func intPtr(value int) *int {
	return &value
}

// mockBacklogLoader is a test double implementing BacklogLoader. It tracks
// cascade writes so tests can assert that initiative service operations
// correctly propagate item-initiative changes to the backlog side.
type mockBacklogLoader struct {
	items      map[string]backlog.BacklogItem // key: "kind/name"
	setCalls   []mockBacklogCall
	clearCalls []mockBacklogCall
	setErr     error
	clearErr   error
}

// mockBacklogCall records a call to SetItemInitiative or ClearItemInitiative.
type mockBacklogCall struct {
	Kind      backlog.BacklogKind
	Name      string
	Value     string // new value for Set; expected value for Clear
	PrevValue string
	Changed   bool
}

func (m *mockBacklogLoader) LoadItem(kind backlog.BacklogKind, name string) (backlog.BacklogItem, error) {
	key := string(kind) + "/" + name
	item, ok := m.items[key]
	if !ok {
		return backlog.BacklogItem{}, fmt.Errorf("item %q not found", key)
	}
	return item, nil
}

func (m *mockBacklogLoader) SetItemInitiative(kind backlog.BacklogKind, name, initiative string) (string, error) {
	if m.setErr != nil {
		return "", m.setErr
	}
	key := string(kind) + "/" + name
	item, ok := m.items[key]
	if !ok {
		return "", fmt.Errorf("item %q not found", key)
	}
	prev := item.Initiative
	item.Initiative = initiative
	m.items[key] = item
	m.setCalls = append(m.setCalls, mockBacklogCall{Kind: kind, Name: name, Value: initiative, PrevValue: prev})
	return prev, nil
}

func (m *mockBacklogLoader) ClearItemInitiative(kind backlog.BacklogKind, name, expected string) (string, bool, error) {
	if m.clearErr != nil {
		return "", false, m.clearErr
	}
	key := string(kind) + "/" + name
	item, ok := m.items[key]
	if !ok {
		m.clearCalls = append(m.clearCalls, mockBacklogCall{Kind: kind, Name: name, Value: expected, PrevValue: "", Changed: false})
		return "", false, nil
	}
	prev := item.Initiative
	if prev != expected {
		m.clearCalls = append(m.clearCalls, mockBacklogCall{Kind: kind, Name: name, Value: expected, PrevValue: prev, Changed: false})
		return prev, false, nil
	}
	item.Initiative = ""
	m.items[key] = item
	m.clearCalls = append(m.clearCalls, mockBacklogCall{Kind: kind, Name: name, Value: expected, PrevValue: prev, Changed: true})
	return prev, true, nil
}

func newTestService(t *testing.T, items map[string]backlog.BacklogItem) *Service {
	t.Helper()
	store := setupTestStore(t)
	loader := &mockBacklogLoader{items: items}
	return NewService(store, loader)
}

func TestService_CreateAndGet(t *testing.T) {
	svc := newTestService(t, nil)

	init, err := svc.Create(CreateRequest{
		Name:        "my-init",
		Title:       "My Initiative",
		Description: "Description",
		Items:       []string{"idea/foo"},
	})
	if err != nil {
		t.Fatalf("Create failed: %v", err)
	}
	if init.Name != "my-init" {
		t.Errorf("expected name my-init, got %q", init.Name)
	}
	if init.Status != "active" {
		t.Errorf("expected status active, got %q", init.Status)
	}
	if init.Mode != "item-level" {
		t.Errorf("expected default mode item-level, got %q", init.Mode)
	}

	result, err := svc.Get("my-init")
	if err != nil {
		t.Fatalf("Get failed: %v", err)
	}
	if result.Initiative.Title != "My Initiative" {
		t.Errorf("expected title 'My Initiative', got %q", result.Initiative.Title)
	}
}

func TestService_Create_DefaultsModeAndNormalizesAcceptanceCriteria(t *testing.T) {
	svc := newTestService(t, nil)

	init, err := svc.Create(CreateRequest{
		Name:               "holistic",
		Title:              "Holistic",
		AcceptanceCriteria: []string{"  system works ", "", "system works", "audit trail preserved"},
	})
	if err != nil {
		t.Fatalf("Create failed: %v", err)
	}
	if init.Mode != "item-level" {
		t.Fatalf("Mode = %q, want item-level", init.Mode)
	}
	want := []string{"system works", "audit trail preserved"}
	if len(init.AcceptanceCriteria) != len(want) {
		t.Fatalf("AcceptanceCriteria = %v, want %v", init.AcceptanceCriteria, want)
	}
	for i := range want {
		if init.AcceptanceCriteria[i] != want[i] {
			t.Errorf("AcceptanceCriteria[%d] = %q, want %q", i, init.AcceptanceCriteria[i], want[i])
		}
	}
}

func TestService_Create_DuplicateName(t *testing.T) {
	svc := newTestService(t, nil)

	_, err := svc.Create(CreateRequest{Name: "dup", Title: "First"})
	if err != nil {
		t.Fatalf("first Create failed: %v", err)
	}

	_, err = svc.Create(CreateRequest{Name: "dup", Title: "Second"})
	if err == nil {
		t.Fatal("expected error for duplicate name")
	}
}

func TestService_Create_MissingName(t *testing.T) {
	svc := newTestService(t, nil)

	_, err := svc.Create(CreateRequest{Name: "", Title: "No Name"})
	if err == nil {
		t.Fatal("expected error for missing name")
	}
}

func TestService_Create_MissingTitle(t *testing.T) {
	svc := newTestService(t, nil)

	_, err := svc.Create(CreateRequest{Name: "test", Title: ""})
	if err == nil {
		t.Fatal("expected error for missing title")
	}
}

func TestService_Update(t *testing.T) {
	svc := newTestService(t, nil)

	_, err := svc.Create(CreateRequest{Name: "upd", Title: "Original"})
	if err != nil {
		t.Fatalf("Create failed: %v", err)
	}

	// Update user-settable fields. Status is explicitly re-asserted as
	// "active" (a no-op transition) to prove active→active is allowed.
	updated, err := svc.Update("upd", UpdateRequest{
		Title:  strPtr("Updated"),
		Status: strPtr("active"),
		Items:  slicePtr([]string{"fix/bar"}),
	})
	if err != nil {
		t.Fatalf("Update failed: %v", err)
	}
	if updated.Title != "Updated" {
		t.Errorf("expected title Updated, got %q", updated.Title)
	}
	if updated.Status != InitiativeStatusActive {
		t.Errorf("expected status to remain active, got %q", updated.Status)
	}
}

// TestService_Update_RejectsTerminalStatus guards the invariant documented in
// internal/initiativereview/doc.go: "Nothing else may write a terminal
// initiative status." PATCH is the one place a user could otherwise bypass
// review-decide.
func TestService_Update_RejectsTerminalStatus(t *testing.T) {
	svc := newTestService(t, nil)

	if _, err := svc.Create(CreateRequest{Name: "term", Title: "Terminal Guard"}); err != nil {
		t.Fatalf("Create failed: %v", err)
	}

	for _, status := range []string{
		InitiativeStatusCompleted,
		InitiativeStatusFailed,
		InitiativeStatusNeedsFollowup,
		InitiativeStatusInReview,
		InitiativeStatusReviewPending,
	} {
		_, err := svc.Update("term", UpdateRequest{Status: strPtr(status)})
		if err == nil {
			t.Errorf("expected PATCH to %q to be rejected, got nil", status)
			continue
		}
		if !errors.Is(err, ErrValidation) {
			t.Errorf("PATCH to %q: expected ErrValidation, got %v", status, err)
		}
	}
}

// TestService_Update_RejectsPatchDuringReview ensures that once
// initiativereview has flipped an initiative into a review-phase status,
// PATCH cannot drag it back without going through review-decide.
func TestService_Update_RejectsPatchDuringReview(t *testing.T) {
	svc := newTestService(t, nil)

	if _, err := svc.Create(CreateRequest{Name: "during", Title: "Mid-Review"}); err != nil {
		t.Fatalf("Create failed: %v", err)
	}
	// Simulate initiativereview writing the status directly via the store
	// (its audit-trail bypass path).
	init, err := svc.store.Load("during")
	if err != nil {
		t.Fatalf("Load failed: %v", err)
	}
	init.Status = InitiativeStatusReviewPending
	if err := svc.store.Save(init); err != nil {
		t.Fatalf("Save failed: %v", err)
	}

	_, err = svc.Update("during", UpdateRequest{Status: strPtr(InitiativeStatusActive)})
	if err == nil {
		t.Fatal("expected PATCH during review to be rejected")
	}
	if !errors.Is(err, ErrValidation) {
		t.Errorf("expected ErrValidation, got %v", err)
	}
}

func TestService_Update_InvalidStatus(t *testing.T) {
	svc := newTestService(t, nil)

	_, err := svc.Create(CreateRequest{Name: "test", Title: "Test"})
	if err != nil {
		t.Fatalf("Create failed: %v", err)
	}

	_, err = svc.Update("test", UpdateRequest{Title: strPtr("Test"), Status: strPtr("invalid")})
	if err == nil {
		t.Fatal("expected error for invalid status")
	}
}

func TestService_Update_NotFound(t *testing.T) {
	svc := newTestService(t, nil)

	_, err := svc.Update("missing", UpdateRequest{Title: strPtr("Test"), Status: strPtr("active")})
	if err == nil {
		t.Fatal("expected error for missing initiative")
	}
}

func TestService_Update_Partial(t *testing.T) {
	svc := newTestService(t, nil)

	_, err := svc.Create(CreateRequest{Name: "partial", Title: "Original", Description: "keep me"})
	if err != nil {
		t.Fatalf("Create failed: %v", err)
	}

	// Partial update on a single field (priority); all others must be
	// preserved including description and status.
	updated, err := svc.Update("partial", UpdateRequest{Priority: intPtr(5)})
	if err != nil {
		t.Fatalf("Update failed: %v", err)
	}
	if updated.Title != "Original" {
		t.Errorf("expected title to remain Original, got %q", updated.Title)
	}
	if updated.Status != InitiativeStatusActive {
		t.Errorf("expected status to remain active, got %q", updated.Status)
	}
	if updated.Description != "keep me" {
		t.Errorf("expected description to remain unchanged, got %q", updated.Description)
	}
	if updated.Priority != 5 {
		t.Errorf("expected priority 5, got %d", updated.Priority)
	}
}

func TestService_Delete(t *testing.T) {
	svc := newTestService(t, nil)

	_, err := svc.Create(CreateRequest{Name: "del", Title: "Delete Me"})
	if err != nil {
		t.Fatalf("Create failed: %v", err)
	}

	if err := svc.Delete("del"); err != nil {
		t.Fatalf("Delete failed: %v", err)
	}

	_, err = svc.Get("del")
	if err == nil {
		t.Fatal("expected error after delete")
	}

	// Idempotent.
	if err := svc.Delete("del"); err != nil {
		t.Fatalf("second Delete should succeed: %v", err)
	}
}

func TestService_List(t *testing.T) {
	svc := newTestService(t, nil)

	items, err := svc.List()
	if err != nil {
		t.Fatalf("List failed: %v", err)
	}
	if len(items) != 0 {
		t.Errorf("expected 0 items, got %d", len(items))
	}

	_, _ = svc.Create(CreateRequest{Name: "b", Title: "B"})
	_, _ = svc.Create(CreateRequest{Name: "a", Title: "A"})

	items, err = svc.List()
	if err != nil {
		t.Fatalf("List failed: %v", err)
	}
	if len(items) != 2 {
		t.Fatalf("expected 2 items, got %d", len(items))
	}
	if items[0].Initiative.Name != "a" {
		t.Errorf("expected first item 'a', got %q", items[0].Initiative.Name)
	}
}

func TestService_ComputeRollup(t *testing.T) {
	archivedAt := "2026-01-02T00:00:00Z"
	loader := map[string]backlog.BacklogItem{
		"idea/completed": {Status: backlog.StatusCompleted},
		"fix/failed":     {Status: backlog.StatusFailed},
		"idea/wip":       {Status: backlog.StatusInProgress},
		"fix/queued":     {Status: backlog.StatusQueued},
		"idea/backlog":   {Status: backlog.StatusBacklog},
		"idea/ready":     {Status: backlog.StatusReady},
		"idea/archived-backlog": {
			Status:     backlog.StatusBacklog,
			ArchivedAt: &archivedAt,
		},
		"fix/archived-wip": {
			Status:     backlog.StatusResearching,
			ArchivedAt: &archivedAt,
		},
		"execute/archived-completed": {
			Status:     backlog.StatusCompleted,
			ArchivedAt: &archivedAt,
		},
	}
	svc := newTestService(t, loader)

	init := &Initiative{
		Items: []string{
			"idea/completed",
			"fix/failed",
			"idea/wip",
			"fix/queued",
			"idea/backlog",
			"idea/ready",
			"idea/archived-backlog",
			"fix/archived-wip",
			"execute/archived-completed",
			"idea/missing", // not in loader
			"invalid-ref",  // invalid format
		},
	}

	rollup, err := svc.ComputeRollup(init)
	if err != nil {
		t.Fatalf("ComputeRollup failed: %v", err)
	}

	if rollup.Total != 11 {
		t.Errorf("expected total 11, got %d", rollup.Total)
	}
	if rollup.Completed != 2 {
		t.Errorf("expected completed 2 (including archived completed), got %d", rollup.Completed)
	}
	if rollup.Failed != 1 {
		t.Errorf("expected failed 1, got %d", rollup.Failed)
	}
	if rollup.InProgress != 2 {
		t.Errorf("expected in_progress 2, got %d", rollup.InProgress)
	}
	// backlog + ready + missing + invalid = 4 pending; archived non-completed
	// items are terminal and should not inflate active progress buckets.
	if rollup.Pending != 4 {
		t.Errorf("expected pending 4, got %d", rollup.Pending)
	}
	if rollup.Archived != 3 {
		t.Errorf("expected archived 3, got %d", rollup.Archived)
	}
}

func TestService_AddItems(t *testing.T) {
	svc := newTestService(t, nil)

	_, err := svc.Create(CreateRequest{
		Name:  "add-test",
		Title: "Add Items Test",
		Items: []string{"idea/foo"},
	})
	if err != nil {
		t.Fatalf("Create failed: %v", err)
	}

	if err := svc.AddItems("add-test", []string{"fix/bar", "idea/foo"}); err != nil {
		t.Fatalf("AddItems failed: %v", err)
	}

	result, err := svc.Get("add-test")
	if err != nil {
		t.Fatalf("Get failed: %v", err)
	}
	if len(result.Initiative.Items) != 2 {
		t.Errorf("expected 2 items (deduplicated), got %d: %v", len(result.Initiative.Items), result.Initiative.Items)
	}
}

func TestService_AddItems_InvalidFormat(t *testing.T) {
	svc := newTestService(t, nil)

	_, err := svc.Create(CreateRequest{
		Name:  "validate-test",
		Title: "Validate Test",
	})
	if err != nil {
		t.Fatalf("Create failed: %v", err)
	}

	tests := []struct {
		name  string
		items []string
	}{
		{"no-slash", []string{"bad-ref"}},
		{"empty-kind", []string{"/name"}},
		{"empty-name", []string{"kind/"}},
		{"spaces-only-kind", []string{" /name"}},
		{"spaces-only-name", []string{"kind/ "}},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := svc.AddItems("validate-test", tt.items)
			if err == nil {
				t.Error("expected error for invalid item reference")
			}
			if !strings.Contains(err.Error(), "invalid item reference") {
				t.Errorf("expected 'invalid item reference' in error, got: %v", err)
			}
		})
	}
}

func TestService_RemoveItems(t *testing.T) {
	svc := newTestService(t, nil)

	_, err := svc.Create(CreateRequest{
		Name:  "rm-test",
		Title: "Remove Items Test",
		Items: []string{"idea/foo", "fix/bar", "execute/baz"},
	})
	if err != nil {
		t.Fatalf("Create failed: %v", err)
	}

	if err := svc.RemoveItems("rm-test", []string{"fix/bar"}); err != nil {
		t.Fatalf("RemoveItems failed: %v", err)
	}

	result, err := svc.Get("rm-test")
	if err != nil {
		t.Fatalf("Get failed: %v", err)
	}
	if len(result.Initiative.Items) != 2 {
		t.Errorf("expected 2 items after removal, got %d", len(result.Initiative.Items))
	}
}

func TestService_ComputeRollup_ItemDeletedFromDisk(t *testing.T) {
	// Simulate an item that was in the initiative but has been deleted from backlog.
	// Only "idea/exists" is in the loader; "fix/deleted" returns an error.
	loader := map[string]backlog.BacklogItem{
		"idea/exists": {Status: backlog.StatusInProgress},
	}
	svc := newTestService(t, loader)

	init := &Initiative{
		Items: []string{
			"idea/exists",
			"fix/deleted", // not in loader — simulates deleted item
		},
	}

	rollup, err := svc.ComputeRollup(init)
	if err != nil {
		t.Fatalf("ComputeRollup should not fail for deleted items: %v", err)
	}

	if rollup.Total != 2 {
		t.Errorf("expected total 2, got %d", rollup.Total)
	}
	if rollup.InProgress != 1 {
		t.Errorf("expected in_progress 1, got %d", rollup.InProgress)
	}
	// Deleted item should be counted as pending (graceful degradation).
	if rollup.Pending != 1 {
		t.Errorf("expected pending 1 (deleted item counted as pending), got %d", rollup.Pending)
	}
}

func TestService_RecreateInitiative_MovesMembersAndPreservesLineage(t *testing.T) {
	loaderItems := map[string]backlog.BacklogItem{
		"execute/first":  {Kind: backlog.KindExecute, Name: "first", Initiative: "stale-init", Status: backlog.StatusBacklog},
		"execute/second": {Kind: backlog.KindExecute, Name: "second", Initiative: "stale-init", Status: backlog.StatusReady},
	}
	svc := newTestService(t, loaderItems)
	if _, err := svc.Create(CreateRequest{Name: "stale-init", Title: "Stale initiative", Items: []string{"execute/first", "execute/second"}}); err != nil {
		t.Fatalf("create source: %v", err)
	}
	if err := svc.RecreateInitiative(context.Background(), "stale-init"); err != nil {
		t.Fatalf("RecreateInitiative: %v", err)
	}
	source, err := svc.store.Load("stale-init")
	if err != nil || source.ArchivedAt == nil {
		t.Fatalf("source after recreate = %#v, %v; want archived", source, err)
	}
	clone, err := svc.store.Load("stale-init-recreated")
	if err != nil {
		t.Fatalf("load clone: %v", err)
	}
	if clone.Status != InitiativeStatusActive || clone.SpawnedFrom != "stale-init" {
		t.Fatalf("clone = %#v, want active lineage-preserving successor", clone)
	}
	for ref, item := range loaderItems {
		if item.Initiative != clone.Name {
			t.Errorf("%s initiative = %q, want %q", ref, item.Initiative, clone.Name)
		}
	}
}
