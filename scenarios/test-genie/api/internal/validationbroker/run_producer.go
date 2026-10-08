package validationbroker

import (
	"context"
	"crypto/sha256"
	"errors"
	"fmt"
	"strings"
	"time"

	"test-genie/internal/execution"
	"test-genie/internal/orchestrator"
	"test-genie/internal/runmanager"
	sharedruns "test-genie/internal/shared/runs"

	commonv1 "github.com/vrooli/vrooli/packages/proto/gen/go/common/v1"
	scenariovalidationv1 "github.com/vrooli/vrooli/packages/proto/gen/go/scenario-validation/v1"
	validationv1 "github.com/vrooli/vrooli/packages/proto/gen/go/test-genie/v1/validation"
	"google.golang.org/protobuf/proto"
	"google.golang.org/protobuf/types/known/timestamppb"
)

type suiteRunManager interface {
	Start(runmanager.StartOptions) (runmanager.StartResult, error)
	Wait(context.Context, string, string) (runmanager.LiveStatus, error)
	Abort(string, string) (runmanager.LiveStatus, error)
	AbortEvidenceProducer(runmanager.StartOptions) (runmanager.LiveStatus, error)
}

type retainedEvidenceSetOwner interface {
	RetainedEvidenceSet(scenario, runID, receiptID, producer, candidateIdentity string) (*scenariovalidationv1.RetainedEvidenceSet, error)
}

// RunProducer translates scenario validation intent into the existing durable
// suite-run authority. Each attachment blocks on the child; observation errors
// reattach to that same durable operation rather than restarting its work.
type RunProducer struct {
	runs     suiteRunManager
	identity IdentityResolver
	gct      GCTEvidenceClient
	planner  execution.ExecutionPlanner
}

const evidenceProducerCleanupReserve = 60 * time.Second

func evidenceProductionDeadline(admittedAt time.Time, timeout time.Duration, cleanupReserve time.Duration) (time.Time, error) {
	maxDuration := time.Duration(int64(^uint64(0) >> 1))
	if admittedAt.IsZero() || timeout <= 0 || cleanupReserve < 0 || timeout > maxDuration-cleanupReserve {
		return time.Time{}, errors.New("pinned evidence producer deadline is invalid")
	}
	return admittedAt.Add(timeout + cleanupReserve), nil
}

func (p *RunProducer) WithExecutionPlanner(planner execution.ExecutionPlanner) *RunProducer {
	p.planner = planner
	return p
}

var errQueueBudget = errors.New("validation queue budget exhausted")

// A failed observer attachment says nothing about the durable child's verdict.
var errEvidencePending = errors.New("durable evidence requires reattachment")
var errSuiteStartUncertain = errors.New("suite start outcome is uncertain")

func NewRunProducer(runs suiteRunManager, identity ...IdentityResolver) *RunProducer {
	producer := &RunProducer{runs: runs}
	if len(identity) > 0 {
		producer.identity = identity[0]
	}
	return producer
}

func (p *RunProducer) WithGCTEvidence(client GCTEvidenceClient) *RunProducer {
	p.gct = client
	return p
}

