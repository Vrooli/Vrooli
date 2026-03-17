package main

import (
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"
)

// [REQ:P1-003a] Provider Configuration Storage - store tests

func TestAIProviderConfigStore_Defaults(t *testing.T) {
	store := NewAIProviderConfigStore()
	configs := store.GetConfigs()
	if len(configs) != 2 {
		t.Fatalf("expected 2 providers, got %d", len(configs))
	}
	// Should be sorted by priority (ollama=1, openrouter=2)
	if configs[0].Name != "ollama" {
		t.Errorf("expected first provider 'ollama', got %q", configs[0].Name)
	}
	if configs[1].Name != "openrouter" {
		t.Errorf("expected second provider 'openrouter', got %q", configs[1].Name)
	}
	if !configs[0].Enabled || !configs[1].Enabled {
		t.Error("expected both providers enabled by default")
	}
}

func TestAIProviderConfigStore_UpdateConfig(t *testing.T) {
	store := NewAIProviderConfigStore()

	// Disable ollama
	ok := store.UpdateConfig("ollama", false, 1, 15, 2)
	if !ok {
		t.Fatal("expected update to succeed")
	}

	configs := store.GetConfigs()
	for _, c := range configs {
		if c.Name == "ollama" {
			if c.Enabled {
				t.Error("expected ollama disabled")
			}
			if c.TimeoutSec != 15 {
				t.Errorf("expected timeout 15, got %d", c.TimeoutSec)
			}
			if c.MaxRetries != 2 {
				t.Errorf("expected max retries 2, got %d", c.MaxRetries)
			}
		}
	}

	// Unknown provider
	if store.UpdateConfig("unknown", true, 1, 30, 0) {
		t.Error("expected update of unknown provider to fail")
	}
}

func TestAIProviderConfigStore_IsEnabled(t *testing.T) {
	store := NewAIProviderConfigStore()
	if !store.IsEnabled("ollama") {
		t.Error("expected ollama enabled")
	}
	store.UpdateConfig("ollama", false, 1, 30, 0)
	if store.IsEnabled("ollama") {
		t.Error("expected ollama disabled after update")
	}
	if store.IsEnabled("unknown") {
		t.Error("expected unknown provider not enabled")
	}
}

func TestAIProviderConfigStore_GetProviderTimeout(t *testing.T) {
	store := NewAIProviderConfigStore()
	if got := store.GetProviderTimeout("ollama"); got != 30*time.Second {
		t.Errorf("expected 30s, got %v", got)
	}
	store.UpdateConfig("ollama", true, 1, 10, 0)
	if got := store.GetProviderTimeout("ollama"); got != 10*time.Second {
		t.Errorf("expected 10s, got %v", got)
	}
	if got := store.GetProviderTimeout("unknown"); got != 30*time.Second {
		t.Errorf("expected 30s default for unknown, got %v", got)
	}
}

// [REQ:P1-003b] Provider Health Dashboard - health tracking

func TestAIProviderConfigStore_HealthTracking(t *testing.T) {
	store := NewAIProviderConfigStore()

	store.RecordSuccess("ollama", 100*time.Millisecond)
	store.RecordSuccess("ollama", 200*time.Millisecond)
	store.RecordError("openrouter")

	health := store.GetHealth()
	for _, h := range health {
		switch h.Name {
		case "ollama":
			if !h.Available {
				t.Error("expected ollama available")
			}
			if h.SuccessCount != 2 {
				t.Errorf("expected 2 successes, got %d", h.SuccessCount)
			}
			if h.ErrorRate != 0 {
				t.Errorf("expected 0 error rate, got %f", h.ErrorRate)
			}
		case "openrouter":
			if h.Available {
				t.Error("expected openrouter not available")
			}
			if h.ErrorCount != 1 {
				t.Errorf("expected 1 error, got %d", h.ErrorCount)
			}
			if h.ErrorRate != 1.0 {
				t.Errorf("expected 1.0 error rate, got %f", h.ErrorRate)
			}
		}
	}
}

// [REQ:P1-003a] Provider Configuration Storage - handler tests

func TestHandleGetAIConfig(t *testing.T) {
	srv := newFakeTestServer()
	req := httptest.NewRequest("GET", "/api/v1/ai/config", nil)
	rec := httptest.NewRecorder()

	srv.handleGetAIConfig(rec, req)

	if rec.Code != http.StatusOK {
		t.Fatalf("expected 200, got %d", rec.Code)
	}

	var resp AIProviderConfigResponse
	if err := json.Unmarshal(rec.Body.Bytes(), &resp); err != nil {
		t.Fatalf("decode: %v", err)
	}
	if len(resp.Providers) != 2 {
		t.Errorf("expected 2 providers, got %d", len(resp.Providers))
	}
	if len(resp.Health) != 2 {
		t.Errorf("expected 2 health entries, got %d", len(resp.Health))
	}
}

