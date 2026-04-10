package promptcatalog

import "testing"

func TestResolveBacklogSkill(t *testing.T) {
	tests := []struct {
		name string
		mode string
		kind string
		want string
	}{
		{name: "workshop non research", mode: "workshop", kind: "idea", want: "swarm-manager-workshop"},
		{name: "workshop research", mode: "workshop", kind: "research", want: "swarm-manager-workshop-research"},
		{name: "initialize any kind", mode: "initialize", kind: "fix", want: "swarm-manager-initialize-backlog"},
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
	if got := SkillUsageCount("swarm-manager-backlog-tools"); got != 6 {
		t.Fatalf("backlog-tools reference count = %d, want 6", got)
	}
	if got := SkillImpactSummary("swarm-manager-backlog-tools"); got != "Referenced by 6 runtime prompt paths." {
		t.Fatalf("unexpected backlog-tools summary: %q", got)
	}
	if got := SkillImpactSummary("spec-sync"); got != "Used directly by 1 runtime prompt path." {
		t.Fatalf("unexpected spec-sync summary: %q", got)
	}
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
