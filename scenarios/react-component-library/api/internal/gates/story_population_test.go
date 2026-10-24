package gates

import (
	"encoding/json"
	"testing"
)

// The five-stage pipeline asks whether evidence exists, not whether it shows a
// component. These tests pin the one measurement that separates the two.
func TestPersistedEvidenceCarriesPopulationCounts(t *testing.T) {
	t.Parallel()

	// Verbatim shape emitted by the harness for react-component-library:Toast 1.0.10,
	// whose specimen returns null: five stages green, nothing on the page.
	raw := []byte(`{"kind":"performance","performance":{"mountMs":2110.6,"commitCount":0,"rerenderCount":0,"nodeCount":0}}`)

	var evidence persistedEvidence
	if err := json.Unmarshal(raw, &evidence); err != nil {
		t.Fatalf("unmarshal performance evidence: %v", err)
	}
	if evidence.Performance.CommitCount != 0 || evidence.Performance.NodeCount != 0 {
		t.Fatalf("blank render must decode as zero commits and zero nodes, got %+v", evidence.Performance)
	}
	if evidence.Performance.MountMS == 0 {
		t.Fatal("mountMs must still decode alongside the population counts")
	}
}

func TestPersistedEvidenceDecodesARenderedStory(t *testing.T) {
	t.Parallel()

	// Verbatim shape from react-component-library:Banner 1.0.13, story "single".
	raw := []byte(`{"kind":"performance","performance":{"mountMs":2.8,"commitCount":4,"rerenderCount":2,"nodeCount":18}}`)

	var evidence persistedEvidence
	if err := json.Unmarshal(raw, &evidence); err != nil {
		t.Fatalf("unmarshal performance evidence: %v", err)
	}
	if evidence.Performance.CommitCount == 0 || evidence.Performance.NodeCount == 0 {
		t.Fatalf("a rendered story must not look empty, got %+v", evidence.Performance)
	}
}

// A review sheet is a roll-up over other subjects and carries zero measurements by
// construction. Reading it as an empty render would fail every asset that has one.
func TestAggregateSubjectsAreExemptFromThePopulationCheck(t *testing.T) {
	t.Parallel()

	if !isAggregateSubject("review-sheet:single,stacked,chrome-match") {
		t.Fatal("review sheets must be recognised as aggregates")
	}
	for _, subject := range []string{"default", "single", "stacked", "chrome-match", ""} {
		if isAggregateSubject(subject) {
			t.Fatalf("subject %q is a rendered specimen, not an aggregate", subject)
		}
	}
}
