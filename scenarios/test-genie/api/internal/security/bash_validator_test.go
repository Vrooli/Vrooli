package security

import (
	"testing"
)

func TestMatchCommandToAllowlist(t *testing.T) {
	validator := NewBashCommandValidator()

	tests := []struct {
		name        string
		pattern     string
		wantMatched bool
		wantHasGlob bool
		wantPrefix  string // Only checked if wantMatched is true
	}{
		{
			name:        "Exact match - ls",
			pattern:     "ls",
			wantMatched: true,
			wantHasGlob: false,
			wantPrefix:  "ls",
		},
		{
			name:        "Prefix match with space - ls -la",
			pattern:     "ls -la",
			wantMatched: true,
			wantHasGlob: false,
			wantPrefix:  "ls",
		},
		{
			name:        "Case insensitive match",
			pattern:     "LS -LA",
			wantMatched: true,
			wantHasGlob: false,
			wantPrefix:  "ls",
		},
		{
			name:        "No match - forbidden command",
			pattern:     "rm -rf /",
			wantMatched: false,
			wantHasGlob: false,
		},
		{
			name:        "Pattern with glob",
			pattern:     "bats *.bats",
			wantMatched: true,
			wantHasGlob: true,
			wantPrefix:  "bats",
		},
		{
			name:        "Pattern with question mark glob",
			pattern:     "ls file?.txt",
			wantMatched: true,
			wantHasGlob: true,
			wantPrefix:  "ls",
		},
		{
			name:        "Whitespace is trimmed",
			pattern:     "  ls  ",
			wantMatched: true,
			wantHasGlob: false,
			wantPrefix:  "ls",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := validator.MatchCommandToAllowlist(tt.pattern)

			if result.Matched != tt.wantMatched {
				t.Errorf("Matched = %v, want %v", result.Matched, tt.wantMatched)
			}

			if result.HasGlob != tt.wantHasGlob {
				t.Errorf("HasGlob = %v, want %v", result.HasGlob, tt.wantHasGlob)
			}

			if tt.wantMatched && result.MatchedPrefix != tt.wantPrefix {
				t.Errorf("MatchedPrefix = %q, want %q", result.MatchedPrefix, tt.wantPrefix)
			}
		})
	}
}

func TestValidateGlobUsage(t *testing.T) {
	tests := []struct {
		name        string
		matchResult CommandMatchResult
		allowGlobs  bool
		wantAllowed bool
	}{
		{
			name: "No glob - always allowed",
			matchResult: CommandMatchResult{
				Matched:       true,
				MatchedPrefix: "cat",
				HasGlob:       false,
				GlobSafe:      false,
			},
			allowGlobs:  true,
			wantAllowed: true,
		},
		{
			name: "Glob with safe command - allowed",
			matchResult: CommandMatchResult{
				Matched:       true,
				MatchedPrefix: "bats",
				HasGlob:       true,
				GlobSafe:      true,
			},
			allowGlobs:  true,
			wantAllowed: true,
		},
		{
			name: "Glob with unsafe command - not allowed",
			matchResult: CommandMatchResult{
				Matched:       true,
				MatchedPrefix: "cat",
				HasGlob:       true,
				GlobSafe:      false,
			},
			allowGlobs:  true,
			wantAllowed: false,
		},
		{
			name: "Glob disabled globally - not allowed even for safe command",
			matchResult: CommandMatchResult{
				Matched:       true,
				MatchedPrefix: "bats",
				HasGlob:       true,
				GlobSafe:      true,
			},
			allowGlobs:  false,
			wantAllowed: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			cfg := DefaultSecurityConfig()
			cfg.AllowGlobPatterns = tt.allowGlobs
			validator := NewBashCommandValidator(WithSecurityConfig(cfg))

			result := validator.ValidateGlobUsage(tt.matchResult)

			if result.Allowed != tt.wantAllowed {
				t.Errorf("Allowed = %v, want %v (reason: %s)", result.Allowed, tt.wantAllowed, result.Reason)
			}

			// All results should have a reason
			if result.Reason == "" {
				t.Error("Expected non-empty Reason")
			}
		})
	}
}

func TestValidateBashPatternIntegration(t *testing.T) {
	validator := NewBashCommandValidator()

	tests := []struct {
		name        string
		pattern     string
		wantErr     bool
		errContains string
	}{
		{
			name:    "Valid simple command",
			pattern: "ls -la",
			wantErr: false,
		},
		{
			name:    "Valid test command",
			pattern: "go test ./...",
			wantErr: false,
		},
		{
			name:    "Valid glob with safe command",
			pattern: "bats *.bats",
			wantErr: false,
		},
		{
			name:        "Invalid command - not in allowlist",
			pattern:     "rm -rf /",
			wantErr:     true,
			errContains: "not in the allowlist",
		},
		{
			name:        "Empty pattern",
			pattern:     "",
			wantErr:     true,
			errContains: "empty bash pattern",
		},
		{
			name:        "Whitespace only pattern",
			pattern:     "   ",
			wantErr:     true,
			errContains: "empty bash pattern",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := validator.ValidateBashPattern(tt.pattern)

			if tt.wantErr && err == nil {
				t.Error("Expected error but got nil")
			}

			if !tt.wantErr && err != nil {
				t.Errorf("Expected no error but got: %v", err)
			}

			if tt.wantErr && tt.errContains != "" && err != nil {
				if !containsBashStr(err.Error(), tt.errContains) {
					t.Errorf("Error %q should contain %q", err.Error(), tt.errContains)
				}
			}
		})
	}
}

func containsBashStr(s, substr string) bool {
	for i := 0; i <= len(s)-len(substr); i++ {
		if s[i:i+len(substr)] == substr {
			return true
		}
	}
	return false
}
