package domain

import (
	"testing"
)

func TestExtraFlags_StructuralValidation(t *testing.T) {
	tests := []struct {
		name    string
		profile *AgentProfile
		wantErr bool
		errMsg  string
	}{
		{
			name: "nil extra flags is valid",
			profile: &AgentProfile{
				Name: "test",

				ExtraFlags: nil, RoleRef: "code.default",
			},
			wantErr: false,
		},
		{
			name: "empty extra flags is valid",
			profile: &AgentProfile{
				Name: "test",

				ExtraFlags: RunnerExtraFlags{}, RoleRef: "code.default",
			},
			wantErr: false,
		},
		{
			name: "valid extra flags",
			profile: &AgentProfile{
				Name: "test",

				ExtraFlags: RunnerExtraFlags{
					RunnerTypeClaudeCode: []string{"--verbose"},
				}, RoleRef: "code.default",
			},
			wantErr: false,
		},
		{
			name: "invalid runner type key",
			profile: &AgentProfile{
				Name: "test",

				ExtraFlags: RunnerExtraFlags{
					RunnerType("bogus"): []string{"--flag"},
				}, RoleRef: "code.default",
			},
			wantErr: true,
			errMsg:  "extraFlags",
		},
		{
			name: "too many flags (over 20)",
			profile: &AgentProfile{
				Name: "test",

				ExtraFlags: RunnerExtraFlags{
					RunnerTypeClaudeCode: func() []string {
						flags := make([]string, 21)
						for i := range flags {
							flags[i] = "--flag"
						}
						return flags
					}(),
				}, RoleRef: "code.default",
			},
			wantErr: true,
			errMsg:  "extraFlags",
		},
		{
			name: "empty flag rejected",
			profile: &AgentProfile{
				Name: "test",

				ExtraFlags: RunnerExtraFlags{
					RunnerTypeClaudeCode: []string{""},
				}, RoleRef: "code.default",
			},
			wantErr: true,
			errMsg:  "extraFlags",
		},
		{
			name: "whitespace-only flag rejected",
			profile: &AgentProfile{
				Name: "test",

				ExtraFlags: RunnerExtraFlags{
					RunnerTypeClaudeCode: []string{"   "},
				}, RoleRef: "code.default",
			},
			wantErr: true,
			errMsg:  "extraFlags",
		},
		{
			name: "flag without dash prefix rejected",
			profile: &AgentProfile{
				Name: "test",

				ExtraFlags: RunnerExtraFlags{
					RunnerTypeClaudeCode: []string{"noDash"},
				}, RoleRef: "code.default",
			},
			wantErr: true,
			errMsg:  "extraFlags",
		},
		{
			name: "pipe shell metacharacter rejected",
			profile: &AgentProfile{
				Name: "test",

				ExtraFlags: RunnerExtraFlags{
					RunnerTypeClaudeCode: []string{"--flag|evil"},
				}, RoleRef: "code.default",
			},
			wantErr: true,
			errMsg:  "extraFlags",
		},
		{
			name: "semicolon shell metacharacter rejected",
			profile: &AgentProfile{
				Name: "test",

				ExtraFlags: RunnerExtraFlags{
					RunnerTypeClaudeCode: []string{"--flag;rm"},
				}, RoleRef: "code.default",
			},
			wantErr: true,
			errMsg:  "extraFlags",
		},
		{
			name: "backtick shell metacharacter rejected",
			profile: &AgentProfile{
				Name: "test",

				ExtraFlags: RunnerExtraFlags{
					RunnerTypeClaudeCode: []string{"--flag`cmd`"},
				}, RoleRef: "code.default",
			},
			wantErr: true,
			errMsg:  "extraFlags",
		},
		{
			name: "dollar sign shell metacharacter rejected",
			profile: &AgentProfile{
				Name: "test",

				ExtraFlags: RunnerExtraFlags{
					RunnerTypeClaudeCode: []string{"--flag$HOME"},
				}, RoleRef: "code.default",
			},
			wantErr: true,
			errMsg:  "extraFlags",
		},
		{
			name: "flag with equals value is valid",
			profile: &AgentProfile{
				Name: "test",

				ExtraFlags: RunnerExtraFlags{
					RunnerTypeClaudeCode: []string{"--timeout=30"},
				}, RoleRef: "code.default",
			},
			wantErr: false,
		},
		{
			name: "multiple runner types valid",
			profile: &AgentProfile{
				Name: "test",

				ExtraFlags: RunnerExtraFlags{
					RunnerTypeClaudeCode: []string{"--verbose"},
					RunnerTypeCodex:      []string{"--verbose"},
				}, RoleRef: "code.default",
			},
			wantErr: false,
		},
		{
			name: "exactly 20 flags is valid",
			profile: &AgentProfile{
				Name: "test",

				ExtraFlags: RunnerExtraFlags{
					RunnerTypeClaudeCode: func() []string {
						flags := make([]string, 20)
						for i := range flags {
							flags[i] = "--flag"
						}
						return flags
					}(),
				}, RoleRef: "code.default",
			},
			wantErr: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := tt.profile.Validate()
			if (err != nil) != tt.wantErr {
				t.Errorf("Validate() error = %v, wantErr %v", err, tt.wantErr)
			}
			if tt.wantErr && err != nil {
				if ve, ok := err.(*ValidationError); ok {
					if tt.errMsg != "" && !containsSubstring(ve.Field, tt.errMsg) {
						t.Errorf("Validate() error field = %v, want containing %v", ve.Field, tt.errMsg)
					}
				}
			}
		})
	}
}

