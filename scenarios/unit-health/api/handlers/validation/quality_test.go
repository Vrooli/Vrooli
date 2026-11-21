package validation

import (
	"encoding/json"
	validationv1 "github.com/vrooli/vrooli/packages/proto/gen/go/unit-health/v1/validation"
	"google.golang.org/protobuf/encoding/protojson"
	"google.golang.org/protobuf/proto"
	"testing"
	"unit-health/internal/testquality"
	internalvalidation "unit-health/internal/validation"
)

func TestUnavailableQualityReportCarriesOwnerGuidance(t *testing.T) {
	got := qualityReportToProto(&testquality.Report{SchemaVersion: testquality.SchemaVersion, CatalogVersion: "1", UnavailableReason: testquality.OwnerUnavailable})
	if got.GetReasonGuidance() != testquality.ReasonGuidance(testquality.OwnerUnavailable) {
		t.Fatalf("guidance lost: %v", got)
	}
	raw, err := protojson.Marshal(got)
	if err != nil {
		t.Fatal(err)
	}
	var decoded validationv1.TestQualityReport
	if err := protojson.Unmarshal(raw, &decoded); err != nil {
		t.Fatal(err)
	}
	if decoded.GetReasonGuidance() != got.GetReasonGuidance() {
		t.Fatal("wire roundtrip lost guidance")
	}
}

func TestPartialQualityPublicProjectionRetainsAllStatesAndLimitations(t *testing.T) {
	var rows []testquality.Result
	for i, status := range []testquality.Status{testquality.CheckedClean, testquality.Violation, testquality.Unknown, testquality.NotApplicable} {
		rows = append(rows, testquality.Result{RuleID: "rule", RuleVersion: "1", SupportProfile: "profile", Target: testquality.Target{Workspace: "api", File: "test.go", TestID: string(rune('a' + i))}, Status: status, Reason: testquality.ReasonNone, Severity: testquality.Error, Enforcement: testquality.Advisory, EvidenceKind: testquality.Static})
	}
	report, err := testquality.BuildReport("1", rows, 10, "")
	if err != nil {
		t.Fatal(err)
	}
	report.MarkIncomplete(testquality.OwnerUnavailable)
	report.MarkIncomplete(testquality.UnsupportedVersion)
	wire := qualityReportToProto(&report)
	if len(wire.GetCollectionLimitations()) != 2 || wire.GetCollectionLimitations()[0].GetGuidance() == "" {
		t.Fatalf("missing limitations: %v", wire)
	}
	if len(wire.GetCoverage()) != 1 {
		t.Fatalf("coverage dropped: %v", wire)
	}
	c := wire.GetCoverage()[0]
	if c.GetAssessed() != 2 || c.GetUnknown() != 1 || c.GetNotApplicable() != 1 || c.GetDiscovered() != 4 {
		t.Fatalf("wrong partial denominator: %v", c)
	}
	for i, row := range wire.GetResults() {
		if row.GetStatus() != qualityStatusToProto(rows[i].Status) {
			t.Fatalf("state %d changed: %v", i, row)
		}
	}
	raw, err := protojson.Marshal(wire)
	if err != nil {
		t.Fatal(err)
	}
	var got validationv1.TestQualityReport
	if err := protojson.Unmarshal(raw, &got); err != nil {
		t.Fatal(err)
	}
	if !proto.Equal(wire, &got) {
		t.Fatal("wire roundtrip changed partial report")
	}
}

func TestEvidenceStagesPublicConversionDoesNotInventReview(t *testing.T) {
	if evidenceStagesToProto(nil) != nil {
		t.Fatal("missing stages became known")
	}
	stages := evidenceStagesToProto(&internalvalidation.EvidenceStages{Configured: "observed", Analyzed: "partial", Executed: "cached", Reviewed: "future", SourceRunID: "original"})
	raw, err := protojson.Marshal(stages)
	if err != nil {
		t.Fatal(err)
	}
	var out validationv1.EvidenceStages
	if err := protojson.Unmarshal(raw, &out); err != nil {
		t.Fatal(err)
	}
	if out.GetReviewed() != "unknown" || out.GetExecuted() != "cached" || out.GetSourceRunId() != "original" {
		t.Fatalf("incorrect public evidence: %s", raw)
	}
}

