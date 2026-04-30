package operatingmode

import (
	"strings"
	"testing"
)

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

func TestValidateRegistryAcceptsCurrentRegistry(t *testing.T) {
	if err := ValidateRegistry(); err != nil {
		t.Fatalf("ValidateRegistry returned error: %v", err)
	}
}

func TestValidateRegistryRejectsInvalidDefinitions(t *testing.T) {
	tests := []struct {
		name   string
		mutate func(map[Mode]Definition)
		want   string
	}{
		{
			name: "invalid transition",
			mutate: func(defs map[Mode]Definition) {
				def := defs[ModeHolisticLoop]
				def.PhaseGraph.Transitions["investigate"] = []Phase{"missing"}
				defs[ModeHolisticLoop] = def
			},
			want: "references unregistered phase",
		},
		{
			name: "artifact outside root",
			mutate: func(defs map[Mode]Definition) {
				def := defs[ModeHolisticLoop]
				phase := def.PhaseGraph.Phases["investigate"]
				phase.OutputArtifacts = []ArtifactDefinition{{Path: "elsewhere/findings.md", Required: true}}
				phase.OutputContract.RequiredArtifacts = phase.OutputArtifacts
				def.PhaseGraph.Phases["investigate"] = phase
				defs[ModeHolisticLoop] = def
			},
			want: "outside mode root",
		},
		{
			name: "non owned profile",
			mutate: func(defs map[Mode]Definition) {
				def := defs[ModeHolisticLoop]
				def.Profile.DefaultProfileKey = "other-scenario/deep-work"
				defs[ModeHolisticLoop] = def
			},
			want: "non-scenario-owned",
		},
		{
			name: "missing prompt skill",
			mutate: func(defs map[Mode]Definition) {
				def := defs[ModeHolisticLoop]
				phase := def.PhaseGraph.Phases["investigate"]
				phase.SkillID = ""
				def.PhaseGraph.Phases["investigate"] = phase
				defs[ModeHolisticLoop] = def
			},
			want: "prompt catalog ID and skill ID are required",
		},
		{
			name: "profile mismatch",
			mutate: func(defs map[Mode]Definition) {
				def := defs[ModeHolisticLoop]
				def.Profile.PhaseProfiles["investigate"] = ProfileAnalysis
				defs[ModeHolisticLoop] = def
			},
			want: "profile mismatch",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			defs := cloneRegistryForTest()
			tt.mutate(defs)
			err := validateDefinitions(defs)
			if err == nil {
				t.Fatalf("validateDefinitions error = nil, want %q", tt.want)
			}
			if !strings.Contains(err.Error(), tt.want) {
				t.Fatalf("validateDefinitions error = %v, want contains %q", err, tt.want)
			}
		})
	}
}

func TestValidatePromptCatalogRejectsRegistryMismatches(t *testing.T) {
	validResolver := func(mode, phase string) (PromptCatalogEntry, bool) {
		def, err := DefinitionFor(Mode(mode))
		if err != nil {
			return PromptCatalogEntry{}, false
		}
		phaseDef, err := def.PhaseDefinition(Phase(phase))
		if err != nil {
			return PromptCatalogEntry{}, false
		}
		return PromptCatalogEntry{CatalogID: phaseDef.CatalogID, SkillID: phaseDef.SkillID}, true
	}
	if err := ValidatePromptCatalog(validResolver); err != nil {
		t.Fatalf("ValidatePromptCatalog(valid) returned error: %v", err)
	}

	tests := []struct {
		name    string
		resolve PromptCatalogResolver
		want    string
	}{
		{
			name:    "nil resolver",
			resolve: nil,
			want:    "resolver is required",
		},
		{
			name: "missing entry",
			resolve: func(string, string) (PromptCatalogEntry, bool) {
				return PromptCatalogEntry{}, false
			},
			want: "missing entry",
		},
		{
			name: "catalog id mismatch",
			resolve: func(mode, phase string) (PromptCatalogEntry, bool) {
				entry, ok := validResolver(mode, phase)
				entry.CatalogID = "wrong"
				return entry, ok
			},
			want: "ID mismatch",
		},
		{
			name: "skill mismatch",
			resolve: func(mode, phase string) (PromptCatalogEntry, bool) {
				entry, ok := validResolver(mode, phase)
				entry.SkillID = "wrong"
				return entry, ok
			},
			want: "skill mismatch",
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := ValidatePromptCatalog(tt.resolve)
			if err == nil {
				t.Fatalf("ValidatePromptCatalog error = nil, want %q", tt.want)
			}
			if !strings.Contains(err.Error(), tt.want) {
				t.Fatalf("ValidatePromptCatalog error = %v, want contains %q", err, tt.want)
			}
		})
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

func cloneRegistryForTest() map[Mode]Definition {
	out := make(map[Mode]Definition, len(registry))
	for mode, def := range registry {
		def.PhaseGraph.Terminal = append([]Phase(nil), def.PhaseGraph.Terminal...)
		def.PhaseGraph.Transitions = clonePhaseTransitions(def.PhaseGraph.Transitions)
		def.PhaseGraph.Phases = clonePhaseDefinitions(def.PhaseGraph.Phases)
		def.Profile.PhaseProfiles = clonePhaseProfiles(def.Profile.PhaseProfiles)
		out[mode] = def
	}
	return out
}

func clonePhaseTransitions(in map[Phase][]Phase) map[Phase][]Phase {
	out := make(map[Phase][]Phase, len(in))
	for phase, next := range in {
		out[phase] = append([]Phase(nil), next...)
	}
	return out
}

func clonePhaseDefinitions(in map[Phase]PhaseDefinition) map[Phase]PhaseDefinition {
	out := make(map[Phase]PhaseDefinition, len(in))
	for phase, def := range in {
		def.OutputArtifacts = append([]ArtifactDefinition(nil), def.OutputArtifacts...)
		def.OutputContract.RequiredArtifacts = append([]ArtifactDefinition(nil), def.OutputContract.RequiredArtifacts...)
		out[phase] = def
	}
	return out
}

func clonePhaseProfiles(in map[Phase]string) map[Phase]string {
	out := make(map[Phase]string, len(in))
	for phase, profile := range in {
		out[phase] = profile
	}
	return out
}
