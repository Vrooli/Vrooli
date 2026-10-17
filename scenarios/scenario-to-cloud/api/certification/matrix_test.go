package certification

import (
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"os"
	"path/filepath"
	"testing"
)

const expectedCaseCount = 94

// TestEmbeddedMatrixMatchesScenarioCopy [REQ:STC-P0-040] proves the embedded
// matrix is byte-identical to certification/matrix.json at the scenario root.
func TestEmbeddedMatrixMatchesScenarioCopy(t *testing.T) {
	data, err := os.ReadFile(filepath.Join("..", "..", "certification", "matrix.json"))
	if err != nil {
		t.Fatalf("read scenario matrix: %v", err)
	}
	sum := sha256.Sum256(data)
	if got, want := EmbeddedMatrixSHA256(), hex.EncodeToString(sum[:]); got != want {
		t.Fatalf("api/certification/matrix.json (%s) is out of sync with certification/matrix.json (%s); copy the scenario file over the embedded one", got, want)
	}
}

// TestMatrixHoldsEveryMandatoryCase [REQ:STC-P0-040] proves every case has a
// stable ID, at least one lane, dimensions and requirement refs, and that the
// mandatory inventory is complete.
func TestMatrixHoldsEveryMandatoryCase(t *testing.T) {
	m, err := LoadEmbedded()
	if err != nil {
		t.Fatalf("load embedded: %v", err)
	}
	if len(m.Cases) != expectedCaseCount {
		t.Fatalf("expected %d cases, got %d", expectedCaseCount, len(m.Cases))
	}
	if m.Source.CaseCount != len(m.Cases) {
		t.Fatalf("source.case_count %d != cases %d", m.Source.CaseCount, len(m.Cases))
	}
	families := map[string]int{}
	for _, c := range m.Cases {
		if !c.Required {
			t.Errorf("%s: mandatory matrix cases must be required", c.ID)
		}
		if len(c.Dimensions) == 0 {
			t.Errorf("%s: no dimensions declared", c.ID)
		}
		if c.Trigger == "" || c.Assertion == "" {
			t.Errorf("%s: trigger and assertion must be non-empty", c.ID)
		}
		families[c.Family]++
	}
	for _, fam := range []string{"AUTH", "PLAN", "RUN", "REACH", "RELEASE", "DATA", "SECRET", "EDGE", "GOV", "UX", "OPS"} {
		if families[fam] == 0 {
			t.Errorf("family %s has no cases", fam)
		}
	}
	if len(m.UnsupportedCells) == 0 {
		t.Fatalf("matrix declares no unsupported cells")
	}
	for _, u := range m.UnsupportedCells {
		if u.Reason == "" || u.PolicyRef == "" {
			t.Errorf("unsupported cell %s must carry a reason and policy ref", u.ID)
		}
	}
}

// TestMatrixRequirementRefsExist [REQ:STC-P0-040] proves every requirement
// reference resolves to a declared requirement in requirements/**/module.json.
func TestMatrixRequirementRefsExist(t *testing.T) {
	m, err := LoadEmbedded()
	if err != nil {
		t.Fatalf("load embedded: %v", err)
	}
	modules, err := filepath.Glob(filepath.Join("..", "..", "requirements", "*", "module.json"))
	if err != nil || len(modules) == 0 {
		t.Fatalf("no requirement modules found: %v", err)
	}
	known := map[string]bool{}
	for _, path := range modules {
		data, err := os.ReadFile(path)
		if err != nil {
			t.Fatalf("read %s: %v", path, err)
		}
		var doc struct {
			Requirements []struct {
				ID string `json:"id"`
			} `json:"requirements"`
		}
		if err := json.Unmarshal(data, &doc); err != nil {
			t.Fatalf("parse %s: %v", path, err)
		}
		for _, r := range doc.Requirements {
			known[r.ID] = true
		}
	}
	for _, c := range m.Cases {
		for _, ref := range c.RequirementRefs {
			if !known[ref] {
				t.Errorf("%s references unknown requirement %s", c.ID, ref)
			}
		}
	}
}

func TestParseRejectsVocabularyDrift(t *testing.T) {
	m, err := LoadEmbedded()
	if err != nil {
		t.Fatalf("load embedded: %v", err)
	}
	m.Cases[0].Lanes = []Lane{"container"}
	if err := m.Validate(); err == nil {
		t.Fatalf("expected lane outside vocabulary to be rejected")
	}
	m, _ = LoadEmbedded()
	m.Cases[1].ID = m.Cases[0].ID
	if err := m.Validate(); err == nil {
		t.Fatalf("expected duplicate case id to be rejected")
	}
}
