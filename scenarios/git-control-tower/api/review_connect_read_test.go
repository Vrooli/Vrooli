package main

import (
	"context"
	"testing"

	"connectrpc.com/connect"
)

func TestReviewConnectReadMethodsValidateRequests(t *testing.T) {
	h := reviewConnectServer{}
	if _, err := h.GetSummary(context.Background(), nil); connect.CodeOf(err) != connect.CodeInvalidArgument {
		t.Fatalf("GetSummary(nil) code = %v, want InvalidArgument", connect.CodeOf(err))
	}
	if _, err := h.GetJobStatus(context.Background(), nil); connect.CodeOf(err) != connect.CodeInvalidArgument {
		t.Fatalf("GetJobStatus(nil) code = %v, want InvalidArgument", connect.CodeOf(err))
	}
}

func TestReviewSummaryToProtoPreservesNestedOwnerEvidence(t *testing.T) {
	source := &ReviewSummaryResponse{
		ScenarioName: "example", Readiness: ReadinessYellow, Timestamp: "now",
		Capabilities:      map[string]bool{"test-genie": true},
		DimensionStatuses: map[string]string{"tests": "available"},
		CheckStatuses:     map[string]CheckStatus{"tests": CheckFailed},
		Dimensions: ReviewDimensions{
			CodeQuality: &CodeQualityDimension{Available: true, Score: .75, Violations: 2, Stale: true, LastScan: "scan", TopIssues: []CodeQualityIssue{{Category: "complexity", Count: 2}}},
			Tests:       &TestsDimension{Available: true, Passed: false, Total: 3, PassedCount: 2, FailedCount: 1, LastRun: "run", Failures: []TestFailure{{Phase: "unit", Error: "failed", Classification: "assertion", Remediation: "fix"}}},
			Standards:   &StandardsDimension{Available: true, BlockingViolations: 1, Warnings: 2, TotalViolations: 3, TopViolations: []StandardsViolationDetail{{FilePath: "a.go", LineNumber: 9, Title: "title", Severity: "error", Recommendation: "repair"}}},
			Visual:      &VisualDimension{Available: true, ScreenshotCount: 4, Stale: false, LatestCapture: &VisualCaptureMeta{CapturedAt: "capture", CommitHash: "abc", ScreenshotCount: 4}},
			Provenance:  &ProvenanceDimension{Available: true, TracedFiles: 5, UntracedFiles: []string{"b.go"}},
		},
	}
	got := reviewSummaryToProto(source)
	if got.GetReadiness() != "yellow" || got.GetCheckStatuses()["tests"] != "failed" || !got.GetCapabilities()["test-genie"] {
		t.Fatalf("summary identity/status data lost: %+v", got)
	}
	if got.GetDimensions().GetTests().GetFailures()[0].GetRemediation() != "fix" || got.GetDimensions().GetVisual().GetLatestCapture().GetCommitHash() != "abc" || got.GetDimensions().GetProvenance().GetUntracedFiles()[0] != "b.go" {
		t.Fatalf("nested dimension details lost: %+v", got.GetDimensions())
	}
}
