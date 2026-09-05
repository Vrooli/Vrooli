package validationbroker

import (
	"context"
	"errors"
	"fmt"
	"strings"
	"time"

	"test-genie/internal/execution"
	"test-genie/internal/orchestrator"
	"test-genie/internal/runmanager"
	sharedruns "test-genie/internal/shared/runs"

	commonv1 "github.com/vrooli/vrooli/packages/proto/gen/go/common/v1"
	validationv1 "github.com/vrooli/vrooli/packages/proto/gen/go/test-genie/v1/validation"
	"google.golang.org/protobuf/types/known/timestamppb"
)

type suiteRunManager interface {
	Start(runmanager.StartOptions) (runmanager.StartResult, error)
	Wait(context.Context, string, string) (runmanager.LiveStatus, error)
	Abort(string, string) (runmanager.LiveStatus, error)
}

// RunProducer translates scenario validation intent into the existing durable
// suite-run authority. It waits exactly once per child; no client owns or polls
// the underlying run.
type RunProducer struct {
	runs     suiteRunManager
	identity IdentityResolver
	gct      GCTEvidenceClient
}

var errQueueBudget = errors.New("validation queue budget exhausted")

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
	if p.identity != nil {
		observed, err := p.identity.Resolve(ctx, intent)
		if err != nil {
			return p.terminalizeResolutionFailure(ctx, receiptID, transition, err)
		}
		resolved = true
		if observed.GetIdentity() != intent.GetExpectedIdentity().GetIdentity() {
			return p.terminalizeIdentityChange(ctx, receiptID, observed, transition, "content identity changed before producer work began")
		}
		admitted = observed
	}
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
		_, err = transition(ctx, receiptID, validationv1.ReceiptState_RECEIPT_STATE_SUCCEEDED, func(receipt *validationv1.ValidationReceipt) error {
			receipt.AchievedStrength = intent.GetRequiredStrength()
			receipt.ObservedIdentity = cloneIdentity(admitted)
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
		} else if child.GetState() != validationv1.ChildOperationState_CHILD_OPERATION_STATE_RUNNING {
			attempt++
			child = nil
		}
		maximumAttempts := int(intent.GetDeadlinePolicy().GetMaximumAttempts())
		for ; attempt <= maximumAttempts; attempt++ {
			runID := ""
			if child != nil {
				runID = child.GetOperationId()
			} else {
				result, err := p.startAfterCapacity(ctx, receiptID, scenario, intent)
				if err != nil {
					reason := validationv1.ValidationReasonCode_VALIDATION_REASON_CODE_PROVIDER_UNAVAILABLE
					if errors.Is(err, errQueueBudget) {
						reason = validationv1.ValidationReasonCode_VALIDATION_REASON_CODE_CAPACITY_UNAVAILABLE
					}
					return p.terminalizeFailure(ctx, receiptID, transition, reason, fmt.Sprintf("start validation child for %s: %v", scenario, err))
				}
				runID = result.RunID
				childID := fmt.Sprintf("%s:attempt:%d", prefix, attempt)
				updated, err := transition(ctx, receiptID, validationv1.ReceiptState_RECEIPT_STATE_RUNNING, func(receipt *validationv1.ValidationReceipt) error {
					receipt.Children = append(receipt.Children, &validationv1.ChildOperation{ChildId: childID, Kind: validationv1.ChildOperationKind_CHILD_OPERATION_KIND_TEST_RUN, State: validationv1.ChildOperationState_CHILD_OPERATION_STATE_RUNNING, Owner: "test-genie", OperationId: runID})
					return nil
				})
				if err != nil {
					return err
				}
				current = updated
			}

			waitCtx, cancel := validationDeadline(ctx, intent)
			status, waitErr := p.runs.Wait(waitCtx, scenario, runID)
			cancel()
			if waitErr != nil {
				_, _ = p.runs.Abort(scenario, runID)
				return p.terminalizeFailure(ctx, receiptID, transition, validationv1.ValidationReasonCode_VALIDATION_REASON_CODE_DEADLINE_EXCEEDED, fmt.Sprintf("validation child %s exceeded its server-owned deadline: %v", runID, waitErr))
			}
			childState := validationv1.ChildOperationState_CHILD_OPERATION_STATE_FAILED
			if status.Status == sharedruns.StatusPassed {
				childState = validationv1.ChildOperationState_CHILD_OPERATION_STATE_SUCCEEDED
			} else if status.Status == sharedruns.StatusAborted {
				childState = validationv1.ChildOperationState_CHILD_OPERATION_STATE_CANCELLED
			}
			updated, err := transition(ctx, receiptID, validationv1.ReceiptState_RECEIPT_STATE_RUNNING, func(receipt *validationv1.ValidationReceipt) error {
				setChildState(receipt, runID, childState, status.Error)
				if !hasEvidence(receipt, runID) {
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

func (p *RunProducer) executeGCTEvidence(ctx context.Context, current *validationv1.ValidationReceipt, intent *validationv1.ValidationIntent, transition TransitionFunc) (*validationv1.ValidationReceipt, error) {
	policy := intent.GetEvidencePolicy()
	if policy == nil || (!policy.GetRequireBehavioralBefore() && !policy.GetRequireSourceSnapshot()) {
		return current, nil
	}
	if p.gct == nil {
		return p.failMissingEvidence(ctx, current, intent, transition, "git-control-tower evidence adapter is unavailable")
	}
	collection := strings.TrimSpace(intent.GetCallerAttributes()["baseline_name"])
	if collection == "" {
		return p.failMissingEvidence(ctx, current, intent, transition, "baseline_name caller attribute is required for GCT evidence")
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
		return p.failMissingEvidence(ctx, current, intent, transition, err.Error())
	}
	state := validationv1.ChildOperationState_CHILD_OPERATION_STATE_FAILED
	if result.Passed {
		state = validationv1.ChildOperationState_CHILD_OPERATION_STATE_SUCCEEDED
	}
	updated, err := transition(ctx, current.GetReceiptId(), validationv1.ReceiptState_RECEIPT_STATE_RUNNING, func(receipt *validationv1.ValidationReceipt) error {
		setChildState(receipt, operationID, state, result.Detail)
		for _, evidence := range result.Evidence {
			if evidence != nil && !hasEvidence(receipt, evidence.GetEvidenceId()) {
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

func (p *RunProducer) startAfterCapacity(ctx context.Context, receiptID, scenario string, intent *validationv1.ValidationIntent) (runmanager.StartResult, error) {
	queueCtx, cancel := queueDeadline(ctx, intent)
	defer cancel()
	for {
		result, err := p.runs.Start(runmanager.StartOptions{
			Input: execution.SuiteExecutionInput{Request: orchestrator.SuiteExecutionRequest{
				ScenarioName:       scenario,
				Target:             "scenario:" + scenario,
				Preset:             presetForStrength(intent.GetRequiredStrength()),
				RequireGateQuality: intent.GetPurpose() == validationv1.ValidationPurpose_VALIDATION_PURPOSE_CERTIFICATION,
				RetainForEvidence:  true,
				RetentionReason:    "validation receipt " + receiptID,
			}},
			Caller: intent.GetCallerScenario(),
		})
		if err == nil {
			return result, nil
		}
		var busy *runmanager.BusyError
		if !errors.As(err, &busy) {
			return runmanager.StartResult{}, err
		}
		if _, waitErr := p.runs.Wait(queueCtx, busy.Scenario, busy.RunID); waitErr != nil {
			return runmanager.StartResult{}, fmt.Errorf("%w behind %s/%s: %v", errQueueBudget, busy.Scenario, busy.RunID, waitErr)
		}
	}
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

func hasEvidence(receipt *validationv1.ValidationReceipt, evidenceID string) bool {
	for _, evidence := range receipt.GetEvidence() {
		if evidence.GetEvidenceId() == evidenceID {
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
func (p *RunProducer) AbortValidation(_ context.Context, receipt *validationv1.ValidationReceipt, _, _ string) error {
	if p == nil || p.runs == nil {
		return fmt.Errorf("suite run manager is unavailable")
	}
	for _, child := range receipt.GetChildren() {
		if child.GetKind() != validationv1.ChildOperationKind_CHILD_OPERATION_KIND_TEST_RUN || child.GetState() != validationv1.ChildOperationState_CHILD_OPERATION_STATE_RUNNING {
			continue
		}
		parts := strings.Split(child.GetChildId(), ":")
		if len(parts) < 3 || parts[0] != "scenario" {
			return fmt.Errorf("validation child %q has no scenario identity", child.GetChildId())
		}
		if _, err := p.runs.Abort(parts[1], child.GetOperationId()); err != nil {
			return fmt.Errorf("abort validation child %s: %w", child.GetOperationId(), err)
		}
	}
	return nil
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
