package memberflow

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func mkMember(team, member string, t Topics) MemberTopics {
	return MemberTopics{Ref: MemberRef{Team: team, Member: member}, Topics: t, Exists: true}
}

func TestRule_ConflictingDrain_OverlappingPrefixes(t *testing.T) {
	members := []MemberTopics{
		mkMember("team-a", "alice", Topics{
			Intake: []IntakeEntry{{Prefix: "research-inbox/*", DrainedBySkill: "router-a"}},
		}),
		mkMember("team-b", "bob", Topics{
			Intake: []IntakeEntry{{Prefix: "research-inbox/audience/*", DrainedBySkill: "router-b"}},
		}),
	}
	r := Validate(members, ValidationOptions{})
	if r.Errors == 0 {
		t.Fatalf("expected conflicting_drain finding, got %v", r.Findings)
	}
	found := false
	for _, f := range r.Findings {
		if f.Rule == "conflicting_drain" {
			found = true
			break
		}
	}
	if !found {
		t.Errorf("no conflicting_drain finding; findings=%v", r.Findings)
	}
}

func TestRule_ConflictingDrain_DisjointPrefixes(t *testing.T) {
	members := []MemberTopics{
		mkMember("team-a", "alice", Topics{
			Intake: []IntakeEntry{{Prefix: "research-inbox/audience/*", DrainedBySkill: "router-a"}},
		}),
		mkMember("team-b", "bob", Topics{
			Intake: []IntakeEntry{{Prefix: "research-inbox/competitor/*", DrainedBySkill: "router-b"}},
		}),
	}
	r := Validate(members, ValidationOptions{})
	for _, f := range r.Findings {
		if f.Rule == "conflicting_drain" {
			t.Errorf("disjoint prefixes should not conflict; got %v", f)
		}
	}
}

func TestRule_OrphanOutput_NoConsumer(t *testing.T) {
	members := []MemberTopics{
		mkMember("team-a", "writer", Topics{
			Output: []OutputEntry{{Prefix: "isolated-output/*", DestinationKind: DestinationKnowledge}},
		}),
	}
	r := Validate(members, ValidationOptions{})
	found := false
	for _, f := range r.Findings {
		if f.Rule == "orphan_output" {
			found = true
		}
	}
	if !found {
		t.Errorf("expected orphan_output, got %v", r.Findings)
	}
}

func TestRule_OrphanOutput_KnowledgeWithConsumer(t *testing.T) {
	members := []MemberTopics{
		mkMember("team-a", "writer", Topics{
			Output: []OutputEntry{{Prefix: "shared-knowledge/*", DestinationKind: DestinationKnowledge}},
		}),
		mkMember("team-b", "reader", Topics{
			Intake:            []IntakeEntry{{Prefix: "shared-knowledge/*", DrainedBySkill: "reader-router"}},
			ExternalProducers: []string{"team-a"},
		}),
	}
	r := Validate(members, ValidationOptions{})
	for _, f := range r.Findings {
		if f.Rule == "orphan_output" {
			t.Errorf("output with consumer should not be orphan; got %v", f)
		}
	}
}

func TestRule_OrphanOutput_NonKnowledgeIsNeverOrphan(t *testing.T) {
	members := []MemberTopics{
		mkMember("team-a", "writer", Topics{
			Output: []OutputEntry{
				{Prefix: "decisions/*", DestinationKind: DestinationDecision},
				{Prefix: "doctrine/*", DestinationKind: DestinationPORFile, DestinationPath: ptr("docs/agent-system/PRIMITIVES.md")},
				{Prefix: "gaps/*", DestinationKind: DestinationCapabilityGap},
			},
		}),
	}
	r := Validate(members, ValidationOptions{})
	for _, f := range r.Findings {
		if f.Rule == "orphan_output" {
			t.Errorf("non-knowledge destination should never be orphan; got %v", f)
		}
	}
}

func TestRule_OrphanInput_NoProducer(t *testing.T) {
	members := []MemberTopics{
		mkMember("team-a", "consumer", Topics{
			Intake: []IntakeEntry{{Prefix: "lonely-input/*", DrainedBySkill: "router"}},
		}),
	}
	r := Validate(members, ValidationOptions{})
	found := false
	for _, f := range r.Findings {
		if f.Rule == "orphan_input" {
			found = true
		}
	}
	if !found {
		t.Errorf("expected orphan_input, got %v", r.Findings)
	}
}

func TestRule_OrphanInput_ExternalProducerSatisfies(t *testing.T) {
	members := []MemberTopics{
		mkMember("team-a", "consumer", Topics{
			Intake:            []IntakeEntry{{Prefix: "external-input/*", DrainedBySkill: "router"}},
			ExternalProducers: []string{"vision-walk"},
		}),
	}
	r := Validate(members, ValidationOptions{})
	for _, f := range r.Findings {
		if f.Rule == "orphan_input" {
			t.Errorf("external producer should satisfy intake; got %v", f)
		}
	}
}

