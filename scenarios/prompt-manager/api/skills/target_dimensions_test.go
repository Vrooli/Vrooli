package skills

import (
	"encoding/json"
	"os"
	"path/filepath"
	"testing"

	"prompt-manager/store"
)

// steerSkillsWithDimensions is the set of steer skills ecosystem-manager may
// select. Each must declare at least one target dimension so the controller's
// dimension → eligible-skills index is never empty for them. The in-vocabulary
// check lives in ecosystem-manager (which owns the dimension SSOT); here we
// only guard that the declaration is present and non-empty.
var steerSkillsWithDimensions = []string{
	"progress", "ux", "refactor", "test", "security", "performance",
	"polish", "explore", "documentation-health",
	"screaming-architecture-audit", "temporal-flow-audit",
}

// TestSkillJSONRoundTripsTargetDimensions proves the new field survives the
// skill.json (store.Skill) marshal/unmarshal boundary.
func TestSkillJSONRoundTripsTargetDimensions(t *testing.T) {
	in := store.Skill{
		ID:               "demo",
		Name:             "Demo",
		Status:           store.StatusActive,
		Entry:            "SKILL.md",
		TargetDimensions: []string{"standards", "tests"},
	}
	raw, err := json.Marshal(in)
	if err != nil {
		t.Fatalf("marshal: %v", err)
	}
	var out store.Skill
	if err := json.Unmarshal(raw, &out); err != nil {
		t.Fatalf("unmarshal: %v", err)
	}
	if len(out.TargetDimensions) != 2 || out.TargetDimensions[0] != "standards" || out.TargetDimensions[1] != "tests" {
		t.Fatalf("targetDimensions not preserved: %v", out.TargetDimensions)
	}
}

// TestToMetadataCarriesTargetDimensions proves the field flows store.Skill →
// Metadata, the first hop toward the catalog Response surface.
func TestToMetadataCarriesTargetDimensions(t *testing.T) {
	adapter := NewStoreAdapter(NewMockSkillStore(), nil)
	md := adapter.toMetadata(store.Skill{
		ID:               "demo",
		Name:             "Demo",
		TargetDimensions: []string{"security"},
	})
	if len(md.TargetDimensions) != 1 || md.TargetDimensions[0] != "security" {
		t.Fatalf("toMetadata dropped targetDimensions: %v", md.TargetDimensions)
	}
	back := adapter.fromMetadata(md, "core")
	if len(back.TargetDimensions) != 1 || back.TargetDimensions[0] != "security" {
		t.Fatalf("fromMetadata dropped targetDimensions: %v", back.TargetDimensions)
	}
}

// TestSteerSkillsDeclareTargetDimensions is the populate guard: every steer
// skill ecosystem-manager may select declares ≥1 target dimension in its
// shipped skill.json.
func TestSteerSkillsDeclareTargetDimensions(t *testing.T) {
	root := filepath.Join("..", "..", "store", "skills", "packs", "core")
	for _, id := range steerSkillsWithDimensions {
		path := filepath.Join(root, id, "skill.json")
		raw, err := os.ReadFile(path)
		if err != nil {
			t.Errorf("%s: cannot read skill.json: %v", id, err)
			continue
		}
		var s store.Skill
		if err := json.Unmarshal(raw, &s); err != nil {
			t.Errorf("%s: invalid skill.json: %v", id, err)
			continue
		}
		if len(s.TargetDimensions) == 0 {
			t.Errorf("%s: declares no targetDimensions; EM selection cannot route findings to it", id)
		}
	}
}