func TestHandleUpdateAIConfig(t *testing.T) {
	srv := newFakeTestServer()

	body := `{"name":"ollama","enabled":false,"timeout_sec":10}`
	req := httptest.NewRequest("PUT", "/api/v1/ai/config", strings.NewReader(body))
	req.Header.Set("Content-Type", "application/json")
	rec := httptest.NewRecorder()

	srv.handleUpdateAIConfig(rec, req)

	if rec.Code != http.StatusOK {
		t.Fatalf("expected 200, got %d: %s", rec.Code, rec.Body.String())
	}

	var resp AIProviderConfigResponse
	if err := json.Unmarshal(rec.Body.Bytes(), &resp); err != nil {
		t.Fatalf("decode: %v", err)
	}
	for _, p := range resp.Providers {
		if p.Name == "ollama" {
			if p.Enabled {
				t.Error("expected ollama disabled")
			}
			if p.TimeoutSec != 10 {
				t.Errorf("expected timeout 10, got %d", p.TimeoutSec)
			}
		}
	}
}

func TestHandleUpdateAIConfig_InvalidTimeout(t *testing.T) {
	srv := newFakeTestServer()

	body := `{"name":"ollama","timeout_sec":999}`
	req := httptest.NewRequest("PUT", "/api/v1/ai/config", strings.NewReader(body))
	req.Header.Set("Content-Type", "application/json")
	rec := httptest.NewRecorder()

	srv.handleUpdateAIConfig(rec, req)

	if rec.Code != http.StatusBadRequest {
		t.Fatalf("expected 400, got %d", rec.Code)
	}
}

func TestHandleUpdateAIConfig_UnknownProvider(t *testing.T) {
	srv := newFakeTestServer()

	body := `{"name":"unknown"}`
	req := httptest.NewRequest("PUT", "/api/v1/ai/config", strings.NewReader(body))
	req.Header.Set("Content-Type", "application/json")
	rec := httptest.NewRecorder()

	srv.handleUpdateAIConfig(rec, req)

	if rec.Code != http.StatusBadRequest {
		t.Fatalf("expected 400, got %d", rec.Code)
	}
}

func TestHandleGetAIHealth(t *testing.T) {
	srv := newFakeTestServer()
	srv.aiConfig.RecordSuccess("ollama", 50*time.Millisecond)

	req := httptest.NewRequest("GET", "/api/v1/ai/health", nil)
	rec := httptest.NewRecorder()

	srv.handleGetAIHealth(rec, req)

	if rec.Code != http.StatusOK {
		t.Fatalf("expected 200, got %d", rec.Code)
	}

	var health []ProviderHealth
	if err := json.Unmarshal(rec.Body.Bytes(), &health); err != nil {
		t.Fatalf("decode: %v", err)
	}
	if len(health) != 2 {
		t.Errorf("expected 2 health entries, got %d", len(health))
	}
}

// [REQ:P1-003a] Business logic - provider config affects generation behavior
func TestGenerateWithConfig_DisabledProvider(t *testing.T) {
	primary := &fakeAIProvider{name: "ollama", result: "should not use"}
	fallback := &fakeAIProvider{name: "openrouter", result: "ls -la"}

	srv := &Server{
		sessions:  NewSessionManagerWithFactory(newFakePTYFactory()),
		events:    NewEventLogger(100),
		metrics:   NewMetrics(),
		aiChain:   NewAIProviderChain(primary, fallback),
		shortcuts: NewShortcutProfileStore(),
		aiConfig:  NewAIProviderConfigStore(),
		workspace: NewMemWorkspaceStore(),
	}

	// Disable ollama
	srv.aiConfig.UpdateConfig("ollama", false, 1, 30, 0)

	cmd, provider, err := srv.generateWithConfig(context.Background(), "test")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if provider != "openrouter" {
		t.Errorf("expected openrouter (ollama disabled), got %q", provider)
	}
	if cmd != "ls -la" {
		t.Errorf("expected 'ls -la', got %q", cmd)
	}
	if primary.called {
		t.Error("disabled provider should not be called")
	}
}

// [REQ:P1-003b] Performance test - status updates <1s
func BenchmarkGetHealth(b *testing.B) {
	store := NewAIProviderConfigStore()
	store.RecordSuccess("ollama", 100*time.Millisecond)
	store.RecordError("openrouter")

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		store.GetHealth()
	}
}
