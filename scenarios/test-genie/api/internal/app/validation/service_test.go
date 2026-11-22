package validation

import (
	"context"
	"path/filepath"
	"testing"

	"connectrpc.com/connect"
	"github.com/vrooli/maturity-go/assessment"
	repocontract "github.com/vrooli/repo-contract-go"
	scenariovalidationv1 "github.com/vrooli/vrooli/packages/proto/gen/go/scenario-validation/v1"

	"test-genie/internal/providerconformance"
)

func repoRoot(t *testing.T) string {
	t.Helper()
	root, err := repocontract.FindRepoRootFromCWD()
	if err != nil {
		t.Fatalf("resolve repository root: %v", err)
	}
	return root
}

func newTestService(t *testing.T) *Service {
	t.Helper()
	root := repoRoot(t)
	spec, err := assessment.LoadSpecFromScenario(filepath.Join(root, "scenarios", "test-genie"))
	if err != nil {
		t.Fatalf("load own spec: %v", err)
	}
	// No probe seam: descriptor-only validation keeps the test hermetic.
	return NewServiceForTest(nil, providerconformance.New(root), spec)
}

// TestValidateScenarioSelfIsContractValid proves the full self loop: Test
// Genie validates its own provider descriptor and the response satisfies the
// same contract, identity, and metrics rules the fleet scan enforces.
func TestValidateScenarioSelfIsContractValid(t *testing.T) {
	service := newTestService(t)
	resp, err := service.ValidateScenario(context.Background(), connect.NewRequest(&scenariovalidationv1.ValidateScenarioRequest{
		Scenario: "test-genie",
	}))
	if err != nil {
		t.Fatalf("ValidateScenario: %v", err)
	}
	msg := resp.Msg
	if msg.GetStatus() != scenariovalidationv1.ValidationStatus_VALIDATION_STATUS_PASSED {
		t.Fatalf("status = %v, want PASSED (assessment: %v)", msg.GetStatus(), msg.GetAssessment().GetFindings())
	}
	if err := assessment.ValidateAssessment(msg.GetAssessment()); err != nil {
		t.Fatalf("assessment contract: %v", err)
	}
	if err := assessment.RequireIdentity("test-genie", "provider-conformance", msg.GetAssessment()); err != nil {
		t.Fatalf("assessment identity: %v", err)
	}
	if msg.GetMetrics() == nil {
		t.Fatal("response must carry execution metrics")
	}
	if msg.GetNativeDetail() == nil {
		t.Fatal("response must carry native detail")
	}
}

func TestValidateScenarioMissingTarget(t *testing.T) {
	service := newTestService(t)
	_, err := service.ValidateScenario(context.Background(), connect.NewRequest(&scenariovalidationv1.ValidateScenarioRequest{}))
	if connect.CodeOf(err) != connect.CodeInvalidArgument {
		t.Fatalf("error code = %v, want InvalidArgument", connect.CodeOf(err))
	}
}

func TestValidateScenarioWithoutSpecFailsClosed(t *testing.T) {
	service := NewServiceForTest(nil, providerconformance.New(repoRoot(t)), nil)
	_, err := service.ValidateScenario(context.Background(), connect.NewRequest(&scenariovalidationv1.ValidateScenarioRequest{
		Scenario: "test-genie",
	}))
	if connect.CodeOf(err) != connect.CodeInternal {
		t.Fatalf("error code = %v, want Internal", connect.CodeOf(err))
	}
}

