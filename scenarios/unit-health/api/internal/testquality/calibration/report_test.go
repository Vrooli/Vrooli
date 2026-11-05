package calibration

import (
	"context"
	"encoding/json"
	"errors"
	"os"
	"path/filepath"
	"testing"

	"unit-health/internal/testquality"
)

func TestReportInventoryAndProfileRegistry(t *testing.T) {
	root := filepath.Join("..", "testdata")
	inventory, err := BuildInventory(root)
	if err != nil {
		t.Fatal(err)
	}
	if inventory.Specified != 81 || inventory.Implemented != 42 || inventory.Retired != 39 || len(inventory.Families) == 0 {
		t.Fatalf("inventory = %+v", inventory)
	}
	RegisterEmittedCodes(func() map[string]struct{} { return map[string]struct{}{"TEST_SKIPPED_OR_ONLY": {}} })
	if _, err := BuildInventory(root); err != nil {
		t.Fatal(err)
	}
	RegisterProfile("report-test", func(context.Context, string, Input) ([]testquality.Result, error) { return nil, nil })
	if _, err := evaluateProfile(context.Background(), "missing", root, Input{}); err == nil {
		t.Fatal("missing profile accepted")
	}
	if _, err := evaluateProfile(context.Background(), "report-test", root, Input{}); err != nil {
		t.Fatal(err)
	}
}

func TestRunPartitionUsesIndependentObservedResults(t *testing.T) {
	root := t.TempDir()
	input := Input{ID: "case-1", Profile: "go-syntax-v1", Kind: "parser", Root: ".", Files: []string{"snippet.go"}}
	expected := Expectation{RuleID: "rule", RuleVersion: "1", File: "snippet.go", TestID: "TestCase", Line: 1, Column: 1, Status: testquality.CheckedClean}
	writeJSON(t, filepath.Join(root, "development.json"), developmentEnvelope{Cases: []Case{{Input: input, Partition: "development", Rationale: "test", Provenance: "review", Expected: []Expectation{expected}}}})
	writeJSON(t, filepath.Join(root, "case-specification.json"), map[string]any{"cases": []map[string]string{{"id": "case-1", "group": "Go"}}})
	RegisterProfile("go-syntax-v1", func(_ context.Context, _ string, got Input) ([]testquality.Result, error) {
		return []testquality.Result{{RuleID: "rule", RuleVersion: "1", Target: testquality.Target{File: got.Files[0], TestID: "TestCase"}, SupportProfile: got.Profile, Status: testquality.CheckedClean, Reason: testquality.ReasonNone, Location: testquality.Location{Line: 1, Column: 1}}}, nil
	})
	for _, partition := range []string{"", "inventory"} {
		report, err := RunPartition(context.Background(), root, partition, "", "rule", true)
		if err != nil {
			t.Fatal(err)
		}
		if len(report.Cases) != 1 || !report.Cases[0].Matched || len(report.Limitations) != 1 {
			t.Fatalf("report = %+v", report)
		}
	}
	if report, err := RunPartition(context.Background(), root, "development", "", "other", false); err != nil {
		t.Fatal(err)
	} else if len(report.Cases) != 0 {
		t.Fatalf("filtered report = %+v", report)
	}
}

func TestAnalyzeVitestFixtureBranches(t *testing.T) {
	root := t.TempDir()
	write := func(name, source string) Input {
		path := filepath.Join(root, name)
		if err := os.WriteFile(path, []byte(source), 0o600); err != nil {
			t.Fatal(err)
		}
		return Input{Profile: "react-vitest-v2", Files: []string{name}}
	}
	for _, tc := range []struct {
		name, source, rule string
		status             testquality.Status
	}{
		{"focused", `test.only("a",()=>expect(1).toBe(1))`, "focused-test", testquality.Violation},
		{"async", `test("a",()=>return expect(Promise.resolve(1)).resolves.toBe(1))`, "async-assertion", testquality.CheckedClean},
		{"async-missing-return", `test("a",()=>expect(Promise.resolve(1)).resolves.toBe(1))`, "async-assertion", testquality.Violation},
		{"bare", `test("a",()=>expect(1);`, "malformed-expectation", testquality.Violation},
		{"skip", `test.skip("a",()=>{})`, "skip-declaration", testquality.Violation},
		{"node-assert", "import assert from \"node:assert/strict\";\ntest(\"a\",()=>assert.equal(1,1))", "assertion-observation", testquality.Unknown},
		{"empty", `test("a",()=>{})`, "assertion-observation", testquality.Violation},
		{"extended", `extended("a",()=>expect(1).toBe(1))`, "focused-test", testquality.CheckedClean},
	} {
		t.Run(tc.name, func(t *testing.T) {
			input := write(tc.name+".ts", tc.source)
			if tc.name == "async-missing-return" {
				input.Files = []string{"async-missing-return.ts"}
				if err := os.WriteFile(filepath.Join(root, input.Files[0]), []byte(`test("a",()=>expect(Promise.resolve(1)).resolves.toBe(1))`), 0o600); err != nil {
					t.Fatal(err)
				}
			}
			rows, err := analyzeVitestFixture(input, root)
			if err != nil {
				t.Fatal(err)
			}
			if rows[0].RuleID != tc.rule || rows[0].Status != tc.status {
				t.Fatalf("rows = %+v", rows)
			}
		})
	}
	unsupported := write("unsupported.ts", `test("a",()=>expect(1).toBe(1))`)
	unsupported.Profile = "react-vitest-v99"
	rows, err := analyzeVitestFixture(unsupported, root)
	if err != nil || rows[0].Status != testquality.Unknown {
		t.Fatalf("unsupported = %+v, %v", rows, err)
	}
	if _, err := analyzeVitestFixture(Input{Profile: "react-vitest-v2", Files: []string{"a.ts", "b.ts"}}, root); err == nil {
		t.Fatal("multiple fixture files accepted")
	}
	bad := write("bad.ts", `const value = 1`)
	if _, err := analyzeVitestFixture(bad, root); err == nil {
		t.Fatal("fixture without test accepted")
	}
}

