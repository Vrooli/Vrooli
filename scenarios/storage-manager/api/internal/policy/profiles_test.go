package policy

import (
	"testing"
	"time"

	"storage-manager/internal/cleanup"
)

// [REQ:CLN-P0-003]
func TestBuildProfileKeepsConservativeConditionalProvidersDisabled(t *testing.T) {
	t.Parallel()

	profile, err := BuildProfile(ProfileConservative, []cleanup.ProviderMetadata{
		{ID: "tmp", SafetyTier: cleanup.SafetyTierSafe, DefaultMode: cleanup.ProviderModeDisabled, DefaultApproval: cleanup.ApprovalModeOperator},
		{ID: "docker", SafetyTier: cleanup.SafetyTierConditional, DefaultMode: cleanup.ProviderModeDisabled, DefaultApproval: cleanup.ApprovalModeOperator},
	})
	if err != nil {
		t.Fatalf("BuildProfile() error = %v", err)
	}
	if profile.Defaults["docker"].Enabled {
		t.Fatal("conservative profile enabled conditional docker provider")
	}
	if profile.Defaults["docker"].ApprovalMode != cleanup.ApprovalModeOperator {
		t.Fatalf("docker approval = %q, want operator", profile.Defaults["docker"].ApprovalMode)
	}
}

func TestBuildProfileAggressiveStillDisablesForbiddenProviders(t *testing.T) {
	t.Parallel()

	profile, err := BuildProfile(ProfileAggressive, []cleanup.ProviderMetadata{
		{ID: "live-db", SafetyTier: cleanup.SafetyTierForbidden, DefaultMode: cleanup.ProviderModeDisabled, DefaultApproval: cleanup.ApprovalModeDisabled},
	})
	if err != nil {
		t.Fatalf("BuildProfile() error = %v", err)
	}
	got := profile.Defaults["live-db"]
	if got.Enabled || got.ApprovalMode != cleanup.ApprovalModeDisabled {
		t.Fatalf("forbidden default = %#v, want disabled", got)
	}
}

func TestValidateProviderPolicyRejectsUnsafeOverrides(t *testing.T) {
	t.Parallel()

	err := ValidateProviderPolicy(
		cleanup.ProviderMetadata{ID: "docker", SafetyTier: cleanup.SafetyTierConditional},
		cleanup.ProviderPolicy{Enabled: true, ApprovalMode: cleanup.ApprovalModeNone},
	)
	if err == nil {
		t.Fatal("ValidateProviderPolicy() expected conditional approval error")
	}
	err = ValidateProviderPolicy(
		cleanup.ProviderMetadata{ID: "live-db", SafetyTier: cleanup.SafetyTierForbidden},
		cleanup.ProviderPolicy{Enabled: true, ApprovalMode: cleanup.ApprovalModeOperator},
	)
	if err == nil {
		t.Fatal("ValidateProviderPolicy() expected forbidden enablement error")
	}
}

func TestBalancedProfileKeepsGoBuildCacheAtSevenDays(t *testing.T) {
	profile, err := BuildProfile(ProfileBalanced, []cleanup.ProviderMetadata{{ID: "go-build-cache", Name: "Go build cache", Version: "v1", OwnerScenario: "storage-manager", SafetyTier: cleanup.SafetyTierSafe, DefaultMode: cleanup.ProviderModeDisabled, DefaultApproval: cleanup.ApprovalModeNone, SupportedPlatforms: []string{"linux"}, IrreversibleEffects: []string{"cache files removed"}, TestSubstitute: "fake"}})
	if err != nil {
		t.Fatal(err)
	}
	got := profile.Defaults["go-build-cache"]
	if !got.Enabled || got.MinAge != 7*24*time.Hour {
		t.Fatalf("go-build-cache balanced policy = %#v", got)
	}
}
