package ai

import (
	"strings"
	"testing"
)

func TestShellBasename(t *testing.T) {
	tests := []struct {
		input string
		want  string
	}{
		{"/bin/bash", "bash"},
		{"/usr/bin/zsh", "zsh"},
		{"bash", "bash"},
		{"", ""},
	}
	for _, tt := range tests {
		got := shellBasename(tt.input)
		if got != tt.want {
			t.Errorf("shellBasename(%q) = %q, want %q", tt.input, got, tt.want)
		}
	}
}

func TestDiscoverSystemContextAndPromptFormatting(t *testing.T) {
	t.Setenv("SHELL", "/bin/fish")
	t.Setenv("WC_EXTRA_TOOLS", "custom-one, missing, custom-two")
	available := map[string]bool{"rg": true, "git": true, "custom-one": true, "custom-two": true}
	ctx := DiscoverSystemContext(func(tool string) (string, error) {
		if available[tool] {
			return "/usr/bin/" + tool, nil
		}
		return "", errToolMissing{}
	})
	if ctx.Shell != "fish" || len(ctx.Tools["Search"]) != 1 || len(ctx.Tools["Custom"]) != 2 {
		t.Fatalf("context = %#v", ctx)
	}
	if CountFoundTools(ctx.Tools) != 4 {
		t.Fatalf("CountFoundTools = %d, want 4", CountFoundTools(ctx.Tools))
	}
	prompt := BuildCommandSystemPrompt(ctx)
	if prompt == CommandSystemPrompt || !containsAll(prompt, "Available tools:", "Search: rg", "Custom: custom-one, custom-two", CommandSystemPrompt) {
		t.Fatalf("enriched prompt = %q", prompt)
	}
	if BuildSuggestSystemPrompt(nil) != SuggestSystemPrompt {
		t.Fatal("nil suggest context did not return base prompt")
	}
}

type errToolMissing struct{}

func (errToolMissing) Error() string { return "missing" }

func containsAll(s string, parts ...string) bool {
	for _, part := range parts {
		if !strings.Contains(s, part) {
			return false
		}
	}
	return true
}
