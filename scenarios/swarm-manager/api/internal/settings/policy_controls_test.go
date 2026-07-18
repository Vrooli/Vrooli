package settings

import (
	"path/filepath"
	"testing"

	"swarm-manager/internal/execution"
	"swarm-manager/internal/testutil"
)

// TestDefaultPolicyControlsEqualsDefaultSettingsProjection pins the contract
// that a settings-store outage (DefaultPolicyControls fallback) behaves
// exactly like a missing settings file (DefaultSettings projection).
func TestDefaultPolicyControlsEqualsDefaultSettingsProjection(t *testing.T) {
	got := DefaultPolicyControls()
	want := ProjectPolicyControls(DefaultSettings())
	if got != want {
		t.Fatalf("DefaultPolicyControls() = %+v, want projection of DefaultSettings() = %+v", got, want)
	}
}

// TestProjectionEquivalence proves the projection is a lossless re-plumbing:
// for a non-default Settings value, every control equals the corresponding
// legacy settings field.
func TestProjectionEquivalence(t *testing.T) {
	s := DefaultSettings()
	s.DefaultMode = "manual"
	s.AutoFixup = true
	s.MaxFixupAttempts = 4
	s.ReviewAgentEnabled = false
	s.MaxAutoRounds = 25
	s.AutoInitializeWorkshop = false
	s.AutoAdvanceWorkshop = false
	s.AutoCascadeWorkshop = false
	s.AutoAdvanceDelaySeconds = 42
	s.AgentMaxTurns = 123
	s.AgentTimeoutSeconds = 700
	s.ReviewCodeQualityMinScore = 80
	s.ReviewTestMinPassRate = 0.5
	s.ReviewMaxBlockingViolations = 2
	s.ReviewMaxWarnings = 7
	s.ReviewRequireScreenshots = false
	s.ReviewRequireTests = false

	c := ProjectPolicyControls(s)

	checks := []struct {
		name string
		got  any
		want any
	}{
		{"Execution.DefaultMode", c.Execution.DefaultMode, s.DefaultMode},
		{"AutoAdvance.AutoInitialize", c.AutoAdvance.AutoInitialize, s.AutoInitializeWorkshop},
		{"AutoAdvance.Enabled", c.AutoAdvance.Enabled, s.AutoAdvanceWorkshop},
		{"AutoAdvance.Cascade", c.AutoAdvance.Cascade, s.AutoCascadeWorkshop},
		{"AutoAdvance.DelaySeconds", c.AutoAdvance.DelaySeconds, s.AutoAdvanceDelaySeconds},
		{"AutoAdvance.MaxAutoRounds", c.AutoAdvance.MaxAutoRounds, s.MaxAutoRounds},
		{"Retry.AutoFixup", c.Retry.AutoFixup, s.AutoFixup},
		{"Retry.MaxFixupAttempts", c.Retry.MaxFixupAttempts, s.MaxFixupAttempts},
		{"Review.AgentEnabled", c.Review.AgentEnabled, s.ReviewAgentEnabled},
		{"Review.CodeQualityMinScore", c.Review.CodeQualityMinScore, s.ReviewCodeQualityMinScore},
		{"Review.TestMinPassRate", c.Review.TestMinPassRate, s.ReviewTestMinPassRate},
		{"Review.MaxBlockingViolations", c.Review.MaxBlockingViolations, s.ReviewMaxBlockingViolations},
		{"Review.MaxWarnings", c.Review.MaxWarnings, s.ReviewMaxWarnings},
		{"Review.RequireScreenshots", c.Review.RequireScreenshots, s.ReviewRequireScreenshots},
		{"Review.RequireTests", c.Review.RequireTests, s.ReviewRequireTests},
		{"Budgets.MaxTurns", c.Budgets.MaxTurns, s.AgentMaxTurns},
		{"Budgets.TimeoutSeconds", c.Budgets.TimeoutSeconds, s.AgentTimeoutSeconds},
	}
	for _, check := range checks {
		if check.got != check.want {
			t.Errorf("%s = %v, want %v", check.name, check.got, check.want)
		}
	}
}

