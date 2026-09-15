package codecs

import (
	"strings"
	"testing"

	"agent-manager/internal/domain"
)

func TestClaudeControlArgsCarriesSkipPermissionsForInteractive(t *testing.T) {
	c := NewClaudeForTest()
	withSkip, err := c.ControlArgs(&domain.RunConfig{Model: "haiku", SkipPermissionPrompt: true})
	if err != nil {
		t.Fatal(err)
	}
	if !containsArg(withSkip, "--dangerously-skip-permissions") {
		t.Fatalf("skip permissions not forwarded to the interactive launch: %q", withSkip)
	}
	withoutSkip, err := c.ControlArgs(&domain.RunConfig{Model: "haiku"})
	if err != nil {
		t.Fatal(err)
	}
	if containsArg(withoutSkip, "--dangerously-skip-permissions") {
		t.Fatalf("skip permissions must not be added when unset: %q", withoutSkip)
	}
}

func TestCodexControlArgsFailsClosedOnUnsupportedEffort(t *testing.T) {
	c := NewCodexForTest()
	args, err := c.ControlArgs(&domain.RunConfig{Model: "gpt-5.6-luna", Effort: domain.EffortHigh})
	if err != nil {
		t.Fatal(err)
	}
	if !containsArg(args, "model_reasoning_effort=high") {
		t.Fatalf("codex high effort not translated: %q", args)
	}

	if _, err := c.ControlArgs(&domain.RunConfig{Model: "gpt-5.6-luna", Effort: domain.EffortMax}); err == nil ||
		!strings.Contains(err.Error(), "no native reasoning effort") {
		t.Fatalf("codex max effort must fail closed, got %v", err)
	}

	args, err = c.ControlArgs(&domain.RunConfig{Model: "gpt-5.6-luna"})
	if err != nil {
		t.Fatal(err)
	}
	if containsArg(args, "-c") {
		t.Fatalf("unset effort must not emit a reasoning-effort config: %q", args)
	}
}

func TestOpenCodeControlArgsUsesOnlyDocumentedProviderVariants(t *testing.T) {
	c := NewOpenCodeForTest()
	tests := []struct {
		name   string
		model  string
		effort domain.Effort
		want   bool
	}{
		{"anthropic high", "anthropic/claude-sonnet", domain.EffortHigh, true},
		{"anthropic max", "anthropic/claude-sonnet", domain.EffortMax, true},
		{"anthropic low", "anthropic/claude-sonnet", domain.EffortLow, false},
		{"openai xhigh", "openai/gpt-5", domain.EffortXHigh, true},
		{"openai max", "openai/gpt-5", domain.EffortMax, false},
		{"google low", "google/gemini", domain.EffortLow, true},
		{"google medium", "google/gemini", domain.EffortMedium, false},
		{"local provider", "ollama/qwen", domain.EffortHigh, false},
		{"missing provider", "gpt-5", domain.EffortHigh, false},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			args, err := c.ControlArgs(&domain.RunConfig{RunnerType: domain.RunnerTypeOpenCode, Model: tt.model, Effort: tt.effort})
			if tt.want {
				if err != nil || !containsArg(args, "--variant") {
					t.Fatalf("ControlArgs() = %q, %v; want documented variant", args, err)
				}
				return
			}
			if err == nil || !strings.Contains(err.Error(), "no documented variant") {
				t.Fatalf("ControlArgs() error = %v, want provider-domain rejection", err)
			}
		})
	}
}
