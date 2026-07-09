package promptcatalog

import (
	"testing"

	"swarm-manager/internal/operatingmode"
)

func TestResolveBacklogSkill(t *testing.T) {
	tests := []struct {
		name string
		mode string
		kind string
		want string
	}{
		{name: "workshop non research", mode: "workshop", kind: "idea", want: "swarm-manager-workshop"},
		{name: "workshop research", mode: "workshop", kind: "research", want: "swarm-manager-workshop-research"},
		{name: "initialize non research", mode: "initialize", kind: "fix", want: "swarm-manager-initialize-backlog"},
		{name: "initialize research", mode: "initialize", kind: "research", want: "swarm-manager-initialize-research"},
		{name: "finalize non research", mode: "finalize", kind: "execute", want: "swarm-manager-workshop-finalize"},
		{name: "finalize research", mode: "finalize", kind: "research", want: "swarm-manager-workshop-research-finalize"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			entry, ok := ResolveBacklogSkill(tt.mode, tt.kind)
			if !ok {
				t.Fatalf("ResolveBacklogSkill(%q, %q) returned no match", tt.mode, tt.kind)
			}
			if entry.SkillID != tt.want {
				t.Fatalf("ResolveBacklogSkill(%q, %q) = %q, want %q", tt.mode, tt.kind, entry.SkillID, tt.want)
			}
		})
	}
}

func TestResolveHelpers(t *testing.T) {
	capture, ok := ResolveCaptureSkill()
	if !ok {
		t.Fatal("ResolveCaptureSkill returned no match")
	}
	if capture.SkillID != "swarm-manager-classify-capture" {
		t.Fatalf("capture skill = %q", capture.SkillID)
	}

	specSync, ok := ResolveSpecSyncSkill()
	if !ok {
		t.Fatal("ResolveSpecSyncSkill returned no match")
	}
	if specSync.SkillID != "spec-sync" {
		t.Fatalf("spec-sync skill = %q", specSync.SkillID)
	}
}

func TestSkillUsageSummary(t *testing.T) {
	if got := SkillUsageCount("swarm-manager-workshop"); got != 1 {
		t.Fatalf("workshop direct usage count = %d, want 1", got)
	}
	if got := SkillUsageCount("swarm-manager-backlog-tools"); got != 8 {
		t.Fatalf("backlog-tools reference count = %d, want 8", got)
	}
	if got := SkillImpactSummary("swarm-manager-backlog-tools"); got != "Referenced by 8 runtime prompt paths." {
		t.Fatalf("unexpected backlog-tools summary: %q", got)
	}
	if got := SkillImpactSummary("spec-sync"); got != "Used directly by 1 runtime prompt path." {
		t.Fatalf("unexpected spec-sync summary: %q", got)
	}
}

func TestResolveInitiativeSkill(t *testing.T) {
	feedback, ok := ResolveInitiativeSkill("feedback")
	if !ok {
		t.Fatal("expected feedback resolver hit")
	}
	if feedback.SkillID != "swarm-manager-initiative-feedback" {
		t.Fatalf("feedback skill = %q", feedback.SkillID)
	}
	cont, ok := ResolveInitiativeSkill("feedback_continue")
	if !ok || cont.SkillID != "swarm-manager-initiative-feedback" {
		t.Fatalf("feedback_continue should map to feedback skill, got ok=%v skill=%q", ok, cont.SkillID)
	}
	review, ok := ResolveInitiativeSkill("review")
	if !ok || review.SkillID != "swarm-manager-initiative-review" {
		t.Fatalf("review skill = %q (ok=%v)", review.SkillID, ok)
	}
	if _, ok := ResolveInitiativeSkill("unknown"); ok {
		t.Fatal("expected unknown purpose to miss")
	}
}

