package heartbeat

import (
	"fmt"
	"testing"

	"prompt-manager/store"
)

func TestDefaultProfileKeyForSpawnMode(t *testing.T) {
	tests := []struct {
		spawnMode string
		want      string
	}{
		{"single-process", DefaultProfileKeyClaudeCode},
		{"multi-process", DefaultProfileKeyCodex},
		{"", DefaultProfileKeyCodex},
	}
	for _, tt := range tests {
		t.Run(fmt.Sprintf("spawnMode=%q", tt.spawnMode), func(t *testing.T) {
			got := DefaultProfileKeyForSpawnMode(tt.spawnMode)
			if got != tt.want {
				t.Errorf("DefaultProfileKeyForSpawnMode(%q) = %q, want %q", tt.spawnMode, got, tt.want)
			}
		})
	}
}

func TestValidateProfileCompatibility(t *testing.T) {
	tests := []struct {
		name      string
		spawnMode string
		runner    string
		wantErr   bool
	}{
		{"single-process with claude-code", "single-process", "RUNNER_TYPE_CLAUDE_CODE", false},
		{"single-process with codex", "single-process", "RUNNER_TYPE_CODEX", true},
		{"multi-process with codex", "multi-process", "RUNNER_TYPE_CODEX", false},
		{"multi-process with claude-code", "multi-process", "RUNNER_TYPE_CLAUDE_CODE", false},
		{"empty spawn mode with codex", "", "RUNNER_TYPE_CODEX", false},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			team := &store.Team{ID: "test-team", SpawnMode: tt.spawnMode}
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
		TeamID:     "t1",
		SpawnMode:  "single-process",
		ProfileKey: "k1",
		RunnerType: "RUNNER_TYPE_CODEX",
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
