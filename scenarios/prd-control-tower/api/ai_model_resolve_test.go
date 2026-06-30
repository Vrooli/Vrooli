package main

import (
	"context"
	"errors"
	"os"
	"testing"
)

// TestMain installs a deterministic stub for the OpenRouter role resolver so the
// rest of the suite stays hermetic: tests that exercise the default-model path
// (no explicit per-request override) do not need a live resource-openrouter
// binary. Tests that specifically validate resolution behavior override the
// seam locally and restore it.
func TestMain(m *testing.M) {
	prev := resolveOpenRouterModelExec
	resolveOpenRouterModelExec = func(context.Context, string) (string, error) {
		return "vendor/test-default-model", nil
	}
	code := m.Run()
	resolveOpenRouterModelExec = prev
	os.Exit(code)
}

func TestResolveOpenRouterModel_ExplicitOverrideWins(t *testing.T) {
	prev := resolveOpenRouterModelExec
	resolveOpenRouterModelExec = func(context.Context, string) (string, error) {
		t.Fatal("exec seam should not run when an explicit override is provided")
		return "", nil
	}
	defer func() { resolveOpenRouterModelExec = prev }()

	got, err := resolveOpenRouterModel("openrouter/vendor/explicit-model")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if got != "vendor/explicit-model" {
		t.Fatalf("expected the override with the openrouter/ prefix stripped, got %q", got)
	}
}

func TestResolveOpenRouterModel_ResolvesRoleViaSeam(t *testing.T) {
	t.Setenv("PRD_OPENROUTER_ROLE", "agent.tools")

	prev := resolveOpenRouterModelExec
	resolveOpenRouterModelExec = func(_ context.Context, role string) (string, error) {
		if role != "agent.tools" {
			t.Fatalf("expected role agent.tools, got %q", role)
		}
		return "vendor/resolved-model\n", nil
	}
	defer func() { resolveOpenRouterModelExec = prev }()

	got, err := resolveOpenRouterModel("")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if got != "vendor/resolved-model" {
		t.Fatalf("expected trimmed resolved slug, got %q", got)
	}
}

func TestResolveOpenRouterModel_FailsClearlyWhenResourceUnavailable(t *testing.T) {
	prev := resolveOpenRouterModelExec
	resolveOpenRouterModelExec = func(context.Context, string) (string, error) {
		return "", errors.New("resource-openrouter not found")
	}
	defer func() { resolveOpenRouterModelExec = prev }()

	if _, err := resolveOpenRouterModel(""); err == nil {
		t.Fatal("expected an error when the resource is unavailable and no override is set")
	}
}

func TestPRDOpenRouterRole_DefaultAndOverride(t *testing.T) {
	t.Setenv("PRD_OPENROUTER_ROLE", "")
	if got := prdOpenRouterRole(); got != "chat.default" {
		t.Fatalf("expected default role chat.default, got %q", got)
	}
	t.Setenv("PRD_OPENROUTER_ROLE", "chat.quality")
	if got := prdOpenRouterRole(); got != "chat.quality" {
		t.Fatalf("expected chat.quality, got %q", got)
	}
}
