package discovery

import (
	"strconv"
	"strings"
	"testing"

	facts "github.com/vrooli/vrooli/packages/proto/gen/go/code-facts/v1/facts"
	common "github.com/vrooli/vrooli/packages/proto/gen/go/common/v1"
	"tidiness-manager/internal/testutil"
)

func TestDefaultRegistryIsProcessScoped(t *testing.T) {
	if first, second := Default(), Default(); first == nil || first != second {
		t.Fatalf("Default must reuse one calibrated registry, got %p and %p", first, second)
	}
}

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
				if c.Origin == "repository-beforeimage" && (c.EvidenceRef == "" || c.OriginalPrediction != "unknown") {
					t.Fatalf("historical source replay must retain evidence and distinguish absent original prediction: %+v", c)
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

func TestScopedTargetCalibrationRequiresExactSourceAndConfig(t *testing.T) {
	precision, recall := 1.0, 0.75
	calibration := Calibration{EvidenceFrames: []CalibrationFrame{
		{FrameID: "incomplete", CompleteCensus: true, AdjudicationComplete: false, ResponseComplete: true, Precision: &precision, Recall: &recall, Targets: []CalibrationTarget{{Scenario: "quality-health", SourceHash: "source-1", ConfigHash: "config-1", ResponseComplete: true}}},
		{FrameID: "exact", PopulationID: "population-1", CompleteCensus: true, AdjudicationComplete: true, ResponseComplete: true, Cases: 4, Precision: &precision, Recall: &recall, Targets: []CalibrationTarget{{Scenario: "quality-health", SourceHash: "source-1", ConfigHash: "config-1", ResponseComplete: true}}},
	}}
	got := calibrationForTarget(calibration, "quality-health", "source-1", "config-1")
	if got.ScopeFrameID != "exact" || got.ScopeCases != 4 || got.Precision == nil || *got.Precision != 1 || got.Recall == nil || *got.Recall != .75 || got.RepresentativeSample {
		t.Fatalf("exact target calibration = %+v", got)
	}
	for _, mismatch := range [][3]string{{"quality-health", "other-source", "config-1"}, {"quality-health", "source-1", "other-config"}, {"other-scenario", "source-1", "config-1"}} {
		got := calibrationForTarget(calibration, mismatch[0], mismatch[1], mismatch[2])
		if got.ScopeFrameID != "" || got.Precision != nil || got.Recall != nil {
			t.Errorf("nonmatching target received a scoped precision: %+v", got)
		}
	}
}

func TestJavaScriptPromptManagerCensusCalibrationIsTargetScoped(t *testing.T) {
	const codeFactsSourceHash = "964beb967782ae919cc2264019fe732d7266170f25b14dee21f468e6a681b7f7"
	const codeFactsConfigHash = "db7a78a9ebd61f3a9bcd0dabafe8857e1dda777a963abf30f13432de0b60b17e"
	registry := New()
	var calibration Calibration
	for _, detector := range registry.Describe() {
		if detector.ID == "duplication.identical-function-body.javascript" {
			calibration = detector.Calibration
			break
		}
	}
	if calibration.Precision != nil || calibration.Recall != nil {
		t.Fatalf("single-target census must not become a fleet estimate: %+v", calibration)
	}
	var census *CalibrationFrame
	for i := range calibration.EvidenceFrames {
		if calibration.EvidenceFrames[i].FrameID == "dr1-prompt-manager-javascript-census-v1" {
			census = &calibration.EvidenceFrames[i]
			break
		}
	}
	if census == nil || census.Cases != 3 || census.TruePositive != 2 || census.FalsePositive != 0 || census.FalseNegative != 0 || census.TrueNegative != 1 {
		t.Fatalf("unexpected complete JavaScript census confusion matrix: %+v", census)
	}
	if census.Precision == nil || *census.Precision != 1 || census.Recall == nil || *census.Recall != 1 || !census.CompleteCensus || !census.AdjudicationComplete || !census.ResponseComplete {
		t.Fatalf("complete source census should expose only its scoped exact-body rates: %+v", census)
	}
	if len(census.Targets) != 1 || census.Targets[0].Scenario != "prompt-manager" || census.Targets[0].SourceHash != codeFactsSourceHash || census.Targets[0].ConfigHash != codeFactsConfigHash || census.Targets[0].SourceManifestSHA256 == "" {
		t.Fatalf("target frame is missing its source/config identity: %+v", census.Targets)
	}
	got := calibrationForTarget(calibration, "prompt-manager", codeFactsSourceHash, codeFactsConfigHash)
	if got.ScopeFrameID != census.FrameID || got.Precision == nil || *got.Precision != 1 || got.Recall == nil || *got.Recall != 1 {
		t.Fatalf("exact source/config should receive the scoped frame: %+v", got)
	}
	for _, mismatch := range [][3]string{{"prompt-manager", "other-source", census.Targets[0].ConfigHash}, {"prompt-manager", census.Targets[0].SourceHash, "other-config"}, {"other-scenario", census.Targets[0].SourceHash, census.Targets[0].ConfigHash}} {
		got := calibrationForTarget(calibration, mismatch[0], mismatch[1], mismatch[2])
		if got.ScopeFrameID != "" || got.Precision != nil || got.Recall != nil {
			t.Errorf("nonmatching target got prompt-manager rates: %+v", got)
		}
	}
}

func TestAuditorFindingsJoinRegistryWithoutChangingPolicySemantics(t *testing.T) {
	registry := New()
	var auditorDetector *Detector
	for _, detector := range registry.Describe() {
		if detector.ID == "standards.scenario-auditor" {
			d := detector
			auditorDetector = &d
		}
	}
	if auditorDetector == nil || auditorDetector.Calibration.Precision != nil || auditorDetector.Calibration.Basis == "" {
		t.Fatalf("policy detector calibration must be explicitly non-precision-based: %+v", auditorDetector)
	}
	report := testutil.GraphReport("/repo/scenarios/demo")
	report.Target.Requested = &facts.CodeTarget{Scenario: "demo"}
	result := registry.RecommendWithAuditorFindings(report, []AuditorFinding{{
		RuleID: "standards.env-validation", Severity: "high", Title: "Validate environment inputs",
		Description: "The service accepts an unvalidated environment value.", FilePath: "api/config.go", LineNumber: 17,
		Recommendation: "Validate the value before use.", Standard: "Vrooli Security", Source: "scenario-auditor",
	}}, 0, 500)
	if result.Count != 1 {
		t.Fatalf("standards findings must not be dropped by reduction min-lines: %+v", result)
	}
	finding := result.Recommendations[0]
	if finding.DetectorID != "standards.scenario-auditor" || finding.PolicyRuleID != "standards.env-validation" || finding.PolicySeverity != "high" || finding.PolicyStandard != "Vrooli Security" || finding.PolicySource != "scenario-auditor" {
		t.Fatalf("auditor policy metadata changed: %+v", finding)
	}
	if finding.NetLines != 0 || finding.ExpectedNetLines != nil || finding.Calibration.Precision != nil || !strings.HasPrefix(finding.Calibration.Basis, "not-applicable") || finding.Action != "Validate the value before use." || finding.Locations[0].Path != "api/config.go" {
		t.Fatalf("policy finding was presented as measured code reduction: %+v", finding)
	}
	if result := registry.RecommendWithAuditorFindings(report, []AuditorFinding{{RuleID: "outside", FilePath: "/repo/scenarios/other/api/a.go"}}, 0, 0); result.Count != 0 || len(result.Warnings) == 0 {
		t.Fatalf("out-of-target Auditor evidence must be rejected with a warning: %+v", result)
	}
}

func TestMeasuredReductionRanksAheadOfPolicyFindings(t *testing.T) {
	precision := 0.5
	registry := &Registry{detectors: []Detector{
		{ID: "measured-reduction", Calibration: Calibration{Precision: &precision}, detect: func([]symbol) []Opportunity {
			return []Opportunity{{Locations: []Location{{Path: "api/a.go", Symbol: "a"}}, RemovableLines: 100, ReplacementLines: 10}}
		}},
		{ID: "standards.scenario-auditor", Calibration: Calibration{Basis: "not-applicable: policy findings are not reduction predictions"}},
	}}
	report := testutil.GraphReport("/repo/scenarios/demo")
	report.Target.Requested = &facts.CodeTarget{Scenario: "demo"}
	findings := []AuditorFinding{{RuleID: "policy-rule", FilePath: "api/b.go", LineNumber: 3}}

	got := registry.RecommendWithAuditorFindings(report, findings, 1, 0)
	if got.Count != 1 || got.Recommendations[0].DetectorID != "measured-reduction" {
		t.Fatalf("top reduction should win the limited ranking: %+v", got)
	}
	all := registry.RecommendWithAuditorFindings(report, findings, 0, 0)
	if all.Count != 2 || all.Recommendations[0].DetectorID != "measured-reduction" || all.Recommendations[1].DetectorID != "standards.scenario-auditor" {
		t.Fatalf("reduction and policy findings were not ordered by expected net lines: %+v", all.Recommendations)
	}
	if got := all.Recommendations[0].ExpectedNetLines; got == nil || *got != 45 {
		t.Fatalf("expected reduction = 45 lines at measured 0.5 precision, got %v", got)
	}
	if got := all.Recommendations[1]; got.NetLines != 0 || got.ExpectedNetLines != nil || got.Calibration.Precision != nil {
		t.Fatalf("policy finding received reduction estimates: %+v", got)
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

func TestDuplicateFunctionBodiesUsesSharedGraphIdentity(t *testing.T) {
	node := func(id, path string, bodyLines int, hash string) *common.CodeGraphNode {
		return testutil.GraphNode(id, "same", path, map[string]string{
			"signature": "func()", "start_line": "3", "end_line": "9",
			"body_lines": strconv.Itoa(bodyLines), "body_hash": hash,
		})
	}
	report := testutil.GraphReport("/repo/scenarios/demo", testutil.GraphUnit{
		ID: "api", Language: "go", RootPath: "/repo/scenarios/demo/api",
		Nodes: []*common.CodeGraphNode{
			node("a", "a.go", 7, "same"),
			node("b", "b.go", 7, "same"),
			node("small", "small.go", 4, "same"),
			node("test", "a_test.go", 7, "same"),
		},
	})

	groups, warnings := DuplicateFunctionBodies(report, 5)
	if len(warnings) != 0 {
		t.Fatalf("unexpected graph warnings: %v", warnings)
	}
	if len(groups) != 1 || groups[0].Lines != 7 {
		t.Fatalf("expected one seven-line production duplicate group, got %+v", groups)
	}
	if got := groups[0].Locations; len(got) != 2 || got[0].Path != "api/a.go" || got[1].Path != "api/b.go" {
		t.Fatalf("unexpected duplicate locations: %+v", got)
	}

	report = testutil.GraphReport("/repo/scenarios/demo", testutil.GraphUnit{
		ID: "ui", Language: "typescript", RootPath: "/repo/scenarios/demo/ui",
		Nodes: []*common.CodeGraphNode{node("a", "a.ts", 7, "same"), node("b", "b.ts", 7, "same")},
	})
	groups, warnings = DuplicateFunctionBodies(report, 5)
	if len(warnings) != 0 || len(groups) != 1 || len(groups[0].Locations) != 2 {
		t.Fatalf("TypeScript graph body identities must use the same detector: groups=%+v warnings=%v", groups, warnings)
	}

	jsNode := func(id, path, language string) *common.CodeGraphNode {
		return testutil.GraphNode(id, id, path, map[string]string{
			"language": language, "signature": "function()", "start_line": "3", "end_line": "9",
			"body_lines": "7", "body_hash": "javascript-same",
		})
	}
	jsTestNode := func(id, path string) *common.CodeGraphNode {
		node := jsNode(id, path, "javascript")
		node.Attributes["is_test"] = "true"
		return node
	}
	report = testutil.GraphReport("/repo/scenarios/demo", testutil.GraphUnit{
		ID: "ui", Language: "typescript", RootPath: "/repo/scenarios/demo/ui",
		Nodes: []*common.CodeGraphNode{jsNode("a", "a.js", "javascript"), jsNode("b", "b.js", "javascript"), jsTestNode("test-a", "a.test.js"), jsTestNode("test-b", "b.test.js")},
	})
	groups, warnings = DuplicateFunctionBodiesForLanguage(report, 5, "javascript")
	if len(warnings) != 0 || len(groups) != 1 || len(groups[0].Locations) != 2 {
		t.Fatalf("JavaScript graph nodes inside a TypeScript project must be language-routed: groups=%+v warnings=%v", groups, warnings)
	}
}

func TestDuplicateFunctionBodiesByLanguageSharesGraphEvidence(t *testing.T) {
	makeNode := func(id, path, language string) *common.CodeGraphNode {
		return testutil.GraphNode(id, id, path, map[string]string{
			"language": language, "signature": "func()", "start_line": "3", "end_line": "9",
			"body_lines": "7", "body_hash": "same-" + language,
		})
	}
	report := testutil.GraphReport("/repo",
		testutil.GraphUnit{ID: "api", Language: "go", RootPath: "/repo/api", Nodes: []*common.CodeGraphNode{makeNode("g1", "a.go", "go"), makeNode("g2", "b.go", "go")}},
		testutil.GraphUnit{ID: "ui", Language: "typescript", RootPath: "/repo/ui", Nodes: []*common.CodeGraphNode{makeNode("t1", "a.ts", "typescript"), makeNode("t2", "b.ts", "typescript")}},
	)
	groups, warnings, evidence := DuplicateFunctionBodiesByLanguageWithEvidence(report, 5)
	if len(warnings) != 0 || len(groups["go"]) != 1 || len(groups["typescript"]) != 1 {
		t.Fatalf("multi-language groups=%+v warnings=%v", groups, warnings)
	}
	for _, path := range []string{"api/a.go", "api/b.go", "ui/a.ts", "ui/b.ts"} {
		if len(evidence[path]) != 1 {
			t.Errorf("language evidence for %s = %v", path, evidence[path])
		}
	}
}

func TestIdenticalBodyReductionKeepsOneOwnerAndAccountsForWrappers(t *testing.T) {
	result := identicalBodies([]symbol{{language: "go", path: "z.go", name: "z", bodyHash: "same", start: 20, end: 39, bodyLines: 20}, {language: "go", path: "a.go", name: "a", bodyHash: "same", start: 1, end: 20, bodyLines: 20}, {language: "go", path: "b.go", name: "b", bodyHash: "same", start: 1, end: 20, bodyLines: 20}})
	if len(result) != 1 || result[0].RemovableLines != 40 || result[0].ReplacementLines != 6 || result[0].Locations[0].Path != "a.go" {
		t.Fatal(result)
	}
}
