package promptcatalog

import (
	"testing"
)

func TestResolveHelpers(t *testing.T) {
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
	if _, ok := Lookup("support-backlog-tools"); ok {
		t.Fatal("retired backlog-tools prompt remains catalogued")
	}
	if got := SkillImpactSummary("spec-sync"); got != "Used directly by 1 runtime prompt path." {
		t.Fatalf("unexpected spec-sync summary: %q", got)
	}
}

func TestGoalContextCatalogEntry(t *testing.T) {
	entry, ok := Lookup("support-goal-context")
	if !ok || entry.SkillID != "swarm-manager-goal-context" {
		t.Fatalf("goal context entry = %+v (ok=%v)", entry, ok)
	}
}