func (p *RunProducer) ExecuteValidation(ctx context.Context, current *validationv1.ValidationReceipt, intent *validationv1.ValidationIntent, transition TransitionFunc) error {
	if p == nil || p.runs == nil {
		return fmt.Errorf("suite run manager is unavailable")
	}
	receiptID := current.GetReceiptId()
	admitted, resolved := intent.GetExpectedIdentity(), false
	if current.GetAdmittedIdentity() != nil && current.GetState() != validationv1.ReceiptState_RECEIPT_STATE_QUEUED {
		admitted = current.GetAdmittedIdentity()
	}
	if p.identity != nil && !hasRunningChild(current) {
		observed, err := p.identity.Resolve(ctx, intent)
		if err != nil {
			return p.terminalizeResolutionFailure(ctx, receiptID, transition, err)
		}
		if observed.GetIdentity() != intent.GetExpectedIdentity().GetIdentity() {
			return p.terminalizeIdentityChange(ctx, receiptID, observed, transition, "content identity changed before producer work began")
		}
		admitted = observed
	}
	resolved = p.identity != nil
	state := current.GetState()
	if state == validationv1.ReceiptState_RECEIPT_STATE_ADMITTED || state == validationv1.ReceiptState_RECEIPT_STATE_RETRY_PENDING {
		state = validationv1.ReceiptState_RECEIPT_STATE_QUEUED
		var err error
		current, err = transition(ctx, receiptID, state, nil)
		if err != nil {
			return err
		}
	}
	updatedCurrent, err := transition(ctx, receiptID, state, func(receipt *validationv1.ValidationReceipt) error {
		receipt.AdmittedIdentity = cloneIdentity(admitted)
		return nil
	})
	if err != nil {
		return err
	}
	current = updatedCurrent
	if intent.GetPurpose() == validationv1.ValidationPurpose_VALIDATION_PURPOSE_EVIDENCE_PRODUCTION {
		pin := intent.GetPinnedEvidenceProducer()
		if pin == nil || pin.GetSourceIdentity() == "" {
			return p.terminalizeFailure(ctx, receiptID, transition, validationv1.ValidationReasonCode_VALIDATION_REASON_CODE_IDENTITY_CHANGED, "pinned producer source identity is unavailable")
		}
		// A durable child is the authority after admission. Re-resolving mutable
		// provider source here would make restart/reattachment depend on current
		// checkout state and could strand the original child.
		if childByID(current, evidenceProducerChildID(pin)) == nil {
			if p.identity == nil {
				return p.terminalizeFailure(ctx, receiptID, transition, validationv1.ValidationReasonCode_VALIDATION_REASON_CODE_IDENTITY_CHANGED, "pinned producer source identity is unavailable")
			}
			observedSource, sourceErr := p.identity.Resolve(ctx, producerSourceIntent(pin))
			if sourceErr != nil {
				return p.terminalizeResolutionFailure(ctx, receiptID, transition, sourceErr)
			}
			if observedSource.GetIdentity() != pin.GetSourceIdentity() {
				return p.terminalizeFailure(ctx, receiptID, transition, validationv1.ValidationReasonCode_VALIDATION_REASON_CODE_IDENTITY_CHANGED, "provider source changed after evidence producer admission")
			}
		}
		return p.executeEvidenceProduction(ctx, current, intent, transition)
	}
	var evidenceErr error
	current, evidenceErr = p.executeGCTEvidence(ctx, current, intent, transition)
	if evidenceErr != nil || terminal(current.GetState()) {
		return evidenceErr
	}
	if intent.GetPurpose() == validationv1.ValidationPurpose_VALIDATION_PURPOSE_REGRESSION_BEFORE {
		if missing := missingRequiredEvidence(current, intent); len(missing) > 0 {
			_, missingErr := p.failMissingEvidence(ctx, current, intent, transition, "required evidence missing: "+strings.Join(missing, ", "))
			return missingErr
		}
		if resolved {
			observed, err := p.identity.Resolve(ctx, intent)
			if err != nil {
				return p.terminalizeResolutionFailure(ctx, receiptID, transition, err)
			}
			if observed.GetIdentity() != admitted.GetIdentity() {
				return p.terminalizeIdentityChange(ctx, receiptID, observed, transition, "relevant inputs changed during behavioral-before capture")
			}
		}
		_, err = transition(ctx, receiptID, validationv1.ReceiptState_RECEIPT_STATE_SUCCEEDED, func(receipt *validationv1.ValidationReceipt) error {
			receipt.AchievedStrength = intent.GetRequiredStrength()
			receipt.ObservedIdentity = cloneIdentity(admitted)
			receipt.Retry = &validationv1.RetryDisposition{Kind: validationv1.RetryKind_RETRY_KIND_NOT_NEEDED, MaximumAttempts: intent.GetDeadlinePolicy().GetMaximumAttempts()}
			receipt.ReasonCode = validationv1.ValidationReasonCode_VALIDATION_REASON_CODE_NONE
			receipt.Detail = "required behavioral-before and source evidence succeeded"
			return nil
		})
		return err
	}
	for index, target := range intent.GetTargets() {
		if target.GetKind() != commonv1.ValidationTargetKind_VALIDATION_TARGET_KIND_SCENARIO {
			return p.terminalizeFailure(ctx, receiptID, transition, validationv1.ValidationReasonCode_VALIDATION_REASON_CODE_INVALID_TARGET, fmt.Sprintf("target %s:%s is not supported by the suite producer", target.GetKind(), target.GetId()))
		}
		scenario := strings.TrimSpace(target.GetId())
		prefix := fmt.Sprintf("scenario:%s:%d", scenario, index+1)
		child, priorAttempts := latestChild(current, prefix)
		if child != nil && child.GetState() == validationv1.ChildOperationState_CHILD_OPERATION_STATE_SUCCEEDED {
			continue
		}
		attempt := priorAttempts
		if attempt == 0 {
			attempt = 1
		} else if child.GetState() != validationv1.ChildOperationState_CHILD_OPERATION_STATE_RUNNING && child.GetState() != validationv1.ChildOperationState_CHILD_OPERATION_STATE_PENDING {
			attempt++
			child = nil
		}
		maximumAttempts := int(intent.GetDeadlinePolicy().GetMaximumAttempts())
		for ; attempt <= maximumAttempts; attempt++ {
			runID := ""
			if child != nil {
				runID = child.GetOperationId()
			} else {
				runID = validationSuiteRunID(receiptID, scenario, attempt)
				childID := fmt.Sprintf("%s:attempt:%d", prefix, attempt)
				updated, err := transition(ctx, receiptID, validationv1.ReceiptState_RECEIPT_STATE_RUNNING, func(receipt *validationv1.ValidationReceipt) error {
					receipt.Children = append(receipt.Children, &validationv1.ChildOperation{ChildId: childID, Kind: validationv1.ChildOperationKind_CHILD_OPERATION_KIND_TEST_RUN, State: validationv1.ChildOperationState_CHILD_OPERATION_STATE_PENDING, Owner: "test-genie", OperationId: runID})
					return nil
				})
				if err != nil {
					return err
				}
				current = updated
			}
			if child == nil || child.GetState() == validationv1.ChildOperationState_CHILD_OPERATION_STATE_PENDING {
				if _, err := p.startAfterCapacity(ctx, receiptID, scenario, intent, runID); err != nil {
					if errors.Is(err, errSuiteStartUncertain) {
						return fmt.Errorf("%w: %v", errEvidencePending, err)
					}
					reason := validationv1.ValidationReasonCode_VALIDATION_REASON_CODE_PROVIDER_UNAVAILABLE
					if errors.Is(err, errQueueBudget) {
						reason = validationv1.ValidationReasonCode_VALIDATION_REASON_CODE_CAPACITY_UNAVAILABLE
					}
					return p.terminalizeFailure(ctx, receiptID, transition, reason, fmt.Sprintf("start validation child for %s: %v", scenario, err))
				}
				updated, err := transition(ctx, receiptID, validationv1.ReceiptState_RECEIPT_STATE_RUNNING, func(receipt *validationv1.ValidationReceipt) error {
					setChildState(receipt, runID, validationv1.ChildOperationState_CHILD_OPERATION_STATE_RUNNING, "")
					return nil
				})
				if err != nil {
					return fmt.Errorf("%w: persist started suite %s: %v", errEvidencePending, runID, err)
				}
				current = updated
			}

			waitCtx, cancel := validationDeadline(ctx, intent)
			status, waitErr := p.runs.Wait(waitCtx, scenario, runID)
			cancel()
			if waitErr != nil {
				return fmt.Errorf("%w: suite %s attachment: %v", errEvidencePending, runID, waitErr)
			}
			childState := validationv1.ChildOperationState_CHILD_OPERATION_STATE_FAILED
			if status.Status == sharedruns.StatusPassed {
				childState = validationv1.ChildOperationState_CHILD_OPERATION_STATE_SUCCEEDED
			} else if status.Status == sharedruns.StatusAborted {
				childState = validationv1.ChildOperationState_CHILD_OPERATION_STATE_CANCELLED
			}
			updated, err := transition(ctx, receiptID, validationv1.ReceiptState_RECEIPT_STATE_RUNNING, func(receipt *validationv1.ValidationReceipt) error {
				setChildState(receipt, runID, childState, status.Error)
				if !hasEvidence(receipt, runID, "test-genie-run") {
					receipt.Evidence = append(receipt.Evidence, &validationv1.EvidenceReference{EvidenceId: runID, Kind: "test-genie-run", Owner: "test-genie", SubjectId: scenario, Uri: fmt.Sprintf("test-genie://runs/%s/%s", scenario, runID)})
				}
				return nil
			})
			if err != nil {
				return err
			}
			current = updated
			if resolved {
				observed, err := p.identity.Resolve(ctx, intent)
				if err != nil {
					return p.terminalizeResolutionFailure(ctx, receiptID, transition, err)
				}
				if observed.GetIdentity() != admitted.GetIdentity() {
					return p.terminalizeIdentityChange(ctx, receiptID, observed, transition, "relevant inputs changed while validation was running; evidence remains attributed to the admitted identity")
				}
			}
			if childState == validationv1.ChildOperationState_CHILD_OPERATION_STATE_CANCELLED {
				_, err := transition(ctx, receiptID, validationv1.ReceiptState_RECEIPT_STATE_CANCELLED, func(receipt *validationv1.ValidationReceipt) error {
					receipt.ReasonCode = validationv1.ValidationReasonCode_VALIDATION_REASON_CODE_ABORTED
					receipt.Detail = fmt.Sprintf("suite run %s for %s was aborted", runID, scenario)
					return nil
				})
				return err
			}
			if childState == validationv1.ChildOperationState_CHILD_OPERATION_STATE_SUCCEEDED {
				break
			}
			if attempt == maximumAttempts {
				_, err := transition(ctx, receiptID, validationv1.ReceiptState_RECEIPT_STATE_FAILED, func(receipt *validationv1.ValidationReceipt) error {
					receipt.Retry = &validationv1.RetryDisposition{Kind: validationv1.RetryKind_RETRY_KIND_EXHAUSTED, Attempt: uint32(attempt), MaximumAttempts: uint32(maximumAttempts), ReasonCode: validationv1.ValidationReasonCode_VALIDATION_REASON_CODE_RETRY_EXHAUSTED}
					receipt.ReasonCode = validationv1.ValidationReasonCode_VALIDATION_REASON_CODE_REQUIRED_EVIDENCE_FAILED
					receipt.Detail = fmt.Sprintf("suite run %s for %s ended %s after %d attempt(s)", runID, scenario, status.Status, attempt)
					return nil
				})
				return err
			}
			updated, err = transition(ctx, receiptID, validationv1.ReceiptState_RECEIPT_STATE_RETRY_PENDING, func(receipt *validationv1.ValidationReceipt) error {
				receipt.Retry = &validationv1.RetryDisposition{Kind: validationv1.RetryKind_RETRY_KIND_SCHEDULED, Attempt: uint32(attempt), MaximumAttempts: uint32(maximumAttempts), ReasonCode: validationv1.ValidationReasonCode_VALIDATION_REASON_CODE_REQUIRED_EVIDENCE_FAILED, RetryAt: timestamppb.Now()}
				return nil
			})
			if err != nil {
				return err
			}
			updated, err = transition(ctx, receiptID, validationv1.ReceiptState_RECEIPT_STATE_QUEUED, nil)
			if err != nil {
				return err
			}
			current, child = updated, nil
		}
	}
	if missing := missingRequiredEvidence(current, intent); len(missing) > 0 {
		_, missingErr := p.failMissingEvidence(ctx, current, intent, transition, "required evidence missing: "+strings.Join(missing, ", "))
		return missingErr
	}
	_, err = transition(ctx, receiptID, validationv1.ReceiptState_RECEIPT_STATE_SUCCEEDED, func(receipt *validationv1.ValidationReceipt) error {
		receipt.AchievedStrength = intent.GetRequiredStrength()
		receipt.ObservedIdentity = cloneIdentity(admitted)
		receipt.Retry = &validationv1.RetryDisposition{Kind: validationv1.RetryKind_RETRY_KIND_NOT_NEEDED, MaximumAttempts: intent.GetDeadlinePolicy().GetMaximumAttempts()}
		receipt.ReasonCode = validationv1.ValidationReasonCode_VALIDATION_REASON_CODE_NONE
		receipt.Detail = "all required validation children succeeded"
		return nil
	})
	return err
}

