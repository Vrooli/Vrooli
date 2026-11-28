package validationbroker

import (
	"context"
	"fmt"
	"strings"
	"time"

	validationv1 "github.com/vrooli/vrooli/packages/proto/gen/go/test-genie/v1/validation"
	"google.golang.org/protobuf/proto"
)

type frozenReplayResolver interface {
	IdentityResolver
	WorkspaceSandboxSealOwner
	ResolveFrozenEvaluatorBundle(context.Context, *validationv1.FrozenReplayExecution) (FrozenEvaluatorBundle, error)
}

func (p *RunProducer) executeFrozenReplay(ctx context.Context, current *validationv1.ValidationReceipt, intent *validationv1.ValidationIntent, transition TransitionFunc) error {
	execution := intent.GetFrozenReplayExecution()
	if p == nil || p.replay == nil || p.identity == nil || current == nil || execution == nil {
		return fmt.Errorf("frozen replay Test Genie producer is unavailable")
	}
	resolver, ok := p.identity.(frozenReplayResolver)
	if !ok {
		return p.terminalizeFailure(ctx, current.GetReceiptId(), transition, validationv1.ValidationReasonCode_VALIDATION_REASON_CODE_PROVIDER_UNAVAILABLE, "frozen replay requires Sandbox and evaluator bundle owner resolution")
	}
	receiptID := current.GetReceiptId()
	childPrefix := "frozen-replay:" + execution.GetCaseId() + ":" + strings.ToLower(execution.GetRole().String()) + ":attempt:"
	attempt := uint32(len(current.GetFrozenReplayAttempts()) + 1)
	if attempt == 0 {
		attempt = 1
	}
	childID := fmt.Sprintf("%s%d", childPrefix, attempt)
	child := childByID(current, childID)
	var admitted *validationv1.SourceIdentity
	if current.GetAdmittedIdentity() != nil {
		admitted = current.GetAdmittedIdentity()
	}
	if child == nil {
		if err := resolver.VerifyReplaySeal(ctx, replaySealReference(execution)); err != nil {
			return p.terminalizeResolutionFailure(ctx, receiptID, transition, err)
		}
		observed, err := resolver.Resolve(ctx, intent)
		if err != nil {
			return p.terminalizeResolutionFailure(ctx, receiptID, transition, err)
		}
		if observed.GetIdentity() != "sha256:"+strings.ToLower(execution.GetSealDigest()) || (admitted != nil && !proto.Equal(admitted, observed)) {
			return p.terminalizeIdentityChange(ctx, receiptID, observed, transition, "workspace seal identity differs from frozen replay candidate")
		}
		admitted = observed
	}
	if admitted == nil {
		return p.terminalizeFailure(ctx, receiptID, transition, validationv1.ValidationReasonCode_VALIDATION_REASON_CODE_PROVIDER_UNAVAILABLE, "admitted frozen replay identity is unavailable")
	}
	state := current.GetState()
	if state == validationv1.ReceiptState_RECEIPT_STATE_ADMITTED || state == validationv1.ReceiptState_RECEIPT_STATE_RETRY_PENDING {
		var err error
		current, err = transition(ctx, receiptID, validationv1.ReceiptState_RECEIPT_STATE_QUEUED, nil)
		if err != nil {
			return err
		}
	}
	if current.GetState() == validationv1.ReceiptState_RECEIPT_STATE_QUEUED {
		var err error
		current, err = transition(ctx, receiptID, validationv1.ReceiptState_RECEIPT_STATE_RUNNING, func(receipt *validationv1.ValidationReceipt) error {
			receipt.AdmittedIdentity = cloneIdentity(admitted)
			return nil
		})
		if err != nil {
			return err
		}
	}
	var result FrozenReplayTestResult
	bundle, err := resolver.ResolveFrozenEvaluatorBundle(ctx, execution)
	if err != nil {
		return p.terminalizeResolutionFailure(ctx, receiptID, transition, err)
	}
	if child == nil {
		duration := intent.GetDeadlinePolicy().GetExecutionBudget().AsDuration()
		if duration <= 0 {
			duration = 10 * time.Minute
		}
		result, err = p.replay.Start(ctx, fmt.Sprintf("test-genie-f9-%s-%d", receiptID, attempt), execution, bundle, duration)
		if err != nil {
			return errEvidencePending // Stable idempotency key makes a lost Start response recoverable.
		}
		if err := validateFrozenReplayResult(result, execution, bundle, ""); err != nil {
			return p.terminalizeFailure(ctx, receiptID, transition, validationv1.ValidationReasonCode_VALIDATION_REASON_CODE_PROVIDER_UNAVAILABLE, err.Error())
		}
		current, err = transition(ctx, receiptID, validationv1.ReceiptState_RECEIPT_STATE_RUNNING, func(receipt *validationv1.ValidationReceipt) error {
			receipt.Children = append(receipt.Children, &validationv1.ChildOperation{ChildId: childID, Kind: validationv1.ChildOperationKind_CHILD_OPERATION_KIND_TEST_RUN, State: validationv1.ChildOperationState_CHILD_OPERATION_STATE_RUNNING, Owner: "workspace-sandbox", OperationId: result.OperationID})
			return nil
		})
		if err != nil {
			return err
		}
	} else {
		result = FrozenReplayTestResult{OperationID: child.GetOperationId(), CaseID: execution.GetCaseId(), Role: execution.GetRole(), SealDigest: execution.GetSealDigest(), EvaluatorBundleID: execution.GetEvaluatorBundleId(), EntryPoint: execution.GetEvaluatorEntryPoint(), Outcome: FrozenReplayRunning}
	}
	if result.Outcome == FrozenReplayRunning {
		duration := intent.GetDeadlinePolicy().GetExecutionBudget().AsDuration()
		if duration <= 0 {
			duration = 10 * time.Minute
		}
		waited, err := p.replay.Wait(ctx, result, duration)
		if err != nil {
			return errEvidencePending
		}
		result = waited
		if result.Outcome == FrozenReplayRunning {
			return errEvidencePending
		}
	}
	if result.Outcome == FrozenReplayInterrupted {
		protoResult := frozenReplayExecutionProtoResult(result, execution)
		maximum := intent.GetDeadlinePolicy().GetMaximumAttempts()
		if maximum == 0 {
			maximum = 1
		}
		if attempt >= maximum {
			_, err := transition(ctx, receiptID, validationv1.ReceiptState_RECEIPT_STATE_FAILED, func(receipt *validationv1.ValidationReceipt) error {
				receipt.FrozenReplayAttempts = append(receipt.FrozenReplayAttempts, protoResult)
				receipt.FrozenReplayResult = protoResult
				receipt.ReasonCode = validationv1.ValidationReasonCode_VALIDATION_REASON_CODE_PROVIDER_UNAVAILABLE
				receipt.Detail = "frozen replay interrupted after maximum attempts"
				if child := childByID(receipt, childID); child != nil {
					child.State = validationv1.ChildOperationState_CHILD_OPERATION_STATE_FAILED
					child.Detail = result.Detail
				}
				return nil
			})
			return err
		}
		_, err := transition(ctx, receiptID, validationv1.ReceiptState_RECEIPT_STATE_RUNNING, func(receipt *validationv1.ValidationReceipt) error {
			receipt.FrozenReplayAttempts = append(receipt.FrozenReplayAttempts, protoResult)
			receipt.FrozenReplayResult = protoResult
			receipt.ReasonCode = validationv1.ValidationReasonCode_VALIDATION_REASON_CODE_PROVIDER_UNAVAILABLE
			receipt.Detail = "frozen replay was interrupted; retry will use a new owner operation after seal verification"
			if child := childByID(receipt, childID); child != nil {
				child.State = validationv1.ChildOperationState_CHILD_OPERATION_STATE_FAILED
				child.Detail = result.Detail
			}
			receipt.Retry = &validationv1.RetryDisposition{Kind: validationv1.RetryKind_RETRY_KIND_SCHEDULED, MaximumAttempts: maximum, ReasonCode: validationv1.ValidationReasonCode_VALIDATION_REASON_CODE_PROVIDER_UNAVAILABLE}
			return nil
		})
		if err != nil {
			return err
		}
		return errEvidencePending
	}
	if err := resolver.VerifyReplaySeal(ctx, replaySealReference(execution)); err != nil {
		result.Outcome = FrozenReplayInfrastructureFailed
		result.SealIntactAfter = false
		result.Detail = "candidate seal verification failed after test: " + err.Error()
	}
	if err := validateFrozenReplayResult(result, execution, FrozenEvaluatorBundle{EvaluatorBundleID: execution.GetEvaluatorBundleId(), EntryPoint: execution.GetEvaluatorEntryPoint()}, childOperation(current, childID)); err != nil {
		result.Outcome = FrozenReplayInfrastructureFailed
		result.Detail = err.Error()
	}
	if result.Detail == "" {
		result.Detail = "frozen replay test completed"
	}
	if len(result.Detail) > 2048 {
		result.Detail = result.Detail[:2048]
	}
	protoOutcome := frozenReplayProtoOutcome(result.Outcome)
	if protoOutcome == validationv1.FrozenReplayTestOutcome_FROZEN_REPLAY_TEST_OUTCOME_UNSPECIFIED {
		protoOutcome = validationv1.FrozenReplayTestOutcome_FROZEN_REPLAY_TEST_OUTCOME_INFRASTRUCTURE_FAILED
	}
	state = validationv1.ReceiptState_RECEIPT_STATE_FAILED
	reason := validationv1.ValidationReasonCode_VALIDATION_REASON_CODE_PROVIDER_UNAVAILABLE
	if result.Outcome == FrozenReplayPassed {
		state = validationv1.ReceiptState_RECEIPT_STATE_SUCCEEDED
		reason = validationv1.ValidationReasonCode_VALIDATION_REASON_CODE_NONE
	} else if result.Outcome == FrozenReplayAssertionFailed {
		reason = validationv1.ValidationReasonCode_VALIDATION_REASON_CODE_REQUIRED_EVIDENCE_FAILED
	}
	_, err = transition(ctx, receiptID, state, func(receipt *validationv1.ValidationReceipt) error {
		resultRecord := frozenReplayExecutionProtoResult(result, execution)
		resultRecord.Outcome = protoOutcome
		receipt.FrozenReplayAttempts = append(receipt.FrozenReplayAttempts, resultRecord)
		receipt.FrozenReplayResult = resultRecord
		receipt.ObservedIdentity = cloneIdentity(admitted)
		receipt.ReasonCode = reason
		receipt.Detail = result.Detail
		if child := childByID(receipt, childID); child != nil {
			if state == validationv1.ReceiptState_RECEIPT_STATE_SUCCEEDED {
				child.State = validationv1.ChildOperationState_CHILD_OPERATION_STATE_SUCCEEDED
			} else {
				child.State = validationv1.ChildOperationState_CHILD_OPERATION_STATE_FAILED
			}
			child.Detail = result.Detail
		}
		if state == validationv1.ReceiptState_RECEIPT_STATE_SUCCEEDED {
			receipt.AchievedStrength = intent.GetRequiredStrength()
			receipt.Retry = &validationv1.RetryDisposition{Kind: validationv1.RetryKind_RETRY_KIND_NOT_NEEDED, MaximumAttempts: intent.GetDeadlinePolicy().GetMaximumAttempts()}
		}
		return nil
	})
	return err
}

