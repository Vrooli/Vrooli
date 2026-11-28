package facts

import (
	"strings"
	"testing"

	factsv1 "github.com/vrooli/vrooli/packages/proto/gen/go/code-facts/v1/facts"
)

func TestImportSearchCLIRequiresAnExplicitScenarioTarget(t *testing.T) {
	imports := []factsv1.FactFamily{factsv1.FactFamily_FACT_FAMILY_IMPORTS}
	for name, target := range map[string]*factsv1.CodeTarget{
		"default":          nil,
		"project":          {Kind: factsv1.TargetKind_TARGET_KIND_PROJECT},
		"empty scenario":   {Kind: factsv1.TargetKind_TARGET_KIND_SCENARIO},
		"invalid scenario": {Kind: factsv1.TargetKind_TARGET_KIND_SCENARIO, Scenario: "../outside"},
	} {
		t.Run(name, func(t *testing.T) {
			if err := validateImportSearchTarget(target, imports); err == nil {
				t.Fatal("unscoped or invalid import target was accepted")
			}
		})
	}
	if err := validateImportSearchTarget(&factsv1.CodeTarget{Kind: factsv1.TargetKind_TARGET_KIND_SCENARIO, Scenario: "demo"}, imports); err != nil {
		t.Fatalf("valid scenario target rejected: %v", err)
	}
	if err := validateImportSearchTarget(nil, []factsv1.FactFamily{factsv1.FactFamily_FACT_FAMILY_SYMBOLS}); err != nil {
		t.Fatalf("ordinary source search acquired import scope requirements: %v", err)
	}
}

func TestImportSearchCLIReportKeepsSourceHashLineAndGeneration(t *testing.T) {
	const digest = "sha256:0123456789abcdef"
	report := (&handlers{}).searchReport(nil, &factsv1.SearchResponse{
		Results: []*factsv1.SearchHit{{
			Title: "scenarios/demo/api/main.go imports context", Path: "scenarios/demo/api/main.go", Text: "imports context",
			FactKind: "imports", SourceHash: digest, Generation: "g17", StartLine: 8, EndLine: 8,
			EvidenceStatus: factsv1.EvidenceStatus_EVIDENCE_STATUS_PROVEN,
		}},
	})
	joined := strings.Join(append(report.Summary, report.Results...), "\n")
	for _, want := range []string{"api/main.go", digest, "generation:g17", "lines:8-8", "EVIDENCE_STATUS_PROVEN"} {
		if !strings.Contains(joined, want) {
			t.Fatalf("import report missing %q: %s", want, joined)
		}
	}
}

func TestImportSearchCLIShowsIncompleteIndexStatus(t *testing.T) {
	report := (&handlers{}).searchReport(nil, &factsv1.SearchResponse{
		DegradedStages: []string{"import_index_incomplete"},
		Results:        []*factsv1.SearchHit{{Title: "Go import evidence incomplete", Path: "scenarios/demo/api/main.go", FactKind: "imports_partial", EvidenceStatus: factsv1.EvidenceStatus_EVIDENCE_STATUS_UNKNOWN}},
	})
	joined := strings.Join(append(report.Summary, report.Results...), "\n")
	if !strings.Contains(joined, "import_index_incomplete") || !strings.Contains(joined, "EVIDENCE_STATUS_UNKNOWN") {
		t.Fatalf("partial import index was not surfaced: %s", joined)
	}
}