func (p *RunProducer) executeEvidenceProduction(ctx context.Context, current *validationv1.ValidationReceipt, intent *validationv1.ValidationIntent, transition TransitionFunc) error {
	pin := intent.GetPinnedEvidenceProducer()
	target := intent.GetTargets()[0].GetId()
	childID := evidenceProducerChildID(pin)
	runID := validationSuiteRunID(current.GetReceiptId(), target, 1)
	if current.GetCreatedAt() == nil || !current.GetCreatedAt().IsValid() || pin.GetTimeoutMilliseconds() > uint64(int64(^uint64(0)>>1)/int64(time.Millisecond)) {
		return p.terminalizeFailure(ctx, current.GetReceiptId(), transition, validationv1.ValidationReasonCode_VALIDATION_REASON_CODE_INVALID_INTENT, "pinned evidence producer deadline is invalid")
	}
	deadline, deadlineErr := evidenceProductionDeadline(current.GetCreatedAt().AsTime(), time.Duration(pin.GetTimeoutMilliseconds())*time.Millisecond, evidenceProducerCleanupReserve)
	if deadlineErr != nil {
		return p.terminalizeFailure(ctx, current.GetReceiptId(), transition, validationv1.ValidationReasonCode_VALIDATION_REASON_CODE_INVALID_INTENT, deadlineErr.Error())
	}
	child := childByID(current, childID)
	if !time.Now().Before(deadline) {
		return p.expireEvidenceProduction(ctx, current, intent, transition, deadline)
	}
	if child != nil && child.GetState() == validationv1.ChildOperationState_CHILD_OPERATION_STATE_SUCCEEDED {
		observed, changed, identityErr := p.verifyEvidenceProductionIdentity(ctx, intent, pin, current.GetAdmittedIdentity())
		if identityErr != nil {
			if changed {
				return p.terminalizeIdentityChange(ctx, current.GetReceiptId(), observed, transition, identityErr.Error())
			}
			return p.terminalizeResolutionFailure(ctx, current.GetReceiptId(), transition, identityErr)
		}
		set, setErr := p.producedEvidenceSet(target, child.GetOperationId(), current, pin, observed)
		if setErr != nil {
			return p.terminalizeFailure(ctx, current.GetReceiptId(), transition, validationv1.ValidationReasonCode_VALIDATION_REASON_CODE_PROVIDER_UNAVAILABLE, "producer output could not be published as retained evidence: "+setErr.Error())
		}
		if !time.Now().Before(deadline) {
			return p.expireEvidenceProduction(ctx, current, intent, transition, deadline)
		}
		_, err := transition(ctx, current.GetReceiptId(), validationv1.ReceiptState_RECEIPT_STATE_SUCCEEDED, func(r *validationv1.ValidationReceipt) error {
			r.ObservedIdentity = cloneIdentity(observed)
			r.ProducedEvidenceSet = set
			r.Detail = "declared evidence producer command completed"
			return nil
		})
		return err
	}
	if child == nil {
		updated, err := transition(ctx, current.GetReceiptId(), validationv1.ReceiptState_RECEIPT_STATE_RUNNING, func(r *validationv1.ValidationReceipt) error {
			r.Children = append(r.Children, &validationv1.ChildOperation{ChildId: childID, Kind: validationv1.ChildOperationKind_CHILD_OPERATION_KIND_TEST_RUN, State: validationv1.ChildOperationState_CHILD_OPERATION_STATE_PENDING, Owner: "test-genie", OperationId: runID})
			return nil
		})
		if err != nil {
			return err
		}
		current = updated
	} else {
		runID = child.GetOperationId()
	}
	if !time.Now().Before(deadline) {
		return p.expireEvidenceProduction(ctx, current, intent, transition, deadline)
	}
	options, err := evidenceProducerStartOptions(current, intent)
	if err != nil {
		return p.terminalizeFailure(ctx, current.GetReceiptId(), transition, validationv1.ValidationReasonCode_VALIDATION_REASON_CODE_INVALID_INTENT, err.Error())
	}
	if _, err := p.runs.Start(options); err != nil {
		var refused *runmanager.AdmissionRefusedError
		if !errors.As(err, &refused) {
			return fmt.Errorf("%w: reconcile producer run start: %v", errEvidencePending, err)
		}
		return p.terminalizeFailure(ctx, current.GetReceiptId(), transition, validationv1.ValidationReasonCode_VALIDATION_REASON_CODE_PROVIDER_UNAVAILABLE, "producer run admission refused: "+err.Error())
	}
	updated, err := transition(ctx, current.GetReceiptId(), validationv1.ReceiptState_RECEIPT_STATE_RUNNING, func(r *validationv1.ValidationReceipt) error {
		setChildState(r, runID, validationv1.ChildOperationState_CHILD_OPERATION_STATE_RUNNING, "")
		return nil
	})
	if err != nil {
		return fmt.Errorf("%w: persist producer run start: %v", errEvidencePending, err)
	}
	if !time.Now().Before(deadline) {
		return p.expireEvidenceProduction(ctx, updated, intent, transition, deadline)
	}
	waitCtx, cancel := context.WithDeadline(ctx, deadline)
	defer cancel()
	status, err := p.runs.Wait(waitCtx, pin.GetProvider(), runID)
	if err != nil {
		if !time.Now().Before(deadline) {
			return p.expireEvidenceProduction(ctx, updated, intent, transition, deadline)
		}
		return fmt.Errorf("%w: observe producer run: %v", errEvidencePending, err)
	}
	if !time.Now().Before(deadline) {
		return p.expireEvidenceProduction(ctx, updated, intent, transition, deadline)
	}
	if waitCtx.Err() != nil {
		return fmt.Errorf("%w: producer wait returned after its observation context was cancelled: %v", errEvidencePending, waitCtx.Err())
	}
	if status.Status != sharedruns.StatusPassed {
		state := validationv1.ReceiptState_RECEIPT_STATE_FAILED
		childState := validationv1.ChildOperationState_CHILD_OPERATION_STATE_FAILED
		reason := validationv1.ValidationReasonCode_VALIDATION_REASON_CODE_PROVIDER_UNAVAILABLE
		if status.Status == sharedruns.StatusAborted {
			state = validationv1.ReceiptState_RECEIPT_STATE_CANCELLED
			childState = validationv1.ChildOperationState_CHILD_OPERATION_STATE_CANCELLED
			reason = validationv1.ValidationReasonCode_VALIDATION_REASON_CODE_ABORTED
		}
		_, err := transition(ctx, updated.GetReceiptId(), state, func(r *validationv1.ValidationReceipt) error {
			setChildState(r, runID, childState, status.Error)
			r.ReasonCode = reason
			r.Detail = "declared evidence producer command ended " + status.Status
			return nil
		})
		if err != nil {
			return fmt.Errorf("%w: persist producer outcome: %w", errEvidencePending, err)
		}
		return nil
	}
	updated, err = transition(ctx, updated.GetReceiptId(), validationv1.ReceiptState_RECEIPT_STATE_RUNNING, func(r *validationv1.ValidationReceipt) error {
		setChildState(r, runID, validationv1.ChildOperationState_CHILD_OPERATION_STATE_SUCCEEDED, "command completed")
		if !hasEvidence(r, runID, "test-genie-run") {
			r.Evidence = append(r.Evidence, &validationv1.EvidenceReference{EvidenceId: runID, Kind: "test-genie-run", Owner: "test-genie", SubjectId: pin.GetProvider(), Uri: fmt.Sprintf("test-genie://runs/%s/%s", pin.GetProvider(), runID)})
		}
		return nil
	})
	if err != nil {
		return err
	}
	if !time.Now().Before(deadline) {
		return p.expireEvidenceProduction(ctx, updated, intent, transition, deadline)
	}
	observedCandidate, changed, identityErr := p.verifyEvidenceProductionIdentity(ctx, intent, pin, updated.GetAdmittedIdentity())
	if identityErr != nil {
		if changed {
			return p.terminalizeIdentityChange(ctx, updated.GetReceiptId(), observedCandidate, transition, identityErr.Error())
		}
		return p.terminalizeResolutionFailure(ctx, updated.GetReceiptId(), transition, identityErr)
	}
	set, setErr := p.producedEvidenceSet(target, runID, updated, pin, observedCandidate)
	if setErr != nil {
		return p.terminalizeFailure(ctx, updated.GetReceiptId(), transition, validationv1.ValidationReasonCode_VALIDATION_REASON_CODE_PROVIDER_UNAVAILABLE, "producer output could not be published as retained evidence: "+setErr.Error())
	}
	if !time.Now().Before(deadline) {
		return p.expireEvidenceProduction(ctx, updated, intent, transition, deadline)
	}
	_, err = transition(ctx, updated.GetReceiptId(), validationv1.ReceiptState_RECEIPT_STATE_SUCCEEDED, func(r *validationv1.ValidationReceipt) error {
		r.ObservedIdentity = cloneIdentity(observedCandidate)
		r.ProducedEvidenceSet = set
		r.ReasonCode = validationv1.ValidationReasonCode_VALIDATION_REASON_CODE_NONE
		r.Detail = "declared evidence producer command completed; provider validation remains separate"
		r.Retry = &validationv1.RetryDisposition{Kind: validationv1.RetryKind_RETRY_KIND_NOT_NEEDED, MaximumAttempts: 1}
		return nil
	})
	return err
}