func replaySealReference(execution *validationv1.FrozenReplayExecution) *validationv1.WorkspaceSandboxSealReference {
	return &validationv1.WorkspaceSandboxSealReference{CaseId: execution.GetCaseId(), Role: execution.GetRole(), SealDigest: execution.GetSealDigest()}
}

func childOperation(receipt *validationv1.ValidationReceipt, childID string) string {
	if child := childByID(receipt, childID); child != nil {
		return child.GetOperationId()
	}
	return ""
}

func validateFrozenReplayResult(result FrozenReplayTestResult, execution *validationv1.FrozenReplayExecution, bundle FrozenEvaluatorBundle, expectedOperationID string) error {
	if !result.Success || result.OperationID == "" || result.CaseID != execution.GetCaseId() || result.Role != execution.GetRole() || result.SealDigest != execution.GetSealDigest() || result.EvaluatorBundleID != execution.GetEvaluatorBundleId() || result.EntryPoint != execution.GetEvaluatorEntryPoint() {
		return fmt.Errorf("Workspace Sandbox returned a mismatched frozen replay identity")
	}
	if expectedOperationID != "" && result.OperationID != expectedOperationID {
		return fmt.Errorf("Workspace Sandbox returned a different operation during recovery")
	}
	if result.Outcome != FrozenReplayRunning && (!resultDigestValid(result.ResultDigest) || !result.SealIntactBefore || !result.SealIntactAfter) {
		if result.Outcome != FrozenReplayInterrupted || !resultDigestValid(result.ResultDigest) {
			return fmt.Errorf("Workspace Sandbox terminal replay result lacks a valid digest or intact seal")
		}
	}
	if bundle.ContentDigest != "" && strings.TrimPrefix(strings.ToLower(result.EvaluatorBundleDigest), "sha256:") != strings.TrimPrefix(strings.ToLower(bundle.ContentDigest), "sha256:") {
		return fmt.Errorf("Workspace Sandbox evaluator content identity differs from the registered bundle")
	}
	if result.Outcome != FrozenReplayRunning && result.Outcome != FrozenReplayPassed && result.Outcome != FrozenReplayAssertionFailed && result.Outcome != FrozenReplayInfrastructureFailed && result.Outcome != FrozenReplayInterrupted {
		return fmt.Errorf("Workspace Sandbox returned an unknown replay test outcome")
	}
	return nil
}

