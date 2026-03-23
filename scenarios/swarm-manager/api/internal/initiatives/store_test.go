package initiatives

import (
	"os"
	"path/filepath"
	"testing"
)

func setupTestStore(t *testing.T) *Store {
	t.Helper()
	dir := t.TempDir()
	return NewStore(dir)
}

func TestStore_SaveAndLoad(t *testing.T) {
	store := setupTestStore(t)

	init := &Initiative{
		Name:        "test-init",
		Title:       "Test Initiative",
		Description: "A test initiative",
		Status:      "active",
		Items:       []string{"idea/foo", "fix/bar"},
		Created:     "2024-01-01T00:00:00Z",
		Updated:     "2024-01-01T00:00:00Z",
	}

	if err := store.Save(init); err != nil {
		t.Fatalf("Save failed: %v", err)
	}

	loaded, err := store.Load("test-init")
	if err != nil {
		t.Fatalf("Load failed: %v", err)
	}

	if loaded.Name != init.Name {
		t.Errorf("expected name %q, got %q", init.Name, loaded.Name)
	}
	if loaded.Title != init.Title {
		t.Errorf("expected title %q, got %q", init.Title, loaded.Title)
	}
	if loaded.Description != init.Description {
		t.Errorf("expected description %q, got %q", init.Description, loaded.Description)
	}
	if loaded.Status != init.Status {
		t.Errorf("expected status %q, got %q", init.Status, loaded.Status)
	}
	if len(loaded.Items) != 2 {
		t.Errorf("expected 2 items, got %d", len(loaded.Items))
	}
}

func TestStore_LoadNotFound(t *testing.T) {
	store := setupTestStore(t)

	_, err := store.Load("nonexistent")
	if err == nil {
		t.Fatal("expected error for nonexistent initiative")
	}
}

func TestStore_Exists(t *testing.T) {
	store := setupTestStore(t)

	if store.Exists("test") {
		t.Error("expected Exists to return false for missing initiative")
	}

	init := &Initiative{
		Name:    "test",
		Title:   "Test",
		Status:  "active",
		Items:   []string{},
		Created: "2024-01-01T00:00:00Z",
		Updated: "2024-01-01T00:00:00Z",
	}
	if err := store.Save(init); err != nil {
		t.Fatalf("Save failed: %v", err)
	}

	if !store.Exists("test") {
		t.Error("expected Exists to return true after save")
	}
}

func TestStore_LoadAll(t *testing.T) {
	store := setupTestStore(t)

	// Empty directory.
	items, err := store.LoadAll()
	if err != nil {
		t.Fatalf("LoadAll on empty dir failed: %v", err)
	}
	if len(items) != 0 {
		t.Errorf("expected 0 items, got %d", len(items))
	}

	// Add two initiatives.
	for _, name := range []string{"beta", "alpha"} {
		init := &Initiative{
			Name:    name,
			Title:   "Title " + name,
			Status:  "active",
			Items:   []string{},
			Created: "2024-01-01T00:00:00Z",
			Updated: "2024-01-01T00:00:00Z",
		}
		if err := store.Save(init); err != nil {
			t.Fatalf("Save %q failed: %v", name, err)
		}
	}

	items, err = store.LoadAll()
	if err != nil {
		t.Fatalf("LoadAll failed: %v", err)
	}
	if len(items) != 2 {
		t.Fatalf("expected 2 items, got %d", len(items))
	}
	// Should be sorted by name.
	if items[0].Name != "alpha" {
		t.Errorf("expected first item alpha, got %q", items[0].Name)
	}
	if items[1].Name != "beta" {
		t.Errorf("expected second item beta, got %q", items[1].Name)
	}
}

func TestStore_LoadAll_NoDirectory(t *testing.T) {
	// Store pointing to a non-existent directory.
	store := &Store{dir: filepath.Join(t.TempDir(), "missing", "path")}
	items, err := store.LoadAll()
	if err != nil {
		t.Fatalf("LoadAll on missing dir should return empty, got error: %v", err)
	}
	if len(items) != 0 {
		t.Errorf("expected 0 items, got %d", len(items))
	}
}

func TestStore_Delete(t *testing.T) {
	store := setupTestStore(t)

	init := &Initiative{
		Name:    "to-delete",
		Title:   "Delete Me",
		Status:  "active",
		Items:   []string{},
		Created: "2024-01-01T00:00:00Z",
		Updated: "2024-01-01T00:00:00Z",
	}
	if err := store.Save(init); err != nil {
		t.Fatalf("Save failed: %v", err)
	}

	if err := store.Delete("to-delete"); err != nil {
		t.Fatalf("Delete failed: %v", err)
	}

	if store.Exists("to-delete") {
		t.Error("initiative should not exist after delete")
	}

	// Delete again should be idempotent.
	if err := store.Delete("to-delete"); err != nil {
		t.Fatalf("second Delete should be idempotent, got: %v", err)
	}
}

func TestStore_Save_EmptyName(t *testing.T) {
	store := setupTestStore(t)
	init := &Initiative{Name: "", Title: "No Name", Status: "active"}
	err := store.Save(init)
	if err == nil {
		t.Fatal("expected error for empty name")
	}
}

func TestStore_LoadAll_SkipsMalformed(t *testing.T) {
	store := setupTestStore(t)

	// Create the directory.
	if err := os.MkdirAll(store.dir, 0o755); err != nil {
		t.Fatalf("mkdir failed: %v", err)
	}

	// Write a valid initiative.
	valid := &Initiative{
		Name:    "valid",
		Title:   "Valid",
		Status:  "active",
		Items:   []string{},
		Created: "2024-01-01T00:00:00Z",
		Updated: "2024-01-01T00:00:00Z",
	}
	if err := store.Save(valid); err != nil {
		t.Fatalf("Save failed: %v", err)
	}

	// Write a malformed JSON file.
	malformedPath := filepath.Join(store.dir, "broken.json")
	if err := os.WriteFile(malformedPath, []byte("{invalid json"), 0o644); err != nil {
		t.Fatalf("write malformed file failed: %v", err)
	}

	items, err := store.LoadAll()
	if err != nil {
		t.Fatalf("LoadAll should succeed despite malformed file: %v", err)
	}
	if len(items) != 1 {
		t.Errorf("expected 1 valid item, got %d", len(items))
	}
	if items[0].Name != "valid" {
		t.Errorf("expected valid, got %q", items[0].Name)
	}
}
