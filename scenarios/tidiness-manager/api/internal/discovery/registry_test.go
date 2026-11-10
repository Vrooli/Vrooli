package discovery

import (
	"strings"
	"testing"

	common "github.com/vrooli/vrooli/packages/proto/gen/go/common/v1"
	"tidiness-manager/internal/testutil"
)

func TestLabelledCorpusAndCalibrationProvenance(t *testing.T) {
	registry := New()
	for _, detector := range registry.detectors {
		for _, c := range corpusCases() {
			if c.DetectorID != detector.ID {
				continue
			}
			t.Run(c.ID, func(t *testing.T) {
				if got := len(detector.detect(c.input())) > 0; got != c.Expected {
					t.Fatalf("detector hit=%v, labelled expectation=%v: %s", got, c.Expected, c.Reason)
				}
				if c.Source == "" || c.Reason == "" {
					t.Fatal("label lacks source or rationale")
				}
			})
		}
		if detector.Calibration.RegressionFailures != 0 {
			t.Fatal(detector.Calibration)
		}
		if detector.Calibration.HistoricalCases == 0 && (detector.Calibration.Precision != nil || detector.Calibration.Recall != nil) {
			t.Fatal("synthetic regression cases must not claim historical precision or recall")
		}
	}
}

func TestRankUsesHistoricalExpectedNetAndReportsUncalibratedCandidates(t *testing.T) {
	registry := New()
	precision := .25
	registry.detectors = []Detector{
		{ID: "large-unproven", detect: func([]symbol) []Opportunity {
			return []Opportunity{{Locations: []Location{{Path: "a.go"}}, RemovableLines: 1000}}
		}},
		{ID: "measured", Calibration: Calibration{Precision: &precision, HistoricalCases: 4}, detect: func([]symbol) []Opportunity {
			return []Opportunity{
				{Locations: []Location{{Path: "c.go"}}, RemovableLines: 120, ReplacementLines: 20},
				{Locations: []Location{{Path: "b.go"}}, RemovableLines: 80, ReplacementLines: 10},
			}
		}},
	}
	report := registry.Recommend(nil, 0, 75)
	if len(report.Recommendations) != 2 || report.Recommendations[0].Locations[0].Path != "c.go" || *report.Recommendations[0].ExpectedNetLines != 25 || report.Recommendations[1].ExpectedNetLines != nil {
		t.Fatalf("ranking/minimum must use net reduction and preserve unknown calibration: %+v", report)
	}
	if got := registry.Recommend(nil, 1, 0); got.Count != 1 || got.Recommendations[0].DetectorID != "measured" {
		t.Fatal(got)
	}
}

func TestGraphEvidenceExcludesTestsGeneratedEscapesAndDuplicateNodes(t *testing.T) {
	node := func(path string) *common.CodeGraphNode {
		return testutil.GraphNode(path, "old", path, map[string]string{"signature": "func()", "start_line": "2", "end_line": "10"})
	}
	report := testutil.GraphReport("/repo/scenarios/demo",
		testutil.GraphUnit{ID: "api", Language: "go", RootPath: "/repo/scenarios/demo/api", Nodes: []*common.CodeGraphNode{node("a.go"), node("a.go"), node("a_test.go"), node("x.pb.go"), node("vendor/a.go"), node("generated/a.go"), node("../../escape.go"), node("/elsewhere/a.go")}},
		testutil.GraphUnit{ID: "escaped", Language: "go", RootPath: "/repo/scenarios/other", Nodes: []*common.CodeGraphNode{node("a.go")}},
	)
	symbols, warnings := graphSymbols(report, false)
	if len(symbols) != 1 || symbols[0].path != "api/a.go" {
		t.Fatalf("unsafe or duplicate evidence admitted: %+v", symbols)
	}
	if len(warnings) != 3 || !strings.Contains(strings.Join(warnings, " "), "outside target") {
		t.Fatalf("missing scope diagnostics: %v", warnings)
	}
}

func TestIdenticalBodyReductionKeepsOneOwnerAndAccountsForWrappers(t *testing.T) {
	result := identicalBodies([]symbol{{path: "z.go", name: "z", bodyHash: "same", start: 20, end: 39, bodyLines: 20}, {path: "a.go", name: "a", bodyHash: "same", start: 1, end: 20, bodyLines: 20}, {path: "b.go", name: "b", bodyHash: "same", start: 1, end: 20, bodyLines: 20}})
	if len(result) != 1 || result[0].RemovableLines != 40 || result[0].ReplacementLines != 6 || result[0].Locations[0].Path != "a.go" {
		t.Fatal(result)
	}
}
