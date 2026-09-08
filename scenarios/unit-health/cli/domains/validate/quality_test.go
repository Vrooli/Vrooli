package validate

import (
	validationv1 "github.com/vrooli/vrooli/packages/proto/gen/go/unit-health/v1/validation"
	"google.golang.org/protobuf/encoding/protojson"
	"strings"
	"testing"
)

func TestEvidenceStagesKeepStaticAndHistoricalUnknownSeparate(t *testing.T) {
	text := strings.Join(evidenceStageLines(&validationv1.EvidenceStages{Configured: "observed", Analyzed: "partial", Executed: "not_requested", Reviewed: "future", SourceRunId: "run"}), "\n")
	for _, want := range []string{"configured=observed", "analyzed=partial", "executed=not_requested", "reviewed=unknown", "source_run=run"} {
		if !strings.Contains(text, want) {
			t.Fatalf("missing %q: %s", want, text)
		}
	}
	if !strings.Contains(strings.Join(evidenceStageLines(nil), ""), "not execution or review evidence") {
		t.Fatal("missing historical disclaimer")
	}
}

func TestFutureQualitySchemaRetainsLocationsWithoutCleanClaims(t *testing.T) {
	report := &validationv1.TestQualityReport{SchemaVersion: "future", CatalogVersion: "1", Results: []*validationv1.QualityCheckResult{{Status: validationv1.QualityCheckStatus_QUALITY_CHECK_STATUS_CHECKED_CLEAN, Target: &validationv1.QualityTestTarget{File: "retained.test.ts", TestId: "test-id"}}}}
	text := strings.Join(qualityLines(report), "\n")
	for _, want := range []string{"unknown scoped result(s)", "unsupported assessment schema", "retained.test.ts", "test-id", "[unknown]"} {
		if !strings.Contains(text, want) {
			t.Fatalf("missing %q: %s", want, text)
		}
	}
	if strings.Contains(text, "[checked_clean]") {
		t.Fatal("future schema became clean evidence")
	}
}

func TestQualityGuidanceRemainsProviderOwned(t *testing.T) {
	report := &validationv1.TestQualityReport{SchemaVersion: "test-quality/v1", CatalogVersion: "1", ReasonGuidance: "Check owner availability.", Results: []*validationv1.QualityCheckResult{{ReasonGuidance: "Inspect native skip rationale."}}}
	text := strings.Join(qualityLines(report), "\n")
	if !strings.Contains(text, "Next: Check owner availability.") || !strings.Contains(text, "Next: Inspect native skip rationale.") {
		t.Fatalf("guidance lost: %s", text)
	}
}

func TestPartialQualityDisplaysFourStatesAndReportedDenominator(t *testing.T) {
	r := &validationv1.TestQualityReport{SchemaVersion: "test-quality/v1", CatalogVersion: "1", TotalResults: 4,
		Coverage:              []*validationv1.QualityAssessmentCoverage{{RuleId: "rule", SupportProfile: "profile", Discovered: 4, Assessed: 2, Unknown: 1, NotApplicable: 1}},
		CollectionLimitations: []*validationv1.QualityCollectionLimitation{{Reason: validationv1.QualityReason_QUALITY_REASON_OWNER_UNAVAILABLE, Guidance: "Check missing collector."}}}
	for _, s := range []validationv1.QualityCheckStatus{validationv1.QualityCheckStatus_QUALITY_CHECK_STATUS_CHECKED_CLEAN, validationv1.QualityCheckStatus_QUALITY_CHECK_STATUS_VIOLATION, validationv1.QualityCheckStatus_QUALITY_CHECK_STATUS_UNKNOWN, validationv1.QualityCheckStatus_QUALITY_CHECK_STATUS_NOT_APPLICABLE} {
		r.Results = append(r.Results, &validationv1.QualityCheckResult{Status: s})
	}
	text := strings.Join(qualityLines(r), "\n")
	for _, want := range []string{"[checked_clean]", "[violation]", "[unknown]", "[not_applicable]", "assessed=2/discovered=4 unknown=1 not_applicable=1", "reported rows only", "Check missing collector."} {
		if !strings.Contains(text, want) {
			t.Fatalf("missing %q: %s", want, text)
		}
	}
}