// TestLegacyAdaptersDerivedFromProjection asserts the legacy execution.Policy
// and execution.ReviewThresholds seams agree with the PolicyControls
// projection for the same persisted settings (they are derived from it, so
// disagreement would mean the derivation drifted).
func TestLegacyAdaptersDerivedFromProjection(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, "settings.json")
	store := NewStore(path)

	s := DefaultSettings()
	s.DefaultMode = "manual"
	s.AutoFixup = true
	s.MaxFixupAttempts = 3
	s.ReviewAgentEnabled = false
	s.ReviewCodeQualityMinScore = 75
	s.ReviewTestMinPassRate = 0.9
	s.ReviewMaxBlockingViolations = 1
	s.ReviewMaxWarnings = 5
	s.ReviewRequireScreenshots = false
	s.ReviewRequireTests = false
	if err := store.Save(s); err != nil {
		t.Fatalf("save settings: %v", err)
	}

	controls, err := NewPolicyControlsAdapter(store).LoadPolicyControls()
	if err != nil {
		t.Fatalf("load policy controls: %v", err)
	}

	policy, err := NewPolicyAdapter(store).LoadPolicy()
	if err != nil {
		t.Fatalf("load policy: %v", err)
	}
	wantPolicy := execution.Policy{
		DefaultMode:        execution.Mode(controls.Execution.DefaultMode),
		AutoFixup:          controls.Retry.AutoFixup,
		MaxFixupAttempts:   controls.Retry.MaxFixupAttempts,
		ReviewAgentEnabled: controls.Review.AgentEnabled,
	}
	if policy != wantPolicy {
		t.Errorf("LoadPolicy() = %+v, want %+v", policy, wantPolicy)
	}

	th, err := NewReviewThresholdsAdapter(store).LoadReviewThresholds()
	if err != nil {
		t.Fatalf("load review thresholds: %v", err)
	}
	wantTh := execution.ReviewThresholds{
		CodeQualityMinScore:   controls.Review.CodeQualityMinScore,
		TestMinPassRate:       controls.Review.TestMinPassRate,
		MaxBlockingViolations: controls.Review.MaxBlockingViolations,
		MaxWarnings:           controls.Review.MaxWarnings,
		RequireScreenshots:    controls.Review.RequireScreenshots,
		RequireTests:          controls.Review.RequireTests,
	}
	if *th != wantTh {
		t.Errorf("LoadReviewThresholds() = %+v, want %+v", *th, wantTh)
	}

	gov, err := NewGovernanceAdapter(store).LoadGovernance()
	if err != nil {
		t.Fatalf("load governance: %v", err)
	}
	if gov.AgentMaxTurns != controls.Budgets.MaxTurns {
		t.Errorf("governance AgentMaxTurns = %d, want %d", gov.AgentMaxTurns, controls.Budgets.MaxTurns)
	}
}

// TestPolicyControlsAdapterBoundsFromNormalize verifies persisted
// out-of-bounds values flow through normalize before projection (the seam
// never sees un-clamped values).
func TestPolicyControlsAdapterBoundsFromNormalize(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, "settings.json")
	testutil.WriteJSONFile(t, path, map[string]any{
		"max_fixup_attempts":         99,
		"max_auto_rounds":            999,
		"auto_advance_delay_seconds": 999,
		"agent_max_turns":            1,
		"agent_timeout_seconds":      1,
	})

	controls, err := NewPolicyControlsAdapter(NewStore(path)).LoadPolicyControls()
	if err != nil {
		t.Fatalf("load policy controls: %v", err)
	}
	if controls.Retry.MaxFixupAttempts != 5 {
		t.Errorf("MaxFixupAttempts = %d, want clamp to 5", controls.Retry.MaxFixupAttempts)
	}
	if controls.AutoAdvance.MaxAutoRounds != 50 {
		t.Errorf("MaxAutoRounds = %d, want clamp to 50", controls.AutoAdvance.MaxAutoRounds)
	}
	if controls.AutoAdvance.DelaySeconds != 120 {
		t.Errorf("DelaySeconds = %d, want clamp to 120", controls.AutoAdvance.DelaySeconds)
	}
	if controls.Budgets.MaxTurns != 5 {
		t.Errorf("Budgets.MaxTurns = %d, want clamp to 5", controls.Budgets.MaxTurns)
	}
	if controls.Budgets.TimeoutSeconds != 60 {
		t.Errorf("Budgets.TimeoutSeconds = %d, want clamp to 60", controls.Budgets.TimeoutSeconds)
	}
}

// TestPolicyProjectionToProto verifies the public projection carries the
// effective controls and classifies every orchestration field exactly once.
func TestPolicyProjectionToProto(t *testing.T) {
	s := DefaultSettings()
	s.AutoAdvanceDelaySeconds = 33
	proj := policyProjectionToProto(s)
	if proj.EffectiveControls == nil {
		t.Fatal("projection missing effective_controls")
	}
	if got := proj.EffectiveControls.AutoAdvanceDelaySeconds; got != 33 {
		t.Errorf("effective auto_advance_delay_seconds = %d, want 33", got)
	}
	if got := proj.EffectiveControls.DefaultMode; got != s.DefaultMode {
		t.Errorf("effective default_mode = %q, want %q", got, s.DefaultMode)
	}

	seen := map[string]bool{}
	for _, c := range proj.Classifications {
		if seen[c.Field] {
			t.Errorf("field %q classified twice", c.Field)
		}
		seen[c.Field] = true
	}
	for _, field := range []string{
		"default_mode", "auto_fixup", "max_fixup_attempts", "review_agent_enabled",
		"max_auto_rounds", "auto_initialize_workshop", "auto_advance_workshop",
		"auto_cascade_workshop", "auto_advance_delay_seconds", "agent_max_turns",
		"agent_timeout_seconds", "review_code_quality_min_score",
		"review_test_min_pass_rate", "review_max_blocking_violations",
		"review_max_warnings", "review_require_screenshots", "review_require_tests",
	} {
		if !seen[field] {
			t.Errorf("field %q missing from classifications", field)
		}
	}
}