func TestRunHoldoutValidationAndComparison(t *testing.T) {
	if _, err := runHoldout(t.TempDir(), "", ""); !errors.Is(err, ErrHoldoutNotFound) {
		t.Fatalf("empty holdout error = %v", err)
	}
	root := t.TempDir()
	if _, err := runHoldout(root, "missing", ""); !errors.Is(err, ErrHoldoutNotFound) {
		t.Fatalf("missing holdout error = %v", err)
	}
	dir := filepath.Join(root, "holdouts", "h1")
	if err := os.MkdirAll(dir, 0o755); err != nil {
		t.Fatal(err)
	}
	writeRaw := func(name, raw string) {
		if err := os.WriteFile(filepath.Join(dir, name), []byte(raw), 0o600); err != nil {
			t.Fatal(err)
		}
	}
	writeRaw("labels.json", `{`)
	writeRaw("observations.json", `{}`)
	if _, err := runHoldout(root, "h1", ""); err == nil {
		t.Fatal("invalid labels accepted")
	}
	writeRaw("labels.json", `{"labelled_at":"2026-09-08"}`)
	writeRaw("observations.json", `{"observed_at":"2026-09-07"}`)
	report, err := runHoldout(root, "h1", "r")
	if err != nil {
		t.Fatal(err)
	}
	if len(report.Limitations) != 1 {
		t.Fatalf("temporal report = %+v", report)
	}
	writeRaw("labels.json", `{"labelled_at":"bad","labels":[]}`)
	writeRaw("observations.json", `{"observed_at":"2026-09-08","observations":[]}`)
	if _, err := runHoldout(root, "h1", ""); err == nil {
		t.Fatal("bad label timestamp accepted")
	}
	writeRaw("labels.json", `{"labelled_at":"2026-09-08","labels":[{"testIdentity":"t","sourceIdentity":"s","ruleVersion":"1","label":"behavioral"}]}`)
	writeRaw("observations.json", `{"observed_at":"2026-09-08","observations":[]}`)
	report, err = runHoldout(root, "h1", "rule")
	if err != nil {
		t.Fatal(err)
	}
	if len(report.Holdout) != 1 || report.Holdout[0].Unknown != 1 || report.Holdout[0].PromotionAllowed {
		t.Fatalf("comparison = %+v", report)
	}
	if report.Holdout[0].WithinBudget {
		t.Fatal("incomplete comparison marked within budget")
	}
}

func TestCommittedAssertionObservationHoldoutIsReviewedAndAdvisory(t *testing.T) {
	report, err := runHoldout(filepath.Join("..", "testdata"), "assertion-observation-go-v1", "assertion-observation")
	if err != nil {
		t.Fatal(err)
	}
	if len(report.Holdout) != 1 {
		t.Fatalf("holdout rows = %+v", report.Holdout)
	}
	row := report.Holdout[0]
	if row.Labelled != 30 || row.Observed != 30 || row.Unknown != 0 || row.Budget != 0.05 || !row.WithinBudget || row.PromotionAllowed {
		t.Fatalf("holdout gate = %+v", row)
	}
}

func writeJSON(t *testing.T, path string, value any) {
	t.Helper()
	data, err := json.Marshal(value)
	if err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(path, data, 0o600); err != nil {
		t.Fatal(err)
	}
}
