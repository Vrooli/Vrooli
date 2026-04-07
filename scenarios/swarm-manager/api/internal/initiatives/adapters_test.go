package initiatives

import (
	"testing"

	"swarm-manager/internal/backlog"
)

func TestBacklogAssignerAdapter_Get(t *testing.T) {
	svc := newTestService(t, nil)

	_, err := svc.Create(CreateRequest{
		Name:        "test-init",
		Title:       "Test Initiative",
		Description: "desc",
		Items:       []string{"idea/foo", "fix/bar"},
	})
	if err != nil {
		t.Fatalf("Create: %v", err)
	}

	adapter := NewBacklogAssignerAdapter(svc)
	snap, err := adapter.Get("test-init")
	if err != nil {
		t.Fatalf("Get: %v", err)
	}

	if snap.Name != "test-init" {
		t.Errorf("expected name test-init, got %q", snap.Name)
	}
	if snap.Title != "Test Initiative" {
		t.Errorf("expected title 'Test Initiative', got %q", snap.Title)
	}
	if snap.Description != "desc" {
		t.Errorf("expected description 'desc', got %q", snap.Description)
	}
	if snap.Status != "active" {
		t.Errorf("expected status active, got %q", snap.Status)
	}
	if len(snap.Items) != 2 {
		t.Fatalf("expected 2 items, got %d", len(snap.Items))
	}
	if snap.Items[0] != "idea/foo" || snap.Items[1] != "fix/bar" {
		t.Errorf("unexpected items: %v", snap.Items)
	}
}

func TestBacklogAssignerAdapter_Get_NotFound(t *testing.T) {
	svc := newTestService(t, nil)
	adapter := NewBacklogAssignerAdapter(svc)

	_, err := adapter.Get("missing")
	if err == nil {
		t.Fatal("expected error for missing initiative")
	}
}

func TestBacklogAssignerAdapter_Get_ItemsCopied(t *testing.T) {
	svc := newTestService(t, nil)

	_, err := svc.Create(CreateRequest{
		Name:  "copy-test",
		Title: "Copy Test",
		Items: []string{"idea/a"},
	})
	if err != nil {
		t.Fatalf("Create: %v", err)
	}

	adapter := NewBacklogAssignerAdapter(svc)
	snap, err := adapter.Get("copy-test")
	if err != nil {
		t.Fatalf("Get: %v", err)
	}

	// Mutate the returned slice to verify it's a copy.
	snap.Items[0] = "idea/mutated"

	snap2, err := adapter.Get("copy-test")
	if err != nil {
		t.Fatalf("Get again: %v", err)
	}
	if snap2.Items[0] != "idea/a" {
		t.Errorf("mutation leaked: expected idea/a, got %q", snap2.Items[0])
	}
}

func TestBacklogAssignerAdapter_Create(t *testing.T) {
	svc := newTestService(t, nil)
	adapter := NewBacklogAssignerAdapter(svc)

	err := adapter.Create(backlog.InitiativeSpec{
		Name:        "new-init",
		Title:       "New Initiative",
		Description: "new desc",
		Status:      "active",
	})
	if err != nil {
		t.Fatalf("Create: %v", err)
	}

	// Verify it was created via the service.
	result, err := svc.Get("new-init")
	if err != nil {
		t.Fatalf("Get after Create: %v", err)
	}
	if result.Initiative.Title != "New Initiative" {
		t.Errorf("expected title 'New Initiative', got %q", result.Initiative.Title)
	}
	if result.Initiative.Description != "new desc" {
		t.Errorf("expected description 'new desc', got %q", result.Initiative.Description)
	}
}

func TestBacklogAssignerAdapter_Update(t *testing.T) {
	svc := newTestService(t, nil)

	_, err := svc.Create(CreateRequest{
		Name:        "upd-init",
		Title:       "Original",
		Description: "orig desc",
	})
	if err != nil {
		t.Fatalf("Create: %v", err)
	}

	adapter := NewBacklogAssignerAdapter(svc)
	err = adapter.Update(backlog.InitiativeSpec{
		Name:        "upd-init",
		Title:       "Updated Title",
		Description: "updated desc",
		Status:      "completed",
	})
	if err != nil {
		t.Fatalf("Update: %v", err)
	}

	result, err := svc.Get("upd-init")
	if err != nil {
		t.Fatalf("Get after Update: %v", err)
	}
	if result.Initiative.Title != "Updated Title" {
		t.Errorf("expected title 'Updated Title', got %q", result.Initiative.Title)
	}
	if result.Initiative.Description != "updated desc" {
		t.Errorf("expected description 'updated desc', got %q", result.Initiative.Description)
	}
	if result.Initiative.Status != "completed" {
		t.Errorf("expected status completed, got %q", result.Initiative.Status)
	}
}

