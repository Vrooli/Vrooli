package validation

import (
	"context"
	"encoding/json"
	"path/filepath"
	"testing"

	"connectrpc.com/connect"
	"github.com/vrooli/maturity-go/assessment"
	repocontract "github.com/vrooli/repo-contract-go"
	scenariovalidationv1 "github.com/vrooli/vrooli/packages/proto/gen/go/scenario-validation/v1"
	"google.golang.org/protobuf/types/known/structpb"

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

func TestProgramRuntimeFaultInjectionRunsGeneratedMatrix(t *testing.T) {
	service := newTestService(t)
	root := repoRoot(t)
	target := filepath.Join(root, "scenarios", "program-runtime")
	service.repoRoot = root
	service.faultSpec, _ = assessment.LoadSpecForPhase(filepath.Join(root, "scenarios", "test-genie"), "fault-injection")
	classes := []string{"unavailable", "degraded", "healthy-empty", "malformed", "partial"}
	rows := make([]map[string]any, 0, len(classes))
	sourceRows := make([]map[string]any, 0, len(classes))
	statuses := map[string]string{"unavailable": "unavailable", "degraded": "partial", "healthy-empty": "failed", "malformed": "failed", "partial": "partial"}
	behaviors := map[string]string{"unavailable": "propagate_child_status_without_retry", "degraded": "propagate_child_status_without_retry", "partial": "propagate_child_status_without_retry", "healthy-empty": "refuse_empty_child_result", "malformed": "refuse_malformed_child_envelope"}
	for _, fault := range classes {
		status := statuses[fault]
		rows = append(rows, map[string]any{
			"program": "fixture.program", "contract_digest": "sha256:fixture", "fault": fault, "passed": true,
			"expected_status": status, "observed_status": status,
			"capability_requirements": []any{map[string]any{"binding": "fixture/read", "effect": "read"}},
			"observable_fallback":     map[string]any{"behavior": behaviors[fault], "status": status},
		})
		parentStatus := status
		if fault == "healthy-empty" || fault == "malformed" {
			parentStatus = "failed"
		}
		sourceRows = append(sourceRows, map[string]any{
			"program": "fixture.program", "fault": fault, "passed": true,
			"execution_kind": "program_runtime_session_with_injected_library_handles", "native_qualification": false,
			"injected_dependencies": []any{"fixture.child"}, "expected_child_status": status,
			"capability_requirements":         []any{map[string]any{"library_program": "fixture.child"}},
			"no_retry":                        true,
			"expected_status_by_dependency": map[string]any{"fixture.child": status},
			"observed_child_envelopes":      []any{map[string]any{"dependency": "fixture.child", "status": status, "errors": []any{}}},
			"parent_status": parentStatus, "parent_status_enum": []any{"ok", "partial", "unavailable", "refused", "failed"},
			"parent_phase": "report", "parent_phase_enum": []any{"report"},
			"observable_fallback": map[string]any{"behavior": "owner_status_or_explicit_degraded_signal", "observed": true,
				"status": parentStatus, "phase": "report", "error_classes": []any{}, "signal_markers": []any{}},
		})
	}
	sourceExecution := map[string]any{"execution_kind": "program_runtime_session_with_injected_library_handles", "native_qualification": false,
		"program_count": 1, "fault_classes": classes, "row_count": len(sourceRows), "passed_rows": len(sourceRows), "failed_rows": 0, "matrix": sourceRows}
	encodedEvidence, _ := json.Marshal(map[string]any{"evidence_kind": "generated_fixture", "native_qualification": false, "program_count": 1, "fault_classes": classes, "row_count": len(rows), "passed_rows": len(rows), "failed_rows": 0, "matrix": rows, "program_source_execution": sourceExecution})
	service.matrixFaultRun = func(_ context.Context, scenarioRoot string) (string, error) {
		if scenarioRoot != target {
			t.Fatalf("runner target=%q want %q", scenarioRoot, target)
		}
		return string(encodedEvidence), nil
	}
	resp, err := service.ValidateScenario(context.Background(), connect.NewRequest(&scenariovalidationv1.ValidateScenarioRequest{Scenario: "program-runtime", Path: target, Phase: "fault-injection", IncludeExecution: true}))
	if err != nil {
		t.Fatalf("ValidateScenario: %v", err)
	}
	if resp.Msg.GetScenario() != "program-runtime" || resp.Msg.GetStatus() != scenariovalidationv1.ValidationStatus_VALIDATION_STATUS_PASSED {
		t.Fatalf("scenario=%q status=%s", resp.Msg.GetScenario(), resp.Msg.GetStatus())
	}
	if err := assessment.RequireProviderContract("test-genie", "fault-injection", resp.Msg.GetAssessment()); err != nil {
		t.Fatalf("provider contract: %v", err)
	}
}

func TestProgramRuntimeFaultInjectionRunsRealGeneratedMatrixInProcess(t *testing.T) {
	service := newTestService(t)
	root := repoRoot(t)
	service.repoRoot = root
	service.faultSpec, _ = assessment.LoadSpecForPhase(filepath.Join(root, "scenarios", "test-genie"), "fault-injection")
	target := filepath.Join(root, "scenarios", "program-runtime")
	resp, err := service.ValidateScenario(context.Background(), connect.NewRequest(&scenariovalidationv1.ValidateScenarioRequest{Scenario: "program-runtime", Path: target, Phase: "fault-injection", IncludeExecution: true}))
	if err != nil {
		t.Fatalf("ValidateScenario: %v", err)
	}
	if resp.Msg.GetStatus() != scenariovalidationv1.ValidationStatus_VALIDATION_STATUS_PASSED {
		t.Fatalf("status=%s, want PASS", resp.Msg.GetStatus())
	}
	var detail structpb.Struct
	if err := resp.Msg.GetNativeDetail().UnmarshalTo(&detail); err != nil {
		t.Fatalf("native detail: %v", err)
	}
	matrix := detail.GetFields()["fault_matrix"].GetStructValue()
	if matrix == nil || matrix.GetFields()["program_count"].GetNumberValue() != 19 || matrix.GetFields()["row_count"].GetNumberValue() != 95 || matrix.GetFields()["failed_rows"].GetNumberValue() != 0 {
		t.Fatalf("incomplete matrix evidence: %v", matrix)
	}
	rows := matrix.GetFields()["matrix"].GetListValue().GetValues()
	if len(rows) != 95 {
		t.Fatalf("matrix rows=%d want 95", len(rows))
	}
	for _, value := range rows {
		row := value.GetStructValue().GetFields()
		if row["capability_requirements"].GetListValue().GetValues() == nil || len(row["capability_requirements"].GetListValue().GetValues()) == 0 {
			t.Fatalf("row omits capability requirement: %v", row)
		}
		fallback := row["observable_fallback"].GetStructValue().GetFields()
		if fallback["behavior"].GetStringValue() == "" || fallback["status"].GetStringValue() == "" {
			t.Fatalf("row omits observable fallback: %v", row)
		}
	}
}

func TestProgramRuntimeFaultInjectionRefusesPathRedirection(t *testing.T) {
	service := newTestService(t)
	service.repoRoot = repoRoot(t)
	service.faultSpec, _ = assessment.LoadSpecForPhase(filepath.Join(service.repoRoot, "scenarios", "test-genie"), "fault-injection")
	service.matrixFaultRun = func(context.Context, string) (string, error) {
		t.Fatal("runner must not receive an arbitrary path")
		return "", nil
	}
	_, err := service.ValidateScenario(context.Background(), connect.NewRequest(&scenariovalidationv1.ValidateScenarioRequest{Scenario: "program-runtime", Path: t.TempDir(), Phase: "fault-injection", IncludeExecution: true}))
	if connect.CodeOf(err) != connect.CodeInvalidArgument {
		t.Fatalf("error code=%v want InvalidArgument", connect.CodeOf(err))
	}
}

func TestGeneratedFaultMatrixRejectsIncompleteEvidence(t *testing.T) {
	_, err := extractProgramRuntimeMatrix(`{"evidence_kind":"generated_fixture","native_qualification":false,"program_count":19,"fault_classes":["unavailable","degraded","healthy-empty","malformed","partial"],"row_count":95,"passed_rows":94,"failed_rows":1}`)
	if err == nil {
		t.Fatal("incomplete generated matrix must fail closed")
	}
}

func TestGeneratedFaultMatrixRejectsMissingParentSourceExecution(t *testing.T) {
	classes := []string{"unavailable", "degraded", "healthy-empty", "malformed", "partial"}
	statuses := map[string]string{"unavailable": "unavailable", "degraded": "partial", "healthy-empty": "failed", "malformed": "failed", "partial": "partial"}
	behaviors := map[string]string{"unavailable": "propagate_child_status_without_retry", "degraded": "propagate_child_status_without_retry", "healthy-empty": "refuse_empty_child_result", "malformed": "refuse_malformed_child_envelope", "partial": "propagate_child_status_without_retry"}
	matrix := make([]any, 0, len(classes))
	for _, fault := range classes {
		status := statuses[fault]
		matrix = append(matrix, map[string]any{"program": "fixture.program", "contract_digest": "sha256:fixture", "capability_requirements": []any{map[string]any{"binding": "fixture/read", "effect": "read"}},
			"fault": fault, "expected_status": status, "observed_status": status, "passed": true,
			"observable_fallback": map[string]any{"behavior": behaviors[fault], "status": status}})
	}
	payload, err := json.Marshal(map[string]any{"evidence_kind": "generated_fixture", "native_qualification": false, "program_count": 1,
		"fault_classes": classes, "row_count": len(matrix), "passed_rows": len(matrix), "failed_rows": 0, "matrix": matrix})
	if err != nil {
		t.Fatalf("marshal fixture: %v", err)
	}
	if _, err := extractProgramRuntimeMatrix(string(payload)); err == nil {
		t.Fatal("matrix without per-program source execution must fail closed")
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