func (p *RunProducer) expireEvidenceProduction(ctx context.Context, current *validationv1.ValidationReceipt, intent *validationv1.ValidationIntent, transition TransitionFunc, deadline time.Time) error {
	childID := evidenceProducerChildID(intent.GetPinnedEvidenceProducer())
	child := childByID(current, childID)
	if child == nil {
		return p.terminalizeFailure(ctx, current.GetReceiptId(), transition, validationv1.ValidationReasonCode_VALIDATION_REASON_CODE_DEADLINE_EXCEEDED, "evidence producer admission expired before a child was launched")
	}
	if err := p.AbortEvidenceValidation(ctx, current, intent, "producer observation deadline expired", "test-genie"); err != nil {
		return fmt.Errorf("%w: expired producer child %s could not be confirmed drained by %s: %v", errEvidencePending, child.GetOperationId(), deadline.UTC().Format(time.RFC3339Nano), err)
	}
	drained := childByID(current, childID)
	_, err := transition(ctx, current.GetReceiptId(), validationv1.ReceiptState_RECEIPT_STATE_FAILED, func(receipt *validationv1.ValidationReceipt) error {
		if drained != nil {
			setChildState(receipt, drained.GetOperationId(), drained.GetState(), drained.GetDetail())
			for _, stored := range receipt.GetChildren() {
				if stored.GetChildId() == childID {
					stored.ReasonCode = drained.GetReasonCode()
					break
				}
			}
		}
		receipt.ReasonCode = validationv1.ValidationReasonCode_VALIDATION_REASON_CODE_DEADLINE_EXCEEDED
		receipt.Detail = "evidence producer observation deadline expired; child was drained without admitting late output"
		receipt.ProducedEvidenceSet = nil
		return nil
	})
	if err != nil {
		return fmt.Errorf("%w: persist expired producer outcome: %w", errEvidencePending, err)
	}
	return nil
}

