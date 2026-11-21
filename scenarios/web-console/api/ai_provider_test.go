package main

import (
	"context"
	"fmt"
	"testing"

	intai "web-console/internal/ai"
)

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