func TestDBMFaultInjectionRunsTheFixedSimulatorTargetAndReturnsTypedEvidence(t *testing.T) {
	service := newTestService(t)
	root := repoRoot(t)
	target := filepath.Join(root, "scenarios", "data-backup-manager")
	faultSpec, err := assessment.LoadSpecForPhase(filepath.Join(root, "scenarios", "test-genie"), "fault-injection")
	if err != nil {
		t.Fatalf("load fault-injection spec: %v", err)
	}
	service.repoRoot = root
	service.faultSpec = faultSpec
	called := false
	service.faultRun = func(_ context.Context, scenarioRoot string) (string, error) {
		called = true
		if scenarioRoot != target {
			t.Fatalf("runner target=%q want %q", scenarioRoot, target)
		}
		return `DB6_FIXTURE_RECEIPTS={"evidence_kind":"fixture","native_qualification":false,"receipt_rows":[{"capability":"backup","evidence_kind":"fixture"}]}`, nil
	}
	resp, err := service.ValidateScenario(context.Background(), connect.NewRequest(&scenariovalidationv1.ValidateScenarioRequest{Scenario: "data-backup-manager", Path: target, Phase: "fault-injection", IncludeExecution: true}))
	if err != nil {
		t.Fatalf("ValidateScenario: %v", err)
	}
	if !called || resp.Msg.GetStatus() != scenariovalidationv1.ValidationStatus_VALIDATION_STATUS_PASSED {
		t.Fatalf("called=%v status=%s", called, resp.Msg.GetStatus())
	}
	if err := assessment.RequireProviderContract("test-genie", "fault-injection", resp.Msg.GetAssessment()); err != nil {
		t.Fatalf("provider contract: %v", err)
	}
	if resp.Msg.GetNativeDetail() == nil || resp.Msg.GetMetrics() == nil {
		t.Fatal("typed native detail and execution metrics are required")
	}
}

func TestDBMFaultInjectionFailsWhenTypedReceiptsAreMissing(t *testing.T) {
	service := newTestService(t)
	root := repoRoot(t)
	target := filepath.Join(root, "scenarios", "data-backup-manager")
	service.repoRoot = root
	service.faultSpec, _ = assessment.LoadSpecForPhase(filepath.Join(root, "scenarios", "test-genie"), "fault-injection")
	service.faultRun = func(context.Context, string) (string, error) {
		return "owner tests completed without typed receipts", nil
	}
	resp, err := service.ValidateScenario(context.Background(), connect.NewRequest(&scenariovalidationv1.ValidateScenarioRequest{Scenario: "data-backup-manager", Path: target, Phase: "fault-injection", IncludeExecution: true}))
	if err != nil {
		t.Fatalf("ValidateScenario: %v", err)
	}
	if resp.Msg.GetStatus() != scenariovalidationv1.ValidationStatus_VALIDATION_STATUS_FAILED {
		t.Fatalf("status=%s want failure", resp.Msg.GetStatus())
	}
	if err := assessment.RequireProviderContract("test-genie", "fault-injection", resp.Msg.GetAssessment()); err != nil {
		t.Fatalf("provider contract: %v", err)
	}
}

func TestDBMFaultInjectionRefusesPathRedirection(t *testing.T) {
	service := newTestService(t)
	service.repoRoot = repoRoot(t)
	service.faultSpec, _ = assessment.LoadSpecForPhase(filepath.Join(service.repoRoot, "scenarios", "test-genie"), "fault-injection")
	service.faultRun = func(context.Context, string) (string, error) {
		t.Fatal("runner must not receive an arbitrary path")
		return "", nil
	}
	_, err := service.ValidateScenario(context.Background(), connect.NewRequest(&scenariovalidationv1.ValidateScenarioRequest{Scenario: "data-backup-manager", Path: t.TempDir(), Phase: "fault-injection", IncludeExecution: true}))
	if connect.CodeOf(err) != connect.CodeInvalidArgument {
		t.Fatalf("error code=%v want InvalidArgument", connect.CodeOf(err))
	}
}

func TestFixRPCsAreHonestlyUnimplemented(t *testing.T) {
	service := newTestService(t)
	if _, err := service.PreviewFix(context.Background(), connect.NewRequest(&scenariovalidationv1.FixRequest{})); connect.CodeOf(err) != connect.CodeUnimplemented {
		t.Fatalf("PreviewFix code = %v, want Unimplemented", connect.CodeOf(err))
	}
	if _, err := service.ApplyFix(context.Background(), connect.NewRequest(&scenariovalidationv1.FixRequest{})); connect.CodeOf(err) != connect.CodeUnimplemented {
		t.Fatalf("ApplyFix code = %v, want Unimplemented", connect.CodeOf(err))
	}
}
