package backlog

import (
	"encoding/json"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"swarm-manager/internal/testutil"
)

// setupTestStore creates a FileStore with a temp root and all kind directories.
func setupTestStore(t *testing.T) (*FileStore, string) {
	t.Helper()
	rootDir := t.TempDir()
	for _, dir := range backlogKindDirs {
		testutil.MakeDir(t, filepath.Join(rootDir, dir))
	}
	return NewFileStore(rootDir), rootDir
}

// writeSpecJSON writes a spec.json file for a given kind/name.
func writeSpecJSON(t *testing.T, rootDir string, kind BacklogKind, name string, data map[string]any) {
	t.Helper()
	dir := filepath.Join(rootDir, backlogKindDirs[kind], name)
	testutil.WriteJSONFile(t, filepath.Join(dir, "spec.json"), data)
}

func TestStore_NewFileStore(t *testing.T) {
	t.Run("creates store with rootDir", func(t *testing.T) {
		s := NewFileStore("/some/path")
		if s == nil {
			t.Fatal("expected non-nil store")
		}
		if s.rootDir != "/some/path" {
			t.Errorf("expected rootDir /some/path, got %s", s.rootDir)
		}
	})
}

func TestStore_KindDir(t *testing.T) {
	s := NewFileStore("/root")
	tests := []struct {
		kind BacklogKind
		want string
	}{
		{KindIdea, "/root/ideas"},
		{KindResearch, "/root/research"},
		{KindFix, "/root/fix"},
		{KindExecute, "/root/execute"},
		{KindChore, "/root/chore"},
	}
	for _, tt := range tests {
		t.Run(string(tt.kind), func(t *testing.T) {
			got := s.KindDir(tt.kind)
			if got != tt.want {
				t.Errorf("KindDir(%s) = %s, want %s", tt.kind, got, tt.want)
			}
		})
	}
}

func TestStore_ItemDir(t *testing.T) {
	s := NewFileStore("/root")
	tests := []struct {
		kind BacklogKind
		name string
		want string
	}{
		{KindIdea, "my-idea", "/root/ideas/my-idea"},
		{KindFix, "bug-123", "/root/fix/bug-123"},
		{KindResearch, "topic-a", "/root/research/topic-a"},
		{KindExecute, "task-1", "/root/execute/task-1"},
		{KindChore, "cleanup", "/root/chore/cleanup"},
	}
	for _, tt := range tests {
		t.Run(string(tt.kind)+"/"+tt.name, func(t *testing.T) {
			got := s.ItemDir(tt.kind, tt.name)
			if got != tt.want {
				t.Errorf("ItemDir(%s, %s) = %s, want %s", tt.kind, tt.name, got, tt.want)
			}
		})
	}
}

