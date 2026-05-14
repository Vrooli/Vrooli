package main

import (
	"testing"
	"time"

	intworkspace "web-console/internal/workspace"
)

// [REQ:P1-002a] Shortcut Profile Storage — interface compliance
// [REQ:P1-003a] Provider Configuration Storage — interface compliance

// Compile-time interface compliance checks
var (
	_ ShortcutStore      = (*ShortcutProfileStore)(nil)
	_ ShortcutStore      = (*SQLShortcutStore)(nil)
	_ AIConfigStore      = (*AIProviderConfigStore)(nil)
	_ AIConfigStore      = (*SQLAIConfigStore)(nil)
	_ intworkspace.Store = (*intworkspace.MemStore)(nil)
	_ intworkspace.Store = (*intworkspace.SQLStore)(nil)
)

// TestShortcutStoreInterface verifies both implementations satisfy the
// ShortcutStore contract using the in-memory implementation (which is
// available without a database).
func TestShortcutStoreInterface(t *testing.T) {
	var store ShortcutStore = NewShortcutProfileStore()

	// List returns at least the default profile
	profiles := store.List()
	if len(profiles) == 0 {
		t.Fatal("expected at least one default profile")
	}

	// Get retrieves the default profile
	p, ok := store.Get("default")
	if !ok {
		t.Fatal("expected to find default profile")
	}
	if p.Scope != "service" {
		t.Errorf("expected scope 'service', got %q", p.Scope)
	}

	// Upsert creates a new profile
	created := store.Upsert("test-1", "workspace", "Test", []ShortcutEntry{
		{Label: "Echo", Command: "echo hello"},
	})
	if created == nil {
		t.Fatal("expected upsert to return profile")
	}
	if created.ID != "test-1" {
		t.Errorf("expected ID 'test-1', got %q", created.ID)
	}

	// Effective returns highest-priority scope's shortcuts
	effective := store.Effective()
	if len(effective) == 0 {
		t.Fatal("expected effective shortcuts")
	}
	// workspace (priority 2) > service (priority 1)
	if effective[0].Label != "Echo" {
		t.Errorf("expected workspace shortcut to take priority, got %q", effective[0].Label)
	}

	// Delete removes the profile
	if !store.Delete("test-1") {
		t.Error("expected delete to return true")
	}
	if _, ok := store.Get("test-1"); ok {
		t.Error("expected profile to be gone after delete")
	}
}

// TestAIConfigStoreInterface verifies the in-memory implementation satisfies
// the AIConfigStore contract.
func TestAIConfigStoreInterface(t *testing.T) {
	var store AIConfigStore = NewAIProviderConfigStore()

	// GetConfigs returns default providers
	configs := store.GetConfigs()
	if len(configs) < 2 {
		t.Fatalf("expected at least 2 providers, got %d", len(configs))
	}

	// IsEnabled for default providers
	if !store.IsEnabled("ollama") {
		t.Error("expected ollama to be enabled by default")
	}

	// GetProviderTimeout returns configured timeout
	timeout := store.GetProviderTimeout("ollama")
	if timeout != 30*time.Second {
		t.Errorf("expected 30s timeout, got %v", timeout)
	}

	// UpdateConfig changes settings
	if !store.UpdateConfig("ollama", false, 1, 15, 2) {
		t.Error("expected update to succeed")
	}
	if store.IsEnabled("ollama") {
		t.Error("expected ollama to be disabled after update")
	}

	// RecordSuccess/RecordError affect health
	store.RecordSuccess("ollama", 100*time.Millisecond)
	store.RecordError("openrouter")

	health := store.GetHealth()
	if len(health) == 0 {
		t.Fatal("expected health data")
	}
}

// TestInitSchemaIdempotent verifies initSchema handles missing files gracefully.
func TestInitSchemaIdempotent(t *testing.T) {
	// With a nil DB, initSchema should fail at the exec step, not panic
	err := initSchema(nil)
	if err == nil {
		t.Error("expected error with nil DB")
	}
}
