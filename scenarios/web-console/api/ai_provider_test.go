package main

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"net/http/httptest"
	"testing"

	intai "web-console/internal/ai"
)

func TestOllamaProvider_UsesChat(t *testing.T) {
	var gotPath string
	var gotBody map[string]any
	ts := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		gotPath = r.URL.Path
		_ = json.NewDecoder(r.Body).Decode(&gotBody)
		_ = json.NewEncoder(w).Encode(map[string]any{
			"message": map[string]string{"content": "test response"},
		})
	}))
	defer ts.Close()

	p := &intai.OllamaProvider{
		BaseURL: ts.URL,
		Model:   "test-model",
		Client:  ts.Client(),
	}

	result, err := p.Generate(context.Background(), "system prompt", "user prompt")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if gotPath != "/api/chat" {
		t.Errorf("expected /api/chat, got %q", gotPath)
	}
	if result != "test response" {
		t.Errorf("expected %q, got %q", "test response", result)
	}

	messages, ok := gotBody["messages"].([]any)
	if !ok || len(messages) != 2 {
		t.Fatalf("expected 2 messages, got %v", gotBody["messages"])
	}
	sysMsg := messages[0].(map[string]any)
	if sysMsg["role"] != "system" || sysMsg["content"] != "system prompt" {
		t.Errorf("unexpected system message: %v", sysMsg)
	}
	userMsg := messages[1].(map[string]any)
	if userMsg["role"] != "user" || userMsg["content"] != "user prompt" {
		t.Errorf("unexpected user message: %v", userMsg)
	}
}

func TestOpenRouterProvider_MissingAPIKey(t *testing.T) {
	noKey := &intai.OpenRouterProvider{Model: "test"}
	_, err := noKey.Generate(context.Background(), "sys", "user")
	if err == nil {
		t.Error("expected error with no API key")
	}
}

func TestAIProviderChain_PassesBothPrompts(t *testing.T) {
	var capturedSys, capturedUser string
	provider := &contextCapturingProvider{
		name:   "test",
		result: "ok",
		capture: func(sys, user string) {
			capturedSys = sys
			capturedUser = user
		},
	}

	chain := intai.NewChain(provider)
	_, _, err := chain.Generate(context.Background(), "my system", "my user")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if capturedSys != "my system" {
		t.Errorf("system prompt = %q, want %q", capturedSys, "my system")
	}
	if capturedUser != "my user" {
		t.Errorf("user prompt = %q, want %q", capturedUser, "my user")
	}
}

func TestAIProviderChain_Failover(t *testing.T) {
	failing := &fakeAIProvider{name: "a", err: fmt.Errorf("fail")}
	working := &fakeAIProvider{name: "b", result: "ok"}

	chain := intai.NewChain(failing, working)
	result, provider, err := chain.Generate(context.Background(), "sys", "user")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if result != "ok" || provider != "b" {
		t.Errorf("got (%q, %q), want (ok, b)", result, provider)
	}
	if !failing.called || !working.called {
		t.Error("both providers should have been called")
	}
}