func TestPartialAssessmentKeepsLocationsAndDenominatorUncertainty(t *testing.T) {
	report := &validationv1.TestQualityReport{SchemaVersion: "test-quality/v1", CatalogVersion: "1", TotalResults: 1,
		UnavailableReason: validationv1.QualityReason_QUALITY_REASON_OWNER_UNAVAILABLE,
		Results:           []*validationv1.QualityCheckResult{{Status: validationv1.QualityCheckStatus_QUALITY_CHECK_STATUS_CHECKED_CLEAN, Target: &validationv1.QualityTestTarget{File: "a.test.ts", TestId: "case-1"}}}}
	text := strings.Join(qualityLines(report), "\n")
	for _, want := range []string{"a.test.ts", "case-1", "[unknown]", "no trustworthy denominator", "severity=", "enforcement="} {
		if !strings.Contains(text, want) {
			t.Fatalf("missing %q: %s", want, text)
		}
	}
	if strings.Contains(text, "[checked_clean]") {
		t.Fatal("unavailable evidence shown as clean")
	}
	if !strings.Contains(strings.Join(executionLines(nil), ""), "not evidence of passing tests") {
		t.Fatal("absent execution is ambiguous")
	}
}

func TestNativeFinalMetadataIsNotLabeledAttemptHistory(t *testing.T) {
	seed, retry := "0", int32(1)
	report := &validationv1.TestQualityReport{SchemaVersion: "test-quality/v1", CatalogVersion: "1", Results: []*validationv1.QualityCheckResult{{RuntimeObservation: &validationv1.QualityRuntimeObservation{RunId: "native", State: "pass", Seed: &seed, RetryCount: &retry}}}}
	text := strings.Join(qualityLines(report), "\n")
	for _, want := range []string{"native final state=pass", "not a complete attempt history", "seed=\"0\"", "retry_count=1"} {
		if !strings.Contains(text, want) {
			t.Fatalf("missing %q: %s", want, text)
		}
	}
	if strings.Contains(text, "retry_ordinal=") {
		t.Fatal("fabricated attempt ordinal")
	}
}

func TestReliabilityDiagnosticDisplaysScopeExclusionsAndOptionalRetry(t *testing.T) {
	seed, retry := "42", int32(0)
	d := &validationv1.Diagnostic{Kind: "reliability", Evidence: "not proof of a flaky test", Reliability: &validationv1.ReliabilityObservation{State: "future", Scope: "command", SampleCount: 1, Passed: 1, ExcludedInfrastructure: 2, ExcludedIncompatible: 3, Seed: &seed, RetryOrdinal: &retry}}
	text := strings.Join(diagnosticLines([]*validationv1.Diagnostic{d}), "\n")
	for _, want := range []string{"not proof", "reliability=unknown", "scope=command", "samples=1", "excluded_infrastructure_or_unclassified=2", "excluded_incompatible=3", "seed=\"42\"", "retry_ordinal=0"} {
		if !strings.Contains(text, want) {
			t.Fatalf("missing %q: %s", want, text)
		}
	}
}

