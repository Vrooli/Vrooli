package initiatives

import (
	"fmt"
	"strings"
	"swarm-manager/internal/backlog"
	"testing"
)

func strPtr(value string) *string {
	return &value
}

func slicePtr(values []string) *[]string {
	return &values
}

// mockBacklogLoader is a test double implementing BacklogLoader.
type mockBacklogLoader struct {
	items map[string]backlog.BacklogItem // key: "kind/name"
}

func (m *mockBacklogLoader) LoadItem(kind backlog.BacklogKind, name string) (backlog.BacklogItem, error) {
	key := string(kind) + "/" + name
	item, ok := m.items[key]
	if !ok {
		return backlog.BacklogItem{}, fmt.Errorf("item %q not found", key)
	}
	return item, nil
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

	result, err := svc.Get("my-init")
	if err != nil {
		t.Fatalf("Get failed: %v", err)
	}
	if result.Initiative.Title != "My Initiative" {
		t.Errorf("expected title 'My Initiative', got %q", result.Initiative.Title)
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

	updated, err := svc.Update("upd", UpdateRequest{
		Title:  strPtr("Updated"),
		Status: strPtr("completed"),
		Items:  slicePtr([]string{"fix/bar"}),
	})
	if err != nil {
		t.Fatalf("Update failed: %v", err)
	}
	if updated.Title != "Updated" {
		t.Errorf("expected title Updated, got %q", updated.Title)
	}
	if updated.Status != "completed" {
		t.Errorf("expected status completed, got %q", updated.Status)
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

	updated, err := svc.Update("partial", UpdateRequest{Status: strPtr("completed")})
	if err != nil {
		t.Fatalf("Update failed: %v", err)
	}
	if updated.Title != "Original" {
		t.Errorf("expected title to remain Original, got %q", updated.Title)
	}
	if updated.Status != "completed" {
		t.Errorf("expected status completed, got %q", updated.Status)
	}
	if updated.Description != "keep me" {
		t.Errorf("expected description to remain unchanged, got %q", updated.Description)
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
	loader := map[string]backlog.BacklogItem{
		"idea/completed": {Status: backlog.StatusCompleted},
		"fix/failed":     {Status: backlog.StatusFailed},
		"idea/wip":       {Status: backlog.StatusInProgress},
		"fix/queued":     {Status: backlog.StatusQueued},
		"idea/backlog":   {Status: backlog.StatusBacklog},
		"idea/ready":     {Status: backlog.StatusReady},
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
			"idea/missing", // not in loader
			"invalid-ref",  // invalid format
		},
	}

	rollup, err := svc.ComputeRollup(init)
	if err != nil {
		t.Fatalf("ComputeRollup failed: %v", err)
	}

	if rollup.Total != 8 {
		t.Errorf("expected total 8, got %d", rollup.Total)
	}
	if rollup.Completed != 1 {
		t.Errorf("expected completed 1, got %d", rollup.Completed)
	}
	if rollup.Failed != 1 {
		t.Errorf("expected failed 1, got %d", rollup.Failed)
	}
	if rollup.InProgress != 2 {
		t.Errorf("expected in_progress 2, got %d", rollup.InProgress)
	}
	// backlog + ready + missing + invalid = 4 pending
	if rollup.Pending != 4 {
		t.Errorf("expected pending 4, got %d", rollup.Pending)
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