func TestNativeFinalObservationPublicRoundTripDoesNotInventAttempts(t *testing.T) {
	seed, retry := "0", 1
	report := &testquality.Report{SchemaVersion: testquality.SchemaVersion, CatalogVersion: "1", Results: []testquality.Result{{EvidenceKind: testquality.Runtime, RuntimeObservation: &testquality.RuntimeObservation{RunID: "native", State: "pass", Seed: &seed, RetryCount: &retry}}}}
	raw, err := protojson.Marshal(qualityReportToProto(report))
	if err != nil {
		t.Fatal(err)
	}
	var out validationv1.TestQualityReport
	if err := protojson.Unmarshal(raw, &out); err != nil {
		t.Fatal(err)
	}
	if len(out.Results) != 1 {
		t.Fatalf("missing row: %s", raw)
	}
	native := out.Results[0].RuntimeObservation
	if native == nil || native.GetRunId() != "native" || native.Seed == nil || native.GetSeed() != "0" || native.RetryCount == nil || native.GetRetryCount() != 1 || native.RetryOrdinal != nil {
		t.Fatalf("lost final native evidence: %s", raw)
	}
}

func TestReliabilityPublicRoundTripPreservesUncertaintyAndZeroRetry(t *testing.T) {
	seed, retry := "", 0
	in := internalvalidation.Diagnostic{Kind: "reliability", Reliability: &internalvalidation.ReliabilityObservation{State: "insufficient_samples", Scope: "command", CohortDigest: "cohort", SampleCount: 1, Passed: 1, ExcludedInfrastructure: 2, ExcludedIncompatible: 3, Seed: &seed, RetryOrdinal: &retry}}
	raw, err := protojson.Marshal(diagnosticToProto(in))
	if err != nil {
		t.Fatal(err)
	}
	var out validationv1.Diagnostic
	if err := protojson.Unmarshal(raw, &out); err != nil {
		t.Fatal(err)
	}
	r := out.GetReliability()
	if r == nil || r.GetState() != "insufficient_samples" || r.GetScope() != "command" || r.GetSampleCount() != 1 || r.GetExcludedInfrastructure() != 2 || r.GetExcludedIncompatible() != 3 || r.Seed == nil || r.RetryOrdinal == nil || r.GetRetryOrdinal() != 0 {
		t.Fatalf("lost uncertainty: %s", raw)
	}
	if diagnosticToProto(internalvalidation.Diagnostic{}).GetReliability() != nil {
		t.Fatal("absent evidence became observed")
	}
}

func TestTraceabilityPublicRoundTripPreservesSkippedAndUnknown(t *testing.T) {
	in := &testquality.TraceabilityReport{SchemaVersion: "requirement-traceability/v1", Links: []testquality.RequirementLink{
		{ID: "UH-CORE-001", Target: testquality.Target{Workspace: "ui", File: "a.test.ts", TestID: "parameter-2"}, Registration: "registered", Execution: testquality.ExecutionSkipped, Reason: testquality.ReasonNone, RunID: "current"},
		{ID: "UH-CORE-010", Registration: "future", Execution: "future", Reason: testquality.MissingInput},
	}}
	data, err := protojson.Marshal(traceabilityToProto(in))
	if err != nil {
		t.Fatal(err)
	}
	var out validationv1.RequirementTraceabilityReport
	if err := protojson.Unmarshal(data, &out); err != nil {
		t.Fatal(err)
	}
	if len(out.Links) != 2 || out.Links[0].Execution != validationv1.RequirementExecutionState_REQUIREMENT_EXECUTION_STATE_SKIPPED || out.Links[0].Target.TestId != "parameter-2" || out.Links[1].Registration != validationv1.RequirementRegistration_REQUIREMENT_REGISTRATION_UNKNOWN || out.Links[1].Execution != validationv1.RequirementExecutionState_REQUIREMENT_EXECUTION_STATE_UNKNOWN {
		t.Fatalf("trace projection: %s", data)
	}
	in.SchemaVersion = "future"
	outFuture := traceabilityToProto(in)
	if outFuture.UnavailableReason != validationv1.QualityReason_QUALITY_REASON_UNSUPPORTED_VERSION || outFuture.Links[0].Execution != validationv1.RequirementExecutionState_REQUIREMENT_EXECUTION_STATE_UNKNOWN {
		t.Fatalf("future schema trusted: %+v", outFuture)
	}
}