func frozenReplayProtoOutcome(outcome FrozenReplayTestOutcome) validationv1.FrozenReplayTestOutcome {
	switch outcome {
	case FrozenReplayPassed:
		return validationv1.FrozenReplayTestOutcome_FROZEN_REPLAY_TEST_OUTCOME_PASSED
	case FrozenReplayAssertionFailed:
		return validationv1.FrozenReplayTestOutcome_FROZEN_REPLAY_TEST_OUTCOME_ASSERTION_FAILED
	case FrozenReplayInfrastructureFailed:
		return validationv1.FrozenReplayTestOutcome_FROZEN_REPLAY_TEST_OUTCOME_INFRASTRUCTURE_FAILED
	case FrozenReplayInterrupted:
		return validationv1.FrozenReplayTestOutcome_FROZEN_REPLAY_TEST_OUTCOME_INTERRUPTED
	default:
		return validationv1.FrozenReplayTestOutcome_FROZEN_REPLAY_TEST_OUTCOME_UNSPECIFIED
	}
}

func frozenReplayExecutionProtoResult(result FrozenReplayTestResult, execution *validationv1.FrozenReplayExecution) *validationv1.FrozenReplayExecutionResult {
	return &validationv1.FrozenReplayExecutionResult{RunId: result.OperationID, SandboxOperationId: result.OperationID, Outcome: frozenReplayProtoOutcome(result.Outcome), CaseId: result.CaseID, Role: result.Role, SealDigest: result.SealDigest, SourceIdentity: "sha256:" + strings.TrimPrefix(strings.ToLower(result.SealDigest), "sha256:"), EvaluatorBundleId: result.EvaluatorBundleID, EvaluatorDigest: execution.GetEvaluatorDigest(), EvaluatorEntryPoint: result.EntryPoint, ResultDigest: result.ResultDigest, SealIntactBefore: result.SealIntactBefore, SealIntactAfter: result.SealIntactAfter, Detail: result.Detail}
}
