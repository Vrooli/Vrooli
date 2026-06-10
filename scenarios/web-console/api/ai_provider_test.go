package main

import (
	"context"
	"fmt"
	"reflect"
	"testing"

	intai "web-console/internal/ai"
)

func TestOllamaProvider_UsesGatewayRole(t *testing.T) {
	var gotArgs []string
	var gotStdin string

	p := &intai.OllamaProvider{
		Role: "chat.default",
		Runner: func(_ context.Context, args []string, stdin string) ([]byte, error) {
			gotArgs = append([]string(nil), args...)
			gotStdin = stdin
			return []byte(`{"response":"test response"}`), nil
		},
	}

	result, err := p.Generate(context.Background(), "system prompt", "user prompt")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	wantArgs := []string{"gateway", "generate", "--role", "chat.default", "--json", "--prompt-stdin"}
	if !reflect.DeepEqual(gotArgs, wantArgs) {
		t.Errorf("args = %v, want %v", gotArgs, wantArgs)
	}
	if result != "test response" {
		t.Errorf("expected %q, got %q", "test response", result)
	}
	if gotStdin != "System:\nsystem prompt\n\nUser:\nuser prompt" {
		t.Errorf("stdin = %q", gotStdin)
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
