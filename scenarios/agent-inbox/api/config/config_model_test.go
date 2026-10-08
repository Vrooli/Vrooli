package config

import (
	"context"
	"errors"
	"testing"
)

func TestResolveModel_UsesExplicitOverride(t *testing.T) {
	c := AIConfig{DefaultRole: "chat.default", DefaultModel: "vendor/explicit-model"}

	// The exec seam must NOT be invoked when an explicit override is set.
	prev := resolveRoleModelExec
	resolveRoleModelExec = func(context.Context, string) (string, error) {
		t.Fatal("exec seam should not run when DefaultModel override is set")
		return "", nil
	}
	defer func() { resolveRoleModelExec = prev }()

	got, err := c.ResolveModel(context.Background())
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if got != "vendor/explicit-model" {
		t.Fatalf("expected explicit override, got %q", got)
	}
}

func TestResolveModel_ResolvesRoleViaSeam(t *testing.T) {
	c := AIConfig{DefaultRole: "chat.quality"}

	prev := resolveRoleModelExec
	resolveRoleModelExec = func(_ context.Context, role string) (string, error) {
		if role != "chat.quality" {
			t.Fatalf("expected role chat.quality, got %q", role)
		}
		return "vendor/resolved-model\n", nil
	}
	defer func() { resolveRoleModelExec = prev }()

	got, err := c.ResolveModel(context.Background())
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if got != "vendor/resolved-model" {
		t.Fatalf("expected trimmed resolved slug, got %q", got)
	}
}

func TestResolveModel_FailsClearlyWhenResourceUnavailable(t *testing.T) {
	c := AIConfig{DefaultRole: "chat.default"}

	prev := resolveRoleModelExec
	sentinel := errors.New("resource-openrouter not found")
	resolveRoleModelExec = func(context.Context, string) (string, error) {
		return "", sentinel
	}
	defer func() { resolveRoleModelExec = prev }()

	if _, err := c.ResolveModel(context.Background()); err == nil {
		t.Fatal("expected an error when the resource is unavailable and no override is set")
	}
}

func TestResolveModel_FailsOnEmptySlug(t *testing.T) {
	c := AIConfig{DefaultRole: "chat.default"}

	prev := resolveRoleModelExec
	resolveRoleModelExec = func(context.Context, string) (string, error) {
		return "  ", nil
	}
	defer func() { resolveRoleModelExec = prev }()

	if _, err := c.ResolveModel(context.Background()); err == nil {
		t.Fatal("expected an error when the resolver returns an empty slug")
	}
}
