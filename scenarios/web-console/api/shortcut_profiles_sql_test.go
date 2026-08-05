package main

import (
	"context"
	"database/sql"
	"encoding/json"
	"testing"
)

func TestSQLShortcutStore_ListSeeded(t *testing.T) {
	db := setupTestDB(t)
	store := NewSQLShortcutStore(db)

	profiles := store.List(context.Background())
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

	p, ok := store.Get(context.Background(), "default")
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
	_, ok = store.Get(context.Background(), "nonexistent")
	if ok {
		t.Error("expected false for non-existent profile")
	}
}

func TestSQLShortcutStore_UpsertCreate(t *testing.T) {
	db := setupTestDB(t)
	store := NewSQLShortcutStore(db)

	created := store.Upsert(context.Background(), "test-1", "workspace", "Test Profile", []ShortcutEntry{
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

	store.Upsert(context.Background(), "upd-1", "service", "V1", []ShortcutEntry{
		{Label: "A", Command: "a"},
	})

	updated := store.Upsert(context.Background(), "upd-1", "service", "V2", []ShortcutEntry{
		{Label: "B", Command: "b"},
	})
	if updated.Name != "V2" {
		t.Errorf("expected updated name 'V2', got %q", updated.Name)
	}

	profiles := store.List(context.Background())
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
	first := store.Upsert(context.Background(), "replay-1", "service", "Same", shortcuts)
	if first == nil {
		t.Fatal("first upsert returned nil")
	}

	// Replay with identical content — updated_at should not change
	second := store.Upsert(context.Background(), "replay-1", "service", "Same", shortcuts)
	if second == nil {
		t.Fatal("second upsert returned nil")
	}
	if first.UpdatedAt != second.UpdatedAt {
		t.Errorf("replay safety: updated_at changed from %q to %q", first.UpdatedAt, second.UpdatedAt)
	}

	// Change content — updated_at should change
	third := store.Upsert(context.Background(), "replay-1", "service", "Changed", shortcuts)
	if third.UpdatedAt == first.UpdatedAt {
		t.Error("expected updated_at to change when content changes")
	}
}

func TestSQLShortcutStore_Delete(t *testing.T) {
	db := setupTestDB(t)
	store := NewSQLShortcutStore(db)

	store.Upsert(context.Background(), "del-1", "service", "ToDelete", []ShortcutEntry{})

	if !store.Delete(context.Background(), "del-1") {
		t.Error("expected Delete to return true")
	}
	if store.Delete(context.Background(), "del-1") {
		t.Error("expected second Delete to return false")
	}
	if store.Delete(context.Background(), "nonexistent") {
		t.Error("expected Delete of nonexistent to return false")
	}
}

func TestSQLShortcutStore_Effective(t *testing.T) {
	db := setupTestDB(t)
	store := NewSQLShortcutStore(db)

	// Default is "service" scope
	effective := store.Effective(context.Background())
	if len(effective) == 0 {
		t.Fatal("expected effective shortcuts from seed data")
	}

	// Add workspace scope — should override service
	store.Upsert(context.Background(), "ws-1", "workspace", "WS", []ShortcutEntry{
		{Label: "WorkspaceCmd", Command: "ws"},
	})
	effective = store.Effective(context.Background())
	if effective[0].Label != "WorkspaceCmd" {
		t.Errorf("workspace should override service, got %q", effective[0].Label)
	}

	// Add parent scope — should override workspace
	store.Upsert(context.Background(), "par-1", "parent", "Parent", []ShortcutEntry{
		{Label: "ParentCmd", Command: "parent"},
	})
	effective = store.Effective(context.Background())
	if effective[0].Label != "ParentCmd" {
		t.Errorf("parent should override workspace, got %q", effective[0].Label)
	}
}

func TestSQLShortcutStore_UnicodeShortcuts(t *testing.T) {
	db := setupTestDB(t)
	store := NewSQLShortcutStore(db)

	created := store.Upsert(context.Background(), "uni-1", "service", "日本語", []ShortcutEntry{
		{Label: "こんにちは", Command: "echo hello", Description: "挨拶"},
	})
	if created == nil {
		t.Fatal("unicode upsert returned nil")
	}
	if created.Name != "日本語" {
		t.Errorf("unicode name: got %q", created.Name)
	}

	p, ok := store.Get(context.Background(), "uni-1")
	if !ok {
		t.Fatal("expected to find unicode profile")
	}
	if p.Shortcuts[0].Label != "こんにちは" {
		t.Errorf("unicode shortcut label: got %q", p.Shortcuts[0].Label)
	}
}

// setLegacyDefaultRow forces the seeded "default" profile to the first known
// legacy (pre-OpenCode/Grok) shortcut list, simulating a DB created before the
// new defaults shipped.
func setLegacyDefaultRow(t *testing.T, db *sql.DB) {
	t.Helper()
	legacy, err := json.Marshal(legacyDefaultShortcutSets[0])
	if err != nil {
		t.Fatalf("marshal legacy default: %v", err)
	}
	if _, err := db.Exec(
		`UPDATE shortcut_profiles SET shortcuts = ? WHERE id = 'default' AND scope = 'service'`,
		string(legacy),
	); err != nil {
		t.Fatalf("seed legacy default row: %v", err)
	}
}

// TestReconcileDefaultShortcutProfile_UpgradesUnmodifiedSeed verifies that a
// persisted default profile still equal to the legacy seed is bumped to the
// current built-in defaults.
func TestReconcileDefaultShortcutProfile_UpgradesUnmodifiedSeed(t *testing.T) {
	db := setupTestDB(t)
	setLegacyDefaultRow(t, db)

	if err := reconcileDefaultShortcutProfile(context.Background(), db); err != nil {
		t.Fatalf("reconcile: %v", err)
	}

	store := NewSQLShortcutStore(db)
	p, ok := store.Get(context.Background(), "default")
	if !ok {
		t.Fatal("default profile missing after reconcile")
	}
	if len(p.Shortcuts) != len(defaultShortcuts) {
		t.Fatalf("expected %d shortcuts after upgrade, got %d", len(defaultShortcuts), len(p.Shortcuts))
	}
	if !shortcutsEqual(p.Shortcuts, defaultShortcuts) {
		t.Errorf("upgraded shortcuts do not match current defaults: %+v", p.Shortcuts)
	}

	// Idempotent: a second run is a no-op (content now differs from legacy).
	if err := reconcileDefaultShortcutProfile(context.Background(), db); err != nil {
		t.Fatalf("reconcile (2nd): %v", err)
	}
	p2, _ := store.Get(context.Background(), "default")
	if !shortcutsEqual(p2.Shortcuts, defaultShortcuts) {
		t.Errorf("second reconcile changed content: %+v", p2.Shortcuts)
	}
}

// TestReconcileDefaultShortcutProfile_PreservesCustomization verifies that a
// user-customized default profile is never overwritten.
func TestReconcileDefaultShortcutProfile_PreservesCustomization(t *testing.T) {
	db := setupTestDB(t)
	custom := []ShortcutEntry{{Label: "Mine", Command: "my-tool", Description: "personal"}}
	store := NewSQLShortcutStore(db)
	store.Upsert(context.Background(), "default", "service", "Default", custom)

	if err := reconcileDefaultShortcutProfile(context.Background(), db); err != nil {
		t.Fatalf("reconcile: %v", err)
	}

	p, _ := store.Get(context.Background(), "default")
	if !shortcutsEqual(p.Shortcuts, custom) {
		t.Errorf("customized default was modified: %+v", p.Shortcuts)
	}
}

// TestReconcileDefaultShortcutProfile_NoRowIsNoop verifies the migration is a
// no-op when no default profile row exists (fresh DBs before seed, or deleted).
func TestReconcileDefaultShortcutProfile_NoRowIsNoop(t *testing.T) {
	db := setupTestDB(t)
	if _, err := db.Exec(`DELETE FROM shortcut_profiles WHERE id = 'default'`); err != nil {
		t.Fatalf("delete default: %v", err)
	}
	if err := reconcileDefaultShortcutProfile(context.Background(), db); err != nil {
		t.Fatalf("reconcile on empty: %v", err)
	}
	store := NewSQLShortcutStore(db)
	if _, ok := store.Get(context.Background(), "default"); ok {
		t.Error("reconcile should not create a default row")
	}
}
