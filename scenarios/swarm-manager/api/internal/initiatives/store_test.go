package initiatives

import (
	"encoding/json"
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

	// Verify the file is at the folder-based path.
	expectedPath := filepath.Join(store.dir, "test-init", initiativeFileName)
	if _, err := os.Stat(expectedPath); os.IsNotExist(err) {
		t.Fatalf("expected file at %s, but it does not exist", expectedPath)
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

func TestStore_InitDir(t *testing.T) {
	store := setupTestStore(t)
	dir := store.InitDir("my-init")
	expected := filepath.Join(store.dir, "my-init")
	if dir != expected {
		t.Errorf("expected %q, got %q", expected, dir)
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

func TestStore_LoadAll_IgnoresRootJsonFiles(t *testing.T) {
	store := setupTestStore(t)

	// Save a valid initiative in folder layout.
	init := &Initiative{
		Name:    "valid",
		Title:   "Valid",
		Status:  "active",
		Items:   []string{},
		Created: "2024-01-01T00:00:00Z",
		Updated: "2024-01-01T00:00:00Z",
	}
	if err := store.Save(init); err != nil {
		t.Fatalf("Save failed: %v", err)
	}

	// Write a stray .json file at the root (old format or temp file).
	if err := os.MkdirAll(store.dir, 0o755); err != nil {
		t.Fatal(err)
	}
	strayPath := filepath.Join(store.dir, "stray.json")
	if err := os.WriteFile(strayPath, []byte(`{"name":"stray"}`), 0o644); err != nil {
		t.Fatal(err)
	}

	items, err := store.LoadAll()
	if err != nil {
		t.Fatalf("LoadAll failed: %v", err)
	}
	// Should only find the folder-based initiative, not the stray file.
	if len(items) != 1 {
		t.Errorf("expected 1 item, got %d", len(items))
	}
	if len(items) > 0 && items[0].Name != "valid" {
		t.Errorf("expected valid, got %q", items[0].Name)
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

	// Write a malformed initiative.json inside a folder.
	brokenDir := filepath.Join(store.dir, "broken")
	if err := os.MkdirAll(brokenDir, 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(brokenDir, initiativeFileName), []byte("{invalid json"), 0o644); err != nil {
		t.Fatal(err)
	}

	items, err := store.LoadAll()
	if err != nil {
		t.Fatalf("LoadAll should succeed despite malformed file: %v", err)
	}
	if len(items) != 1 {
		t.Errorf("expected 1 valid item, got %d", len(items))
	}
	if len(items) > 0 && items[0].Name != "valid" {
		t.Errorf("expected valid, got %q", items[0].Name)
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

	// Write an extra file to ensure the entire folder is removed.
	extraFile := filepath.Join(store.InitDir("to-delete"), "notes.md")
	if err := os.WriteFile(extraFile, []byte("some notes"), 0o644); err != nil {
		t.Fatal(err)
	}

	if err := store.Delete("to-delete"); err != nil {
		t.Fatalf("Delete failed: %v", err)
	}

	if store.Exists("to-delete") {
		t.Error("initiative should not exist after delete")
	}
	// Verify the entire directory is gone.
	if _, err := os.Stat(store.InitDir("to-delete")); !os.IsNotExist(err) {
		t.Error("initiative directory should not exist after delete")
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

func TestStore_Migrate(t *testing.T) {
	store := setupTestStore(t)

	// Create the initiatives directory.
	if err := os.MkdirAll(store.dir, 0o755); err != nil {
		t.Fatal(err)
	}

	// Write old-style flat files.
	for _, name := range []string{"alpha", "beta"} {
		data, _ := json.Marshal(Initiative{
			Name:    name,
			Title:   "Title " + name,
			Status:  "active",
			Items:   []string{},
			Created: "2024-01-01T00:00:00Z",
			Updated: "2024-01-01T00:00:00Z",
		})
		oldPath := filepath.Join(store.dir, name+".json")
		if err := os.WriteFile(oldPath, data, 0o644); err != nil {
			t.Fatal(err)
		}
	}

	if err := store.Migrate(); err != nil {
		t.Fatalf("Migrate failed: %v", err)
	}

	// Verify old files are gone and new layout exists.
	for _, name := range []string{"alpha", "beta"} {
		oldPath := filepath.Join(store.dir, name+".json")
		if _, err := os.Stat(oldPath); !os.IsNotExist(err) {
			t.Errorf("old file %s should not exist after migration", oldPath)
		}
		newPath := filepath.Join(store.dir, name, initiativeFileName)
		if _, err := os.Stat(newPath); os.IsNotExist(err) {
			t.Errorf("new file %s should exist after migration", newPath)
		}
	}

	// Verify the initiatives can be loaded.
	items, err := store.LoadAll()
	if err != nil {
		t.Fatalf("LoadAll after migration failed: %v", err)
	}
	if len(items) != 2 {
		t.Fatalf("expected 2 items after migration, got %d", len(items))
	}
}

func TestStore_Migrate_Idempotent(t *testing.T) {
	store := setupTestStore(t)

	// Create a properly migrated initiative.
	init := &Initiative{
		Name:    "already-migrated",
		Title:   "Already Migrated",
		Status:  "active",
		Items:   []string{},
		Created: "2024-01-01T00:00:00Z",
		Updated: "2024-01-01T00:00:00Z",
	}
	if err := store.Save(init); err != nil {
		t.Fatal(err)
	}

	// Running Migrate should be a no-op.
	if err := store.Migrate(); err != nil {
		t.Fatalf("Migrate should succeed on already-migrated store: %v", err)
	}

	// Verify nothing was corrupted.
	loaded, err := store.Load("already-migrated")
	if err != nil {
		t.Fatalf("Load after idempotent migration failed: %v", err)
	}
	if loaded.Title != "Already Migrated" {
		t.Errorf("expected title 'Already Migrated', got %q", loaded.Title)
	}
}

func TestStore_Migrate_NoOldFiles(t *testing.T) {
	store := setupTestStore(t)

	// Empty directory — nothing to migrate.
	if err := store.Migrate(); err != nil {
		t.Fatalf("Migrate on empty dir should succeed: %v", err)
	}
}

func TestStore_Migrate_MissingDirectory(t *testing.T) {
	// Store pointing to nonexistent path.
	store := &Store{dir: filepath.Join(t.TempDir(), "nonexistent")}
	if err := store.Migrate(); err != nil {
		t.Fatalf("Migrate on missing dir should succeed: %v", err)
	}
}
