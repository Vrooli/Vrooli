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
		{ModePhasedPlanDrain, "execute", ProfileDeepWork},
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

func TestCatalogDerivesModesFromRegistry(t *testing.T) {
	svc := newTestService(t, t.TempDir(), &fakeAgent{}, &fakePrompts{})
	catalog, err := svc.Catalog()
	if err != nil {
		t.Fatalf("Catalog returned error: %v", err)
	}
	if len(catalog.Modes) != len(Modes()) {
		t.Fatalf("Catalog mode count = %d, want %d", len(catalog.Modes), len(Modes()))
	}
	found := map[string]ModeCatalogEntry{}
	defaultCount := 0
	for _, mode := range catalog.Modes {
		found[mode.Mode] = mode
		if mode.Default {
			defaultCount++
		}
		if mode.Label == "" || mode.TargetKind == "" || mode.RunStrategy == "" || mode.WorkspaceTabID == "" {
			t.Fatalf("catalog mode has missing fields: %+v", mode)
		}
	}
	if defaultCount != 1 || !found[string(ModeItemLevel)].Default {
		t.Fatalf("default mode state = count %d item-level=%+v", defaultCount, found[string(ModeItemLevel)])
	}
	if found[string(ModeItemLevel)].SupportsPhases {
		t.Fatal("item-level supports phases in catalog")
	}
	if !found[string(ModeItemLevel)].Capabilities.UsesItemExecutionFlow {
		t.Fatal("item-level catalog capabilities should declare item execution flow")
	}
	if found[string(ModeItemLevel)].Capabilities.CanStartPhases {
		t.Fatal("item-level catalog capabilities should not allow phase starts")
	}
	holisticCapabilities := found[string(ModeHolisticLoop)].Capabilities
	if !holisticCapabilities.SupportsPhases || !holisticCapabilities.CanStartPhases {
		t.Fatalf("holistic-loop phase capabilities = %+v, want phase support", holisticCapabilities)
	}
	if !holisticCapabilities.CanCompleteItems || !holisticCapabilities.CanApplyBacklogSyncProposals {
		t.Fatalf("holistic-loop backlog capabilities = %+v, want sync support", holisticCapabilities)
	}
	if !holisticCapabilities.RequiresAcceptanceCriteria || !holisticCapabilities.SupportsArtifacts {
		t.Fatalf("holistic-loop workspace capabilities = %+v, want criteria and artifacts", holisticCapabilities)
	}
	if !found[string(ModePhasedPlanDrain)].Capabilities.SupportsHandoffs {
		t.Fatal("phased-plan-drain catalog capabilities should declare handoff support")
	}
	if got := len(found[string(ModeHolisticLoop)].Phases); got != len(MustDefinition(ModeHolisticLoop).PhaseGraph.Phases) {
		t.Fatalf("holistic-loop phase count = %d", got)
	}
	if got := len(found[string(ModePhasedPlanDrain)].Phases); got != len(MustDefinition(ModePhasedPlanDrain).PhaseGraph.Phases) {
		t.Fatalf("phased-plan-drain phase count = %d", got)
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
			name: "invalid activity purpose",
			mutate: func(defs map[Mode]Definition) {
				def := defs[ModeHolisticLoop]
				phase := def.PhaseGraph.Phases["investigate"]
				phase.ActivityPurpose = "Invalid-Purpose"
				def.PhaseGraph.Phases["investigate"] = phase
				defs[ModeHolisticLoop] = def
			},
			want: "activity purpose must be a lowercase snake-case token",
		},
		{
			name: "invalid lock purpose",
			mutate: func(defs map[Mode]Definition) {
				def := defs[ModeHolisticLoop]
				phase := def.PhaseGraph.Phases["investigate"]
				phase.LockPurpose = "Invalid-Purpose"
				def.PhaseGraph.Phases["investigate"] = phase
				defs[ModeHolisticLoop] = def
			},
			want: "lock purpose must be a lowercase snake-case token",
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
		{
			name: "unknown result binding kind",
			mutate: func(defs map[Mode]Definition) {
				def := defs[ModePhasedPlanDrain]
				phase := def.PhaseGraph.Phases["execute"]
				phase.ResultBindings = []ResultBinding{{
					Kind: "mystery",
					Artifact: ArtifactDefinition{
						Path:        "modes/phased-plan-drain/legacy-state.json",
						ContentType: "application/json",
						Required:    true,
					},
				}}
				def.PhaseGraph.Phases["execute"] = phase
				defs[ModePhasedPlanDrain] = def
			},
			want: "unknown kind",
		},
		{
			name: "result binding artifact must be declared output",
			mutate: func(defs map[Mode]Definition) {
				def := defs[ModePhasedPlanDrain]
				phase := def.PhaseGraph.Phases["execute"]
				phase.OutputArtifacts = nil
				phase.OutputContract.RequiredArtifacts = nil
				phase.ResultBindings = []ResultBinding{{
					Kind: ResultBindingProgressArtifact,
					Artifact: ArtifactDefinition{
						Path:        "modes/phased-plan-drain/legacy-state.json",
						ContentType: "application/json",
						Required:    true,
					},
				}}
				def.PhaseGraph.Phases["execute"] = phase
				defs[ModePhasedPlanDrain] = def
			},
			want: "is not declared as a phase output",
		},
		{
			name: "replan metrics phase must exist",
			mutate: func(defs map[Mode]Definition) {
				def := defs[ModeHolisticLoop]
				def.Metrics.ReplanSamplePhases = []Phase{"missing"}
				defs[ModeHolisticLoop] = def
			},
			want: "metrics replan phase",
		},
		{
			name: "acceptance metrics phase must require verdict",
			mutate: func(defs map[Mode]Definition) {
				def := defs[ModeHolisticLoop]
				def.Metrics.AcceptanceSamplePhases = []Phase{"execute"}
				defs[ModeHolisticLoop] = def
			},
			want: "must require a verdict",
		},
		{
			name: "acceptance metrics require accepted verdicts",
			mutate: func(defs map[Mode]Definition) {
				def := defs[ModeHolisticLoop]
				def.Metrics.AcceptedVerdicts = nil
				defs[ModeHolisticLoop] = def
			},
			want: "requires accepted verdict values",
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
		return ExpectedPromptCatalogEntry(mode, phase)
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
		{
			name: "output paths mismatch",
			resolve: func(mode, phase string) (PromptCatalogEntry, bool) {
				entry, ok := validResolver(mode, phase)
				entry.OutputPaths = []string{"wrong"}
				return entry, ok
			},
			want: "output paths mismatch",
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
		{ModeHolisticLoop, "review", "holistic_loop_review"},
		{ModePhasedPlanDrain, "execute", "phased_plan_execute_next"},
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

// TestLoaderDerivesCommonAuthoringPolicy proves the data loader derives the
// implicit authoring defaults a mode author does not spell out — round root
// from artifact root, prompt catalog IDs from prefix + phase (or prompt suffix),
// lock purpose from activity purpose, and event-source tags from the mode id —
// so authoring a mode is a minimal data edit (the Phase-7 self-serve path).
func TestLoaderDerivesCommonAuthoringPolicy(t *testing.T) {
	const doc = `{
	  "kind": "operating-mode",
	  "id": "synthetic",
	  "label": "Synthetic",
	  "description": "Synthetic mode exercising loader derivation defaults.",
	  "best_for": ["Exercising loader derivation"],
	  "not_for": ["Production work"],
	  "tradeoffs": ["Test-only"],
	  "when_in_doubt_pick_instead": "item-level",
	  "target": { "kind": "initiative" },
	  "run_strategy": { "kind": "single_phase_run" },
	  "prompt": { "catalog_prefix": "swarm-manager-synthetic" },
	  "artifact": { "root": "modes/synthetic" },
	  "profile": { "default_profile_key": "swarm-manager/deep-work" },
	  "backlog_sync": {
	    "capabilities": ["read_only", "propose_mutations", "mark_complete", "create_followups", "update_scope"],
	    "requires_run_id": true,
	    "requires_membership": true,
	    "apply_mode": "operator-gated"
	  },
	  "metrics": { "accepted_verdicts": ["accept", "accepted"] },
	  "lock": { "initiative_exclusive": true },
	  "ui": { "workspace_tab_id": "operating-mode" },
	  "phase_graph": {
	    "start_phase": "draft",
	    "terminal": ["review"],
	    "phases": [
	      {
	        "id": "draft",
	        "kind": "investigate",
	        "activity_purpose": "synthetic_draft",
        "reads": ["OPERATING_MODE", "PHASE", "ROUND_NUMBER", "OPERATOR_NOTE", "PRIOR_ROUNDS_JSON", "INITIATIVE_NAME", "MEMBER_ITEMS_JSON"],
	        "profile_key": "swarm-manager/deep-work",
	        "declared_output": {
	          "fields": [{ "name": "progress", "type": "object", "required": true, "description": "Progress state." }]
	        },
	        "result_bindings": [
	          { "kind": "progress_artifact", "artifact": { "path": "modes/synthetic/draft-progress.json", "content_type": "application/json", "required": true } }
	        ],
	        "metrics": { "counts_replan_sample": true },
	        "transitions": [{ "when": { "op": "always" }, "to": ["review"] }]
	      },
	      {
	        "id": "review",
	        "kind": "review",
	        "activity_purpose": "synthetic_review",
        "reads": ["OPERATING_MODE", "PHASE", "ROUND_NUMBER", "OPERATOR_NOTE", "PRIOR_ROUNDS_JSON", "INITIATIVE_NAME", "MEMBER_ITEMS_JSON"],
	        "profile_key": "swarm-manager/analysis",
	        "requires_criteria": true,
	        "prompt": { "suffix": "final-review" },
	        "declared_output": {
	          "fields": [{ "name": "verdict", "type": "string", "required": true, "description": "Acceptance verdict." }]
	        },
	        "metrics": { "counts_acceptance_sample": true },
	        "transitions": [{ "when": { "op": "always" }, "to": ["draft"] }]
	      }
	    ]
	  }
	}`

	def, err := LoadModeDefinition([]byte(doc))
	if err != nil {
		t.Fatalf("LoadModeDefinition: %v", err)
	}

	if def.Target.Kind != TargetInitiative {
		t.Fatalf("target = %q, want %q", def.Target.Kind, TargetInitiative)
	}
	if def.Artifact.RoundRoot != "modes/synthetic/rounds" {
		t.Fatalf("round root = %q", def.Artifact.RoundRoot)
	}
	if !def.BacklogSync.RequiresRunID || !def.BacklogSync.RequiresMembership || def.BacklogSync.EventSource != "synthetic" {
		t.Fatalf("backlog policy not derived from mode: %+v", def.BacklogSync)
	}
	if def.Metrics.EventSource != "synthetic" {
		t.Fatalf("metrics event source = %q, want synthetic", def.Metrics.EventSource)
	}
	if !def.Lock.InitiativeExclusive || def.UI.WorkspaceTabID != "operating-mode" {
		t.Fatalf("lock/UI policy not derived: lock=%+v ui=%+v", def.Lock, def.UI)
	}

	draft := def.PhaseGraph.Phases["draft"]
	if draft.CatalogID != "swarm-manager-synthetic-draft" || draft.SkillID != draft.CatalogID {
		t.Fatalf("draft prompt IDs = %q/%q", draft.CatalogID, draft.SkillID)
	}
	if draft.LockPurpose != draft.ActivityPurpose {
		t.Fatalf("draft lock purpose = %q, want activity purpose %q", draft.LockPurpose, draft.ActivityPurpose)
	}
	if got := len(draft.OutputContract.RequiredArtifacts); got != 1 {
		t.Fatalf("draft required artifacts = %d, want 1", got)
	}
	if got := len(draft.ResultBindings); got != 1 {
		t.Fatalf("draft result bindings = %d, want 1", got)
	}
	if draft.OutputArtifacts[0].Path != "modes/synthetic/draft-progress.json" {
		t.Fatalf("draft output artifacts = %+v", draft.OutputArtifacts)
	}

	review := def.PhaseGraph.Phases["review"]
	if review.CatalogID != "swarm-manager-synthetic-final-review" {
		t.Fatalf("review catalog ID = %q", review.CatalogID)
	}
	if !review.OutputContract.RequiresVerdict || !review.RequiresCriteria {
		t.Fatalf("review contract did not preserve review requirements: %+v criteria=%v", review.OutputContract, review.RequiresCriteria)
	}
	if got := def.Profile.PhaseProfiles["review"]; got != ProfileAnalysis {
		t.Fatalf("review profile policy = %q, want %q", got, ProfileAnalysis)
	}
	if !def.Metrics.CountsReplanSample("draft") || def.Metrics.CountsReplanSample("review") {
		t.Fatalf("replan metrics policy = %+v", def.Metrics)
	}
	if !def.Metrics.CountsAcceptanceSample("review") || def.Metrics.CountsAcceptanceSample("draft") {
		t.Fatalf("acceptance metrics policy = %+v", def.Metrics)
	}
	if !def.Metrics.IsAcceptedVerdict("ACCEPTED") || def.Metrics.IsAcceptedVerdict("request_changes") {
		t.Fatalf("accepted verdict policy = %+v", def.Metrics)
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

// cloneRegistryForTest deep-copies the loaded registry so a test can mutate a
// definition in isolation (to drive the validators negatively) without
// corrupting the process registry shared across tests.
func cloneRegistryForTest() map[Mode]Definition {
	registryMu.RLock()
	defer registryMu.RUnlock()
	out := make(map[Mode]Definition, len(registry))
	for mode, def := range registry {
		def.PhaseGraph.Terminal = append([]Phase(nil), def.PhaseGraph.Terminal...)
		def.PhaseGraph.Transitions = clonePhaseTransitions(def.PhaseGraph.Transitions)
		def.PhaseGraph.Guards = clonePhaseGuards(def.PhaseGraph.Guards)
		def.PhaseGraph.Phases = clonePhaseDefinitions(def.PhaseGraph.Phases)
		def.Profile.PhaseProfiles = clonePhaseProfiles(def.Profile.PhaseProfiles)
		def.Metrics = cloneMetricsPolicy(def.Metrics)
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

func clonePhaseGuards(in map[Phase][]GuardedTransition) map[Phase][]GuardedTransition {
	if in == nil {
		return nil
	}
	out := make(map[Phase][]GuardedTransition, len(in))
	for phase, guards := range in {
		cloned := make([]GuardedTransition, len(guards))
		for i, gt := range guards {
			gt.To = append([]Phase(nil), gt.To...)
			cloned[i] = gt
		}
		out[phase] = cloned
	}
	return out
}

func clonePhaseDefinitions(in map[Phase]PhaseDefinition) map[Phase]PhaseDefinition {
	out := make(map[Phase]PhaseDefinition, len(in))
	for phase, def := range in {
		def.OutputArtifacts = append([]ArtifactDefinition(nil), def.OutputArtifacts...)
		def.ResultBindings = append([]ResultBinding(nil), def.ResultBindings...)
		def.OutputContract.RequiredArtifacts = append([]ArtifactDefinition(nil), def.OutputContract.RequiredArtifacts...)
		def.AutoStartAfter = append([]Phase(nil), def.AutoStartAfter...)
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

func cloneMetricsPolicy(in MetricsPolicy) MetricsPolicy {
	in.ReplanSamplePhases = append([]Phase(nil), in.ReplanSamplePhases...)
	in.AcceptanceSamplePhases = append([]Phase(nil), in.AcceptanceSamplePhases...)
	in.AcceptedVerdicts = append([]string(nil), in.AcceptedVerdicts...)
	return in
}

func TestDefinitionsHaveDecisionMetadata(t *testing.T) {
	for _, mode := range Modes() {
		def, err := DefinitionFor(mode)
		if err != nil {
			t.Fatalf("DefinitionFor(%q): %v", mode, err)
		}
		if len(def.BestFor) == 0 {
			t.Errorf("mode %q BestFor is empty", mode)
		}
		if len(def.NotFor) == 0 {
			t.Errorf("mode %q NotFor is empty", mode)
		}
		if len(def.Tradeoffs) == 0 {
			t.Errorf("mode %q Tradeoffs is empty", mode)
		}
		for i, entry := range def.BestFor {
			if strings.TrimSpace(entry) == "" {
				t.Errorf("mode %q BestFor[%d] is blank", mode, i)
			}
		}
		for i, entry := range def.NotFor {
			if strings.TrimSpace(entry) == "" {
				t.Errorf("mode %q NotFor[%d] is blank", mode, i)
			}
		}
		for i, entry := range def.Tradeoffs {
			if strings.TrimSpace(entry) == "" {
				t.Errorf("mode %q Tradeoffs[%d] is blank", mode, i)
			}
		}
	}
}

func TestWhenInDoubtPickInsteadReferencesRegisteredMode(t *testing.T) {
	itemLevel := MustDefinition(ModeItemLevel)
	if itemLevel.WhenInDoubtPickInstead != "" {
		t.Errorf("item-level WhenInDoubtPickInstead = %q, want empty (item-level is the safe default)", itemLevel.WhenInDoubtPickInstead)
	}
	for _, mode := range []Mode{ModeHolisticLoop, ModePhasedPlanDrain} {
		def := MustDefinition(mode)
		if def.WhenInDoubtPickInstead == "" {
			t.Errorf("mode %q WhenInDoubtPickInstead is empty; only item-level may be empty", mode)
			continue
		}
		if def.WhenInDoubtPickInstead == mode {
			t.Errorf("mode %q WhenInDoubtPickInstead = self", mode)
		}
		if !ValidateMode(string(def.WhenInDoubtPickInstead)) {
			t.Errorf("mode %q WhenInDoubtPickInstead %q is not a registered mode", mode, def.WhenInDoubtPickInstead)
		}
	}
}

func TestValidateRegistryRejectsMissingDecisionMetadata(t *testing.T) {
	tests := []struct {
		name   string
		mutate func(map[Mode]Definition)
		want   string
	}{
		{
			name: "empty best_for",
			mutate: func(defs map[Mode]Definition) {
				def := defs[ModeHolisticLoop]
				def.BestFor = nil
				defs[ModeHolisticLoop] = def
			},
			want: "best_for requires at least one entry",
		},
		{
			name: "empty not_for",
			mutate: func(defs map[Mode]Definition) {
				def := defs[ModeHolisticLoop]
				def.NotFor = nil
				defs[ModeHolisticLoop] = def
			},
			want: "not_for requires at least one entry",
		},
		{
			name: "empty tradeoffs",
			mutate: func(defs map[Mode]Definition) {
				def := defs[ModeHolisticLoop]
				def.Tradeoffs = nil
				defs[ModeHolisticLoop] = def
			},
			want: "tradeoffs requires at least one entry",
		},
		{
			name: "blank entry",
			mutate: func(defs map[Mode]Definition) {
				def := defs[ModeHolisticLoop]
				def.BestFor = []string{"   "}
				defs[ModeHolisticLoop] = def
			},
			want: "cannot be blank",
		},
		{
			name: "self reference",
			mutate: func(defs map[Mode]Definition) {
				def := defs[ModeHolisticLoop]
				def.WhenInDoubtPickInstead = ModeHolisticLoop
				defs[ModeHolisticLoop] = def
			},
			want: "cannot reference itself",
		},
		{
			name: "unregistered fallback",
			mutate: func(defs map[Mode]Definition) {
				def := defs[ModeHolisticLoop]
				def.WhenInDoubtPickInstead = Mode("nope")
				defs[ModeHolisticLoop] = def
			},
			want: "references unregistered mode",
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
