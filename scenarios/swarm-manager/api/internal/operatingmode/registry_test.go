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
