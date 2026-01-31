package ai

import (
	"strings"
	"testing"

	"browser-automation-studio/cli/internal/appctx"
)

func TestNavigatorInfo_Struct(t *testing.T) {
	info := NavigatorInfo{
		Type:              "playwright",
		Available:         true,
		Description:       "AI navigation via vision models",
		AllowedSources:    []string{"ui", "cli", "api"},
		UnavailableReason: "",
		CreditPolicy: CreditPolicyInfo{
			RequiresCredits:  true,
			CreditsPerStep:   2,
			BypassConditions: []string{"byok", "resource_openrouter"},
		},
	}

	if info.Type != "playwright" {
		t.Errorf("Type = %q, want %q", info.Type, "playwright")
	}
	if !info.Available {
		t.Error("Available = false, want true")
	}
	if len(info.AllowedSources) != 3 {
		t.Errorf("AllowedSources length = %d, want 3", len(info.AllowedSources))
	}
	if !info.CreditPolicy.RequiresCredits {
		t.Error("CreditPolicy.RequiresCredits = false, want true")
	}
	if info.CreditPolicy.CreditsPerStep != 2 {
		t.Errorf("CreditPolicy.CreditsPerStep = %d, want 2", info.CreditPolicy.CreditsPerStep)
	}
}

func TestCreditPolicyInfo_Struct(t *testing.T) {
	policy := CreditPolicyInfo{
		RequiresCredits:  true,
		CreditsPerStep:   5,
		BypassConditions: []string{"byok"},
	}

	if !policy.RequiresCredits {
		t.Error("RequiresCredits = false, want true")
	}
	if policy.CreditsPerStep != 5 {
		t.Errorf("CreditsPerStep = %d, want 5", policy.CreditsPerStep)
	}
	if len(policy.BypassConditions) != 1 {
		t.Errorf("BypassConditions length = %d, want 1", len(policy.BypassConditions))
	}
}

func TestNavigatorsResponse_Struct(t *testing.T) {
	resp := NavigatorsResponse{
		Navigators: []NavigatorInfo{
			{
				Type:        "playwright",
				Available:   true,
				Description: "Playwright navigator",
			},
			{
				Type:              "claude_code",
				Available:         false,
				Description:       "Claude Code navigator",
				UnavailableReason: "not installed",
			},
		},
		Default: "playwright",
	}

	if len(resp.Navigators) != 2 {
		t.Fatalf("Navigators length = %d, want 2", len(resp.Navigators))
	}

	if resp.Default != "playwright" {
		t.Errorf("Default = %q, want %q", resp.Default, "playwright")
	}

	// First navigator should be available
	if !resp.Navigators[0].Available {
		t.Error("Navigators[0].Available = false, want true")
	}

	// Second navigator should be unavailable
	if resp.Navigators[1].Available {
		t.Error("Navigators[1].Available = true, want false")
	}
	if resp.Navigators[1].UnavailableReason != "not installed" {
		t.Errorf("Navigators[1].UnavailableReason = %q, want %q",
			resp.Navigators[1].UnavailableReason, "not installed")
	}
}

func TestRunNavigators_Help(t *testing.T) {
	ctx := &appctx.Context{
		Name:    "test-cli",
		Version: "1.0.0",
	}

	// Test --help flag
	err := runNavigators(ctx, []string{"--help"})
	if err != nil {
		t.Errorf("runNavigators(--help) error = %v, want nil", err)
	}

	// Test -h flag
	err = runNavigators(ctx, []string{"-h"})
	if err != nil {
		t.Errorf("runNavigators(-h) error = %v, want nil", err)
	}
}

func TestRunNavigators_InvalidArgs(t *testing.T) {
	ctx := &appctx.Context{
		Name:    "test-cli",
		Version: "1.0.0",
	}

	tests := []struct {
		name    string
		args    []string
		wantErr string
	}{
		{
			name:    "unknown option",
			args:    []string{"--unknown"},
			wantErr: "unknown option",
		},
		{
			name:    "unexpected argument",
			args:    []string{"extra"},
			wantErr: "unexpected argument",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := runNavigators(ctx, tt.args)
			if err == nil {
				t.Error("expected error, got nil")
				return
			}
			if !strings.Contains(err.Error(), tt.wantErr) {
				t.Errorf("error = %q, want to contain %q", err.Error(), tt.wantErr)
			}
		})
	}
}