func TestFileScopedDiagnosticsSurvivePublicProtoRoundTrip(t *testing.T) {
	report, err := testquality.BuildReport("1", []testquality.Result{{RuleID: "malformed-expectation", RuleVersion: "1", Target: testquality.Target{Workspace: "ui", File: "a.test.ts", Scope: "file"}, SupportProfile: "vitest-syntax-1.6.9", Status: testquality.Violation, Reason: testquality.ReasonNone, Diagnostics: []testquality.Diagnostic{{RuleID: "vitest/valid-expect", MessageID: "matcherNotFound", Message: "native detail", Location: testquality.Location{Line: 3, Column: 2}}}}}, 10, "")
	if err != nil {
		t.Fatal(err)
	}
	wire := qualityReportToProto(&report)
	data, err := protojson.Marshal(wire)
	if err != nil {
		t.Fatal(err)
	}
	var decoded validationv1.TestQualityReport
	if err := protojson.Unmarshal(data, &decoded); err != nil {
		t.Fatal(err)
	}
	if decoded.Results[0].Target.Scope != "file" || decoded.Results[0].Target.TestId != "" || decoded.Coverage[0].Scope != "file" || decoded.Results[0].Diagnostics[0].Location.Column != 2 {
		t.Fatalf("lost file scope or native detail: %s", data)
	}
}

func TestPersistedQualityResponsePreservesUnknownThroughPublicMapping(t *testing.T) {
	for _, tc := range []struct {
		name, persisted string
		reason          validationv1.QualityReason
	}{
		{"historical", `{}`, validationv1.QualityReason_QUALITY_REASON_UNSPECIFIED},
		{"future", `{"TestQuality":{"schemaVersion":"test-quality/v99","catalogVersion":"1","results":[{"ruleId":"assertion-observation","status":"checked_clean"}]}}`, validationv1.QualityReason_QUALITY_REASON_UNSUPPORTED_VERSION},
		{"missing", `{"TestQuality":{"results":[{"status":"checked_clean"}]}}`, validationv1.QualityReason_QUALITY_REASON_MISSING_ANALYSIS},
	} {
		t.Run(tc.name, func(t *testing.T) {
			var saved internalvalidation.Response = protoMappingFixture()
			if err := json.Unmarshal([]byte(tc.persisted), &saved); err != nil {
				t.Fatal(err)
			}
			wire, err := responseToProto(saved, testSpec(t))
			if err != nil {
				t.Fatal(err)
			}
			if tc.name == "historical" {
				if wire.TestQuality != nil {
					t.Fatal("historical response invented analysis")
				}
				return
			}
			if wire.TestQuality.UnavailableReason != tc.reason || wire.TestQuality.Results[0].Status != validationv1.QualityCheckStatus_QUALITY_CHECK_STATUS_UNKNOWN || len(wire.TestQuality.Coverage) != 0 {
				t.Fatalf("unsafe historical projection: %v", wire.TestQuality)
			}
		})
	}
}

