package main

import "testing"

func TestSQLShortcutStore_ListSeeded(t *testing.T) {
	db := setupTestDB(t)
	store := NewSQLShortcutStore(db)

	profiles := store.List()
	if len(profiles) != 1 {
		t.Fatalf("expected 1 seeded profile, got %d", len(profiles))
	}
	if profiles[0].ID != "default" {
		t.Errorf("expected seeded profile id 'default', got %q", profiles[0].ID)
	}
	if profiles[0].Scope != "service" {
		t.Errorf("expected scope 'service', got %q", profiles[0].Scope)
	}
}

func TestSQLShortcutStore_Get(t *testing.T) {
	db := setupTestDB(t)
	store := NewSQLShortcutStore(db)

	p, ok := store.Get("default")
	if !ok {
		t.Fatal("expected to find 'default' profile")
	}
	if p.Name != "Default" {
		t.Errorf("name: got %q", p.Name)
	}
	if len(p.Shortcuts) < 1 {
		t.Error("expected at least 1 shortcut in default profile")
	}

	// Non-existent
	_, ok = store.Get("nonexistent")
	if ok {
		t.Error("expected false for non-existent profile")
	}
}

func TestSQLShortcutStore_UpsertCreate(t *testing.T) {
	db := setupTestDB(t)
	store := NewSQLShortcutStore(db)

	created := store.Upsert("test-1", "workspace", "Test Profile", []ShortcutEntry{
		{Label: "Hello", Command: "echo hello", Description: "greet"},
	})
	if created == nil {
		t.Fatal("Upsert returned nil")
	}
	if created.ID != "test-1" {
		t.Errorf("id: got %q", created.ID)
	}
	if created.Scope != "workspace" {
		t.Errorf("scope: got %q", created.Scope)
	}
	if len(created.Shortcuts) != 1 {
		t.Fatalf("shortcuts count: got %d", len(created.Shortcuts))
	}
	if created.Shortcuts[0].Label != "Hello" {
		t.Errorf("shortcut label: got %q", created.Shortcuts[0].Label)
	}
}

func TestSQLShortcutStore_UpsertUpdate(t *testing.T) {
	db := setupTestDB(t)
	store := NewSQLShortcutStore(db)

	store.Upsert("upd-1", "service", "V1", []ShortcutEntry{
		{Label: "A", Command: "a"},
	})

	updated := store.Upsert("upd-1", "service", "V2", []ShortcutEntry{
		{Label: "B", Command: "b"},
	})
	if updated.Name != "V2" {
		t.Errorf("expected updated name 'V2', got %q", updated.Name)
	}

	profiles := store.List()
	count := 0
	for _, p := range profiles {
		if p.ID == "upd-1" {
			count++
		}
	}
	if count != 1 {
		t.Errorf("expected exactly 1 profile with id 'upd-1', got %d", count)
	}
}

func TestSQLShortcutStore_UpsertReplaySafety(t *testing.T) {
	db := setupTestDB(t)
	store := NewSQLShortcutStore(db)

	shortcuts := []ShortcutEntry{{Label: "X", Command: "x"}}
	first := store.Upsert("replay-1", "service", "Same", shortcuts)
	if first == nil {
		t.Fatal("first upsert returned nil")
	}

	// Replay with identical content — updated_at should not change
	second := store.Upsert("replay-1", "service", "Same", shortcuts)
	if second == nil {
		t.Fatal("second upsert returned nil")
	}
	if first.UpdatedAt != second.UpdatedAt {
		t.Errorf("replay safety: updated_at changed from %q to %q", first.UpdatedAt, second.UpdatedAt)
	}

	// Change content — updated_at should change
	third := store.Upsert("replay-1", "service", "Changed", shortcuts)
	if third.UpdatedAt == first.UpdatedAt {
		t.Error("expected updated_at to change when content changes")
	}
}

func TestSQLShortcutStore_Delete(t *testing.T) {
	db := setupTestDB(t)
	store := NewSQLShortcutStore(db)

	store.Upsert("del-1", "service", "ToDelete", []ShortcutEntry{})

	if !store.Delete("del-1") {
		t.Error("expected Delete to return true")
	}
	if store.Delete("del-1") {
		t.Error("expected second Delete to return false")
	}
	if store.Delete("nonexistent") {
		t.Error("expected Delete of nonexistent to return false")
	}
}

func TestSQLShortcutStore_Effective(t *testing.T) {
	db := setupTestDB(t)
	store := NewSQLShortcutStore(db)

	// Default is "service" scope
	effective := store.Effective()
	if len(effective) == 0 {
		t.Fatal("expected effective shortcuts from seed data")
	}

	// Add workspace scope — should override service
	store.Upsert("ws-1", "workspace", "WS", []ShortcutEntry{
		{Label: "WorkspaceCmd", Command: "ws"},
	})
	effective = store.Effective()
	if effective[0].Label != "WorkspaceCmd" {
		t.Errorf("workspace should override service, got %q", effective[0].Label)
	}

	// Add parent scope — should override workspace
	store.Upsert("par-1", "parent", "Parent", []ShortcutEntry{
		{Label: "ParentCmd", Command: "parent"},
	})
	effective = store.Effective()
	if effective[0].Label != "ParentCmd" {
		t.Errorf("parent should override workspace, got %q", effective[0].Label)
	}
}

func TestSQLShortcutStore_UnicodeShortcuts(t *testing.T) {
	db := setupTestDB(t)
	store := NewSQLShortcutStore(db)

	created := store.Upsert("uni-1", "service", "日本語", []ShortcutEntry{
		{Label: "こんにちは", Command: "echo hello", Description: "挨拶"},
	})
	if created == nil {
		t.Fatal("unicode upsert returned nil")
	}
	if created.Name != "日本語" {
		t.Errorf("unicode name: got %q", created.Name)
	}

	p, ok := store.Get("uni-1")
	if !ok {
		t.Fatal("expected to find unicode profile")
	}
	if p.Shortcuts[0].Label != "こんにちは" {
		t.Errorf("unicode shortcut label: got %q", p.Shortcuts[0].Label)
	}
}
