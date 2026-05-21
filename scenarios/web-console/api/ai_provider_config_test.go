package main

import (
	"context"
	"testing"
	"time"
	"web-console/internal/events"
	"web-console/internal/metrics"
	"web-console/internal/ptyfake"

	intai "web-console/internal/ai"

	"connectrpc.com/connect"

	aiv1 "github.com/vrooli/vrooli/packages/proto/gen/go/web-console/v1/ai"

	intworkspace "web-console/internal/workspace"
)

// [REQ:P1-003a] Provider Configuration Storage - store tests

func TestAIProviderConfigStore_Defaults(t *testing.T) {
	store := intai.NewMemConfigStore()
	configs := store.GetConfigs()
	if len(configs) != 2 {
		t.Fatalf("expected 2 providers, got %d", len(configs))
	}
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
	store := intai.NewMemConfigStore()

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

	if store.UpdateConfig("unknown", true, 1, 30, 0) {
		t.Error("expected update of unknown provider to fail")
	}
}

func TestAIProviderConfigStore_IsEnabled(t *testing.T) {
	store := intai.NewMemConfigStore()
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
	store := intai.NewMemConfigStore()
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
	store := intai.NewMemConfigStore()

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

// [REQ:P1-003a] Provider Configuration Storage - Connect handler tests

func TestConnect_GetAIConfig(t *testing.T) {
	srv := newFakeTestServer()
	h := newAIConnectHandlerForServer(srv)

	resp, err := h.GetConfig(context.Background(), connect.NewRequest(&aiv1.GetConfigRequest{}))
	if err != nil {
		t.Fatalf("GetConfig: %v", err)
	}
	if len(resp.Msg.GetProviders()) != 2 {
		t.Errorf("expected 2 providers, got %d", len(resp.Msg.GetProviders()))
	}
	if len(resp.Msg.GetHealth()) != 2 {
		t.Errorf("expected 2 health entries, got %d", len(resp.Msg.GetHealth()))
	}
}

func TestConnect_UpdateAIConfig(t *testing.T) {
	srv := newFakeTestServer()
	h := newAIConnectHandlerForServer(srv)

	enabled := false
	timeout := int32(10)
	resp, err := h.UpdateConfig(context.Background(), connect.NewRequest(&aiv1.UpdateConfigRequest{
		Name:          "ollama",
		Enabled:       enabled,
		HasEnabled:    true,
		TimeoutSec:    timeout,
		HasTimeoutSec: true,
	}))
	if err != nil {
		t.Fatalf("UpdateConfig: %v", err)
	}
	for _, p := range resp.Msg.GetProviders() {
		if p.GetName() == "ollama" {
			if p.GetEnabled() {
				t.Error("expected ollama disabled")
			}
			if p.GetTimeoutSec() != 10 {
				t.Errorf("expected timeout 10, got %d", p.GetTimeoutSec())
			}
		}
	}
}

func TestConnect_UpdateAIConfig_InvalidTimeout(t *testing.T) {
	srv := newFakeTestServer()
	h := newAIConnectHandlerForServer(srv)

	_, err := h.UpdateConfig(context.Background(), connect.NewRequest(&aiv1.UpdateConfigRequest{
		Name:          "ollama",
		TimeoutSec:    999,
		HasTimeoutSec: true,
	}))
	if err == nil {
		t.Fatal("expected error for invalid timeout")
	}
	if got := connect.CodeOf(err); got != connect.CodeInvalidArgument {
		t.Errorf("expected CodeInvalidArgument, got %v", got)
	}
}

func TestConnect_UpdateAIConfig_UnknownProvider(t *testing.T) {
	srv := newFakeTestServer()
	h := newAIConnectHandlerForServer(srv)

	_, err := h.UpdateConfig(context.Background(), connect.NewRequest(&aiv1.UpdateConfigRequest{
		Name: "unknown",
	}))
	if err == nil {
		t.Fatal("expected error for unknown provider")
	}
	if got := connect.CodeOf(err); got != connect.CodeInvalidArgument {
		t.Errorf("expected CodeInvalidArgument, got %v", got)
	}
}

func TestConnect_GetAIHealth(t *testing.T) {
	srv := newFakeTestServer()
	srv.aiConfig.RecordSuccess("ollama", 50*time.Millisecond)

	h := newAIConnectHandlerForServer(srv)
	resp, err := h.GetHealth(context.Background(), connect.NewRequest(&aiv1.GetHealthRequest{}))
	if err != nil {
		t.Fatalf("GetHealth: %v", err)
	}
	if len(resp.Msg.GetHealth()) != 2 {
		t.Errorf("expected 2 health entries, got %d", len(resp.Msg.GetHealth()))
	}
}

// [REQ:P1-003a] Business logic - provider config affects generation behavior
func TestGenerateWithConfig_DisabledProvider(t *testing.T) {
	primary := &fakeAIProvider{name: "ollama", result: "should not use"}
	fallback := &fakeAIProvider{name: "openrouter", result: "ls -la"}

	srv := &Server{
		sessions:  newSessionManagerWithFactory(ptyfake.NewFactory()),
		events:    events.NewLogger(100),
		metrics:   metrics.New(),
		aiChain:   intai.NewChain(primary, fallback),
		shortcuts: NewShortcutProfileStore(),
		aiConfig:  intai.NewMemConfigStore(),
		workspace: intworkspace.NewMemStore(),
	}
	srv.ai = intai.NewService(srv.aiChain, srv.aiConfig, nil, srv.events, &srv.metrics.AIGenerations, &srv.metrics.AISuggestions)

	srv.aiConfig.UpdateConfig("ollama", false, 1, 30, 0)

	cmd, provider, err := srv.ai.Execute(context.Background(), intai.CommandSystemPrompt, "test")
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
	store := intai.NewMemConfigStore()
	store.RecordSuccess("ollama", 100*time.Millisecond)
	store.RecordError("openrouter")

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		store.GetHealth()
	}
}