func TestStore_LoadItemFromPath(t *testing.T) {
	t.Run("valid spec.json", func(t *testing.T) {
		store, rootDir := setupTestStore(t)
		writeSpecJSON(t, rootDir, KindIdea, "test-idea", map[string]any{
			"title":       "Test Idea",
			"description": "A test description",
			"status":      "backlog",
			"priority":    3,
			"tags":        []string{"ui", "backend"},
			"created":     "2025-01-01T00:00:00Z",
			"updated":     "2025-01-02T00:00:00Z",
		})

		specPath := filepath.Join(rootDir, "ideas", "test-idea", "spec.json")
		item, err := store.LoadItemFromPath(KindIdea, specPath)
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
		if item.Name != "test-idea" {
			t.Errorf("Name = %s, want test-idea", item.Name)
		}
		if item.Title != "Test Idea" {
			t.Errorf("Title = %s, want Test Idea", item.Title)
		}
		if item.Kind != KindIdea {
			t.Errorf("Kind = %s, want idea", item.Kind)
		}
		if item.Priority != 3 {
			t.Errorf("Priority = %d, want 3", item.Priority)
		}
		if len(item.Tags) != 2 {
			t.Errorf("Tags len = %d, want 2", len(item.Tags))
		}
	})

	t.Run("missing file returns error", func(t *testing.T) {
		store, rootDir := setupTestStore(t)
		specPath := filepath.Join(rootDir, "ideas", "nonexistent", "spec.json")
		_, err := store.LoadItemFromPath(KindIdea, specPath)
		if err == nil {
			t.Fatal("expected error for missing file")
		}
	})

	t.Run("malformed JSON returns error", func(t *testing.T) {
		store, rootDir := setupTestStore(t)
		dir := filepath.Join(rootDir, "ideas", "bad-json")
		testutil.MakeDir(t, dir)
		if err := os.WriteFile(filepath.Join(dir, "spec.json"), []byte("{not valid json}"), 0o644); err != nil {
			t.Fatal(err)
		}
		_, err := store.LoadItemFromPath(KindIdea, filepath.Join(dir, "spec.json"))
		if err == nil {
			t.Fatal("expected error for malformed JSON")
		}
	})

	t.Run("nil tags are normalized to empty slice", func(t *testing.T) {
		store, rootDir := setupTestStore(t)
		writeSpecJSON(t, rootDir, KindFix, "no-tags", map[string]any{
			"title":    "No Tags",
			"status":   "backlog",
			"priority": 5,
			"created":  "2025-01-01T00:00:00Z",
		})
		specPath := filepath.Join(rootDir, "fix", "no-tags", "spec.json")
		item, err := store.LoadItemFromPath(KindFix, specPath)
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
		if item.Tags == nil {
			t.Error("Tags should be non-nil empty slice, got nil")
		}
		if len(item.Tags) != 0 {
			t.Errorf("Tags len = %d, want 0", len(item.Tags))
		}
	})

	t.Run("legacy status 'done' normalized to completed", func(t *testing.T) {
		store, rootDir := setupTestStore(t)
		writeSpecJSON(t, rootDir, KindIdea, "done-item", map[string]any{
			"title":    "Done Item",
			"status":   "done",
			"priority": 5,
			"created":  "2025-01-01T00:00:00Z",
		})
		specPath := filepath.Join(rootDir, "ideas", "done-item", "spec.json")
		item, err := store.LoadItemFromPath(KindIdea, specPath)
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
		if item.Status != StatusCompleted {
			t.Errorf("Status = %s, want %s", item.Status, StatusCompleted)
		}
	})

	t.Run("legacy status 'complete' normalized to completed", func(t *testing.T) {
		store, rootDir := setupTestStore(t)
		writeSpecJSON(t, rootDir, KindIdea, "complete-item", map[string]any{
			"title":    "Complete Item",
			"status":   "complete",
			"priority": 5,
			"created":  "2025-01-01T00:00:00Z",
		})
		specPath := filepath.Join(rootDir, "ideas", "complete-item", "spec.json")
		item, err := store.LoadItemFromPath(KindIdea, specPath)
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
		if item.Status != StatusCompleted {
			t.Errorf("Status = %s, want %s", item.Status, StatusCompleted)
		}
	})

	t.Run("unknown status normalized to backlog", func(t *testing.T) {
		store, rootDir := setupTestStore(t)
		writeSpecJSON(t, rootDir, KindIdea, "weird-status", map[string]any{
			"title":    "Weird Status",
			"status":   "banana",
			"priority": 5,
			"created":  "2025-01-01T00:00:00Z",
		})
		specPath := filepath.Join(rootDir, "ideas", "weird-status", "spec.json")
		item, err := store.LoadItemFromPath(KindIdea, specPath)
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
		if item.Status != StatusBacklog {
			t.Errorf("Status = %s, want %s", item.Status, StatusBacklog)
		}
	})

	t.Run("priority clamped to valid range", func(t *testing.T) {
		store, rootDir := setupTestStore(t)

		// Priority 0 should become 5
		writeSpecJSON(t, rootDir, KindIdea, "low-pri", map[string]any{
			"title":    "Low Priority",
			"status":   "backlog",
			"priority": 0,
			"created":  "2025-01-01T00:00:00Z",
		})
		specPath := filepath.Join(rootDir, "ideas", "low-pri", "spec.json")
		item, err := store.LoadItemFromPath(KindIdea, specPath)
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
		if item.Priority != 5 {
			t.Errorf("Priority = %d, want 5 (default for <1)", item.Priority)
		}

		// Priority 99 should become 10
		writeSpecJSON(t, rootDir, KindIdea, "high-pri", map[string]any{
			"title":    "High Priority",
			"status":   "backlog",
			"priority": 99,
			"created":  "2025-01-01T00:00:00Z",
		})
		specPath = filepath.Join(rootDir, "ideas", "high-pri", "spec.json")
		item, err = store.LoadItemFromPath(KindIdea, specPath)
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
		if item.Priority != 10 {
			t.Errorf("Priority = %d, want 10 (clamped from 99)", item.Priority)
		}
	})

	t.Run("missing created backfilled from updated", func(t *testing.T) {
		store, rootDir := setupTestStore(t)
		writeSpecJSON(t, rootDir, KindIdea, "no-created", map[string]any{
			"title":    "No Created",
			"status":   "backlog",
			"priority": 5,
			"updated":  "2025-06-15T12:00:00Z",
		})
		specPath := filepath.Join(rootDir, "ideas", "no-created", "spec.json")
		item, err := store.LoadItemFromPath(KindIdea, specPath)
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
		if item.Created != "2025-06-15T12:00:00Z" {
			t.Errorf("Created = %s, want 2025-06-15T12:00:00Z", item.Created)
		}
	})

	t.Run("missing created backfilled from file mtime when updated also empty", func(t *testing.T) {
		store, rootDir := setupTestStore(t)
		writeSpecJSON(t, rootDir, KindIdea, "no-timestamps", map[string]any{
			"title":    "No Timestamps",
			"status":   "backlog",
			"priority": 5,
		})
		specPath := filepath.Join(rootDir, "ideas", "no-timestamps", "spec.json")
		item, err := store.LoadItemFromPath(KindIdea, specPath)
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
		if strings.TrimSpace(item.Created) == "" {
			t.Error("Created should be backfilled from mtime, got empty")
		}
	})

	t.Run("name derived from parent directory", func(t *testing.T) {
		store, rootDir := setupTestStore(t)
		writeSpecJSON(t, rootDir, KindChore, "my-chore-item", map[string]any{
			"title":    "Some Title",
			"name":     "wrong-name",
			"status":   "backlog",
			"priority": 5,
			"created":  "2025-01-01T00:00:00Z",
		})
		specPath := filepath.Join(rootDir, "chore", "my-chore-item", "spec.json")
		item, err := store.LoadItemFromPath(KindChore, specPath)
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
		if item.Name != "my-chore-item" {
			t.Errorf("Name = %s, want my-chore-item (derived from directory)", item.Name)
		}
	})
}

