package skills

import (
	"encoding/json"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"prompt-manager/internal/store"
)

func strPtr(s string) *string { return &s }

// TestSkillJSONRoundTripsProgrammaticHome proves the field survives the
// skill.json (store.Skill) marshal/unmarshal boundary, both set and unset.
func TestSkillJSONRoundTripsProgrammaticHome(t *testing.T) {
	in := store.Skill{
		ID:               "demo",
		Name:             "Demo",
		Status:           store.StatusActive,
		Entry:            "SKILL.md",
		ProgrammaticHome: strPtr("test-genie:architecture"),
	}
	raw, err := json.Marshal(in)
	if err != nil {
		t.Fatalf("marshal: %v", err)
	}
	var out store.Skill
	if err := json.Unmarshal(raw, &out); err != nil {
		t.Fatalf("unmarshal: %v", err)
	}
	if out.ProgrammaticHome == nil || *out.ProgrammaticHome != "test-genie:architecture" {
		t.Fatalf("programmaticHome not preserved: %v", out.ProgrammaticHome)
	}

	// Unset must round-trip as nil and omit from JSON (no "programmaticHome" key).
	rawUnset, err := json.Marshal(store.Skill{ID: "demo", Name: "Demo", Status: store.StatusActive, Entry: "SKILL.md"})
	if err != nil {
		t.Fatalf("marshal unset: %v", err)
	}
	var generic map[string]any
	if err := json.Unmarshal(rawUnset, &generic); err != nil {
		t.Fatalf("unmarshal unset: %v", err)
	}
	if _, present := generic["programmaticHome"]; present {
		t.Fatalf("unset programmaticHome should be omitted from JSON, got: %s", rawUnset)
	}
}

func TestProgramRuntimeSkillDeclaresHighArityDiscoveryVocabulary(t *testing.T) {
	root := filepath.Join("..", "..", "..", "store", "skills", "packs", "core")
	skill := readSkillJSON(t, filepath.Join(root, "program-runtime", "skill.json"))
	description := strings.ToLower(skill.Description)
	for _, phrase := range []string{"high-arity", "multi-scenario", "fan out", "cross-scenario", "discard intermediate data", "bounded results", "tool-call loops"} {
		if !strings.Contains(description, phrase) {
			t.Errorf("program-runtime description missing discovery phrase %q: %q", phrase, skill.Description)
		}
	}
	tags := make(map[string]struct{}, len(skill.Tags))
	for _, tag := range skill.Tags {
		tags[strings.ToLower(tag)] = struct{}{}
	}
	for _, tag := range []string{"high arity", "multi-scenario", "cross-scenario", "fan-out", "discard intermediate data", "return bounded results", "tool-call compression"} {
		if _, ok := tags[tag]; !ok {
			t.Errorf("program-runtime tags missing %q: %v", tag, skill.Tags)
		}
	}
}

// TestToMetadataCarriesProgrammaticHome proves the field flows store.Skill →
// Metadata → store.Skill without being dropped by the adapter.
func TestToMetadataCarriesProgrammaticHome(t *testing.T) {
	adapter := NewStoreAdapter(NewMockSkillStore(), nil)
	md := adapter.toMetadata(store.Skill{
		ID:               "demo",
		Name:             "Demo",
		ProgrammaticHome: strPtr("scenario-auditor:some-rule"),
	})
	if md.ProgrammaticHome == nil || *md.ProgrammaticHome != "scenario-auditor:some-rule" {
		t.Fatalf("toMetadata dropped programmaticHome: %v", md.ProgrammaticHome)
	}
	back := adapter.fromMetadata(md, "core")
	if back.ProgrammaticHome == nil || *back.ProgrammaticHome != "scenario-auditor:some-rule" {
		t.Fatalf("fromMetadata dropped programmaticHome: %v", back.ProgrammaticHome)
	}
}

// TestMetadataChangedDetectsProgrammaticHome proves a change in the pointer is
// detected (so a graduation write actually persists a new revision).
func TestMetadataChangedDetectsProgrammaticHome(t *testing.T) {
	base := Metadata{ID: "demo", Name: "Demo"}
	graduated := Metadata{ID: "demo", Name: "Demo", ProgrammaticHome: strPtr("test-genie:architecture")}
	if !metadataChanged(base, graduated) {
		t.Fatal("metadataChanged should detect nil → value")
	}
	if !metadataChanged(graduated, base) {
		t.Fatal("metadataChanged should detect value → nil")
	}
	if metadataChanged(graduated, graduated) {
		t.Fatal("metadataChanged should not flag identical programmaticHome")
	}
}

// TestFilterWithoutProgrammaticHome proves the frontier query keeps only skills
// whose detection has not graduated (nil or empty pointer).
func TestFilterWithoutProgrammaticHome(t *testing.T) {
	skills := []Metadata{
		{ID: "still-agentic", Modes: []string{"steer"}},
		{ID: "graduated", Modes: []string{"steer"}, ProgrammaticHome: strPtr("test-genie:architecture")},
		{ID: "empty-pointer", Modes: []string{"steer"}, ProgrammaticHome: strPtr("  ")},
	}
	got := Filter(skills, FilterOptions{Modes: []string{"steer"}, WithoutProgrammaticHome: true})
	if len(got) != 2 {
		t.Fatalf("expected 2 frontier skills, got %d: %+v", len(got), got)
	}
	for _, s := range got {
		if s.ID == "graduated" {
			t.Fatalf("graduated skill should be excluded from frontier query")
		}
	}
}

// TestScreamingArchitectureGraduated is the Phase 3 pilot proof: the shipped
// screaming-architecture-audit skill.json carries the test-genie:architecture
// pointer and is therefore EXCLUDED from the frontier query, while the other
// quality-auditor steer skills remain INCLUDED.
func TestScreamingArchitectureGraduated(t *testing.T) {
	root := filepath.Join("..", "..", "..", "store", "skills", "packs", "core")

	// The pilot skill must carry the pointer.
	pilot := readSkillJSON(t, filepath.Join(root, "screaming-architecture-audit", "skill.json"))
	if pilot.ProgrammaticHome == nil || *pilot.ProgrammaticHome != "test-genie:architecture" {
		t.Fatalf("screaming-architecture-audit must declare programmaticHome=test-genie:architecture, got: %v", pilot.ProgrammaticHome)
	}

	// The other 6 quality-auditor rotation skills must NOT have graduated yet.
	stillFrontier := []string{
		"boundary-of-responsibility-enforcement",
		"seam-discovery-and-enforcement",
		"invariant-discovery-and-enforcement",
		"cognitive-load-reduction",
		"decision-boundary-extraction",
		"code-cleanup",
	}
	for _, id := range stillFrontier {
		s := readSkillJSON(t, filepath.Join(root, id, "skill.json"))
		if s.ProgrammaticHome != nil && *s.ProgrammaticHome != "" {
			t.Errorf("%s unexpectedly carries programmaticHome=%q; rotation proof assumes it is still on the frontier", id, *s.ProgrammaticHome)
		}
	}
}

func readSkillJSON(t *testing.T, path string) store.Skill {
	t.Helper()
	raw, err := os.ReadFile(path)
	if err != nil {
		t.Fatalf("cannot read %s: %v", path, err)
	}
	var s store.Skill
	if err := json.Unmarshal(raw, &s); err != nil {
		t.Fatalf("invalid skill.json %s: %v", path, err)
	}
	return s
}