func TestRule_OrphanInput_PeerProducerSatisfies(t *testing.T) {
	members := []MemberTopics{
		mkMember("team-a", "writer", Topics{
			Output: []OutputEntry{{Prefix: "shared/*", DestinationKind: DestinationKnowledge}},
		}),
		mkMember("team-b", "reader", Topics{
			Intake: []IntakeEntry{{Prefix: "shared/*", DrainedBySkill: "router"}},
		}),
	}
	r := Validate(members, ValidationOptions{})
	for _, f := range r.Findings {
		if f.Rule == "orphan_input" {
			t.Errorf("peer producer should satisfy intake; got %v", f)
		}
	}
}

func TestRule_MissingDrainSkill(t *testing.T) {
	members := []MemberTopics{
		mkMember("team-a", "consumer", Topics{
			Intake:            []IntakeEntry{{Prefix: "x/*", DrainedBySkill: "no-such-skill"}},
			ExternalProducers: []string{"operator"},
		}),
	}
	opts := ValidationOptions{SkillIDs: map[string]bool{"valid-router": true}}
	r := Validate(members, opts)
	found := false
	for _, f := range r.Findings {
		if f.Rule == "missing_drain_skill" {
			found = true
		}
	}
	if !found {
		t.Errorf("expected missing_drain_skill; findings=%v", r.Findings)
	}
}

func TestRule_MissingDrainSkill_SkippedWhenRegistryEmpty(t *testing.T) {
	members := []MemberTopics{
		mkMember("team-a", "consumer", Topics{
			Intake:            []IntakeEntry{{Prefix: "x/*", DrainedBySkill: "anything"}},
			ExternalProducers: []string{"operator"},
		}),
	}
	r := Validate(members, ValidationOptions{}) // SkillIDs nil
	for _, f := range r.Findings {
		if f.Rule == "missing_drain_skill" {
			t.Errorf("rule should be skipped when SkillIDs is empty; got %v", f)
		}
	}
}

func TestRule_DanglingPORSink(t *testing.T) {
	repoRoot := t.TempDir()
	if err := os.MkdirAll(filepath.Join(repoRoot, "docs", "agent-system"), 0o755); err != nil {
		t.Fatalf("mkdir: %v", err)
	}
	if err := os.WriteFile(filepath.Join(repoRoot, "docs", "agent-system", "EXISTING.md"), []byte("real"), 0o644); err != nil {
		t.Fatalf("write: %v", err)
	}

	members := []MemberTopics{
		mkMember("team-a", "writer", Topics{
			Output: []OutputEntry{
				{Prefix: "good/*", DestinationKind: DestinationPORFile, DestinationPath: ptr("docs/agent-system/EXISTING.md")},
				{Prefix: "bad/*", DestinationKind: DestinationPORFile, DestinationPath: ptr("docs/agent-system/MISSING.md")},
			},
		}),
	}
	r := Validate(members, ValidationOptions{RepoRoot: repoRoot})
	missingFound := 0
	for _, f := range r.Findings {
		if f.Rule == "dangling_por_sink" {
			missingFound++
			if !strings.Contains(f.Detail, "MISSING") {
				t.Errorf("dangling rule fired on existing file: %v", f)
			}
		}
	}
	if missingFound != 1 {
		t.Errorf("expected exactly 1 dangling_por_sink finding, got %d (findings=%v)", missingFound, r.Findings)
	}
}

func TestValidate_RealStoreCanary(t *testing.T) {
	// The canary backfill (marketing-crew + monetization + meta-opt + ...) on
	// the real store should validate clean for orphan rules. dangling_por_sink
	// will fire only if a member declares a por_file destination with a
	// missing path.
	storeDir := "/home/matthalloran8/Vrooli/scenarios/prompt-manager/store"
	if _, err := os.Stat(storeDir); err != nil {
		t.Skip("real store not available in this environment")
	}
	members, err := LoadAll(storeDir)
	if err != nil {
		t.Fatalf("LoadAll: %v", err)
	}
	skillIDs, err := LoadSkillIDs(storeDir)
	if err != nil {
		t.Fatalf("LoadSkillIDs: %v", err)
	}
	repoRoot := filepath.Join(storeDir, "..", "..", "..")
	repoRoot, _ = filepath.Abs(repoRoot)
	r := Validate(members, ValidationOptions{
		RepoRoot: repoRoot,
		SkillIDs: skillIDs,
	})
	if r.Errors > 0 {
		for _, f := range r.Findings {
			t.Logf("[%s] %s %s %s", f.Severity, f.Rule, f.Member, f.Detail)
		}
		t.Errorf("real-store validation produced %d errors and %d warnings", r.Errors, r.Warnings)
	}
}

func TestExitCode(t *testing.T) {
	tests := []struct {
		name   string
		result ValidationResult
		want   int
	}{
		{"clean", ValidationResult{}, 0},
		{"warnings only", ValidationResult{Warnings: 5}, 0},
		{"any error", ValidationResult{Errors: 1, Warnings: 2}, 1},
	}
	for _, tt := range tests {
		if got := tt.result.ExitCode(); got != tt.want {
			t.Errorf("%s: ExitCode = %d, want %d", tt.name, got, tt.want)
		}
	}
}
