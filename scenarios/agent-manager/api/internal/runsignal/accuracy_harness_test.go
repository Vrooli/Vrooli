package runsignal

import (
	"path/filepath"
	"testing"
)

// The structural corpus scores every shipped detector for the conformance
// response; the precision corpus (precision_test.go) is the gate on real runs.
func TestStructuralCorpusMeetsItsThresholds(t *testing.T) {
	root, err := filepath.Abs(filepath.Join("..", "..", "..", "..", ".."))
	if err != nil {
		t.Fatal(err)
	}
	results, err := ClassificationAccuracy(root)
	if err != nil {
		t.Fatal(err)
	}
	for _, r := range results {
		if r.Precision < r.Threshold || r.Recall < r.Threshold {
			t.Errorf("%s precision=%.2f recall=%.2f below %.2f", r.ID, r.Precision, r.Recall, r.Threshold)
		}
	}
}
