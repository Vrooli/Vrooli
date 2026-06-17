package validationprovider

import (
	"context"
	"errors"
	"testing"
	"time"

	"connectrpc.com/connect"

	"test-genie/internal/shared"

	architecturev1 "github.com/vrooli/vrooli/packages/proto/gen/go/architecture/v1"
	commonv1 "github.com/vrooli/vrooli/packages/proto/gen/go/common/v1"
	scenariovalidationv1 "github.com/vrooli/vrooli/packages/proto/gen/go/scenario-validation/v1"
)

type fakeClient struct {
	resp *scenariovalidationv1.ValidateScenarioResponse
	err  error
}

func (f fakeClient) ValidateScenario(context.Context, *connect.Request[scenariovalidationv1.ValidateScenarioRequest]) (*connect.Response[scenariovalidationv1.ValidateScenarioResponse], error) {
	if f.err != nil {
		return nil, f.err
	}
	return connect.NewResponse(f.resp), nil
}

func testProvider(optional bool) Provider {
	return Provider{
		Phase:            "proto",
		ProviderScenario: "proto-health",
		FindingSource:    architecturev1.FindingSource_FINDING_SOURCE_PROTO,
		Emoji:            "P",
		Optional:         optional,
		Timeout:          time.Second,
	}
}

func testAssessment(severity string) *commonv1.MaturityAssessment {
	a := &commonv1.MaturityAssessment{
		Scenario: "demo",
		Provider: "proto-health",
		Phase:    "proto",
		Version:  "1",
		Local:    &commonv1.LocalMaturityAssessment{CurrentLevel: "L1", NextLevel: "L2"},
		FindingsBySeverity: map[string]int32{
			severity: 1,
		},
	}
	if severity != "" {
		a.Findings = []*commonv1.AssessmentFinding{{
			Code:        "proto.gen_out_of_sync",
			Severity:    severity,
			Title:       "Generated protos stale",
			Message:     "Regenerate proto artifacts",
			Location:    "packages/proto/gen",
			Remediation: "run make generate",
			Maturity: &commonv1.FindingMaturity{
				Dimension:    "proto-health",
				GlobalImpact: commonv1.GlobalImpact_GLOBAL_IMPACT_FOUNDATION_BLOCKER,
			},
		}}
	}
	return a
}

func TestRunFailedStatusEmitsFindingAndFails(t *testing.T) {
	prevResolve, prevClient := ResolveBaseURL, NewClient
	ResolveBaseURL = func(context.Context, string) (string, error) { return "http://provider", nil }
	NewClient = func(time.Duration, string) Client {
		return fakeClient{resp: &scenariovalidationv1.ValidateScenarioResponse{
			Scenario:   "demo",
			Status:     scenariovalidationv1.ValidationStatus_VALIDATION_STATUS_FAILED,
			Assessment: testAssessment("SEVERITY_ERROR"),
		}}
	}
	t.Cleanup(func() { ResolveBaseURL, NewClient = prevResolve, prevClient })

	got := Run(context.Background(), testProvider(false), "demo")
	if got.Success {
		t.Fatal("expected failed shared status to fail the phase")
	}
	if got.FailureClass != shared.FailureClassTestFailure {
		t.Fatalf("FailureClass = %q, want test_failure", got.FailureClass)
	}
	if got.Remediation != "Run `proto-health validate scenario demo` for details." {
		t.Fatalf("Remediation = %q", got.Remediation)
	}
	if len(got.Findings) != 1 || got.Findings[0].GetSource() != architecturev1.FindingSource_FINDING_SOURCE_PROTO {
		t.Fatalf("expected one PROTO finding, got %+v", got.Findings)
	}
	if got.Summary.LocalCurrentLevel != "L1" || got.Summary.LocalNextLevel != "L2" {
		t.Fatalf("summary local = %q/%q, want L1/L2", got.Summary.LocalCurrentLevel, got.Summary.LocalNextLevel)
	}
}

func TestRunFailedStatusUsesProviderDetailCommand(t *testing.T) {
	prevResolve, prevClient := ResolveBaseURL, NewClient
	ResolveBaseURL = func(context.Context, string) (string, error) { return "http://provider", nil }
	NewClient = func(time.Duration, string) Client {
		return fakeClient{resp: &scenariovalidationv1.ValidateScenarioResponse{
			Scenario:   "demo",
			Status:     scenariovalidationv1.ValidationStatus_VALIDATION_STATUS_FAILED,
			Assessment: testAssessment("SEVERITY_ERROR"),
		}}
	}
	t.Cleanup(func() { ResolveBaseURL, NewClient = prevResolve, prevClient })

	provider := testProvider(false)
	provider.DetailCommand = "scenario-dependency-analyzer health {{scenario}}"
	got := Run(context.Background(), provider, "demo")
	if got.Remediation != "Run `scenario-dependency-analyzer health demo` for details." {
		t.Fatalf("Remediation = %q", got.Remediation)
	}
}

func TestRunSummaryCountsFindingSeverityAliases(t *testing.T) {
	prevResolve, prevClient := ResolveBaseURL, NewClient
	ResolveBaseURL = func(context.Context, string) (string, error) { return "http://provider", nil }
	NewClient = func(time.Duration, string) Client {
		return fakeClient{resp: &scenariovalidationv1.ValidateScenarioResponse{
			Scenario:   "demo",
			Status:     scenariovalidationv1.ValidationStatus_VALIDATION_STATUS_PASSED,
			Assessment: testAssessment("FINDING_SEVERITY_WARNING"),
		}}
	}
	t.Cleanup(func() { ResolveBaseURL, NewClient = prevResolve, prevClient })

	got := Run(context.Background(), testProvider(false), "demo")
	if got.Summary.Warnings != 1 {
		t.Fatalf("Warnings = %d, want 1", got.Summary.Warnings)
	}
}

func TestRunOptionalProviderUnavailableSkips(t *testing.T) {
	prevResolve := ResolveBaseURL
	ResolveBaseURL = func(context.Context, string) (string, error) { return "", errors.New("not running") }
	t.Cleanup(func() { ResolveBaseURL = prevResolve })

	got := Run(context.Background(), testProvider(true), "demo")
	if !got.Success || !got.Summary.Skipped {
		t.Fatalf("optional unavailable provider should skip successfully, got success=%v summary=%+v", got.Success, got.Summary)
	}
	if len(got.Findings) != 0 {
		t.Fatalf("skip should emit no findings, got %+v", got.Findings)
	}
}

func TestRunMissingAssessmentIsMaturityContract(t *testing.T) {
	prevResolve, prevClient := ResolveBaseURL, NewClient
	ResolveBaseURL = func(context.Context, string) (string, error) { return "http://provider", nil }
	NewClient = func(time.Duration, string) Client {
		return fakeClient{resp: &scenariovalidationv1.ValidateScenarioResponse{
			Scenario: "demo",
			Status:   scenariovalidationv1.ValidationStatus_VALIDATION_STATUS_PASSED,
		}}
	}
	t.Cleanup(func() { ResolveBaseURL, NewClient = prevResolve, prevClient })

	got := Run(context.Background(), testProvider(false), "demo")
	if got.FailureClass != shared.FailureClassMaturityContract {
		t.Fatalf("FailureClass = %q, want maturity_contract", got.FailureClass)
	}
}