func (p *RunProducer) producedEvidenceSet(target, runID string, receipt *validationv1.ValidationReceipt, pin *validationv1.PinnedEvidenceProducer, identity *validationv1.SourceIdentity) (*scenariovalidationv1.RetainedEvidenceSet, error) {
	owner, ok := p.runs.(retainedEvidenceSetOwner)
	if !ok {
		return nil, errors.New("run manager does not expose owner-published catalog evidence")
	}
	if identity == nil || strings.TrimSpace(identity.GetIdentity()) == "" {
		return nil, errors.New("candidate identity is unavailable")
	}
	set, err := owner.RetainedEvidenceSet(target, runID, receipt.GetReceiptId(), pin.GetProducer(), identity.GetIdentity())
	if err != nil {
		return nil, err
	}
	if set.GetProducerReceiptId() != receipt.GetReceiptId() || set.GetProducer() != pin.GetProducer() || set.GetTarget() != target || set.GetRunId() != runID || set.GetCandidateIdentity() != identity.GetIdentity() {
		return nil, errors.New("run manager returned mismatched retained evidence identity")
	}
	return set, nil
}

func evidenceProducerStartOptions(current *validationv1.ValidationReceipt, intent *validationv1.ValidationIntent) (runmanager.StartOptions, error) {
	pin := intent.GetPinnedEvidenceProducer()
	if pin == nil || len(intent.GetTargets()) != 1 {
		return runmanager.StartOptions{}, errors.New("pinned evidence producer input is incomplete")
	}
	target := intent.GetTargets()[0].GetId()
	runID := validationSuiteRunID(current.GetReceiptId(), target, 1)
	if child := childByID(current, evidenceProducerChildID(pin)); child != nil {
		runID = child.GetOperationId()
	}
	argv := append([]string(nil), pin.GetArgv()...)
	for index, arg := range argv {
		argv[index] = strings.ReplaceAll(strings.ReplaceAll(arg, "{run_id}", runID), "{output_dir}", "/dev/shm/tg-output")
		if strings.Contains(argv[index], "{") || strings.Contains(argv[index], "}") {
			return runmanager.StartOptions{}, errors.New("pinned producer has unresolved argument placeholder")
		}
	}
	request := orchestrator.SuiteExecutionRequest{ScenarioName: pin.GetProvider(), RunID: runID, ValidationRun: true, RetainForEvidence: true, RetentionReason: "declared evidence producer receipt " + current.GetReceiptId()}
	input := execution.SuiteExecutionInput{Request: request, EvidenceProducer: &execution.EvidenceProducerCommand{Provider: pin.GetProvider(), Name: pin.GetProducer(), Argv: argv, WorkingDirectory: pin.GetWorkingDirectory(), OutputRoot: pin.GetOutputRoot(), Timeout: time.Duration(pin.GetTimeoutMilliseconds()) * time.Millisecond, MaximumOutputBytes: int64(pin.GetMaximumOutputBytes()), MutatesLifecycle: pin.GetMutatesLifecycle(), DescriptorDigest: pin.GetDescriptorDigest(), SourceIdentity: pin.GetSourceIdentity()}}
	return runmanager.StartOptions{Input: input}, nil
}

func evidenceProducerChildID(pin *validationv1.PinnedEvidenceProducer) string {
	return "scenario:" + pin.GetProvider() + ":evidence:" + pin.GetProducer()
}

func (p *RunProducer) verifyEvidenceProductionIdentity(ctx context.Context, intent *validationv1.ValidationIntent, pin *validationv1.PinnedEvidenceProducer, admitted *validationv1.SourceIdentity) (*validationv1.SourceIdentity, bool, error) {
	if p.identity == nil {
		return nil, false, errors.New("evidence producer identity resolver is unavailable at completion")
	}
	observedCandidate, err := p.identity.Resolve(ctx, intent)
	if err != nil {
		return nil, false, err
	}
	if observedCandidate.GetIdentity() != admitted.GetIdentity() {
		return observedCandidate, true, fmt.Errorf("candidate identity changed during evidence production: admitted %s, observed %s", admitted.GetIdentity(), observedCandidate.GetIdentity())
	}
	observedProvider, err := p.identity.Resolve(ctx, producerSourceIntent(pin))
	if err != nil {
		return observedCandidate, false, err
	}
	if observedProvider.GetIdentity() != pin.GetSourceIdentity() {
		return observedCandidate, true, fmt.Errorf("provider source identity changed during evidence production: pinned %s, observed %s", pin.GetSourceIdentity(), observedProvider.GetIdentity())
	}
	return observedCandidate, false, nil
}

func (p *RunProducer) executeGCTEvidence(ctx context.Context, current *validationv1.ValidationReceipt, intent *validationv1.ValidationIntent, transition TransitionFunc) (*validationv1.ValidationReceipt, error) {
	policy := intent.GetEvidencePolicy()
	if policy == nil || (!policy.GetRequireBehavioralBefore() && !policy.GetRequireSourceSnapshot()) {
		return current, nil
	}
	if p.gct == nil {
		return p.failMissingEvidence(ctx, current, intent, transition, "git-control-tower evidence adapter is unavailable")
	}
	collection := strings.TrimSpace(intent.GetBehavioralPrior())
	if collection == "" {
		return p.failMissingEvidence(ctx, current, intent, transition, "behavioral prior is required for GCT evidence")
	}
	targets := make([]string, 0, len(intent.GetTargets()))
	for _, target := range intent.GetTargets() {
		if target.GetKind() == commonv1.ValidationTargetKind_VALIDATION_TARGET_KIND_SCENARIO && strings.TrimSpace(target.GetId()) != "" {
			targets = append(targets, strings.TrimSpace(target.GetId()))
		}
	}
	if intent.GetPurpose() == validationv1.ValidationPurpose_VALIDATION_PURPOSE_REGRESSION_BEFORE {
		req := GCTCaptureRequest{Collection: collection, ParentReceiptID: current.GetReceiptId(), Targets: targets, Paths: contentInputPaths(intent), Actor: intent.GetCallerScenario()}
		return p.driveGCTChild(ctx, current, intent, transition, validationv1.ChildOperationKind_CHILD_OPERATION_KIND_GCT_BASELINE_COLLECTION, "gct-capture:"+collection, collection,
			func(callCtx context.Context) (string, error) { return p.gct.StartCapture(callCtx, req) },
			func(callCtx context.Context) (GCTEvidenceResult, error) { return p.gct.WaitCapture(callCtx, req) })
	}
	operationID := "receipt-" + current.GetReceiptId()
	req := GCTDiffRequest{Collection: collection, ParentReceiptID: current.GetReceiptId(), OperationID: operationID, Targets: targets, RequireSourceSnapshot: policy.GetRequireSourceSnapshot()}
	return p.driveGCTChild(ctx, current, intent, transition, validationv1.ChildOperationKind_CHILD_OPERATION_KIND_GCT_COLLECTION_DIFF, "gct-diff:"+collection, operationID,
		func(callCtx context.Context) (string, error) { return p.gct.StartDiff(callCtx, req) },
		func(callCtx context.Context) (GCTEvidenceResult, error) { return p.gct.WaitDiff(callCtx, req) })
}

