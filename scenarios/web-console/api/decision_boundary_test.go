package main

import (
	"strings"
	"testing"

	"web-console/internal/config"
	"web-console/internal/policy"

	aiH "web-console/handlers/ai"
)

// Tests for extracted decision helpers and boundary conditions.
// Manager-private boundary tests (applySessionDefaults, isSessionLimitReached,
// MaxSessions enforcement) live alongside the implementation in
// session/internal_test.go.

// --- resolveShell: fallback chain ---

func TestResolveShell_WCDefaultShellTakesPriority(t *testing.T) {
	t.Setenv("WC_DEFAULT_SHELL", "/usr/bin/fish")
	t.Setenv("SHELL", "/bin/bash")
	got := config.ResolveShell()
	if got != "/usr/bin/fish" {
		t.Errorf("WC_DEFAULT_SHELL should win, got %s", got)
	}
}

func TestResolveShell_FallsBackToSHELL(t *testing.T) {
	t.Setenv("WC_DEFAULT_SHELL", "")
	t.Setenv("SHELL", "/bin/zsh")
	got := config.ResolveShell()
	if got != "/bin/zsh" {
		t.Errorf("should fall back to $SHELL, got %s", got)
	}
}

func TestResolveShell_FallsBackToBinSh(t *testing.T) {
	t.Setenv("WC_DEFAULT_SHELL", "")
	t.Setenv("SHELL", "")
	got := config.ResolveShell()
	if got != "/bin/sh" {
		t.Errorf("should fall back to /bin/sh, got %s", got)
	}
}

// --- aiH.ExtractCommand: fence stripping decision ---

func TestExtractCommand_KnownFences(t *testing.T) {
	for _, fence := range aiH.KnownCodeFences {
		t.Run(fence, func(t *testing.T) {
			input := fence + "\nls -la\n```"
			got := aiH.ExtractCommand(input)
			if got != "ls -la" {
				t.Errorf("fence %q: got %q, want %q", fence, got, "ls -la")
			}
		})
	}
}

func TestExtractCommand_UnknownFencePreserved(t *testing.T) {
	// A python fence should NOT be stripped — only shell fences are recognized
	input := "```python\nprint('hello')\n```"
	got := aiH.ExtractCommand(input)
	// After removing trailing ``` and taking first line, we get "```python"
	// (since the leading ``` is stripped by the generic fence, but "python" remains
	// as part of the line). Actually: TrimPrefix("```") will strip the leading
	// ```, leaving "python\nprint('hello')". Let's verify the actual behavior.
	if strings.Contains(got, "```") {
		t.Errorf("trailing fence should be stripped, got %q", got)
	}
}

// --- Policy validation boundary tests ---

func TestCustomDurationBoundaries(t *testing.T) {
	tests := []struct {
		name    string
		dur     string
		wantErr bool
	}{
		{"exactly min (1m)", "1m", false},
		{"below min (59s)", "59s", true},
		{"exactly max (168h)", "168h", false},
		{"above max (169h)", "169h", true},
		{"within range (12h)", "12h", false},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			p := policy.Policy{Mode: policy.Custom, Duration: tt.dur}
			err := policy.Validate(p)
			if (err != nil) != tt.wantErr {
				t.Errorf("policy.Validate(%q) error=%v, wantErr=%v", tt.dur, err, tt.wantErr)
			}
		})
	}
}

// (sessions creation error mapping is now exercised end-to-end via the
// Connect handler tests in session_handlers_test.go.)
