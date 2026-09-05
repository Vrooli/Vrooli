package validation

import (
	"context"
	"crypto/sha256"
	"fmt"
	"strings"
	"time"

	planmodel "plan-manager/internal/planmodel"

	commonv1 "github.com/vrooli/vrooli/packages/proto/gen/go/common/v1"
	validationv1 "github.com/vrooli/vrooli/packages/proto/gen/go/test-genie/v1/validation"
	"google.golang.org/protobuf/types/known/durationpb"
)

// ReceiptClient is Plan Manager's only validation execution dependency.
// Test Genie owns every lifecycle transition behind this narrow typed seam.
type ReceiptClient interface {
	CreateValidation(context.Context, *validationv1.ValidationIntent) (*validationv1.ValidationReceipt, error)
	GetValidation(context.Context, string) (*validationv1.ValidationReceipt, error)
}

func (s *service) startReceiptValidation(ctx context.Context, request ValidationTicketRequest, plan planmodel.Plan, boundary planmodel.ChangeBoundary, refs []planmodel.Reference, projection ValidationOperation) (ValidationOperation, bool, error) {
	scenarios := uniqueSortedStrings(append([]string(nil), projection.SelectedMembers...))
	if len(scenarios) == 0 {
		scenarios = uniqueSortedStrings(boundary.AffectedScenarios())
		for _, ref := range refs {
			if scenario := scenarioFromTarget(ref.Target); scenario != "" {
				scenarios = append(scenarios, scenario)
			}
		}
		scenarios = uniqueSortedStrings(scenarios)
	}
	if len(scenarios) == 0 {
		return ValidationOperation{}, false, fmt.Errorf("validation receipt requires at least one affected scenario")
	}
	intent := &validationv1.ValidationIntent{
		SchemaVersion:     1,
		IdempotencyKey:    receiptIdempotencyKey(request, projection),
		CallerScenario:    "plan-manager",
		CallerExecutionId: strings.TrimSpace(request.ExecutionID),
		PlanId:            plan.ID,
		PhaseId:           request.PhaseID,
		Purpose:           validationv1.ValidationPurpose_VALIDATION_PURPOSE_PHASE,
		RequiredStrength:  validationv1.ValidationStrength_VALIDATION_STRENGTH_TARGETED,
		ReusePolicy:       &validationv1.ReusePolicy{Mode: validationv1.ReuseMode_REUSE_MODE_ATTACH_OR_TERMINAL, MaximumAge: durationpb.New(24 * time.Hour)},
		ConcurrencyPolicy: &validationv1.ConcurrencyPolicy{Mode: validationv1.ConcurrencyMode_CONCURRENCY_MODE_SHARED_COMPATIBLE, MaximumParallelism: maxValidationConcurrency},
		DeadlinePolicy:    &validationv1.DeadlinePolicy{QueueBudget: durationpb.New(defaultQueueBudget), ExecutionBudget: durationpb.New(defaultExecutionBudget), MaximumAttempts: 2},
		EvidencePolicy:    &validationv1.EvidencePolicy{RequireBehavioralBefore: strings.TrimSpace(plan.BaselineSet.Name) != "", RequireSourceSnapshot: len(boundary.RepoPaths()) > 0, RequiredEvidenceKinds: []string{"test-genie-run"}},
		CallerAttributes: map[string]string{
			"baseline_name":    plan.BaselineSet.Name,
			"scope_generation": fmt.Sprintf("%d", request.ScopeGeneration),
		},
	}
	if intent.GetEvidencePolicy().GetRequireBehavioralBefore() {
		intent.EvidencePolicy.RequiredEvidenceKinds = append(intent.EvidencePolicy.RequiredEvidenceKinds, "gct-collection-diff")
	}
	if intent.GetEvidencePolicy().GetRequireSourceSnapshot() {
		intent.EvidencePolicy.RequiredEvidenceKinds = append(intent.EvidencePolicy.RequiredEvidenceKinds, "gct-source-snapshot")
	}
	if request.PhaseID == "" {
		intent.Purpose = validationv1.ValidationPurpose_VALIDATION_PURPOSE_CERTIFICATION
		intent.RequiredStrength = validationv1.ValidationStrength_VALIDATION_STRENGTH_CERTIFICATION
	}
	for index, scenario := range scenarios {
		intent.Targets = append(intent.Targets, &commonv1.ValidationTarget{Kind: commonv1.ValidationTargetKind_VALIDATION_TARGET_KIND_SCENARIO, Id: scenario, Root: "scenarios/" + scenario})
		intent.ContentInputs = append(intent.ContentInputs, &validationv1.ContentInputRoot{Name: "scenario-" + scenario, Root: "scenarios/" + scenario, Dependency: index > 0, Selections: []*validationv1.InputSelection{{Glob: "**", Required: true}}})
	}
	if paths := uniqueSortedStrings(boundary.RepoPaths()); len(paths) > 0 {
		root := &validationv1.ContentInputRoot{Name: "plan-boundary", Root: ".", Dependency: true}
		for _, path := range paths {
			root.Selections = append(root.Selections, &validationv1.InputSelection{Glob: path, Required: false})
		}
		intent.ContentInputs = append(intent.ContentInputs, root)
	}
	receipt, err := s.receipts.CreateValidation(ctx, intent)
	if err != nil {
		return ValidationOperation{}, false, err
	}
	projection.ID = receipt.GetReceiptId()
	projection.Result.ID = projection.ID + ":result"
	projection.Result.OperationID = projection.ID
	projection.Children = nil
	projection.ProducerWaitArgv = []string{"test-genie", "validation", "wait", "--wait-id", "plan-manager-" + projection.ID, projection.ID, "--json"}
	projection.SyncArgv = []string{"plan-manager", "validate", "sync", projection.ID}
	projection = projectReceipt(projection, receipt, s.now())
	stored, created, err := s.operations.CreateOperation(ctx, projection)
	if err != nil {
		return ValidationOperation{}, false, err
	}
	reused := !created || receipt.GetCompatibility().GetKind() != validationv1.CompatibilityKind_COMPATIBILITY_KIND_NEW_WORK
	return stored, reused, nil
}

