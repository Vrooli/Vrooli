package runsignal

import (
	"path/filepath"
	"testing"

	"agent-manager/internal/transcriptredact"
)

func TestClassificationAccuracyHarness(t *testing.T) {
	root, err := filepath.Abs(filepath.Join("..", "..", "..", "..", ".."))
	if err != nil {
		t.Fatal(err)
	}
	results, err := ClassificationAccuracy(root)
	if err != nil {
		t.Fatal(err)
	}
	if len(results) == 0 {
		t.Fatal("no detector accuracy results")
	}
	for _, result := range results {
		if result.Precision < result.Threshold || result.Recall < result.Threshold {
			t.Fatalf("%s precision=%.2f recall=%.2f below %.2f", result.ID, result.Precision, result.Recall, result.Threshold)
		}
		t.Logf("%s precision=%.2f recall=%.2f", result.ID, result.Precision, result.Recall)
	}
}

func TestClassificationAccuracyHarnessRejectsBrokenDetector(t *testing.T) {
	labels := accuracyLabels{Expected: []string{"episode/repeated-work"}, Cases: []accuracyCase{{Name: "missing-positive", Expected: []string{"episode/repeated-work"}}}}
	scores, err := scoreDetectors(labels)
	if err != nil {
		t.Fatal(err)
	}
	if scores["episode/repeated-work"].recall() != 0 {
		t.Fatal("broken detector unexpectedly passed recall")
	}
}

func TestClassificationCorpusIsCanonicalRedacted(t *testing.T) {
	violations, err := transcriptredact.ScanDir("testdata/classification")
	if err != nil {
		t.Fatal(err)
	}
	if len(violations) != 0 {
		t.Fatalf("classification corpus is not redacted: %v", violations)
	}
}
