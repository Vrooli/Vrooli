package captures

import (
	"encoding/json"
	"os"
	"path/filepath"
	"testing"

	"swarm-manager/internal/backlog"
)

func TestBacklogItemCreatorAdapter_SaveItem(t *testing.T) {
	rootDir := t.TempDir()
	store := backlog.NewFileStore(rootDir)
	adapter := NewBacklogItemCreatorAdapter(store)

	// Create the kind directory so SaveItem can create the item dir.
	kindDir := store.KindDir(backlog.KindExecute)
	if err := os.MkdirAll(kindDir, 0o755); err != nil {
		t.Fatal(err)
	}

	err := adapter.SaveItem(BacklogItemDraft{Kind: "execute", Name: "my-task", Title: "My Task", Description: "A test task", Priority: 7, Tags: []string{"ops", "infra"}, SpawnedFrom: "cap-test"})
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
	if item.Priority != 7 {
		t.Errorf("expected priority 7, got %d", item.Priority)
	}
	if len(item.Tags) != 2 || item.Tags[0] != "ops" || item.Tags[1] != "infra" {
		t.Errorf("unexpected tags: %v", item.Tags)
	}
	if item.Kind != backlog.KindExecute {
		t.Errorf("expected kind execute, got %q", item.Kind)
	}
	if item.SpawnedFrom != "cap-test" {
		t.Errorf("expected spawned_from cap-test, got %q", item.SpawnedFrom)
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

	err := adapter.SaveItem(BacklogItemDraft{Kind: "idea", Name: "dup-item", Title: "First", Description: "first desc", Priority: 5})
	if err != nil {
		t.Fatalf("first SaveItem: %v", err)
	}

	err = adapter.SaveItem(BacklogItemDraft{Kind: "idea", Name: "dup-item", Title: "Second", Description: "second desc", Priority: 5})
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

	err := adapter.SaveItem(BacklogItemDraft{Kind: "fix", Name: "ts-test", Title: "Timestamp Test", Description: "desc", Priority: 5})
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
