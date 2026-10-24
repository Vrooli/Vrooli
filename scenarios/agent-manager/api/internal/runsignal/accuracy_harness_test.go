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

func TestSplitCorpusBySourceRunKeepsLineageTogether(t *testing.T) {
	cases := []CorpusCase{
		{Name: "run-a-window-1", SourceRun: "run-a"},
		{Name: "run-a-window-2", SourceRun: "run-a"},
		{Name: "run-b-window-1", SourceRun: "run-b"},
		{Name: "run-c-window-1", SourceRun: "run-c"},
	}
	split, err := SplitCorpusBySourceRun(cases, map[string]struct{}{"run-c": {}})
	if err != nil {
		t.Fatal(err)
	}
	if len(split.Development) != 3 || len(split.HeldOut) != 1 || split.HeldOut[0].SourceRun != "run-c" {
		t.Fatalf("split=%+v", split)
	}
	for _, item := range split.Development {
		if item.SourceRun == "run-c" {
			t.Fatalf("held-out lineage leaked into development: %+v", split)
		}
	}
}

func TestSplitCorpusBySourceRunRejectsMissingLineage(t *testing.T) {
	if _, err := SplitCorpusBySourceRun([]CorpusCase{{Name: "case-without-lineage"}}, map[string]struct{}{"run-a": {}}); err == nil {
		t.Fatal("missing source-run lineage was accepted")
	}
}
