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
// NOTE: "explore" is intentionally absent — it is an experimentation mode
// selected manually/ad-hoc, not routed by the objective controller, so it
// declares no targetDimensions (EM-P7; see ecosystem-manager DIMENSIONS.md).
var steerSkillsWithDimensions = []string{
	"progress", "ux", "refactor", "test", "security", "performance",
	"polish", "documentation-health", "storage-steer",
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

// loadDimensionSSOT reads the canonical dimension vocabulary from the
// maturity-go package (which owns the SSOT). Returns nil when the cross-package
// layout is unavailable so the in-vocabulary guard skips rather than fails in an
// isolated checkout.
func loadDimensionSSOT(t *testing.T) map[string]bool {
	t.Helper()
	dir, err := os.Getwd()
	if err != nil {
		t.Fatalf("getwd: %v", err)
	}
	var ssotPath string
	for i := 0; i < 8; i++ {
		candidate := filepath.Join(dir, "packages", "maturity-go", "dimensions", "dimensions.json")
		if _, err := os.Stat(candidate); err == nil {
			ssotPath = candidate
			break
		}
		parent := filepath.Dir(dir)
		if parent == dir {
			break
		}
		dir = parent
	}
	if ssotPath == "" {
		return nil
	}
	raw, err := os.ReadFile(ssotPath)
	if err != nil {
		t.Fatalf("read dimensions SSOT %s: %v", ssotPath, err)
	}
	var doc struct {
		Dimensions []struct {
			ID string `json:"id"`
		} `json:"dimensions"`
	}
	if err := json.Unmarshal(raw, &doc); err != nil {
		t.Fatalf("parse dimensions SSOT: %v", err)
	}
	set := make(map[string]bool, len(doc.Dimensions))
	for _, d := range doc.Dimensions {
		set[d.ID] = true
	}
	return set
}

// TestSteerSkillTargetDimensionsInVocabulary tightens the populate guard beyond
// presence: every declared targetDimension across ALL shipped skill.json packs
// must be a member of the canonical dimension SSOT (packages/maturity-go). This
// is the prompt-manager-side mirror of EM's cross-package vocabulary guard — a
// typo here would make the skill silently unselectable. Catching it on both
// sides means neither side's CI can let an out-of-vocabulary dimension through.
func TestSteerSkillTargetDimensionsInVocabulary(t *testing.T) {
	ssot := loadDimensionSSOT(t)
	if ssot == nil {
		t.Skip("maturity-go dimension SSOT not reachable from this checkout; in-vocabulary guard skipped")
	}
	packs := filepath.Join("..", "..", "store", "skills", "packs")
	var declared int
	err := filepath.WalkDir(packs, func(path string, d os.DirEntry, err error) error {
		if err != nil {
			return err
		}
		if d.IsDir() || d.Name() != "skill.json" {
			return nil
		}
		raw, err := os.ReadFile(path)
		if err != nil {
			t.Errorf("%s: read: %v", path, err)
			return nil
		}
		var s store.Skill
		if err := json.Unmarshal(raw, &s); err != nil {
			t.Errorf("%s: invalid skill.json: %v", path, err)
			return nil
		}
		for _, dim := range s.TargetDimensions {
			declared++
			if !ssot[dim] {
				t.Errorf("%s: skill %q declares targetDimension %q not in the dimensions SSOT; the controller would silently drop it",
					path, s.ID, dim)
			}
		}
		return nil
	})
	if err != nil {
		t.Fatalf("walk %s: %v", packs, err)
	}
	t.Logf("in-vocabulary guard: %d targetDimensions validated against the SSOT", declared)
}