func (p *RunProducer) driveGCTChild(ctx context.Context, current *validationv1.ValidationReceipt, intent *validationv1.ValidationIntent, transition TransitionFunc, kind validationv1.ChildOperationKind, childID, expectedOperationID string, start func(context.Context) (string, error), wait func(context.Context) (GCTEvidenceResult, error)) (*validationv1.ValidationReceipt, error) {
	operationID := expectedOperationID
	child := childByID(current, childID)
	if child != nil && child.GetState() == validationv1.ChildOperationState_CHILD_OPERATION_STATE_SUCCEEDED {
		return current, nil
	}
	if child == nil {
		started, err := start(ctx)
		if err != nil {
			return p.failMissingEvidence(ctx, current, intent, transition, err.Error())
		}
		operationID = started
		updated, err := transition(ctx, current.GetReceiptId(), validationv1.ReceiptState_RECEIPT_STATE_RUNNING, func(receipt *validationv1.ValidationReceipt) error {
			receipt.Children = append(receipt.Children, &validationv1.ChildOperation{ChildId: childID, Kind: kind, State: validationv1.ChildOperationState_CHILD_OPERATION_STATE_RUNNING, Owner: "git-control-tower", OperationId: operationID})
			return nil
		})
		if err != nil {
			return current, err
		}
		current = updated
	} else {
		operationID = child.GetOperationId()
	}
	waitCtx, cancel := validationDeadline(ctx, intent)
	result, err := wait(waitCtx)
	cancel()
	if err != nil {
		return current, fmt.Errorf("%w: GCT %s attachment: %v", errEvidencePending, operationID, err)
	}
	state := validationv1.ChildOperationState_CHILD_OPERATION_STATE_FAILED
	if result.Passed {
		state = validationv1.ChildOperationState_CHILD_OPERATION_STATE_SUCCEEDED
	}
	updated, err := transition(ctx, current.GetReceiptId(), validationv1.ReceiptState_RECEIPT_STATE_RUNNING, func(receipt *validationv1.ValidationReceipt) error {
		setChildState(receipt, operationID, state, result.Detail)
		for _, evidence := range result.Evidence {
			if evidence != nil && !hasEvidence(receipt, evidence.GetEvidenceId(), evidence.GetKind()) {
				receipt.Evidence = append(receipt.Evidence, evidence)
			}
			if evidence != nil && evidence.GetKind() == "gct-source-snapshot" && childByID(receipt, "gct-source:"+evidence.GetEvidenceId()) == nil {
				receipt.Children = append(receipt.Children, &validationv1.ChildOperation{ChildId: "gct-source:" + evidence.GetEvidenceId(), Kind: validationv1.ChildOperationKind_CHILD_OPERATION_KIND_SOURCE_SNAPSHOT, State: state, Owner: "git-control-tower", OperationId: evidence.GetEvidenceId(), Detail: result.Detail})
			}
		}
		return nil
	})
	if err != nil {
		return current, err
	}
	if !result.Passed {
		return p.failMissingEvidence(ctx, updated, intent, transition, result.Detail)
	}
	return updated, nil
}

func (p *RunProducer) failMissingEvidence(ctx context.Context, current *validationv1.ValidationReceipt, intent *validationv1.ValidationIntent, transition TransitionFunc, detail string) (*validationv1.ValidationReceipt, error) {
	if intent.GetEvidencePolicy().GetAllowAuthorizedDegradation() {
		updated, err := transition(ctx, current.GetReceiptId(), validationv1.ReceiptState_RECEIPT_STATE_DEGRADED, func(receipt *validationv1.ValidationReceipt) error {
			receipt.ReasonCode = validationv1.ValidationReasonCode_VALIDATION_REASON_CODE_REQUIRED_EVIDENCE_MISSING
			receipt.Detail = detail
			receipt.Degradation = &validationv1.Degradation{Authorized: true, AuthorizedBy: intent.GetCallerScenario(), AuthorizationReason: "intent explicitly authorizes missing GCT evidence", MissingEvidenceKinds: []string{"git-control-tower"}, ReasonCode: validationv1.ValidationReasonCode_VALIDATION_REASON_CODE_REQUIRED_EVIDENCE_MISSING}
			return nil
		})
		return updated, err
	}
	updated, err := transition(ctx, current.GetReceiptId(), validationv1.ReceiptState_RECEIPT_STATE_FAILED, func(receipt *validationv1.ValidationReceipt) error {
		receipt.ReasonCode = validationv1.ValidationReasonCode_VALIDATION_REASON_CODE_REQUIRED_EVIDENCE_MISSING
		receipt.Detail = detail
		return nil
	})
	return updated, err
}

func childByID(receipt *validationv1.ValidationReceipt, childID string) *validationv1.ChildOperation {
	for _, child := range receipt.GetChildren() {
		if child.GetChildId() == childID {
			return child
		}
	}
	return nil
}

func hasRunningChild(receipt *validationv1.ValidationReceipt) bool {
	for _, child := range receipt.GetChildren() {
		if child.GetState() == validationv1.ChildOperationState_CHILD_OPERATION_STATE_RUNNING || child.GetState() == validationv1.ChildOperationState_CHILD_OPERATION_STATE_PENDING {
			return true
		}
	}
	return false
}

func missingRequiredEvidence(receipt *validationv1.ValidationReceipt, intent *validationv1.ValidationIntent) []string {
	present := make(map[string]struct{}, len(receipt.GetEvidence()))
	for _, evidence := range receipt.GetEvidence() {
		present[evidence.GetKind()] = struct{}{}
	}
	var missing []string
	for _, kind := range intent.GetEvidencePolicy().GetRequiredEvidenceKinds() {
		if _, ok := present[kind]; !ok {
			missing = append(missing, kind)
		}
	}
	return missing
}

func contentInputPaths(intent *validationv1.ValidationIntent) []string {
	var paths []string
	for _, root := range intent.GetContentInputs() {
		if root.GetRoot() != "." {
			continue
		}
		for _, selection := range root.GetSelections() {
			if strings.TrimSpace(selection.GetGlob()) != "" {
				paths = append(paths, strings.TrimSpace(selection.GetGlob()))
			}
		}
	}
	return paths
}

