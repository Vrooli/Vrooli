package lifecycle

import (
	"strings"
	"testing"

	lifecyclev1 "github.com/vrooli/vrooli/packages/proto/gen/go/template-manager/v1/lifecycle"
)

func TestValidateReportSurfacesStatusAndWarnings(t *testing.T) {
	h := &handlers{}
	report := h.validateReport(nil, &lifecyclev1.ValidateTemplateResponse{
		Mode:        "shallow",
		Template:    "react-vite",
		Count:       1,
		Status:      "pass",
		IssuesCount: 0,
		Warnings:    []string{"--retain-temp only applies to deep validation; it has no effect in shallow mode"},
	})
	if len(report.Summary) != 1 || !strings.Contains(report.Summary[0], "status=pass") {
		t.Fatalf("summary = %#v, want status=pass", report.Summary)
	}
	if !strings.Contains(strings.Join(report.Results, "\n"), "warning:") {
		t.Fatalf("results = %#v, want the retain-temp warning surfaced", report.Results)
	}
}

func TestCleanupReportExplainsSkippedRuns(t *testing.T) {
	h := &handlers{}
	report := h.cleanupReport(nil, &lifecyclev1.CleanupRunsResponse{
		Matched: 0,
		Removed: 0,
		DryRun:  true,
		Message: "0 eligible, 0 removed (dry-run), 1 skipped, 0 failed",
		Skipped: 1,
		SkippedRuns: []*lifecyclev1.CleanupSkippedRun{{
			RunId:  "run-123",
			Reason: "retained run; use --include-retained or --run run-123",
		}},
	})
	joined := strings.Join(report.Changes, "\n")
	if !strings.Contains(joined, "skipped run-123") || !strings.Contains(joined, "--include-retained") {
		t.Fatalf("changes = %#v, want the skip reason and --include-retained hint", report.Changes)
	}
	if len(report.Result) != 1 || !strings.Contains(report.Result[0], "skipped 1") {
		t.Fatalf("result = %#v, want the skipped count", report.Result)
	}
}

func TestDesignValidateReportGroupsPerKit(t *testing.T) {
	h := &handlers{}
	report := h.designValidateReport(nil, &lifecyclev1.ValidateDesignKitsResponse{
		Count:       2,
		Status:      "fail",
		IssuesCount: 1,
		Issues: []*lifecyclev1.DesignValidationIssue{
			{Kit: "broken", Path: "metadata.json", Message: "metadata id must match folder name"},
		},
		Results: []*lifecyclev1.DesignKitValidationResult{
			{Kit: "clean", Status: "pass"},
			{Kit: "broken", Status: "fail", Issues: []*lifecyclev1.DesignValidationIssue{
				{Kit: "broken", Path: "metadata.json", Message: "metadata id must match folder name"},
			}},
		},
	})
	joined := strings.Join(report.Results, "\n")
	if !strings.Contains(joined, "clean [pass]") || !strings.Contains(joined, "broken [fail]") {
		t.Fatalf("results = %#v, want per-kit status lines", report.Results)
	}
	if len(report.Summary) != 1 || !strings.Contains(report.Summary[0], "status=fail") {
		t.Fatalf("summary = %#v, want status=fail", report.Summary)
	}
}
