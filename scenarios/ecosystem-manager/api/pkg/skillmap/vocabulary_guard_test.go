package skillmap

import (
	"encoding/json"
	"os"
	"path/filepath"
	"testing"

	"github.com/vrooli/maturity-go/dimensions"
)

// skillJSON is the minimal shape we need from a prompt-manager skill.json: its
// id and the controller-routing target dimensions it declares.
type skillJSON struct {
	ID               string   `json:"id"`
	TargetDimensions []string `json:"targetDimensions"`
}

// findPromptManagerPacks walks up from the test's working directory to locate
// the sibling prompt-manager skill store. Returns "" (and the test skips) when
// the cross-scenario layout is unavailable (e.g. an isolated checkout).
func findPromptManagerPacks(t *testing.T) string {
	t.Helper()
	dir, err := os.Getwd()
	if err != nil {
		t.Fatalf("getwd: %v", err)
	}
	for i := 0; i < 8; i++ {
		candidate := filepath.Join(dir, "prompt-manager", "store", "skills", "packs")
		if info, err := os.Stat(candidate); err == nil && info.IsDir() {
			return candidate
		}
		parent := filepath.Dir(dir)
		if parent == dir {
			break
		}
		dir = parent
	}
	return ""
}

// TestSkillTargetDimensionsInVocabulary is the cross-repo vocabulary guard
// (EM-P3): every targetDimensions value declared by any prompt-manager skill
// must be a member of the dimensions SSOT this scenario owns. A typo'd dimension
// would otherwise be SILENTLY dropped by skillmap (log.Printf warn + exclude),
// making the skill unselectable with no test failure. This test turns that into
// a hard failure at its source.
func TestSkillTargetDimensionsInVocabulary(t *testing.T) {
	packs := findPromptManagerPacks(t)
	if packs == "" {
		t.Skip("prompt-manager skill store not reachable from this checkout; cross-repo guard skipped")
	}

	var checked, declared int
	err := filepath.WalkDir(packs, func(path string, d os.DirEntry, err error) error {
		if err != nil {
			return err
		}
		if d.IsDir() || d.Name() != "skill.json" {
			return nil
		}
		checked++
		raw, err := os.ReadFile(path)
		if err != nil {
			t.Errorf("%s: read: %v", path, err)
			return nil
		}
		var s skillJSON
		if err := json.Unmarshal(raw, &s); err != nil {
			t.Errorf("%s: invalid skill.json: %v", path, err)
			return nil
		}
		for _, raw := range s.TargetDimensions {
			declared++
			if !dimensions.IsValid(dimensions.Dimension(raw)) {
				rel, _ := filepath.Rel(packs, path)
				t.Errorf("skill %q (%s) declares targetDimension %q which is not in the dimensions SSOT (packages/maturity-go/dimensions/dimensions.json); the controller would silently drop it — fix the typo or add the dimension to the SSOT",
					s.ID, rel, raw)
			}
		}
		return nil
	})
	if err != nil {
		t.Fatalf("walk %s: %v", packs, err)
	}
	if checked == 0 {
		t.Fatalf("no skill.json files found under %s — guard would silently pass", packs)
	}
	t.Logf("vocabulary guard: %d skill.json checked, %d targetDimensions validated against the SSOT", checked, declared)
}

// TestVocabularyGuardCatchesTypo proves the guard's mechanism: a deliberately
// mistyped dimension is rejected by the same IsValid check the file walk uses.
// (A fixture file on disk would itself trip TestSkillTargetDimensionsInVocabulary,
// so the negative case is asserted in-memory here.)
func TestVocabularyGuardCatchesTypo(t *testing.T) {
	if dimensions.IsValid(dimensions.Dimension("standardz")) {
		t.Fatal("expected a mistyped dimension to be rejected by the SSOT")
	}
	if !dimensions.IsValid(dimensions.Dimension("standards")) {
		t.Fatal("a valid dimension must pass — guard would be vacuous otherwise")
	}
}
