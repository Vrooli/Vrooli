package captures

import (
	"encoding/json"
	"os"
	"path/filepath"
	"testing"

	"swarm-manager/internal/backlog"
)

func TestBacklogItemCreatorAdapter_ItemDir(t *testing.T) {
	rootDir := t.TempDir()
	store := backlog.NewFileStore(rootDir)
	adapter := NewBacklogItemCreatorAdapter(store)

	dir := adapter.ItemDir("execute", "my-task")
	expected := store.ItemDir(backlog.KindExecute, "my-task")
	if dir != expected {
		t.Errorf("expected %q, got %q", expected, dir)
	}
}

func TestBacklogItemCreatorAdapter_ItemDir_AllKinds(t *testing.T) {
	rootDir := t.TempDir()
	store := backlog.NewFileStore(rootDir)
	adapter := NewBacklogItemCreatorAdapter(store)

	kinds := []string{"idea", "research", "fix", "execute", "chore"}
	for _, kind := range kinds {
		dir := adapter.ItemDir(kind, "test-item")
		expected := store.ItemDir(backlog.BacklogKind(kind), "test-item")
		if dir != expected {
			t.Errorf("kind %q: expected %q, got %q", kind, expected, dir)
		}
	}
}

func TestBacklogItemCreatorAdapter_SaveItem(t *testing.T) {
	rootDir := t.TempDir()
	store := backlog.NewFileStore(rootDir)
	adapter := NewBacklogItemCreatorAdapter(store)

	// Create the kind directory so SaveItem can create the item dir.
	kindDir := store.KindDir(backlog.KindExecute)
	if err := os.MkdirAll(kindDir, 0o755); err != nil {
		t.Fatal(err)
	}

	err := adapter.SaveItem("execute", "my-task", "My Task", "A test task", []string{"ops", "infra"})
	if err != nil {
		t.Fatalf("SaveItem: %v", err)
	}

	// Verify the item was saved by loading it from the store.
	item, err := store.LoadItem(backlog.KindExecute, "my-task")
	if err != nil {
		t.Fatalf("LoadItem: %v", err)
	}
	if item.Name != "my-task" {
		t.Errorf("expected name my-task, got %q", item.Name)
	}
	if item.Title != "My Task" {
		t.Errorf("expected title 'My Task', got %q", item.Title)
	}
	if item.Description != "A test task" {
		t.Errorf("expected description 'A test task', got %q", item.Description)
	}
	if item.Status != backlog.StatusBacklog {
		t.Errorf("expected status backlog, got %q", item.Status)
	}
	if item.Priority != 5 {
		t.Errorf("expected priority 5, got %d", item.Priority)
	}
	if len(item.Tags) != 2 || item.Tags[0] != "ops" || item.Tags[1] != "infra" {
		t.Errorf("unexpected tags: %v", item.Tags)
	}
	if item.Kind != backlog.KindExecute {
		t.Errorf("expected kind execute, got %q", item.Kind)
	}
}

func TestBacklogItemCreatorAdapter_SaveItem_DuplicateReturnsError(t *testing.T) {
	rootDir := t.TempDir()
	store := backlog.NewFileStore(rootDir)
	adapter := NewBacklogItemCreatorAdapter(store)

	kindDir := store.KindDir(backlog.KindIdea)
	if err := os.MkdirAll(kindDir, 0o755); err != nil {
		t.Fatal(err)
	}

	err := adapter.SaveItem("idea", "dup-item", "First", "first desc", nil)
	if err != nil {
		t.Fatalf("first SaveItem: %v", err)
	}

	err = adapter.SaveItem("idea", "dup-item", "Second", "second desc", nil)
	if err == nil {
		t.Fatal("expected error for duplicate item")
	}
}

func TestBacklogItemCreatorAdapter_SaveItem_SetsTimestamps(t *testing.T) {
	rootDir := t.TempDir()
	store := backlog.NewFileStore(rootDir)
	adapter := NewBacklogItemCreatorAdapter(store)

	kindDir := store.KindDir(backlog.KindFix)
	if err := os.MkdirAll(kindDir, 0o755); err != nil {
		t.Fatal(err)
	}

	err := adapter.SaveItem("fix", "ts-test", "Timestamp Test", "desc", nil)
	if err != nil {
		t.Fatalf("SaveItem: %v", err)
	}

	// Read the raw spec.json to verify timestamps were set.
	specPath := filepath.Join(store.ItemDir(backlog.KindFix, "ts-test"), "spec.json")
	data, err := os.ReadFile(specPath)
	if err != nil {
		t.Fatalf("ReadFile: %v", err)
	}
	var raw map[string]any
	if err := json.Unmarshal(data, &raw); err != nil {
		t.Fatalf("Unmarshal: %v", err)
	}
	if raw["created"] == nil || raw["created"] == "" {
		t.Error("expected created timestamp to be set")
	}
	if raw["updated"] == nil || raw["updated"] == "" {
		t.Error("expected updated timestamp to be set")
	}
}
