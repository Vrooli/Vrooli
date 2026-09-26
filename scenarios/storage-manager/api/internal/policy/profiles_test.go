package policy

import (
	"encoding/json"
	"testing"
	"time"

	"storage-manager/internal/cleanup"
)

func TestProviderPolicyDoesNotPersistControllerOnlyFreshReclaim(t *testing.T) {
	payload, err := json.Marshal(cleanup.ProviderPolicy{Enabled: true, AllowFreshReclaim: true})
	if err != nil {
		t.Fatal(err)
	}
	if string(payload) != `{"Enabled":true,"MinAge":0,"MaxBytes":0,"ApprovalMode":""}` {
		t.Fatalf("serialized provider policy = %s, controller-only capability leaked", payload)
	}
}

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

func TestReconcileAddsMissingProvidersWithoutChangingExistingPolicy(t *testing.T) {
	providers := []cleanup.ProviderMetadata{
		{ID: "a", Name: "A", Version: "v1", OwnerScenario: "storage-manager", SafetyTier: cleanup.SafetyTierSafe, DefaultMode: cleanup.ProviderModeEnabled, DefaultApproval: cleanup.ApprovalModeNone, SupportedPlatforms: []string{"linux"}, IrreversibleEffects: []string{"none"}, TestSubstitute: "fake"},
		{ID: "b", Name: "B", Version: "v1", OwnerScenario: "storage-manager", SafetyTier: cleanup.SafetyTierSafe, DefaultMode: cleanup.ProviderModeEnabled, DefaultApproval: cleanup.ApprovalModeNone, SupportedPlatforms: []string{"linux"}, IrreversibleEffects: []string{"none"}, TestSubstitute: "fake"},
	}
	existing := map[string]cleanup.ProviderPolicy{"a": {Enabled: false, MinAge: 99 * time.Hour, ApprovalMode: cleanup.ApprovalModeOperator}}
	version, got, added, err := ReconcilePolicy(ProfileBalanced, "old", existing, time.Time{}, providers)
	if err != nil {
		t.Fatal(err)
	}
	if len(added) != 1 || added[0] != "b" || len(got) != 2 {
		t.Fatalf("added=%v policies=%#v", added, got)
	}
	if got["a"] != existing["a"] {
		t.Fatalf("existing operator policy changed: %#v", got["a"])
	}
	if version == "old" || version != StableVersion(ProfileBalanced, got) {
		t.Fatalf("reconciled version = %q, want new stable fingerprint", version)
	}
}
