package dimensions

import (
	"encoding/json"
	"os"
	"path/filepath"
	"testing"

	architecturev1 "github.com/vrooli/vrooli/packages/proto/gen/go/architecture/v1"
)

// fixtureAudit is a minimal projection of test-genie's SuiteExecutionResult,
// just enough to exercise the dimension mapping. P3's ingestion package owns
// the full parse shape; here we only need phase names and finding sources.
type fixtureAudit struct {
	Phases []struct {
		Findings []struct {
			Source   int32  `json:"source"`
			Code     string `json:"code"`
			StableID string `json:"stable_id"`
		} `json:"findings"`
	} `json:"phases"`
}

func loadFixture(t *testing.T) fixtureAudit {
	t.Helper()
	raw, err := os.ReadFile(filepath.Join("testdata", "testgenie_audit_fixture.json"))
	if err != nil {
		t.Fatalf("read fixture: %v", err)
	}
	var fx fixtureAudit
	if err := json.Unmarshal(raw, &fx); err != nil {
		t.Fatalf("parse fixture: %v", err)
	}
	return fx
}

// TestEverySourceMapsToValidDimension is the anti-drift guard over test-genie's
// finding sources: if test-genie's proto adds or renames a FindingSource, this
// fails until the SSOT map (dimensions.json) is updated. UNSPECIFIED is the one
// source intentionally left unmapped.
func TestEverySourceMapsToValidDimension(t *testing.T) {
	for val, name := range architecturev1.FindingSource_name {
		src := architecturev1.FindingSource(val)
		if src == architecturev1.FindingSource_FINDING_SOURCE_UNSPECIFIED {
			if _, ok := ForSource(src); ok {
				t.Errorf("UNSPECIFIED source must not map to a dimension")
			}
			continue
		}
		dim, ok := ForSource(src)
		if !ok {
			t.Errorf("FindingSource %q (%d) has no dimension mapping in dimensions.json", name, val)
			continue
		}
		if !IsValid(dim) {
			t.Errorf("FindingSource %q maps to non-vocabulary dimension %q", name, dim)
		}
	}
}

// TestFixtureFindingsMapToExactlyOneDimension asserts every finding category in
// the captured audit fixture resolves to exactly one valid dimension. Map
// semantics give "exactly one"; this proves coverage over real source values.
func TestFixtureFindingsMapToExactlyOneDimension(t *testing.T) {
	fx := loadFixture(t)
	seenSources := map[int32]bool{}
	for _, ph := range fx.Phases {
		for _, f := range ph.Findings {
			seenSources[f.Source] = true
			dim, ok := ForSource(architecturev1.FindingSource(f.Source))
			if !ok {
				t.Errorf("finding %q: source %d unmapped", f.Code, f.Source)
				continue
			}
			if !IsValid(dim) {
				t.Errorf("finding %q: dimension %q not in vocabulary", f.Code, dim)
			}
		}
	}
	// The fixture must exercise every real source so the coverage claim holds.
	for val, name := range architecturev1.FindingSource_name {
		if architecturev1.FindingSource(val) == architecturev1.FindingSource_FINDING_SOURCE_UNSPECIFIED {
			continue
		}
		if !seenSources[int32(val)] {
			t.Errorf("fixture does not exercise FindingSource %q (%d); add a finding so the mapping is proven", name, val)
		}
	}
}

func TestVocabularyIsNonEmptyAndUnique(t *testing.T) {
	all := All()
	if len(all) == 0 {
		t.Fatal("vocabulary is empty")
	}
	seen := map[Dimension]bool{}
	for _, d := range all {
		if seen[d] {
			t.Errorf("duplicate dimension %q", d)
		}
		seen[d] = true
		if !IsValid(d) {
			t.Errorf("All() returned non-valid dimension %q", d)
		}
		if Describe(d) == "" {
			t.Errorf("dimension %q has no description", d)
		}
	}
}