func TestStore_LoadItem(t *testing.T) {
	t.Run("valid item", func(t *testing.T) {
		store, rootDir := setupTestStore(t)
		writeSpecJSON(t, rootDir, KindExecute, "task-1", map[string]any{
			"title":    "Execute Task",
			"status":   "ready",
			"priority": 7,
			"tags":     []string{"deploy"},
			"created":  "2025-01-01T00:00:00Z",
		})
		item, err := store.LoadItem(KindExecute, "task-1")
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
		if item.Title != "Execute Task" {
			t.Errorf("Title = %s, want Execute Task", item.Title)
		}
		if item.Kind != KindExecute {
			t.Errorf("Kind = %s, want execute", item.Kind)
		}
	})

	t.Run("missing item returns error", func(t *testing.T) {
		store, _ := setupTestStore(t)
		_, err := store.LoadItem(KindIdea, "does-not-exist")
		if err == nil {
			t.Fatal("expected error for missing item")
		}
	})

	t.Run("invalid kind returns empty path lookup error", func(t *testing.T) {
		store, _ := setupTestStore(t)
		_, err := store.LoadItem(BacklogKind("invalid"), "some-item")
		if err == nil {
			t.Fatal("expected error for invalid kind")
		}
	})
}

func TestStore_LoadAll(t *testing.T) {
	t.Run("empty dirs return empty slice", func(t *testing.T) {
		store, _ := setupTestStore(t)
		items, err := store.LoadAll(nil)
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
		if len(items) != 0 {
			t.Errorf("expected 0 items, got %d", len(items))
		}
		// Should be non-nil empty slice
		if items == nil {
			t.Error("expected non-nil empty slice")
		}
	})

	t.Run("loads items from single kind", func(t *testing.T) {
		store, rootDir := setupTestStore(t)
		writeSpecJSON(t, rootDir, KindIdea, "idea-1", map[string]any{
			"title":    "Idea 1",
			"status":   "backlog",
			"priority": 5,
			"created":  "2025-01-01T00:00:00Z",
		})
		writeSpecJSON(t, rootDir, KindIdea, "idea-2", map[string]any{
			"title":    "Idea 2",
			"status":   "ready",
			"priority": 3,
			"created":  "2025-01-02T00:00:00Z",
		})

		items, err := store.LoadAll([]BacklogKind{KindIdea})
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
		if len(items) != 2 {
			t.Errorf("expected 2 items, got %d", len(items))
		}
		for _, item := range items {
			if item.Kind != KindIdea {
				t.Errorf("expected kind idea, got %s", item.Kind)
			}
		}
	})

	t.Run("loads items from multiple kinds", func(t *testing.T) {
		store, rootDir := setupTestStore(t)
		writeSpecJSON(t, rootDir, KindIdea, "idea-a", map[string]any{
			"title":    "Idea A",
			"status":   "backlog",
			"priority": 5,
			"created":  "2025-01-01T00:00:00Z",
		})
		writeSpecJSON(t, rootDir, KindFix, "fix-a", map[string]any{
			"title":    "Fix A",
			"status":   "backlog",
			"priority": 8,
			"created":  "2025-01-01T00:00:00Z",
		})
		writeSpecJSON(t, rootDir, KindChore, "chore-a", map[string]any{
			"title":    "Chore A",
			"status":   "backlog",
			"priority": 2,
			"created":  "2025-01-01T00:00:00Z",
		})

		items, err := store.LoadAll([]BacklogKind{KindIdea, KindFix})
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
		if len(items) != 2 {
			t.Errorf("expected 2 items (idea + fix), got %d", len(items))
		}
	})

	t.Run("nil kinds loads all kinds", func(t *testing.T) {
		store, rootDir := setupTestStore(t)
		writeSpecJSON(t, rootDir, KindIdea, "idea-x", map[string]any{
			"title": "X", "status": "backlog", "priority": 5, "created": "2025-01-01T00:00:00Z",
		})
		writeSpecJSON(t, rootDir, KindFix, "fix-x", map[string]any{
			"title": "X", "status": "backlog", "priority": 5, "created": "2025-01-01T00:00:00Z",
		})
		writeSpecJSON(t, rootDir, KindResearch, "research-x", map[string]any{
			"title": "X", "status": "backlog", "priority": 5, "created": "2025-01-01T00:00:00Z",
		})

		items, err := store.LoadAll(nil)
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
		if len(items) != 3 {
			t.Errorf("expected 3 items, got %d", len(items))
		}
	})

	t.Run("missing kind directory is not an error", func(t *testing.T) {
		rootDir := t.TempDir()
		// Only create the ideas directory, not others
		testutil.MakeDir(t, filepath.Join(rootDir, "ideas"))
		store := NewFileStore(rootDir)

		items, err := store.LoadAll([]BacklogKind{KindIdea, KindFix})
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
		if len(items) != 0 {
			t.Errorf("expected 0 items, got %d", len(items))
		}
	})

	t.Run("directories without spec.json are skipped", func(t *testing.T) {
		store, rootDir := setupTestStore(t)
		// Create a directory without spec.json
		testutil.MakeDir(t, filepath.Join(rootDir, "ideas", "no-spec"))
		// Create one with spec.json
		writeSpecJSON(t, rootDir, KindIdea, "has-spec", map[string]any{
			"title": "Has Spec", "status": "backlog", "priority": 5, "created": "2025-01-01T00:00:00Z",
		})

		items, err := store.LoadAll([]BacklogKind{KindIdea})
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
		if len(items) != 1 {
			t.Errorf("expected 1 item, got %d", len(items))
		}
		if len(items) > 0 && items[0].Name != "has-spec" {
			t.Errorf("expected item name has-spec, got %s", items[0].Name)
		}
	})
}

