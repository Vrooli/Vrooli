package validationprovider

import (
	"testing"

	assessmentpkg "github.com/vrooli/maturity-go/assessment"
	commonv1 "github.com/vrooli/vrooli/packages/proto/gen/go/common/v1"
	scenariovalidationv1 "github.com/vrooli/vrooli/packages/proto/gen/go/scenario-validation/v1"
)

func TestTranslateThreadsMetrics(t *testing.T) {
	provider := testProvider(false)
	resp := &scenariovalidationv1.ValidateScenarioResponse{
		Scenario:   "demo",
		Status:     scenariovalidationv1.ValidationStatus_VALIDATION_STATUS_PASSED,
		Assessment: testAssessment(""),
		Metrics: &commonv1.ExecutionMetrics{
			WallClockMs: 1840,
			Stages:      []*commonv1.Stage{{Name: "analyze", DurationMs: 1510}},
			Environment: &commonv1.CaptureEnvironment{Os: "linux", Arch: "amd64", NumCpu: 4},
		},
	}

	out := translate(provider, "demo", resp)
	if out.Metrics == nil {
		t.Fatal("translate dropped metrics")
	}
	if out.Metrics.GetWallClockMs() != 1840 {
		t.Fatalf("wall_clock_ms = %d, want 1840", out.Metrics.GetWallClockMs())
	}
	if len(out.Metrics.GetStages()) != 1 || out.Metrics.GetStages()[0].GetName() != "analyze" {
		t.Fatalf("stages not threaded: %+v", out.Metrics.GetStages())
	}
}

func TestTranslateToleratesNilMetrics(t *testing.T) {
	provider := testProvider(false)
	resp := &scenariovalidationv1.ValidateScenarioResponse{
		Scenario:   "demo",
		Status:     scenariovalidationv1.ValidationStatus_VALIDATION_STATUS_PASSED,
		Assessment: testAssessment(""),
		// No metrics: the contract for an un-migrated provider.
	}
	out := translate(provider, "demo", resp)
	if out.Metrics != nil {
		t.Fatalf("expected nil metrics for un-migrated provider, got %+v", out.Metrics)
	}
	if !out.Success {
		t.Fatal("nil metrics should not affect success")
	}
}

func TestTranslateFailedStatusMessageNamesTheFailureSource(t *testing.T) {
	provider := testProvider(false)
	cases := []struct {
		name       string
		assessment *commonv1.MaturityAssessment
		want       string
	}{
		{"typed errors", testAssessment("SEVERITY_ERROR"), "1 proto finding(s) at ERROR or BLOCKER severity"},
		{"untyped blocking codes", func() *commonv1.MaturityAssessment {
			a := testAssessment("")
			a.Local.BlockingFindingCodes = []string{"skill-set.role_missing:usage"}
			a.Presentation = assessmentpkg.BuildPhasePresentation(a)
			return a
		}(), "proto failed with untyped blocking codes (provider proto-health reported no ERROR or BLOCKER findings): skill-set.role_missing:usage"},
		{"no reason", testAssessment(""), "proto failed without ERROR or BLOCKER findings (provider proto-health gave no reason)"},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			out := translate(provider, "demo", &scenariovalidationv1.ValidateScenarioResponse{
				Scenario:   "demo",
				Status:     scenariovalidationv1.ValidationStatus_VALIDATION_STATUS_FAILED,
				Assessment: tc.assessment,
			})
			if out.Success || out.Error == nil || out.Error.Error() != tc.want {
				t.Fatalf("error = %v, want %q", out.Error, tc.want)
			}
		})
	}
}
