package promptcatalog

import (
	"testing"
)

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
	if _, ok := Lookup("backlog-workshop"); ok {
		t.Fatal("retired workshop prompt remains catalogued")
	}
	if got := SkillUsageCount("swarm-manager-backlog-tools"); got != 1 {
		t.Fatalf("backlog-tools reference count = %d, want 1", got)
	}
	if got := SkillImpactSummary("swarm-manager-backlog-tools"); got != "Referenced by 1 runtime prompt path." {
		t.Fatalf("unexpected backlog-tools summary: %q", got)
	}
	if got := SkillImpactSummary("spec-sync"); got != "Used directly by 1 runtime prompt path." {
		t.Fatalf("unexpected spec-sync summary: %q", got)
	}
}

func TestResolveInitiativeSkill(t *testing.T) {
	review, ok := ResolveInitiativeSkill("review")
	if !ok || review.SkillID != "swarm-manager-initiative-review" {
		t.Fatalf("review skill = %q (ok=%v)", review.SkillID, ok)
	}
	if _, ok := ResolveInitiativeSkill("unknown"); ok {
		t.Fatal("expected unknown purpose to miss")
	}
}

func TestResolveInitiativeModeSkill(t *testing.T) {
	if _, ok := ResolveInitiativeModeSkill("item-level", "execute"); ok {
		t.Fatal("workflow composition must not resolve retired initiative-mode prompts")
	}
}

func TestInitiativeModePromptCatalogEntriesComeFromRegistry(t *testing.T) {
	if _, ok := ResolveInitiativeModeSkill("holistic-loop", "investigate"); ok {
		t.Fatal("retired operating modes must not publish runtime prompts")
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