func validationSuiteRunID(receiptID, scenario string, attempt int) string {
	digest := sha256.Sum256([]byte(fmt.Sprintf("%s\x00%s\x00%d", receiptID, scenario, attempt)))
	return fmt.Sprintf("validation-%x", digest[:16])
}

func cloneIntentEvidenceSets(sets []*scenariovalidationv1.RetainedEvidenceSet) []*scenariovalidationv1.RetainedEvidenceSet {
	cloned := make([]*scenariovalidationv1.RetainedEvidenceSet, 0, len(sets))
	for _, set := range sets {
		if set != nil {
			cloned = append(cloned, proto.Clone(set).(*scenariovalidationv1.RetainedEvidenceSet))
		}
	}
	return cloned
}

func (p *RunProducer) startAfterCapacity(ctx context.Context, receiptID, scenario string, intent *validationv1.ValidationIntent, runID string) (runmanager.StartResult, error) {
	queueCtx, cancel := queueDeadline(ctx, intent)
	defer cancel()
	for {
		request, _, err := resolvedSuiteRequest(queueCtx, p.planner, scenario, intent)
		if err != nil {
			return runmanager.StartResult{}, err
		}
		request.RetainForEvidence = true
		request.RetentionReason = "validation receipt " + receiptID
		request.RunID = runID
		request.RetainedEvidenceSets = cloneIntentEvidenceSets(intent.GetRetainedEvidenceSets())
		result, err := p.runs.Start(runmanager.StartOptions{
			Input:  execution.SuiteExecutionInput{Request: request},
			Caller: intent.GetCallerScenario(),
		})
		if err == nil {
			if result.RunID != runID {
				return runmanager.StartResult{}, fmt.Errorf("%w: explicit suite run returned %q, expected %q", errSuiteStartUncertain, result.RunID, runID)
			}
			return result, nil
		}
		var busy *runmanager.BusyError
		if !errors.As(err, &busy) {
			var refused *runmanager.AdmissionRefusedError
			if errors.As(err, &refused) {
				// Refusal is permanent for this request, but an earlier lost
				// response may belong to live work. Settle only after observing it.
				observeCtx, cancel := validationDeadline(ctx, intent)
				prior, observeErr := p.runs.Wait(observeCtx, scenario, runID)
				cancel()
				if errors.Is(observeErr, sharedruns.ErrRunNotFound) || (observeErr == nil && suiteTerminal(prior.Status)) {
					return runmanager.StartResult{}, err
				}
			}
			return runmanager.StartResult{}, fmt.Errorf("%w: %v", errSuiteStartUncertain, err)
		}
		if _, waitErr := p.runs.Wait(queueCtx, busy.Scenario, busy.RunID); waitErr != nil {
			return runmanager.StartResult{}, fmt.Errorf("%w behind %s/%s: %v", errQueueBudget, busy.Scenario, busy.RunID, waitErr)
		}
	}
}

// suiteRequest is shared by admission planning and actual execution. Retention
// metadata belongs to the receipt, not to validation configuration identity.
func suiteRequest(scenario string, intent *validationv1.ValidationIntent) orchestrator.SuiteExecutionRequest {
	return orchestrator.SuiteExecutionRequest{
		ScenarioName:       scenario,
		Target:             "scenario:" + scenario,
		Preset:             presetForStrength(intent.GetRequiredStrength()),
		Phases:             append([]string(nil), intent.GetPhases()...),
		RequireGateQuality: intent.GetPurpose() == validationv1.ValidationPurpose_VALIDATION_PURPOSE_CERTIFICATION,
		ValidationRun:      true,
	}
}

func resolvedSuiteRequest(ctx context.Context, planner execution.ExecutionPlanner, scenario string, intent *validationv1.ValidationIntent) (orchestrator.SuiteExecutionRequest, *execution.ExecutionPlanPreview, error) {
	request := suiteRequest(scenario, intent)
	if planner == nil {
		return request, nil, nil
	}
	preview, err := planner.Preview(ctx, request)
	if err != nil {
		return request, nil, err
	}
	if preview == nil || len(preview.Phases) == 0 {
		return request, nil, fmt.Errorf("suite planner returned no runnable phases for %s", scenario)
	}
	for _, phase := range preview.Phases {
		request.ResolvedPhases = append(request.ResolvedPhases, phase.Name)
	}
	return request, preview, nil
}

func latestChild(receipt *validationv1.ValidationReceipt, prefix string) (*validationv1.ChildOperation, int) {
	var latest *validationv1.ChildOperation
	count := 0
	for _, child := range receipt.GetChildren() {
		if child.GetChildId() == prefix || strings.HasPrefix(child.GetChildId(), prefix+":attempt:") {
			latest = child
			count++
		}
	}
	return latest, count
}

func hasEvidence(receipt *validationv1.ValidationReceipt, evidenceID, kind string) bool {
	for _, evidence := range receipt.GetEvidence() {
		if evidence.GetEvidenceId() == evidenceID && evidence.GetKind() == kind {
			return true
		}
	}
	return false
}

func (p *RunProducer) terminalizeResolutionFailure(ctx context.Context, receiptID string, transition TransitionFunc, cause error) error {
	_, err := transition(ctx, receiptID, validationv1.ReceiptState_RECEIPT_STATE_FAILED, func(receipt *validationv1.ValidationReceipt) error {
		receipt.ReasonCode = validationv1.ValidationReasonCode_VALIDATION_REASON_CODE_UNRESOLVED_INPUT
		receipt.Detail = cause.Error()
		return nil
	})
	return err
}

func (p *RunProducer) terminalizeIdentityChange(ctx context.Context, receiptID string, observed *validationv1.SourceIdentity, transition TransitionFunc, detail string) error {
	_, err := transition(ctx, receiptID, validationv1.ReceiptState_RECEIPT_STATE_FAILED, func(receipt *validationv1.ValidationReceipt) error {
		receipt.ObservedIdentity = cloneIdentity(observed)
		receipt.ReasonCode = validationv1.ValidationReasonCode_VALIDATION_REASON_CODE_IDENTITY_CHANGED
		receipt.Detail = detail
		return nil
	})
	return err
}

func (p *RunProducer) terminalizeFailure(ctx context.Context, receiptID string, transition TransitionFunc, reason validationv1.ValidationReasonCode, detail string) error {
	_, err := transition(ctx, receiptID, validationv1.ReceiptState_RECEIPT_STATE_FAILED, func(receipt *validationv1.ValidationReceipt) error {
		receipt.ReasonCode = reason
		receipt.Detail = detail
		return nil
	})
	return err
}

// AbortValidation stops every still-active suite child. The receipt service
// owns the subsequent durable CANCELLED transition.
func (p *RunProducer) AbortValidation(ctx context.Context, receipt *validationv1.ValidationReceipt, _, _ string) error {
	return p.abortValidation(ctx, receipt, nil)
}

