package runner

import (
	"testing"

	"agent-manager/internal/domain"
)

func TestClassifyModelError_Codex(t *testing.T) {
	cases := []struct {
		name   string
		stderr string
		want   ModelErrorKind
	}{
		{"unknown model phrasing", "error: unknown model 'gpt-5-codex'", ModelErrorUnavailable},
		{"model not found", "the model gpt-5-codex was not found", ModelErrorUnavailable},
		{"invalid model", "invalid model identifier", ModelErrorUnavailable},
		{"deprecated", "model gpt-5-codex is deprecated", ModelErrorUnavailable},
		{"no longer available", "model gpt-5-codex is no longer available", ModelErrorUnavailable},
		{"unsupported", "unsupported model 'gpt-5.2-codex'", ModelErrorUnavailable},
		{"rate limit (transient, not model)", "rate limit exceeded", ModelErrorTransient},
		{"network reset (transient)", "connection reset by peer", ModelErrorTransient},
		{"generic failure", "the agent exited unexpectedly", ModelErrorNone},
		{"empty stderr", "", ModelErrorNone},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			got := ClassifyModelError(domain.RunnerTypeCodex, tc.stderr, 1)
			if got != tc.want {
				t.Fatalf("ClassifyModelError(%q) = %v, want %v", tc.stderr, got, tc.want)
			}
		})
	}
}

func TestClassifyModelError_ClaudeCode(t *testing.T) {
	cases := []struct {
		stderr string
		want   ModelErrorKind
	}{
		{"Unknown model: sonnet-99", ModelErrorUnavailable},
		{"model not found for 'opus-99'", ModelErrorUnavailable},
		{"permission denied", ModelErrorNone},
	}
	for _, tc := range cases {
		got := ClassifyModelError(domain.RunnerTypeClaudeCode, tc.stderr, 1)
		if got != tc.want {
			t.Fatalf("ClassifyModelError(%q) = %v, want %v", tc.stderr, got, tc.want)
		}
	}
}

func TestClassifyModelError_OpenCode(t *testing.T) {
	if got := ClassifyModelError(domain.RunnerTypeOpenCode, "invalid model 'claude-bogus'", 1); got != ModelErrorUnavailable {
		t.Fatalf("expected ModelErrorUnavailable, got %v", got)
	}
	if got := ClassifyModelError(domain.RunnerTypeOpenCode, "something unrelated went wrong", 1); got != ModelErrorNone {
		t.Fatalf("expected ModelErrorNone, got %v", got)
	}
}

func TestClassifyModelError_UnknownRunnerType(t *testing.T) {
	if got := ClassifyModelError(domain.RunnerType("nonexistent"), "unknown model foo", 1); got != ModelErrorNone {
		t.Fatalf("expected unknown runner type to return ModelErrorNone, got %v", got)
	}
}