func TestTraceabilityLabelsSeparateLinksExecutionAndOwnership(t *testing.T) {
	r := &validationv1.RequirementTraceabilityReport{SchemaVersion: "requirement-traceability/v1", Requirements: []*validationv1.RequirementTraceScope{{RequirementId: "UH-CORE-010", Applicability: validationv1.RequirementApplicability_REQUIREMENT_APPLICABILITY_NOT_APPLICABLE}}, Links: []*validationv1.RequirementTraceLink{
		{RequirementId: "UH-CORE-001", Target: &validationv1.QualityTestTarget{File: "a.test.ts", TestId: "case-2"}, Registration: validationv1.RequirementRegistration_REQUIREMENT_REGISTRATION_REGISTERED, Execution: validationv1.RequirementExecutionState_REQUIREMENT_EXECUTION_STATE_SKIPPED, RunId: "current"},
		{RequirementId: "UH-CORE-999", Registration: validationv1.RequirementRegistration(999), Execution: validationv1.RequirementExecutionState(999)},
	}}
	text := strings.Join(traceabilityLines(r), "\n")
	for _, want := range []string{"not behavioral proof", "UH-CORE-010 unit responsibility: not_applicable", "UH-CORE-001 [registered] a.test.ts:case-2 execution=skipped", "UH-CORE-999 [unknown]", "execution=unknown"} {
		if !strings.Contains(text, want) {
			t.Fatalf("missing %q in %s", want, text)
		}
	}
	r.UnavailableReason = validationv1.QualityReason_QUALITY_REASON_OWNER_UNAVAILABLE
	text = strings.Join(traceabilityLines(r), "\n")
	if strings.Contains(text, "[registered]") || strings.Contains(text, "unit responsibility:") || !strings.Contains(text, "Registry: unknown") {
		t.Fatalf("unavailable registry appeared authoritative: %s", text)
	}
	for _, historical := range []*validationv1.RequirementTraceabilityReport{nil, {}, {SchemaVersion: "future"}} {
		if text := strings.Join(traceabilityLines(historical), "\n"); !strings.Contains(text, "unknown") {
			t.Fatal(text)
		}
	}
}

func TestQualityFileScopeAndNativeDetailAreVisible(t *testing.T) {
	report := &validationv1.TestQualityReport{SchemaVersion: "test-quality/v1", CatalogVersion: "1", TotalResults: 1, Results: []*validationv1.QualityCheckResult{{Target: &validationv1.QualityTestTarget{Scope: "file", File: "a.test.ts"}, Diagnostics: []*validationv1.QualityNativeDiagnostic{{NativeRuleId: "vitest/valid-expect", MessageId: "matcherNotFound", Message: "native detail", Location: &validationv1.QualitySourceLocation{Line: 3, Column: 2}}}}}}
	text := strings.Join(qualityLines(report), "\n")
	for _, expected := range []string{"[file scope]", "a.test.ts:3:2", "matcherNotFound", "native detail"} {
		if !strings.Contains(text, expected) {
			t.Fatalf("missing %s: %s", expected, text)
		}
	}
}

func TestQualityUnknownWireEnumsAndHistoricalAbsence(t *testing.T) {
	for _, input := range []string{`{}`, `{"status":0}`, `{"status":999}`} {
		var row validationv1.QualityCheckResult
		if err := protojson.Unmarshal([]byte(input), &row); err != nil {
			t.Fatal(err)
		}
		if qualityStatusLabel(row.Status) != "unknown" {
			t.Fatalf("future or missing status became clean: %s", input)
		}
	}
	for _, report := range []*validationv1.TestQualityReport{nil, {}, {SchemaVersion: "future"}, {SchemaVersion: "test-quality/v1", UnavailableReason: validationv1.QualityReason(999)}} {
		if got := strings.Join(qualityLines(report), "\n"); !strings.Contains(got, "Test quality: unknown") {
			t.Fatalf("unavailable report not unknown: %s", got)
		}
	}
}

func TestQualityCleanIsScopedAndTruncationVisible(t *testing.T) {
	r := &validationv1.TestQualityReport{SchemaVersion: "test-quality/v1", CatalogVersion: "1", TotalResults: 2, Truncated: true, DetailsRef: "run:one/details",
		Results: []*validationv1.QualityCheckResult{{RuleId: "assertion-observation", RuleVersion: "1", Status: validationv1.QualityCheckStatus_QUALITY_CHECK_STATUS_CHECKED_CLEAN,
			Target: &validationv1.QualityTestTarget{File: "x_test.go", TestId: "TestX"}, EvidenceKind: validationv1.QualityEvidenceKind_QUALITY_EVIDENCE_KIND_STATIC,
			Enforcement: validationv1.QualityEnforcement_QUALITY_ENFORCEMENT_ADVISORY}}}
	got := strings.Join(qualityLines(r), "\n")
	for _, expected := range []string{"[checked_clean]", "not behavioral certification", "Showing 1 of 2", "run:one/details", "QUALITY_EVIDENCE_KIND_STATIC", "QUALITY_ENFORCEMENT_ADVISORY"} {
		if !strings.Contains(got, expected) {
			t.Fatalf("missing %q: %s", expected, got)
		}
	}
}