func TestQualityReportAbsentPreservesHistoricalUncertainty(t *testing.T) {
	if qualityReportToProto(nil) != nil {
		t.Fatal("absent analysis became a report")
	}
	var historical validationv1.ValidateScenarioResponse
	if err := protojson.Unmarshal([]byte(`{}`), &historical); err != nil {
		t.Fatal(err)
	}
	if historical.TestQuality != nil {
		t.Fatal("historical response manufactures analysis")
	}
}

func TestQualityStatusMappingNeverDefaultsClean(t *testing.T) {
	for _, tc := range []struct {
		in   testquality.Status
		want validationv1.QualityCheckStatus
	}{
		{testquality.Violation, validationv1.QualityCheckStatus_QUALITY_CHECK_STATUS_VIOLATION},
		{testquality.CheckedClean, validationv1.QualityCheckStatus_QUALITY_CHECK_STATUS_CHECKED_CLEAN},
		{testquality.Unknown, validationv1.QualityCheckStatus_QUALITY_CHECK_STATUS_UNKNOWN},
		{testquality.NotApplicable, validationv1.QualityCheckStatus_QUALITY_CHECK_STATUS_NOT_APPLICABLE},
		{"", validationv1.QualityCheckStatus_QUALITY_CHECK_STATUS_UNKNOWN},
		{"future", validationv1.QualityCheckStatus_QUALITY_CHECK_STATUS_UNKNOWN},
	} {
		if got := qualityStatusToProto(tc.in); got != tc.want {
			t.Fatalf("status %q: %v, want %v", tc.in, got, tc.want)
		}
	}
}

func TestQualityReasonsAndReportRoundtrip(t *testing.T) {
	for _, reason := range testquality.Reasons() {
		mapped := qualityReasonToProto(reason)
		if mapped == validationv1.QualityReason_QUALITY_REASON_UNSPECIFIED {
			t.Fatalf("unmapped reason %s", reason)
		}
		in := &validationv1.QualityCheckResult{Reason: mapped, Status: validationv1.QualityCheckStatus_QUALITY_CHECK_STATUS_UNKNOWN}
		data, err := protojson.Marshal(in)
		if err != nil {
			t.Fatal(err)
		}
		var out validationv1.QualityCheckResult
		if err := protojson.Unmarshal(data, &out); err != nil {
			t.Fatal(err)
		}
		if !proto.Equal(in, &out) {
			t.Fatalf("reason lost: %s", reason)
		}
	}
	in := &testquality.Report{SchemaVersion: testquality.SchemaVersion, CatalogVersion: "1", TotalResults: 4, Truncated: true, DetailsRef: "run:one/quality",
		Coverage: []testquality.Coverage{{RuleID: "rule", SupportProfile: "profile", Discovered: 4, Assessed: 2, Unknown: 1, NotApplicable: 1}},
		Results: []testquality.Result{{RuleID: "rule", RuleVersion: "1", Status: testquality.CheckedClean, Reason: testquality.ReasonNone,
			Target: testquality.Target{Workspace: "api", File: "x_test.go", TestID: "TestX"}, SupportProfile: "profile", TestKind: "unit",
			Location: testquality.Location{Line: 12, Column: 3}, EvidenceKind: testquality.Static, Severity: testquality.Error, Enforcement: testquality.Advisory,
			EvidenceRefs: []string{"source:x_test.go"}, Limitations: []string{"static only"}}}}
	wire := qualityReportToProto(in)
	data, err := protojson.Marshal(wire)
	if err != nil {
		t.Fatal(err)
	}
	var out validationv1.TestQualityReport
	if err := protojson.Unmarshal(data, &out); err != nil {
		t.Fatal(err)
	}
	if !proto.Equal(wire, &out) {
		t.Fatal("report roundtrip lost fields")
	}
	if out.TotalResults != 4 || out.Coverage[0].Assessed != 2 || out.Results[0].Location.Line != 12 || out.Results[0].Enforcement != validationv1.QualityEnforcement_QUALITY_ENFORCEMENT_ADVISORY {
		t.Fatalf("incorrect projection: %v", &out)
	}
}