func (p *RunProducer) AbortEvidenceValidation(ctx context.Context, receipt *validationv1.ValidationReceipt, intent *validationv1.ValidationIntent, _, _ string) error {
	if intent == nil || intent.GetPurpose() != validationv1.ValidationPurpose_VALIDATION_PURPOSE_EVIDENCE_PRODUCTION || intent.GetPinnedEvidenceProducer() == nil {
		return errors.New("pinned evidence producer intent is unavailable during abort")
	}
	if p == nil || p.runs == nil {
		return errors.New("suite run manager is unavailable")
	}
	options, err := evidenceProducerStartOptions(receipt, intent)
	if err != nil {
		return err
	}
	status, err := p.runs.AbortEvidenceProducer(options)
	if err != nil {
		return fmt.Errorf("abort pinned evidence producer %s: %w", options.Input.Request.RunID, err)
	}
	if !suiteTerminal(status.Status) {
		return fmt.Errorf("evidence producer %s has not settled after abort: %s", options.Input.Request.RunID, status.Status)
	}
	producerChildID := evidenceProducerChildID(intent.GetPinnedEvidenceProducer())
	for _, child := range receipt.GetChildren() {
		if child.GetChildId() == producerChildID && (child.GetState() == validationv1.ChildOperationState_CHILD_OPERATION_STATE_RUNNING || child.GetState() == validationv1.ChildOperationState_CHILD_OPERATION_STATE_PENDING) {
			child.State, child.Detail, child.ReasonCode = suiteChildTerminalState(status.Status), status.Error, childReasonCode(status.Status)
		}
	}
	for _, child := range receipt.GetChildren() {
		if child.GetChildId() == producerChildID || child.GetKind() != validationv1.ChildOperationKind_CHILD_OPERATION_KIND_TEST_RUN || (child.GetState() != validationv1.ChildOperationState_CHILD_OPERATION_STATE_RUNNING && child.GetState() != validationv1.ChildOperationState_CHILD_OPERATION_STATE_PENDING) {
			continue
		}
		parts := strings.Split(child.GetChildId(), ":")
		if len(parts) < 3 || parts[0] != "scenario" {
			return fmt.Errorf("validation child %q has no scenario identity", child.GetChildId())
		}
		settled, abortErr := p.runs.Abort(parts[1], child.GetOperationId())
		if abortErr != nil {
			return fmt.Errorf("abort validation child %s: %w", child.GetOperationId(), abortErr)
		}
		if !suiteTerminal(settled.Status) {
			return fmt.Errorf("validation child %s has not settled after abort: %s", child.GetOperationId(), settled.Status)
		}
		child.State, child.Detail, child.ReasonCode = suiteChildTerminalState(settled.Status), settled.Error, childReasonCode(settled.Status)
	}
	return nil
}

func (p *RunProducer) abortValidation(ctx context.Context, receipt *validationv1.ValidationReceipt, intent *validationv1.ValidationIntent) error {
	if p == nil || p.runs == nil {
		return fmt.Errorf("suite run manager is unavailable")
	}
	for _, child := range receipt.GetChildren() {
		if child.GetKind() != validationv1.ChildOperationKind_CHILD_OPERATION_KIND_TEST_RUN || (child.GetState() != validationv1.ChildOperationState_CHILD_OPERATION_STATE_RUNNING && child.GetState() != validationv1.ChildOperationState_CHILD_OPERATION_STATE_PENDING) {
			continue
		}
		parts := strings.Split(child.GetChildId(), ":")
		if len(parts) < 3 || parts[0] != "scenario" {
			return fmt.Errorf("validation child %q has no scenario identity", child.GetChildId())
		}
		var status runmanager.LiveStatus
		var err error
		if intent != nil && child.GetChildId() == evidenceProducerChildID(intent.GetPinnedEvidenceProducer()) {
			options, optionsErr := evidenceProducerStartOptions(receipt, intent)
			if optionsErr != nil {
				return optionsErr
			}
			status, err = p.runs.AbortEvidenceProducer(options)
		} else {
			status, err = p.runs.Abort(parts[1], child.GetOperationId())
		}
		if err != nil {
			return fmt.Errorf("abort validation child %s: %w", child.GetOperationId(), err)
		}
		if !suiteTerminal(status.Status) {
			return fmt.Errorf("validation child %s has not settled after abort: %s", child.GetOperationId(), status.Status)
		}
		child.State, child.Detail, child.ReasonCode = suiteChildTerminalState(status.Status), status.Error, childReasonCode(status.Status)
	}
	return nil
}

func suiteChildTerminalState(status string) validationv1.ChildOperationState {
	switch status {
	case sharedruns.StatusPassed:
		return validationv1.ChildOperationState_CHILD_OPERATION_STATE_SUCCEEDED
	case sharedruns.StatusAborted:
		return validationv1.ChildOperationState_CHILD_OPERATION_STATE_CANCELLED
	default:
		return validationv1.ChildOperationState_CHILD_OPERATION_STATE_FAILED
	}
}

func childReasonCode(status string) validationv1.ValidationReasonCode {
	if status == sharedruns.StatusAborted {
		return validationv1.ValidationReasonCode_VALIDATION_REASON_CODE_ABORTED
	}
	if status == sharedruns.StatusPassed {
		return validationv1.ValidationReasonCode_VALIDATION_REASON_CODE_NONE
	}
	return validationv1.ValidationReasonCode_VALIDATION_REASON_CODE_PROVIDER_UNAVAILABLE
}

func suiteTerminal(status string) bool {
	return status == sharedruns.StatusPassed || status == sharedruns.StatusFailed || status == sharedruns.StatusAborted
}

func presetForStrength(strength validationv1.ValidationStrength) string {
	switch strength {
	case validationv1.ValidationStrength_VALIDATION_STRENGTH_SMOKE:
		return "smoke"
	case validationv1.ValidationStrength_VALIDATION_STRENGTH_TARGETED:
		return "quick"
	default:
		return "comprehensive"
	}
}

func validationDeadline(parent context.Context, intent *validationv1.ValidationIntent) (context.Context, context.CancelFunc) {
	budget := 30 * time.Minute
	if policy := intent.GetDeadlinePolicy(); policy != nil {
		if requested := policy.GetExecutionBudget(); requested != nil && requested.AsDuration() > 0 {
			budget = requested.AsDuration()
		}
	}
	return context.WithTimeout(parent, budget)
}

func queueDeadline(parent context.Context, intent *validationv1.ValidationIntent) (context.Context, context.CancelFunc) {
	budget := 10 * time.Minute
	if policy := intent.GetDeadlinePolicy(); policy != nil {
		if requested := policy.GetQueueBudget(); requested != nil && requested.AsDuration() > 0 {
			budget = requested.AsDuration()
		}
	}
	return context.WithTimeout(parent, budget)
}

func setChildState(receipt *validationv1.ValidationReceipt, operationID string, state validationv1.ChildOperationState, detail string) {
	for _, child := range receipt.GetChildren() {
		if child.GetOperationId() == operationID {
			child.State = state
			child.Detail = detail
			return
		}
	}
}
