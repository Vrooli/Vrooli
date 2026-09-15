package validate

import (
	"context"
	"testing"

	"connectrpc.com/connect"

	"github.com/vrooli/cli-core/cliapp"
	cliapptest "github.com/vrooli/cli-core/cliapptest"
	validationv1 "github.com/vrooli/vrooli/packages/proto/gen/go/unit-health/v1/validation"
)

type validationClient struct {
	response *validationv1.ValidateScenarioResponse
	request  *validationv1.ValidateScenarioRequest
}

func (c *validationClient) ValidateScenario(_ context.Context, request *connect.Request[validationv1.ValidateScenarioRequest]) (*connect.Response[validationv1.ValidateScenarioResponse], error) {
	c.request = request.Msg
	return connect.NewResponse(c.response), nil
}

func (validationClient) RunCalibration(context.Context, *connect.Request[validationv1.RunCalibrationRequest]) (*connect.Response[validationv1.RunCalibrationResponse], error) {
	return nil, nil
}

func (validationClient) ReadTestBody(context.Context, *connect.Request[validationv1.ReadTestBodyRequest]) (*connect.Response[validationv1.ReadTestBodyResponse], error) {
	return nil, nil
}

func (validationClient) RunMutationPilot(context.Context, *connect.Request[validationv1.RunMutationPilotRequest]) (*connect.Response[validationv1.RunMutationPilotResponse], error) {
	return nil, nil
}

// [REQ:UH-CORE-005]
func TestValidationHandlerRendersRichHumanReport(t *testing.T) {
	h := newHandlers(nil)
	h.client = &validationClient{response: &validationv1.ValidateScenarioResponse{
		Scenario: "demo", Status: "passed", Counts: &validationv1.ValidationCounts{Warnings: 1, Workspaces: 1}, Maturity: &validationv1.MaturitySummary{Label: "L3"},
		Findings:           []*validationv1.ValidationFinding{{Severity: "warning", Code: "RULE", FilePath: "a.go", Message: "finding", Evidence: "evidence", Expected: "expected", Observed: "observed", WhyItMatters: "why", Remediation: "fix", SourceCommand: "go test"}},
		SuppressedFindings: []*validationv1.ValidationFinding{{Code: "OLD", Severity: "info"}}, Workspaces: []*validationv1.TestWorkspace{{Id: "api", Language: "go", CanonicalFramework: "go test", Status: "ready", DegradedReason: "none"}},
		EvidenceStages:   &validationv1.EvidenceStages{Configured: "observed", Analyzed: "partial", Executed: "passed", Reviewed: "not_supplied"},
		Plan:             &validationv1.ExecutionPlan{Commands: []*validationv1.PlannedCommand{{Name: "test", Command: "go test ./...", WorkingDirectory: "api", TimeoutSeconds: 30}}, Notes: "plan note"},
		ProjectionChecks: []*validationv1.ProjectionCheck{{Status: "pass", WorkspaceId: "api", Key: "coverage", Owner: "policy", FilePath: "testing.json", PolicyValue: "75", NativeValue: "75", Remediation: "none"}},
		CommandResults:   []*validationv1.CommandResult{{Name: "test", Status: "passed", ExitCode: 0, DurationMs: 4}}, Coverage: []*validationv1.CoverageTarget{{SurfaceId: "api", Language: "go", CoveredLines: 8, TotalLines: 10, Threshold: 75}}, Diagnostics: []*validationv1.Diagnostic{{Kind: "runtime", Message: "stable", WorkspaceId: "api", Evidence: "none"}}, NextSteps: []string{"keep going"},
	}}
	schema := cliapp.ArgSchema{Positionals: []cliapp.Positional{{Name: "scenario"}}, Flags: []cliapp.Flag{{Name: "path"}, {Name: "workspace"}, {Name: "execution", Bool: true}, {Name: "fast-test-only", Bool: true}, {Name: "reviewed-cohort"}, {Name: "reviewed-source-identity"}, {Name: "reviewed-observation-count"}}}
	ctx, out := cliapptest.NewCapturedRunContext(nil, schema, cliapptest.TestRunContextOptions{Positionals: map[string]string{"scenario": "demo"}})
	if err := h.validateScenario(ctx); err != nil {
		t.Fatal(err)
	}
	if out.Len() == 0 {
		t.Fatal("validation handler rendered no output")
	}
}

func TestValidationHandlerReportsFailedStatusAndNilResponse(t *testing.T) {
	h := newHandlers(nil)
	h.client = &validationClient{response: &validationv1.ValidateScenarioResponse{Scenario: "demo", Status: "failed", Counts: &validationv1.ValidationCounts{Errors: 1}}}
	schema := cliapp.ArgSchema{Positionals: []cliapp.Positional{{Name: "scenario"}}, Flags: []cliapp.Flag{{Name: "path"}, {Name: "workspace"}, {Name: "execution", Bool: true}, {Name: "fast-test-only", Bool: true}, {Name: "reviewed-cohort"}, {Name: "reviewed-source-identity"}, {Name: "reviewed-observation-count"}}}
	ctx, _ := cliapptest.NewCapturedRunContext(nil, schema, cliapptest.TestRunContextOptions{Positionals: map[string]string{"scenario": "demo"}})
	if err := h.validateScenario(ctx); err == nil {
		t.Fatal("failed validation returned nil")
	}
	h.client = &validationClient{}
	if err := h.validateScenario(ctx); err == nil {
		t.Fatal("nil response accepted")
	}
}

func TestValidationHandlerDisablesCacheForExecutedReads(t *testing.T) {
	client := &validationClient{response: &validationv1.ValidateScenarioResponse{Scenario: "demo", Status: "passed", Counts: &validationv1.ValidationCounts{}}}
	h := newHandlers(nil)
	h.client = client
	schema := cliapp.ArgSchema{Positionals: []cliapp.Positional{{Name: "scenario"}}, Flags: []cliapp.Flag{{Name: "path"}, {Name: "workspace"}, {Name: "execution", Bool: true}, {Name: "fast-test-only", Bool: true}, {Name: "reviewed-cohort"}, {Name: "reviewed-source-identity"}, {Name: "reviewed-observation-count"}}}
	ctx, _ := cliapptest.NewCapturedRunContext(nil, schema, cliapptest.TestRunContextOptions{Positionals: map[string]string{"scenario": "demo"}, BoolFlags: map[string]bool{"execution": true}})
	if err := h.validateScenario(ctx); err != nil {
		t.Fatal(err)
	}
	if client.request == nil || client.request.GetUseCache() {
		t.Fatalf("executed request reused cache: %+v", client.request)
	}
}

func TestValidationRenderingHelpersCoverEmptyAndRichBranches(t *testing.T) {
	if len(workspaceLines(nil)) != 0 || len(planLines(nil)) != 0 || len(projectionLines(nil)) != 0 || len(coverageLines(nil)) != 0 || len(diagnosticLines(nil)) != 0 {
		t.Fatal("empty helper inputs should remain empty")
	}
	_ = executionLines(nil)
	_ = executionLines([]*validationv1.CommandResult{{Name: "x", Status: "failed", ExitCode: 1, FailureClass: "timeout", DurationMs: 3}})
}
