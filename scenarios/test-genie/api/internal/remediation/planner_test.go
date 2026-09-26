package remediation

import (
	"reflect"
	"testing"
	"time"
)

func TestBuildPlanDeduplicatesRanksAndBundlesStableFindings(t *testing.T) {
	plan := BuildPlan(Evidence{
		SourceExecutionID: "execution-1", SourceRunID: "run-1", Scenario: "demo", CompletedAt: time.Now(),
		Phases: []Phase{{Name: "structure", Provider: "architecture-cartographer", ResultGating: "blocking"}, {Name: "docs", Provider: "knowledge-observatory", ResultGating: "advisory"}},
		Findings: []Finding{
			{StableID: "afid:advisory", Severity: "warning", Class: "heuristic", Phase: "docs", Locations: []string{"docs/README.md"}},
			{StableID: "afid:blocker", Severity: "blocker", Class: "deterministic", Phase: "structure", Locations: []string{"api/internal/foo.go"}},
			{StableID: "afid:blocker", Severity: "blocker", Class: "deterministic", Phase: "structure", Locations: []string{"api/internal/foo.go"}},
		},
	})
	if plan.Degraded {
		t.Fatalf("plan unexpectedly degraded: %+v", plan.DegradedReasons)
	}
	if len(plan.Findings) != 2 {
		t.Fatalf("findings = %d, want 2", len(plan.Findings))
	}
	if plan.Findings[0].StableID != "afid:blocker" || !plan.Findings[0].Gating {
		t.Fatalf("ranking = %+v", plan.Findings)
	}
	if len(plan.Findings[0].Occurrences) != 2 {
		t.Fatalf("occurrences = %+v", plan.Findings[0].Occurrences)
	}
	if len(plan.Bundles) != 2 || !plan.Bundles[0].Gating {
		t.Fatalf("bundles = %+v", plan.Bundles)
	}
}

func TestBuildPlanDegradesRatherThanFabricatingWithoutStableEvidence(t *testing.T) {
	plan := BuildPlan(Evidence{SourceExecutionID: "e", SourceRunID: "r", Scenario: "demo", CompletedAt: time.Now(), Phases: []Phase{{Name: "unit"}}, Findings: []Finding{{Phase: "unit", Code: "missing"}}})
	if !plan.Degraded || len(plan.Findings) != 0 || len(plan.Bundles) != 0 {
		t.Fatalf("want explicit degradation, got %+v", plan)
	}
}

func TestValidateSelectionAndCompareUseStableIDs(t *testing.T) {
	plan := BuildPlan(Evidence{SourceExecutionID: "e", SourceRunID: "r", Scenario: "demo", CompletedAt: time.Now(), Phases: []Phase{{Name: "unit"}}, Findings: []Finding{{StableID: "one", Phase: "unit"}, {StableID: "two", Phase: "unit"}}})
	selected, err := ValidateSelection(plan, []string{"two", "one", "two"})
	if err != nil {
		t.Fatal(err)
	}
	if !reflect.DeepEqual(selected, []string{"one", "two"}) {
		t.Fatalf("selected = %v", selected)
	}
	if _, err := ValidateSelection(plan, []string{"unknown"}); err == nil {
		t.Fatal("unknown selector accepted")
	}
	delta := Compare([]string{"one", "two"}, []Finding{{StableID: "two"}, {StableID: "three"}}, true)
	if !reflect.DeepEqual(delta.Resolved, []string{"one"}) || !reflect.DeepEqual(delta.Remaining, []string{"two"}) || !reflect.DeepEqual(delta.New, []string{"three"}) {
		t.Fatalf("delta = %+v", delta)
	}
}

func TestComparePlanPreservesSeverityChangesAndSkippedPhases(t *testing.T) {
	source := BuildPlan(Evidence{SourceExecutionID: "e", SourceRunID: "r", Scenario: "demo", CompletedAt: time.Now(), Phases: []Phase{{Name: "unit"}, {Name: "docs"}}, Findings: []Finding{
		{StableID: "same", Severity: "warning", Phase: "unit"},
		{StableID: "skipped", Severity: "error", Phase: "docs"},
	}})
	delta := ComparePlan(source, []string{"same", "skipped"}, []Finding{{StableID: "same", Severity: "error", Phase: "unit"}, {StableID: "new", Severity: "warning", Phase: "unit"}}, true, []string{"unit"})
	if !reflect.DeepEqual(delta.Remaining, []string{"same"}) || !reflect.DeepEqual(delta.ChangedSeverity, []string{"same"}) || !reflect.DeepEqual(delta.Skipped, []string{"skipped"}) || !reflect.DeepEqual(delta.New, []string{"new"}) {
		t.Fatalf("delta = %+v", delta)
	}
}

func TestValidateRequirementSelectionUsesImmutableRequirementEvidence(t *testing.T) {
	plan := BuildPlan(Evidence{SourceExecutionID: "e", SourceRunID: "r", Scenario: "demo", CompletedAt: time.Now()})
	plan.Requirements = []RequirementEvidence{{ID: "REQ-1", Title: "Evidence retained", LiveStatus: "failed", Validations: []string{"go:go test:failed"}}}
	selected, err := ValidateRequirementSelection(plan, []string{"REQ-1"})
	if err != nil || !reflect.DeepEqual(selected, []string{"REQ-1"}) {
		t.Fatalf("selected=%v err=%v", selected, err)
	}
	if _, err := ValidateRequirementSelection(plan, []string{"REQ-unknown"}); err == nil {
		t.Fatal("unknown requirement selector accepted")
	}
}
