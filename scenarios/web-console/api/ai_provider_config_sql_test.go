package main

import (
	"context"
	"testing"
	"time"

	intai "web-console/internal/ai"
)

func TestSQLAIConfigStore_GetConfigsSeeded(t *testing.T) {
	db := setupTestDB(t)
	store := intai.NewSQLConfigStore(context.Background(), db)

	configs := store.GetConfigs(context.Background())
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
	store := intai.NewSQLConfigStore(context.Background(), db)

	if !store.IsEnabled(context.Background(), "ollama") {
		t.Error("expected ollama to be enabled")
	}
	if !store.IsEnabled(context.Background(), "openrouter") {
		t.Error("expected openrouter to be enabled")
	}
	if store.IsEnabled(context.Background(), "nonexistent") {
		t.Error("expected false for non-existent provider")
	}
}

func TestSQLAIConfigStore_GetProviderTimeout(t *testing.T) {
	db := setupTestDB(t)
	store := intai.NewSQLConfigStore(context.Background(), db)

	timeout := store.GetProviderTimeout(context.Background(), "ollama")
	if timeout != 30*time.Second {
		t.Errorf("expected 30s, got %v", timeout)
	}

	// Non-existent returns default
	timeout = store.GetProviderTimeout(context.Background(), "nonexistent")
	if timeout != intai.DefaultProviderTimeout {
		t.Errorf("expected default timeout %v, got %v", intai.DefaultProviderTimeout, timeout)
	}
}

func TestSQLAIConfigStore_UpdateConfig(t *testing.T) {
	db := setupTestDB(t)
	store := intai.NewSQLConfigStore(context.Background(), db)

	// Disable ollama, change timeout
	if !store.UpdateConfig(context.Background(), "ollama", false, 1, 15, 3) {
		t.Error("expected UpdateConfig to return true")
	}

	if store.IsEnabled(context.Background(), "ollama") {
		t.Error("expected ollama disabled after update")
	}

	timeout := store.GetProviderTimeout(context.Background(), "ollama")
	if timeout != 15*time.Second {
		t.Errorf("expected 15s timeout, got %v", timeout)
	}

	configs := store.GetConfigs(context.Background())
	for _, c := range configs {
		if c.Name == "ollama" {
			if c.MaxRetries != 3 {
				t.Errorf("max_retries: got %d, want 3", c.MaxRetries)
			}
		}
	}

	// Non-existent provider
	if store.UpdateConfig(context.Background(), "nonexistent", true, 1, 30, 0) {
		t.Error("expected false for non-existent provider")
	}
}

func TestSQLAIConfigStore_HealthTracking(t *testing.T) {
	db := setupTestDB(t)
	store := intai.NewSQLConfigStore(context.Background(), db)

	// Record some activity
	store.RecordSuccess(context.Background(), "ollama", 50*time.Millisecond)
	store.RecordSuccess(context.Background(), "ollama", 100*time.Millisecond)
	store.RecordError(context.Background(), "openrouter")

	health := store.GetHealth(context.Background())
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
	store := intai.NewSQLConfigStore(context.Background(), db)

	// Recording for unknown provider should not panic
	store.RecordSuccess(context.Background(), "unknown-provider", 10*time.Millisecond)
	store.RecordError(context.Background(), "unknown-provider")
}
