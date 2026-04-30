package operatingmode

import "testing"

func TestRegistryDefinesRequiredModes(t *testing.T) {
	for _, mode := range []Mode{ModeItemLevel, ModeHolisticLoop, ModePhasedPlanDrain} {
		def, err := DefinitionFor(mode)
		if err != nil {
			t.Fatalf("DefinitionFor(%q): %v", mode, err)
		}
		if def.Mode != mode {
			t.Fatalf("DefinitionFor(%q).Mode = %q", mode, def.Mode)
		}
	}
}

func TestNormalizeModeDefaultsBlankToItemLevel(t *testing.T) {
	if got := NormalizeMode(" "); got != ModeItemLevel {
		t.Fatalf("NormalizeMode(blank) = %q, want %q", got, ModeItemLevel)
	}
}

func TestInitiativeModesCarryPhaseProfilePolicy(t *testing.T) {
	cases := []struct {
		mode  Mode
		phase Phase
		want  string
	}{
		{ModeHolisticLoop, "investigate", ProfileDeepWork},
		{ModeHolisticLoop, "review", ProfileAnalysis},
		{ModePhasedPlanDrain, "execute_next", ProfileDeepWork},
		{ModePhasedPlanDrain, "classify_progress", ProfileAnalysis},
	}

	for _, tc := range cases {
		def, err := DefinitionFor(tc.mode)
		if err != nil {
			t.Fatalf("DefinitionFor(%q): %v", tc.mode, err)
		}
		phase, err := def.PhaseDefinition(tc.phase)
		if err != nil {
			t.Fatalf("PhaseDefinition(%q, %q): %v", tc.mode, tc.phase, err)
		}
		if phase.ProfileKey != tc.want {
			t.Errorf("%s/%s profile = %q, want %q", tc.mode, tc.phase, phase.ProfileKey, tc.want)
		}
		if got := def.Profile.PhaseProfiles[tc.phase]; got != tc.want {
			t.Errorf("%s/%s profile policy = %q, want %q", tc.mode, tc.phase, got, tc.want)
		}
	}
}

func TestRequiredProfileKeysReturnsScenarioOwnedRegistryProfiles(t *testing.T) {
	keys, err := RequiredProfileKeys()
	if err != nil {
		t.Fatalf("RequiredProfileKeys returned error: %v", err)
	}
	want := []string{ProfileAnalysis, ProfileDeepWork, ProfileDefault}
	if len(keys) != len(want) {
		t.Fatalf("RequiredProfileKeys len = %d, want %d: %v", len(keys), len(want), keys)
	}
	for i := range want {
		if keys[i] != want[i] {
			t.Fatalf("RequiredProfileKeys[%d] = %q, want %q; got %v", i, keys[i], want[i], keys)
		}
	}
}

func TestRequiredProfileKeysRejectsNonScenarioOwnedProfile(t *testing.T) {
	original := registry[ModeHolisticLoop]
	modified := original
	modified.Profile.DefaultProfileKey = "other-scenario/deep-work"
	registry[ModeHolisticLoop] = modified
	t.Cleanup(func() {
		registry[ModeHolisticLoop] = original
	})

	if _, err := RequiredProfileKeys(); err == nil {
		t.Fatal("expected non-scenario-owned profile key to fail")
	}
}

func TestInitiativeModePhasesCarryStableActivityPurposes(t *testing.T) {
	cases := []struct {
		mode  Mode
		phase Phase
		want  string
	}{
		{ModeHolisticLoop, "investigate", "holistic_loop_investigate"},
		{ModeHolisticLoop, "plan", "holistic_loop_plan"},
		{ModeHolisticLoop, "execute", "holistic_loop_execute"},
		{ModeHolisticLoop, "review", "holistic_loop_review"},
		{ModePhasedPlanDrain, "prepare_plan", "phased_plan_prepare"},
		{ModePhasedPlanDrain, "execute_next", "phased_plan_execute_next"},
		{ModePhasedPlanDrain, "classify_progress", "phased_plan_classify_progress"},
		{ModePhasedPlanDrain, "review", "phased_plan_review"},
	}

	for _, tc := range cases {
		def, err := DefinitionFor(tc.mode)
		if err != nil {
			t.Fatalf("DefinitionFor(%q): %v", tc.mode, err)
		}
		phase, err := def.PhaseDefinition(tc.phase)
		if err != nil {
			t.Fatalf("PhaseDefinition(%q, %q): %v", tc.mode, tc.phase, err)
		}
		if phase.ActivityPurpose != tc.want {
			t.Fatalf("%s/%s activity purpose = %q, want %q", tc.mode, tc.phase, phase.ActivityPurpose, tc.want)
		}
		if phase.LockPurpose != phase.ActivityPurpose {
			t.Fatalf("%s/%s lock purpose = %q, want activity purpose %q", tc.mode, tc.phase, phase.LockPurpose, phase.ActivityPurpose)
		}
	}
}

func TestUnknownModeFailsClosed(t *testing.T) {
	if ValidateMode("not-a-mode") {
		t.Fatal("ValidateMode accepted unknown mode")
	}
	if _, err := DefinitionFor("not-a-mode"); err == nil {
		t.Fatal("DefinitionFor accepted unknown mode")
	}
}
