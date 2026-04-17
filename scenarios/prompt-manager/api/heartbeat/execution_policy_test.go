package heartbeat

import (
	"fmt"
	"prompt-manager/teamconfig"
	"testing"
)

func TestDefaultProfileKeyForRuntimeMode(t *testing.T) {
	tests := []struct {
		runtimeMode string
		want        string
	}{
		{teamconfig.RuntimeModeSingleProcess, DefaultProfileKeyClaudeCode},
		{teamconfig.RuntimeModeMultiProcess, DefaultProfileKeyCodex},
		{"", DefaultProfileKeyCodex},
	}
	for _, tt := range tests {
		t.Run(fmt.Sprintf("runtimeMode=%q", tt.runtimeMode), func(t *testing.T) {
			got := DefaultProfileKeyForRuntimeMode(tt.runtimeMode)
			if got != tt.want {
				t.Errorf("DefaultProfileKeyForRuntimeMode(%q) = %q, want %q", tt.runtimeMode, got, tt.want)
			}
		})
	}
}

func TestValidateProfileCompatibility(t *testing.T) {
	tests := []struct {
		name        string
		runtimeMode string
		runner      string
		wantErr     bool
	}{
		{"single-process with claude-code", teamconfig.RuntimeModeSingleProcess, "RUNNER_TYPE_CLAUDE_CODE", false},
		{"single-process with codex", teamconfig.RuntimeModeSingleProcess, "RUNNER_TYPE_CODEX", true},
		{"multi-process with codex", teamconfig.RuntimeModeMultiProcess, "RUNNER_TYPE_CODEX", false},
		{"multi-process with claude-code", teamconfig.RuntimeModeMultiProcess, "RUNNER_TYPE_CLAUDE_CODE", false},
		{"empty runtime mode with codex", "", "RUNNER_TYPE_CODEX", false},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			team := newIndependentTestTeam("test-team", "Test Team")
			team.Runtime.Mode = tt.runtimeMode
			profile := &AgentProfile{ProfileKey: "test-key", RunnerType: tt.runner}
			err := validateProfileCompatibility(team, profile)
			if (err != nil) != tt.wantErr {
				t.Errorf("validateProfileCompatibility() error = %v, wantErr %v", err, tt.wantErr)
			}
			if tt.wantErr && !IsProfileMismatch(err) {
				t.Errorf("expected ProfileMismatchError, got %T", err)
			}
		})
	}
}

func TestIsProfileMismatch(t *testing.T) {
	err := &ProfileMismatchError{
		TeamID:      "t1",
		RuntimeMode: teamconfig.RuntimeModeSingleProcess,
		ProfileKey:  "k1",
		RunnerType:  "RUNNER_TYPE_CODEX",
	}
	if !IsProfileMismatch(err) {
		t.Error("expected IsProfileMismatch to return true for ProfileMismatchError")
	}

	wrapped := fmt.Errorf("outer: %w", err)
	if !IsProfileMismatch(wrapped) {
		t.Error("expected IsProfileMismatch to return true for wrapped ProfileMismatchError")
	}

	if IsProfileMismatch(fmt.Errorf("unrelated error")) {
		t.Error("expected IsProfileMismatch to return false for unrelated error")
	}
}