func TestResolveInitiativeModeSkill(t *testing.T) {
	for _, mode := range []operatingmode.Mode{operatingmode.ModeHolisticLoop, operatingmode.ModePhasedPlanDrain} {
		def, err := operatingmode.DefinitionFor(mode)
		if err != nil {
			t.Fatalf("DefinitionFor(%q): %v", mode, err)
		}
		for phase, phaseDef := range def.PhaseGraph.Phases {
			entry, ok := ResolveInitiativeModeSkill(string(mode), string(phase))
			if phaseDef.Delegated() {
				// A delegated phase has no prompt of its own — its execution
				// surface resolves through the sub-mode's phases.
				if ok {
					t.Fatalf("ResolveInitiativeModeSkill(%q, %q) resolved %q for a delegated phase", mode, phase, entry.ID)
				}
				continue
			}
			if !ok {
				t.Fatalf("ResolveInitiativeModeSkill(%q, %q) missed", mode, phase)
			}
			if entry.ID != phaseDef.CatalogID {
				t.Fatalf("%s/%s catalog id = %q, want %q", mode, phase, entry.ID, phaseDef.CatalogID)
			}
			if entry.SkillID != phaseDef.SkillID {
				t.Fatalf("%s/%s skill = %q, want %q", mode, phase, entry.SkillID, phaseDef.SkillID)
			}
			expected, ok := operatingmode.ExpectedPromptCatalogEntry(string(mode), string(phase))
			if !ok {
				t.Fatalf("ExpectedPromptCatalogEntry(%q, %q) missed", mode, phase)
			}
			if !sameStrings(entry.OutputPaths, expected.OutputPaths) {
				t.Fatalf("%s/%s output paths = %v, want %v", mode, phase, entry.OutputPaths, expected.OutputPaths)
			}
		}
	}

	if _, ok := ResolveInitiativeModeSkill("item-level", "execute"); ok {
		t.Fatal("item-level should not resolve through initiative mode phase catalog")
	}
	if _, ok := ResolveInitiativeModeSkill("holistic-loop", "unknown"); ok {
		t.Fatal("unknown phase should miss")
	}
}

func TestInitiativeModePromptCatalogEntriesComeFromRegistry(t *testing.T) {
	for _, expected := range operatingmode.PromptCatalogEntries() {
		entry, ok := ResolveInitiativeModeSkill(expected.Mode, expected.Phase)
		if !ok {
			t.Fatalf("ResolveInitiativeModeSkill(%q, %q) missed generated entry", expected.Mode, expected.Phase)
		}
		if entry.ID != expected.CatalogID || entry.SkillID != expected.SkillID {
			t.Fatalf("%s/%s prompt IDs = %q/%q, want %q/%q", expected.Mode, expected.Phase, entry.ID, entry.SkillID, expected.CatalogID, expected.SkillID)
		}
		if entry.Title != expected.Title || entry.Trigger != expected.Trigger || entry.Purpose != expected.Purpose {
			t.Fatalf("%s/%s prompt metadata drifted: got title=%q trigger=%q purpose=%q", expected.Mode, expected.Phase, entry.Title, entry.Trigger, entry.Purpose)
		}
		if !sameStrings(entry.OutputPaths, expected.OutputPaths) {
			t.Fatalf("%s/%s output paths = %v, want %v", expected.Mode, expected.Phase, entry.OutputPaths, expected.OutputPaths)
		}
	}
}

func sameStrings(a, b []string) bool {
	if len(a) != len(b) {
		return false
	}
	for i := range a {
		if a[i] != b[i] {
			return false
		}
	}
	return true
}

func TestVariableKeysForSkill(t *testing.T) {
	keys := VariableKeysForSkill("swarm-manager-classify-capture")
	if len(keys) != 2 {
		t.Fatalf("expected classify variable keys, got %v", keys)
	}
	if keys[0] != "CAPTURE_ID" || keys[1] != "CAPTURE_TEXT" {
		t.Fatalf("unexpected classify variable keys: %v", keys)
	}
}