func TestStore_SaveItem(t *testing.T) {
	t.Run("round-trip save and load", func(t *testing.T) {
		store, rootDir := setupTestStore(t)
		// Pre-create the item directory
		testutil.MakeDir(t, filepath.Join(rootDir, "ideas", "save-test"))

		original := BacklogItem{
			Name:        "save-test",
			Title:       "Save Test",
			Description: "Testing save functionality",
			Status:      StatusReady,
			Priority:    7,
			Tags:        []string{"test", "save"},
			Created:     "2025-01-01T00:00:00Z",
			Updated:     "2025-01-02T00:00:00Z",
			Kind:        KindIdea,
		}
		if err := store.SaveItem(original); err != nil {
			t.Fatalf("SaveItem error: %v", err)
		}

		loaded, err := store.LoadItem(KindIdea, "save-test")
		if err != nil {
			t.Fatalf("LoadItem error: %v", err)
		}
		if loaded.Title != original.Title {
			t.Errorf("Title = %s, want %s", loaded.Title, original.Title)
		}
		if loaded.Description != original.Description {
			t.Errorf("Description = %s, want %s", loaded.Description, original.Description)
		}
		if loaded.Status != original.Status {
			t.Errorf("Status = %s, want %s", loaded.Status, original.Status)
		}
		if loaded.Priority != original.Priority {
			t.Errorf("Priority = %d, want %d", loaded.Priority, original.Priority)
		}
		if len(loaded.Tags) != len(original.Tags) {
			t.Errorf("Tags len = %d, want %d", len(loaded.Tags), len(original.Tags))
		}
	})

	t.Run("empty tags saved correctly", func(t *testing.T) {
		store, rootDir := setupTestStore(t)
		testutil.MakeDir(t, filepath.Join(rootDir, "fix", "empty-tags"))

		item := BacklogItem{
			Name:     "empty-tags",
			Title:    "Empty Tags",
			Status:   StatusBacklog,
			Priority: 5,
			Tags:     []string{},
			Created:  "2025-01-01T00:00:00Z",
			Kind:     KindFix,
		}
		if err := store.SaveItem(item); err != nil {
			t.Fatalf("SaveItem error: %v", err)
		}

		loaded, err := store.LoadItem(KindFix, "empty-tags")
		if err != nil {
			t.Fatalf("LoadItem error: %v", err)
		}
		if loaded.Tags == nil {
			t.Error("Tags should be non-nil empty slice")
		}
		if len(loaded.Tags) != 0 {
			t.Errorf("Tags len = %d, want 0", len(loaded.Tags))
		}
	})

	t.Run("missing kind returns error", func(t *testing.T) {
		store, _ := setupTestStore(t)
		item := BacklogItem{
			Name:  "no-kind",
			Title: "No Kind",
		}
		err := store.SaveItem(item)
		if err == nil {
			t.Fatal("expected error for missing kind")
		}
		if !strings.Contains(err.Error(), "kind is required") {
			t.Errorf("error message = %q, want to contain 'kind is required'", err.Error())
		}
	})

	t.Run("preserves unknown fields on rewrite", func(t *testing.T) {
		store, rootDir := setupTestStore(t)
		itemDir := filepath.Join(rootDir, "ideas", "preserve-test")
		testutil.MakeDir(t, itemDir)

		// Write initial spec with an extra field
		initial := map[string]any{
			"title":           "Preserve Test",
			"status":          "backlog",
			"priority":        5,
			"created":         "2025-01-01T00:00:00Z",
			"archive_reason":  "superseded",
			"custom_metadata": map[string]any{"key": "value"},
		}
		testutil.WriteJSONFile(t, filepath.Join(itemDir, "spec.json"), initial)

		// Save updated item
		item := BacklogItem{
			Name:     "preserve-test",
			Title:    "Updated Title",
			Status:   StatusReady,
			Priority: 8,
			Tags:     []string{},
			Created:  "2025-01-01T00:00:00Z",
			Updated:  "2025-01-03T00:00:00Z",
			Kind:     KindIdea,
		}
		if err := store.SaveItem(item); err != nil {
			t.Fatalf("SaveItem error: %v", err)
		}

		// Read raw JSON and verify extra fields are preserved
		data, err := os.ReadFile(filepath.Join(itemDir, "spec.json"))
		if err != nil {
			t.Fatalf("ReadFile error: %v", err)
		}
		var raw map[string]any
		if err := json.Unmarshal(data, &raw); err != nil {
			t.Fatalf("Unmarshal error: %v", err)
		}
		if raw["archive_reason"] != "superseded" {
			t.Errorf("archive_reason was not preserved, got %v", raw["archive_reason"])
		}
		if raw["title"] != "Updated Title" {
			t.Errorf("title = %v, want Updated Title", raw["title"])
		}
	})

	t.Run("research_target always cleaned from saved JSON", func(t *testing.T) {
		store, rootDir := setupTestStore(t)
		testutil.MakeDir(t, filepath.Join(rootDir, "research", "with-target"))

		// Pre-seed a spec.json with a legacy research_target field.
		specPath := filepath.Join(rootDir, "research", "with-target", "spec.json")
		if err := os.WriteFile(specPath, []byte(`{"title":"Research Item","status":"backlog","priority":5,"created":"2025-01-01T00:00:00Z","research_target":"idea"}`), 0o644); err != nil {
			t.Fatalf("WriteFile error: %v", err)
		}

		item := BacklogItem{
			Name:     "with-target",
			Title:    "Research Item",
			Status:   StatusBacklog,
			Priority: 5,
			Tags:     []string{},
			Created:  "2025-01-01T00:00:00Z",
			Kind:     KindResearch,
		}
		if err := store.SaveItem(item); err != nil {
			t.Fatalf("SaveItem error: %v", err)
		}

		// Read raw JSON to verify research_target is removed.
		data, err := os.ReadFile(specPath)
		if err != nil {
			t.Fatalf("ReadFile error: %v", err)
		}
		var raw map[string]any
		if err := json.Unmarshal(data, &raw); err != nil {
			t.Fatalf("Unmarshal error: %v", err)
		}
		if _, exists := raw["research_target"]; exists {
			t.Error("research_target should be removed during save")
		}
	})

	t.Run("save to new file creates spec.json", func(t *testing.T) {
		store, rootDir := setupTestStore(t)
		testutil.MakeDir(t, filepath.Join(rootDir, "chore", "new-item"))

		item := BacklogItem{
			Name:     "new-item",
			Title:    "Brand New",
			Status:   StatusBacklog,
			Priority: 5,
			Tags:     []string{},
			Created:  "2025-01-01T00:00:00Z",
			Kind:     KindChore,
		}
		if err := store.SaveItem(item); err != nil {
			t.Fatalf("SaveItem error: %v", err)
		}

		specPath := filepath.Join(rootDir, "chore", "new-item", "spec.json")
		testutil.AssertFileExists(t, specPath)
	})
}