// =============================================================================
// PROFILE VALIDATE WITH FEATURES TESTS
// =============================================================================

func TestAgentProfile_Validate_WithFeatures(t *testing.T) {
	tests := []struct {
		name    string
		profile *AgentProfile
		wantErr bool
	}{
		{
			name: "profile with EnableBrowser true is valid",
			profile: &AgentProfile{
				Name: "test",

				Features: FeatureFlags{EnableBrowser: true}, RoleRef: "code.default",
			},
			wantErr: false,
		},
		{
			name: "profile with EnableBrowser false is valid",
			profile: &AgentProfile{
				Name: "test",

				Features: FeatureFlags{EnableBrowser: false}, RoleRef: "code.default",
			},
			wantErr: false,
		},
		{
			name: "profile with zero features is valid",
			profile: &AgentProfile{
				Name: "test",

				Features: FeatureFlags{}, RoleRef: "code.default",
			},
			wantErr: false,
		},
		{
			name: "profile with features and valid extra flags",
			profile: &AgentProfile{
				Name: "test",

				Features: FeatureFlags{EnableBrowser: true},
				ExtraFlags: RunnerExtraFlags{
					RunnerTypeClaudeCode: []string{"--verbose"},
				}, RoleRef: "code.default",
			},
			wantErr: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := tt.profile.Validate()
			if (err != nil) != tt.wantErr {
				t.Errorf("Validate() error = %v, wantErr %v", err, tt.wantErr)
			}
		})
	}
}

// =============================================================================
// RUNCONFIG APPLYPROFILE DEEP COPY TESTS
// =============================================================================

