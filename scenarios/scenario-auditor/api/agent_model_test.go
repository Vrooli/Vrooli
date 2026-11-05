package main

import (
	"fmt"
	"strings"
	"testing"
)

// init installs a deterministic role resolver for the test binary so the rest
// of the suite does not shell out to resource-openrouter (which may be
// unavailable in CI) and gets a stable default model value.
func init() {
	resolveAgentModelRole = func(role string) (string, error) {
		return "z-ai/glm-5.2", nil
	}
}

func TestComputeAgentModelResolvesRoleToOpencodeRef(t *testing.T) {
	t.Setenv(envAgentModel, "")
	t.Setenv(envAgentRole, "")

	var gotRole string
	prev := resolveAgentModelRole
	resolveAgentModelRole = func(role string) (string, error) {
		gotRole = role
		return "z-ai/glm-5.2", nil
	}
	defer func() { resolveAgentModelRole = prev }()

	model, err := computeAgentModel()
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if gotRole != defaultAgentRole {
		t.Errorf("expected default role %q, got %q", defaultAgentRole, gotRole)
	}
	if model != "openrouter/z-ai/glm-5.2" {
		t.Errorf("expected opencode-style ref openrouter/z-ai/glm-5.2, got %q", model)
	}
}

func TestComputeAgentModelHonorsExplicitOverrideVerbatim(t *testing.T) {
	t.Setenv(envAgentModel, "openrouter/x-ai/grok-code-fast-1")
	t.Setenv(envAgentRole, "")

	resolveAgentModelRole = func(role string) (string, error) {
		t.Fatalf("policy resolver should not be called when override is set")
		return "", nil
	}
	defer func() {
		resolveAgentModelRole = func(role string) (string, error) { return "z-ai/glm-5.2", nil }
	}()

	model, err := computeAgentModel()
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if model != "openrouter/x-ai/grok-code-fast-1" {
		t.Errorf("expected override honored verbatim, got %q", model)
	}
}

func TestComputeAgentModelUsesConfiguredRole(t *testing.T) {
	t.Setenv(envAgentModel, "")
	t.Setenv(envAgentRole, "code.default")

	var gotRole string
	prev := resolveAgentModelRole
	resolveAgentModelRole = func(role string) (string, error) {
		gotRole = role
		return "anthropic/claude", nil
	}
	defer func() { resolveAgentModelRole = prev }()

	if _, err := computeAgentModel(); err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if gotRole != "code.default" {
		t.Errorf("expected configured role code.default, got %q", gotRole)
	}
}

func TestComputeAgentModelFailsWithoutResourceOrOverride(t *testing.T) {
	t.Setenv(envAgentModel, "")
	t.Setenv(envAgentRole, "")

	prev := resolveAgentModelRole
	resolveAgentModelRole = func(role string) (string, error) {
		return "", fmt.Errorf("resource-openrouter unavailable")
	}
	defer func() { resolveAgentModelRole = prev }()

	model, err := computeAgentModel()
	if err == nil {
		t.Fatalf("expected error when resolution fails and no override is set, got model %q", model)
	}
	if model != "" {
		t.Errorf("expected no concrete fallback slug, got %q", model)
	}
	if !strings.Contains(err.Error(), envAgentModel) {
		t.Errorf("expected error to mention the override env var %q, got %v", envAgentModel, err)
	}
}

func TestComputeAgentModelRejectsEmptyResolvedSlug(t *testing.T) {
	t.Setenv(envAgentModel, "")
	t.Setenv(envAgentRole, "")

	prev := resolveAgentModelRole
	resolveAgentModelRole = func(role string) (string, error) {
		return "   ", nil
	}
	defer func() { resolveAgentModelRole = prev }()

	if _, err := computeAgentModel(); err == nil {
		t.Fatalf("expected error when resolver returns an empty slug")
	}
}
