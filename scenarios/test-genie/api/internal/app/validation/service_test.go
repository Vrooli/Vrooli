package validation

import (
	"context"
	"path/filepath"
	"runtime"
	"testing"

	"connectrpc.com/connect"
	"github.com/vrooli/maturity-go/assessment"
	scenariovalidationv1 "github.com/vrooli/vrooli/packages/proto/gen/go/scenario-validation/v1"

	"test-genie/internal/providerconformance"
)

func repoRoot(t *testing.T) string {
	t.Helper()
	_, file, _, ok := runtime.Caller(0)
	if !ok {
		t.Fatal("runtime.Caller failed")
	}
	return filepath.Clean(filepath.Join(filepath.Dir(file), "..", "..", "..", "..", "..", ".."))
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

func TestFixRPCsAreHonestlyUnimplemented(t *testing.T) {
	service := newTestService(t)
	if _, err := service.PreviewFix(context.Background(), connect.NewRequest(&scenariovalidationv1.FixRequest{})); connect.CodeOf(err) != connect.CodeUnimplemented {
		t.Fatalf("PreviewFix code = %v, want Unimplemented", connect.CodeOf(err))
	}
	if _, err := service.ApplyFix(context.Background(), connect.NewRequest(&scenariovalidationv1.FixRequest{})); connect.CodeOf(err) != connect.CodeUnimplemented {
		t.Fatalf("ApplyFix code = %v, want Unimplemented", connect.CodeOf(err))
	}
}