func TestRunConfig_ApplyProfile_CopiesFeatures(t *testing.T) {
	t.Run("features are copied by value", func(t *testing.T) {
		profile := &AgentProfile{
			Name: "test",

			Features: FeatureFlags{EnableBrowser: true}, RoleRef: "code.default",
		}
		cfg := DefaultRunConfig()
		cfg.ApplyProfile(profile)

		if !cfg.Features.EnableBrowser {
			t.Error("expected Features.EnableBrowser to be true after ApplyProfile")
		}

		// Mutate profile features - should not affect config
		profile.Features.EnableBrowser = false
		if !cfg.Features.EnableBrowser {
			t.Error("expected config Features to remain unchanged after mutating profile")
		}
	})

	t.Run("extra flags are deep copied", func(t *testing.T) {
		profile := &AgentProfile{
			Name: "test",

			ExtraFlags: RunnerExtraFlags{
				RunnerTypeClaudeCode: []string{"--verbose", "--allowedTools"},
			}, RoleRef: "code.default",
		}
		cfg := DefaultRunConfig()
		cfg.ApplyProfile(profile)

		if len(cfg.ExtraFlags) != 1 {
			t.Fatalf("expected 1 runner type in ExtraFlags, got %d", len(cfg.ExtraFlags))
		}
		if len(cfg.ExtraFlags[RunnerTypeClaudeCode]) != 2 {
			t.Fatalf("expected 2 flags, got %d", len(cfg.ExtraFlags[RunnerTypeClaudeCode]))
		}

		// Mutate profile extra flags - should not affect config
		profile.ExtraFlags[RunnerTypeClaudeCode][0] = "--mutated"
		if cfg.ExtraFlags[RunnerTypeClaudeCode][0] != "--verbose" {
			t.Errorf("expected config ExtraFlags to remain '--verbose', got %q", cfg.ExtraFlags[RunnerTypeClaudeCode][0])
		}

		// Add new flag to profile - should not affect config
		profile.ExtraFlags[RunnerTypeClaudeCode] = append(profile.ExtraFlags[RunnerTypeClaudeCode], "--extra")
		if len(cfg.ExtraFlags[RunnerTypeClaudeCode]) != 2 {
			t.Errorf("expected config ExtraFlags length to remain 2, got %d", len(cfg.ExtraFlags[RunnerTypeClaudeCode]))
		}
	})

	t.Run("nil extra flags remain nil in config", func(t *testing.T) {
		profile := &AgentProfile{
			Name: "test",

			ExtraFlags: nil, RoleRef: "code.default",
		}
		cfg := DefaultRunConfig()
		cfg.ApplyProfile(profile)

		if cfg.ExtraFlags != nil {
			t.Errorf("expected nil ExtraFlags, got %v", cfg.ExtraFlags)
		}
	})

	t.Run("nil profile is no-op", func(t *testing.T) {
		cfg := DefaultRunConfig()
		originalType := cfg.RunnerType
		cfg.ApplyProfile(nil)

		if cfg.RunnerType != originalType {
			t.Error("expected RunnerType to remain unchanged after nil profile")
		}
	})
}

// =============================================================================
// HELPER TESTS
// =============================================================================

func Test_hasStringOverlap(t *testing.T) {
	tests := []struct {
		name string
		a    []string
		b    []string
		want bool
	}{
		{"no overlap", []string{"a", "b"}, []string{"c", "d"}, false},
		{"has overlap", []string{"a", "b"}, []string{"b", "c"}, true},
		{"empty a", []string{}, []string{"a"}, false},
		{"empty b", []string{"a"}, []string{}, false},
		{"both empty", []string{}, []string{}, false},
		{"nil slices", nil, nil, false},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := hasStringOverlap(tt.a, tt.b); got != tt.want {
				t.Errorf("hasStringOverlap() = %v, want %v", got, tt.want)
			}
		})
	}
}

func Test_hasRunnerOverlap(t *testing.T) {
	tests := []struct {
		name string
		a    []RunnerType
		b    []RunnerType
		want bool
	}{
		{"no overlap", []RunnerType{RunnerTypeClaudeCode}, []RunnerType{RunnerTypeCodex}, false},
		{"has overlap", []RunnerType{RunnerTypeClaudeCode, RunnerTypeCodex}, []RunnerType{RunnerTypeCodex}, true},
		{"empty", []RunnerType{}, []RunnerType{RunnerTypeClaudeCode}, false},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := hasRunnerOverlap(tt.a, tt.b); got != tt.want {
				t.Errorf("hasRunnerOverlap() = %v, want %v", got, tt.want)
			}
		})
	}
}

// Helper function for substring matching
func containsSubstring(s, substr string) bool {
	return len(s) >= len(substr) && (s == substr || len(s) > 0 && (s[:len(substr)] == substr || containsSubstring(s[1:], substr)))
}