func TestBacklogAssignerAdapter_Replace(t *testing.T) {
	svc := newTestService(t, nil)

	_, err := svc.Create(CreateRequest{
		Name:  "repl-init",
		Title: "Original",
		Items: []string{"idea/a"},
	})
	if err != nil {
		t.Fatalf("Create: %v", err)
	}

	adapter := NewBacklogAssignerAdapter(svc)
	err = adapter.Replace(backlog.InitiativeSnapshot{
		Name:        "repl-init",
		Title:       "Replaced",
		Description: "replaced desc",
		Status:      "completed",
		Items:       []string{"fix/x", "fix/y"},
	})
	if err != nil {
		t.Fatalf("Replace: %v", err)
	}

	result, err := svc.Get("repl-init")
	if err != nil {
		t.Fatalf("Get after Replace: %v", err)
	}
	if result.Initiative.Title != "Replaced" {
		t.Errorf("expected title Replaced, got %q", result.Initiative.Title)
	}
	if result.Initiative.Description != "replaced desc" {
		t.Errorf("expected description 'replaced desc', got %q", result.Initiative.Description)
	}
	if result.Initiative.Status != "completed" {
		t.Errorf("expected status completed, got %q", result.Initiative.Status)
	}
	if len(result.Initiative.Items) != 2 {
		t.Fatalf("expected 2 items, got %d", len(result.Initiative.Items))
	}
	if result.Initiative.Items[0] != "fix/x" || result.Initiative.Items[1] != "fix/y" {
		t.Errorf("unexpected items: %v", result.Initiative.Items)
	}
}

func TestBacklogAssignerAdapter_Replace_ItemsCopied(t *testing.T) {
	svc := newTestService(t, nil)

	_, err := svc.Create(CreateRequest{
		Name:  "copy-repl",
		Title: "Copy Replace",
	})
	if err != nil {
		t.Fatalf("Create: %v", err)
	}

	items := []string{"idea/a", "idea/b"}
	adapter := NewBacklogAssignerAdapter(svc)
	err = adapter.Replace(backlog.InitiativeSnapshot{
		Name:   "copy-repl",
		Title:  "Copy Replace",
		Status: "active",
		Items:  items,
	})
	if err != nil {
		t.Fatalf("Replace: %v", err)
	}

	// Mutate original slice.
	items[0] = "idea/mutated"

	result, err := svc.Get("copy-repl")
	if err != nil {
		t.Fatalf("Get: %v", err)
	}
	if result.Initiative.Items[0] != "idea/a" {
		t.Errorf("mutation leaked into store: expected idea/a, got %q", result.Initiative.Items[0])
	}
}

func TestBacklogAssignerAdapter_Delete(t *testing.T) {
	svc := newTestService(t, nil)

	_, err := svc.Create(CreateRequest{
		Name:  "del-init",
		Title: "Delete Me",
	})
	if err != nil {
		t.Fatalf("Create: %v", err)
	}

	adapter := NewBacklogAssignerAdapter(svc)
	if err := adapter.Delete("del-init"); err != nil {
		t.Fatalf("Delete: %v", err)
	}

	_, err = svc.Get("del-init")
	if err == nil {
		t.Fatal("expected error after delete")
	}
}

func TestBacklogAssignerAdapter_AddItems(t *testing.T) {
	svc := newTestService(t, nil)

	_, err := svc.Create(CreateRequest{
		Name:  "add-init",
		Title: "Add Items",
		Items: []string{"idea/existing"},
	})
	if err != nil {
		t.Fatalf("Create: %v", err)
	}

	adapter := NewBacklogAssignerAdapter(svc)
	if err := adapter.AddItems("add-init", []string{"fix/new"}); err != nil {
		t.Fatalf("AddItems: %v", err)
	}

	result, err := svc.Get("add-init")
	if err != nil {
		t.Fatalf("Get after AddItems: %v", err)
	}
	if len(result.Initiative.Items) != 2 {
		t.Errorf("expected 2 items, got %d", len(result.Initiative.Items))
	}
}
