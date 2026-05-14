package main

import (
	"testing"
	"time"

	intai "web-console/internal/ai"
)

func TestSQLAIConfigStore_GetConfigsSeeded(t *testing.T) {
	db := setupTestDB(t)
	store := intai.NewSQLConfigStore(db)

	configs := store.GetConfigs()
	if len(configs) != 2 {
		t.Fatalf("expected 2 seeded configs, got %d", len(configs))
	}

	// Should be sorted by priority
	if configs[0].Name != "ollama" {
		t.Errorf("first config should be ollama (priority 1), got %q", configs[0].Name)
	}
	if configs[1].Name != "openrouter" {
		t.Errorf("second config should be openrouter (priority 2), got %q", configs[1].Name)
	}
}

func TestSQLAIConfigStore_IsEnabled(t *testing.T) {
	db := setupTestDB(t)
	store := intai.NewSQLConfigStore(db)

	if !store.IsEnabled("ollama") {
		t.Error("expected ollama to be enabled")
	}
	if !store.IsEnabled("openrouter") {
		t.Error("expected openrouter to be enabled")
	}
	if store.IsEnabled("nonexistent") {
		t.Error("expected false for non-existent provider")
	}
}

func TestSQLAIConfigStore_GetProviderTimeout(t *testing.T) {
	db := setupTestDB(t)
	store := intai.NewSQLConfigStore(db)

	timeout := store.GetProviderTimeout("ollama")
	if timeout != 30*time.Second {
		t.Errorf("expected 30s, got %v", timeout)
	}

	// Non-existent returns default
	timeout = store.GetProviderTimeout("nonexistent")
	if timeout != intai.DefaultProviderTimeout {
		t.Errorf("expected default timeout %v, got %v", intai.DefaultProviderTimeout, timeout)
	}
}

func TestSQLAIConfigStore_UpdateConfig(t *testing.T) {
	db := setupTestDB(t)
	store := intai.NewSQLConfigStore(db)

	// Disable ollama, change timeout
	if !store.UpdateConfig("ollama", false, 1, 15, 3) {
		t.Error("expected UpdateConfig to return true")
	}

	if store.IsEnabled("ollama") {
		t.Error("expected ollama disabled after update")
	}

	timeout := store.GetProviderTimeout("ollama")
	if timeout != 15*time.Second {
		t.Errorf("expected 15s timeout, got %v", timeout)
	}

	configs := store.GetConfigs()
	for _, c := range configs {
		if c.Name == "ollama" {
			if c.MaxRetries != 3 {
				t.Errorf("max_retries: got %d, want 3", c.MaxRetries)
			}
		}
	}

	// Non-existent provider
	if store.UpdateConfig("nonexistent", true, 1, 30, 0) {
		t.Error("expected false for non-existent provider")
	}
}

func TestSQLAIConfigStore_HealthTracking(t *testing.T) {
	db := setupTestDB(t)
	store := intai.NewSQLConfigStore(db)

	// Record some activity
	store.RecordSuccess("ollama", 50*time.Millisecond)
	store.RecordSuccess("ollama", 100*time.Millisecond)
	store.RecordError("openrouter")

	health := store.GetHealth()
	if len(health) != 2 {
		t.Fatalf("expected 2 health entries, got %d", len(health))
	}

	for _, h := range health {
		switch h.Name {
		case "ollama":
			if !h.Available {
				t.Error("ollama should be available")
			}
			if h.SuccessCount != 2 {
				t.Errorf("ollama success count: got %d", h.SuccessCount)
			}
		case "openrouter":
			if h.Available {
				t.Error("openrouter should not be available")
			}
			if h.ErrorCount != 1 {
				t.Errorf("openrouter error count: got %d", h.ErrorCount)
			}
		}
	}
}

func TestSQLAIConfigStore_HealthIgnoresUnknown(t *testing.T) {
	db := setupTestDB(t)
	store := intai.NewSQLConfigStore(db)

	// Recording for unknown provider should not panic
	store.RecordSuccess("unknown-provider", 10*time.Millisecond)
	store.RecordError("unknown-provider")
}