func receiptIdempotencyKey(request ValidationTicketRequest, projection ValidationOperation) string {
	if explicit := strings.TrimSpace(request.IdempotencyKey); explicit != "" {
		return explicit
	}
	material := strings.Join([]string{
		projection.PlanID,
		projection.PhaseID,
		strings.TrimSpace(request.ExecutionID),
		fmt.Sprintf("%d", request.ScopeGeneration),
		projection.ScopeFingerprint,
	}, "\x00")
	sum := sha256.Sum256([]byte(material))
	return fmt.Sprintf("plan-manager:%x", sum[:])
}

func projectReceipt(operation ValidationOperation, receipt *validationv1.ValidationReceipt, now string) ValidationOperation {
	if receipt == nil {
		return operation
	}
	operation.ID = receipt.GetReceiptId()
	operation.QueueReason = receipt.GetDetail()
	operation.Children = make([]ValidationChild, 0, len(receipt.GetChildren()))
	for _, child := range receipt.GetChildren() {
		status := ChildQueued
		if child.GetState() == validationv1.ChildOperationState_CHILD_OPERATION_STATE_RUNNING {
			status = ChildRunning
		} else if child.GetState() == validationv1.ChildOperationState_CHILD_OPERATION_STATE_SUCCEEDED || child.GetState() == validationv1.ChildOperationState_CHILD_OPERATION_STATE_FAILED || child.GetState() == validationv1.ChildOperationState_CHILD_OPERATION_STATE_CANCELLED {
			status = ChildTerminal
		}
		verdict := VerdictUnknown
		if child.GetState() == validationv1.ChildOperationState_CHILD_OPERATION_STATE_SUCCEEDED {
			verdict = VerdictPass
		} else if child.GetState() == validationv1.ChildOperationState_CHILD_OPERATION_STATE_FAILED {
			verdict = VerdictFail
		}
		operation.Children = append(operation.Children, ValidationChild{ID: child.GetChildId(), ExternalID: child.GetOperationId(), Status: status, Verdict: verdict, Detail: child.GetDetail(), Oracle: true})
	}
	if receiptTerminal(receipt.GetState()) {
		operation.Status, operation.TerminalAt, operation.QueueReason = OperationTerminal, now, ""
		verdict := VerdictUnknown
		if receipt.GetState() == validationv1.ReceiptState_RECEIPT_STATE_SUCCEEDED {
			verdict = VerdictPass
		} else if receipt.GetState() == validationv1.ReceiptState_RECEIPT_STATE_FAILED || receipt.GetState() == validationv1.ReceiptState_RECEIPT_STATE_CANCELLED {
			verdict = VerdictFail
		}
		operation.Result = &Result{ID: operation.ID + ":result", PlanID: operation.PlanID, PhaseID: operation.PhaseID, Verdict: verdict, Staleness: planmodel.StalenessFresh, Detail: receipt.GetDetail(), RanAt: now, ExecutionID: operation.ExecutionID, OperationID: operation.ID, ScopeGeneration: operation.ScopeGeneration, FullInventory: operation.FullInventory, RequiredMembers: append([]string(nil), operation.RequiredMembers...), SelectedMembers: append([]string(nil), operation.SelectedMembers...)}
	} else if receipt.GetState() == validationv1.ReceiptState_RECEIPT_STATE_RUNNING || receipt.GetState() == validationv1.ReceiptState_RECEIPT_STATE_ATTACHED {
		operation.Status = OperationRunning
	} else {
		operation.Status = OperationQueued
	}
	return operation
}

func receiptTerminal(state validationv1.ReceiptState) bool {
	switch state {
	case validationv1.ReceiptState_RECEIPT_STATE_SUCCEEDED, validationv1.ReceiptState_RECEIPT_STATE_FAILED, validationv1.ReceiptState_RECEIPT_STATE_DEGRADED, validationv1.ReceiptState_RECEIPT_STATE_CANCELLED, validationv1.ReceiptState_RECEIPT_STATE_SUPERSEDED:
		return true
	default:
		return false
	}
}