func TestCheckDependencies_FailOpen(t *testing.T) {
	store, rootDir := setupTestStore(t)

	// Create a completed dependency.
	writeSpecJSON(t, rootDir, KindIdea, "dep-done", map[string]any{
		"title":    "Done Dep",
		"status":   "completed",
		"priority": 3,
		"created":  "2025-01-01T00:00:00Z",
	})

	// Create an incomplete dependency.
	writeSpecJSON(t, rootDir, KindIdea, "dep-pending", map[string]any{
		"title":    "Pending Dep",
		"status":   "ready",
		"priority": 3,
		"created":  "2025-01-01T00:00:00Z",
	})

	t.Run("deleted/archived dep is not unmet", func(t *testing.T) {
		// A dependency whose spec no longer exists on disk is presumed
		// completed and subsequently archived/deleted. It must never
		// block execution — cleaning up past work is a valid workflow.
		unmet, err := store.CheckDependencies([]string{"idea/nonexistent-item"})
		if err != nil {
			t.Fatalf("expected no error for missing dep, got: %v", err)
		}
		if len(unmet) != 0 {
			t.Errorf("missing dep should be treated as satisfied (archived), got unmet: %v", unmet)
		}
	})

	t.Run("unparseable ref is skipped not blocking", func(t *testing.T) {
		// Unparseable refs cannot be validated and should not block execution.
		unmet, err := store.CheckDependencies([]string{"bad-ref-no-slash"})
		if err != nil {
			t.Fatalf("expected no error for bad ref, got: %v", err)
		}
		if len(unmet) != 0 {
			t.Errorf("unparseable ref should be skipped, got unmet: %v", unmet)
		}
	})

	t.Run("completed dep is not unmet", func(t *testing.T) {
		unmet, err := store.CheckDependencies([]string{"idea/dep-done"})
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
		if len(unmet) != 0 {
			t.Errorf("expected no unmet deps for completed item, got: %v", unmet)
		}
	})

	t.Run("incomplete dep is unmet", func(t *testing.T) {
		unmet, err := store.CheckDependencies([]string{"idea/dep-pending"})
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
		if len(unmet) != 1 || unmet[0] != "idea/dep-pending" {
			t.Errorf("expected [idea/dep-pending] as unmet, got: %v", unmet)
		}
	})

	t.Run("mixed completed, archived, and incomplete deps", func(t *testing.T) {
		// Only the incomplete (on-disk, non-completed) dep should be unmet.
		// The completed dep is satisfied; the missing dep is presumed archived.
		unmet, err := store.CheckDependencies([]string{
			"idea/dep-done",         // completed on disk → satisfied
			"idea/nonexistent-item", // missing on disk → presumed archived → satisfied
			"idea/dep-pending",      // exists on disk, status=ready → unmet
		})
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
		if len(unmet) != 1 {
			t.Fatalf("expected 1 unmet dep (only the incomplete one), got %d: %v", len(unmet), unmet)
		}
		if unmet[0] != "idea/dep-pending" {
			t.Errorf("expected unmet dep to be idea/dep-pending, got: %s", unmet[0])
		}
	})
}
