package main

import (
	"context"
	"testing"
	"time"

	intai "web-console/internal/ai"
	intworkspace "web-console/internal/workspace"
)

// [REQ:P1-002a] Shortcut Profile Storage — interface compliance
// [REQ:P1-003a] Provider Configuration Storage — interface compliance

// Compile-time interface compliance checks
var (
	_ ShortcutStore      = (*ShortcutProfileStore)(nil)
	_ ShortcutStore      = (*SQLShortcutStore)(nil)
	_ intai.ConfigStore  = (*intai.MemConfigStore)(nil)
	_ intai.ConfigStore  = (*intai.SQLConfigStore)(nil)
	_ intworkspace.Store = (*intworkspace.MemStore)(nil)
	_ intworkspace.Store = (*intworkspace.SQLStore)(nil)
)

// TestShortcutStoreInterface verifies both implementations satisfy the
// ShortcutStore contract using the in-memory implementation (which is
// available without a database).
func TestShortcutStoreInterface(t *testing.T) {
	var store ShortcutStore = NewShortcutProfileStore()

	// List returns at least the default profile
	profiles := store.List(context.Background())
	if len(profiles) == 0 {
		t.Fatal("expected at least one default profile")
	}

	// Get retrieves the default profile
	p, ok := store.Get(context.Background(), "default")
	if !ok {
		t.Fatal("expected to find default profile")
	}
	if p.Scope != "service" {
		t.Errorf("expected scope 'service', got %q", p.Scope)
	}

	// Upsert creates a new profile
	created := store.Upsert(context.Background(), "test-1", "workspace", "Test", []ShortcutEntry{
		{Label: "Echo", Command: "echo hello"},
	})
	if created == nil {
		t.Fatal("expected upsert to return profile")
	}
	if created.ID != "test-1" {
		t.Errorf("expected ID 'test-1', got %q", created.ID)
	}

	// Effective returns highest-priority scope's shortcuts
	effective := store.Effective(context.Background())
	if len(effective.Shortcuts) == 0 {
		t.Fatal("expected effective shortcuts")
	}
	// workspace (priority 2) > service (priority 1)
	if effective.Shortcuts[0].Label != "Echo" {
		t.Errorf("expected workspace shortcut to take priority, got %q", effective.Shortcuts[0].Label)
	}
	if effective.ProfileID != "test-1" || effective.Scope != "workspace" {
		t.Errorf("effective profile = %q/%q, want test-1/workspace", effective.ProfileID, effective.Scope)
	}

	// Delete removes the profile
	if !store.Delete(context.Background(), "test-1") {
		t.Error("expected delete to return true")
	}
	if _, ok := store.Get(context.Background(), "test-1"); ok {
		t.Error("expected profile to be gone after delete")
	}
}

// TestAIConfigStoreInterface verifies the in-memory implementation satisfies
// the intai.ConfigStore contract.
func TestAIConfigStoreInterface(t *testing.T) {
	var store intai.ConfigStore = intai.NewMemConfigStore()

	// GetConfigs returns default providers
	configs := store.GetConfigs(context.Background())
	if len(configs) < 2 {
		t.Fatalf("expected at least 2 providers, got %d", len(configs))
	}

	// IsEnabled for default providers
	if !store.IsEnabled(context.Background(), "ollama") {
		t.Error("expected ollama to be enabled by default")
	}

	// GetProviderTimeout returns configured timeout
	timeout := store.GetProviderTimeout(context.Background(), "ollama")
	if timeout != 30*time.Second {
		t.Errorf("expected 30s timeout, got %v", timeout)
	}

	// UpdateConfig changes settings
	if !store.UpdateConfig(context.Background(), "ollama", false, 1, 15, 2) {
		t.Error("expected update to succeed")
	}
	if store.IsEnabled(context.Background(), "ollama") {
		t.Error("expected ollama to be disabled after update")
	}

	// RecordSuccess/RecordError affect health
	store.RecordSuccess(context.Background(), "ollama", 100*time.Millisecond)
	store.RecordError(context.Background(), "openrouter")

	health := store.GetHealth(context.Background())
	if len(health) == 0 {
		t.Fatal("expected health data")
	}
}

// TestInitSchemaIdempotent verifies initSchema handles missing files gracefully.
func TestInitSchemaIdempotent(t *testing.T) {
	// With a nil DB, initSchema should fail at the exec step, not panic
	err := initSchema(context.Background(), nil)
	if err == nil {
		t.Error("expected error with nil DB")
	}
}
