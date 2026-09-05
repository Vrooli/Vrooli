package execution

import (
	"bytes"
	"context"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"path/filepath"
	"strings"

	"swarm-manager/internal/agentmanager"
	"swarm-manager/internal/apierr"
	"swarm-manager/internal/stringsx"
	"swarm-manager/internal/transitionrun"
	"swarm-manager/internal/transitionrunner"
	"swarm-manager/internal/transitions"

	domainpb "github.com/vrooli/vrooli/packages/proto/gen/go/agent-manager/v1/domain"
	executionv1 "github.com/vrooli/vrooli/packages/proto/gen/go/plan-manager/v1/execution"
	sharedv1 "github.com/vrooli/vrooli/packages/proto/gen/go/plan-manager/v1/shared"
	"google.golang.org/protobuf/encoding/protojson"
	"google.golang.org/protobuf/proto"
	"google.golang.org/protobuf/types/known/structpb"
)

type phasedPlanResult struct {
	Outcome string                  `json:"outcome"`
	Summary string                  `json:"summary,omitempty"`
	Reason  string                  `json:"reason,omitempty"`
	Blocker phasedPlanResultBlocker `json:"blocker,omitempty"`
	Handoff string                  `json:"handoff,omitempty"`
}

type phasedPlanSnapshot struct {
	PlanReference     string
	FrontierDigest    string
	ExecutionID       string
	ProjectRoot       string
	PlanExecutionID   string
	EntityKind        string
	EntityName        string
	EntityVersion     string
	MaxSlices         int
	WriteScope        []string
	SliceApprovalMode string
}

type phasedPlanResultBlocker struct {
	Code      string `json:"code,omitempty"`
	Summary   string `json:"summary,omitempty"`
	Retryable bool   `json:"retryable,omitempty"`
}

func (b *phasedPlanResultBlocker) UnmarshalJSON(data []byte) error {
	var text string
	if json.Unmarshal(data, &text) == nil {
		b.Summary = text
		return nil
	}
	type blockerAlias phasedPlanResultBlocker
	var value blockerAlias
	if err := json.Unmarshal(data, &value); err != nil {
		return err
	}
	*b = phasedPlanResultBlocker(value)
	return nil
}

// PhasedPlanApplyResult reports whether the transition was newly applied or
// was an idempotent replay of a previously completed callback.
type PhasedPlanApplyResult struct {
	Record     Record `json:"record"`
	Idempotent bool   `json:"idempotent"`
}

// SetPhasedPlanWorkflow installs a workflow transport and builds a local
// runner from it.
//
// TEST-ONLY SEAM. Production composition never calls this: it constructs one
// shared runner and injects it through SetTransitionRunner. Do not add a
// production caller, and do not gate behavior on the field it sets — code that
// did exactly that reported "phased-plan workflow service is not available" for
// every real plan execution, because nothing ever set it.
func (s *Service) SetPhasedPlanWorkflow(workflow agentmanager.WorkflowInvoker) {
	s.mu.Lock()
	defer s.mu.Unlock()
	s.phasedPlanWorkflow = workflow
	s.configureLocalTransitionRunner()
}

// SetWorkWorkflow installs a workflow transport for bounded follow-up and
// correction work.
//
// TEST-ONLY SEAM. See SetPhasedPlanWorkflow.
func (s *Service) SetWorkWorkflow(workflow agentmanager.WorkflowInvoker) {
	s.mu.Lock()
	defer s.mu.Unlock()
	s.workWorkflow = workflow
	s.configureLocalTransitionRunner()
}

// SetSpecSyncWorkflow installs a workflow transport for scenario spec-sync.
//
// TEST-ONLY SEAM. See SetPhasedPlanWorkflow.
func (s *Service) SetSpecSyncWorkflow(workflow agentmanager.WorkflowInvoker) {
	s.mu.Lock()
	defer s.mu.Unlock()
	s.specSyncWorkflow = workflow
	s.configureLocalTransitionRunner()
}

// SetWorkflowStartGuard applies transition policy to a locally-installed
// transport. In production the guard is attached to the shared runner's
// transport at composition, so this is a no-op there.
func (s *Service) SetWorkflowStartGuard(guard agentmanager.WorkflowStartGuard) {
	if workflow, ok := s.phasedPlanWorkflow.(*agentmanager.WorkflowService); ok {
		workflow.SetStartGuard(guard)
	}
	if workflow, ok := s.workWorkflow.(*agentmanager.WorkflowService); ok {
		workflow.SetStartGuard(guard)
	}
	if workflow, ok := s.specSyncWorkflow.(*agentmanager.WorkflowService); ok {
		workflow.SetStartGuard(guard)
	}
}

func buildPhasedPlanSnapshot(item backlogItem, record Record, planHandle, projectRoot string, rendered renderedPlanContent) (phasedPlanSnapshot, error) {
	itemBytes, err := marshalAuthoredBacklogItem(item)
	if err != nil {
		return phasedPlanSnapshot{}, fmt.Errorf("encode backlog snapshot: %w", err)
	}
	if rendered.Plan == nil {
		return phasedPlanSnapshot{}, fmt.Errorf("build phased-plan snapshot: canonical structured plan is required")
	}
	frontierJSON, err := authoredPlanFrontierJSON(rendered.Plan)
	if err != nil {
		return phasedPlanSnapshot{}, fmt.Errorf("build phased-plan snapshot: %w", err)
	}
	frontier := digestStrings(planHandle, frontierJSON)
	return phasedPlanSnapshot{
		PlanReference: planHandle, FrontierDigest: frontier, ExecutionID: record.ExecutionID,
		ProjectRoot: filepath.Clean(projectRoot),
		EntityKind:  item.Kind, EntityName: item.Name, EntityVersion: digestStrings(string(itemBytes)),
		MaxSlices: firstPositive(record.MaxSlices, 6), WriteScope: append([]string(nil), item.AcceptanceAllow...), PlanExecutionID: record.PlanManagerExecutionID,
	}, nil
}

// marshalAuthoredBacklogItem excludes lifecycle metadata that the execution
// service updates as part of the normal queue/start/finish flow. Including
// status or updated in the immutable subject version made a workflow reject
// its own terminal result after the backlog moved from queued to in_progress.
// Authored content remains covered, so a real edit still fails closed.
func marshalAuthoredBacklogItem(item backlogItem) ([]byte, error) {
	item.Status = ""
	item.Updated = ""
	item.PlanAcceptance = nil
	return json.Marshal(item)
}

func authoredPlanFrontierJSON(plan *sharedv1.Plan) (string, error) {
	stable, ok := proto.Clone(plan).(*sharedv1.Plan)
	if !ok || stable == nil {
		return "", fmt.Errorf("clone canonical structured plan")
	}
	stable.Status = sharedv1.PlanStatus_PLAN_STATUS_UNSPECIFIED
	stable.ContentHash = ""
	stable.UpdatedAt = ""
	stable.WorkPosture = sharedv1.WorkPosture_WORK_POSTURE_UNSPECIFIED
	stable.WorkPostureSource = sharedv1.WorkPostureSource_WORK_POSTURE_SOURCE_UNSPECIFIED
	stable.WorkPostureDetail = ""
	stable.Mirror = nil
	stable.SupersededBy = nil
	stripReferenceRuntimeState(stable.References)
	stripRelevantContextRuntimeState(stable.RelevantContext)
	if stable.RegressionAnchor != nil {
		stable.RegressionAnchor.CapturedAt = ""
	}
	for _, phase := range stable.Phases {
		phase.Status = sharedv1.PhaseStatus_PHASE_STATUS_UNSPECIFIED
		phase.BaselineScope = nil
		phase.LastValidation = nil
		stripReferenceRuntimeState(phase.References)
		stripRelevantContextRuntimeState(phase.RelevantContext)
	}
	data, err := (protojson.MarshalOptions{UseProtoNames: true}).Marshal(stable)
	if err != nil {
		return "", fmt.Errorf("encode authored plan frontier: %w", err)
	}
	var canonical any
	if err := json.Unmarshal(data, &canonical); err != nil {
		return "", fmt.Errorf("decode authored plan frontier: %w", err)
	}
	var canonicalJSON bytes.Buffer
	encoder := json.NewEncoder(&canonicalJSON)
	encoder.SetEscapeHTML(false)
	if err := encoder.Encode(canonical); err != nil {
		return "", fmt.Errorf("canonicalize authored plan frontier: %w", err)
	}
	data = bytes.TrimSuffix(canonicalJSON.Bytes(), []byte{'\n'})
	return string(data), nil
}

func stripReferenceRuntimeState(references []*sharedv1.Reference) {
	for _, reference := range references {
		reference.Resolution = sharedv1.ReferenceResolution_REFERENCE_RESOLUTION_UNSPECIFIED
		reference.Staleness = sharedv1.StalenessTier_STALENESS_TIER_UNSPECIFIED
		reference.ChangeFactor = 0
	}
}

func stripRelevantContextRuntimeState(items []*sharedv1.RelevantContextItem) {
	for _, item := range items {
		item.Status = sharedv1.RelevantContextStatus_RELEVANT_CONTEXT_STATUS_UNSPECIFIED
		item.StatusDetail = ""
	}
}

func firstPositive(value, fallback int) int {
	if value > 0 {
		return value
	}
	return fallback
}

func (snapshot phasedPlanSnapshot) input() (*structpb.Value, error) {
	writeScope := make([]any, len(snapshot.WriteScope))
	for i, path := range snapshot.WriteScope {
		writeScope[i] = path
	}
	return structpb.NewValue(map[string]any{
		"projectRoot":     snapshot.ProjectRoot,
		"plan":            map[string]any{"reference": snapshot.PlanReference, "frontierDigest": snapshot.FrontierDigest},
		"planExecutionId": snapshot.PlanExecutionID,
		"consumer": map[string]any{
			"executionId": snapshot.ExecutionID, "entityKind": snapshot.EntityKind,
			"entityName": snapshot.EntityName, "entityVersion": snapshot.EntityVersion,
		},
		"constraints": map[string]any{"maxSlices": snapshot.MaxSlices, "writeScope": writeScope, "sliceApprovalMode": firstNonEmpty(snapshot.SliceApprovalMode, string(transitions.GateModeManual))},
	})
}

func digestStrings(parts ...string) string {
	h := sha256.New()
	for _, part := range parts {
		_, _ = h.Write([]byte(part))
		_, _ = h.Write([]byte{0})
	}
	return "sha256:" + hex.EncodeToString(h.Sum(nil))
}

// ApprovePhasedPlanWorkflow durably records the operator decision before
// delivering its idempotent workflow signal.
func (s *Service) ApprovePhasedPlanWorkflow(ctx context.Context, executionID, actor string) (Record, error) {
	s.mu.Lock()
	defer s.mu.Unlock()
	records, idx, err := s.loadRecordLocked(executionID)
	if err != nil {
		return Record{}, err
	}
	record := records[idx]
	correlation, err := s.transitionCorrelation(record)
	if err != nil || correlation.TransitionKey != "plan.execute" {
		return Record{}, apierr.BadRequest("execution is not owned by a phased-plan workflow")
	}
	// Apply state lives on the shared correlation, not on this record. Keeping a
	// second copy here meant two sources of truth for whether a terminal result
	// had already been consumed.
	if correlation.ApplyState == transitionrun.ApplyStateComplete || isWorkflowConsumerTerminal(record.Status) {
		return Record{}, apierr.Conflict("execution workflow is already terminal")
	}
	if strings.TrimSpace(correlation.ApprovalTime) == "" {
		correlation, err = s.transitionRunner.UpdateCorrelation(correlation.ExecutionID, func(value *transitionrun.Correlation) error {
			value.ApprovalTime = nowRFC3339()
			value.ApprovalActor = strings.TrimSpace(actor)
			return nil
		})
		if err != nil {
			return Record{}, err
		}
	}
	payload, err := structpb.NewValue(map[string]any{"executionId": record.ExecutionID, "actor": correlation.ApprovalActor})
	if err != nil {
		return Record{}, apierr.Internal("encode workflow approval: %s", err.Error())
	}
	var signalErr error
	if s.transitionRunner != nil {
		signalErr = s.transitionRunner.Signal(ctx, correlation.ExecutionID, "slice_approved", payload, "approve-"+record.ExecutionID)
	} else if signaler, ok := s.phasedPlanWorkflow.(interface {
		SignalWorkflow(context.Context, string, string, *structpb.Value, string) error
	}); ok {
		signalErr = signaler.SignalWorkflow(ctx, correlation.ExecutionID, "slice_approved", payload, "approve-"+record.ExecutionID)
	} else {
		return Record{}, apierr.BadRequest("approval is not supported by current workflow service")
	}
	if signalErr != nil {
		return Record{}, wrapAgentError(signalErr)
	}
	return record, nil
}

// RebasePhasedPlanWorkflow migrates a legacy subject-version pin after an
// execution service changed its authored backlog projection. The operator
// must name the migration, and the current authored plan frontier must still
// equal the pinned frontier. This preserves the meaningful stale-write guard
// while allowing an already-terminal legacy workflow to be applied.
func (s *Service) RebasePhasedPlanWorkflow(ctx context.Context, executionID, actor, reason string) (Record, error) {
	actor = strings.TrimSpace(actor)
	reason = strings.TrimSpace(reason)
	if actor == "" || reason == "" {
		return Record{}, apierr.BadRequest("workflow rebase requires actor and reason")
	}
	s.mu.Lock()
	records, idx, err := s.loadRecordLocked(executionID)
	if err != nil {
		s.mu.Unlock()
		return Record{}, err
	}
	record := records[idx]
	correlation, err := s.transitionCorrelation(record)
	s.mu.Unlock()
	if err != nil || correlation.TransitionKey != "plan.execute" {
		return Record{}, apierr.BadRequest("execution is not owned by a phased-plan workflow")
	}
	if isWorkflowConsumerTerminal(record.Status) || correlation.ApplyState == transitionrun.ApplyStateComplete {
		return Record{}, apierr.Conflict("execution workflow is already terminal")
	}
	if s.transitionRunner == nil {
		return Record{}, apierr.Unavailable("transition runner is not configured")
	}
	current, err := s.transitionRunner.BuildInput(ctx, "plan.execute", record.ExecutionID)
	if err != nil {
		return Record{}, apierr.BadGateway("rebuild current phased-plan input: %s", err)
	}
	if current.FrontierDigest != correlation.FrontierDigest {
		return Record{}, apierr.Conflict("cannot rebase subject version after the authored plan frontier changed")
	}
	declaredOutcomes, err := s.transitionRunner.DeclaredOutcomes("plan.execute")
	if err != nil {
		return Record{}, apierr.BadGateway("read current phased-plan transition contract: %s", err)
	}
	if _, err := s.transitionRunner.UpdateCorrelation(correlation.ExecutionID, func(value *transitionrun.Correlation) error {
		if value.ApplyState == transitionrun.ApplyStateComplete {
			return apierr.Conflict("execution workflow is already terminal")
		}
		if value.FrontierDigest != current.FrontierDigest {
			return apierr.Conflict("cannot rebase subject version after the authored plan frontier changed")
		}
		value.EntityVersion = current.EntityVersion
		value.DeclaredOutcomes = append([]string(nil), declaredOutcomes...)
		value.EntityVersionRebasedBy = actor
		value.EntityVersionRebasedAt = nowRFC3339()
		value.EntityVersionRebasedReason = reason
		return nil
	}); err != nil {
		return Record{}, err
	}
	// Return the persisted record directly. Get also reconciles active executions,
	// which would make an operator-only rebase unexpectedly advance lifecycle state.
	s.mu.Lock()
	defer s.mu.Unlock()
	records, idx, err = s.loadRecordLocked(executionID)
	if err != nil {
		return Record{}, err
	}
	return records[idx], nil
}

// ApplyPhasedPlanWorkflow validates and applies one authorized terminal result.
// A persisted claim is sufficient to resume after a process crash without
// collecting mutable external state again.
func (s *Service) ApplyPhasedPlanWorkflow(ctx context.Context, executionID string) (PhasedPlanApplyResult, error) {
	if s.transitionRunner == nil {
		return PhasedPlanApplyResult{}, agentmanager.ErrNotAvailable
	}
	workflowExecutionID, err := s.transitionExecutionID(ctx, executionID)
	if err != nil {
		return PhasedPlanApplyResult{}, err
	}
	before, err := s.transitionRunner.GetCorrelation(workflowExecutionID)
	if err != nil {
		return PhasedPlanApplyResult{}, err
	}
	if _, err := s.transitionRunner.ApplyExecution(ctx, workflowExecutionID); err != nil {
		return PhasedPlanApplyResult{}, err
	}
	record, err := s.Get(ctx, executionID)
	if err != nil {
		return PhasedPlanApplyResult{}, err
	}
	return PhasedPlanApplyResult{Record: record, Idempotent: before.ApplyState == "complete"}, nil
}

func (s *Service) applyPlanExecuteTransition(ctx context.Context, executionID string, outcome transitionrunner.Outcome) error {
	s.mu.Lock()
	defer s.mu.Unlock()
	records, idx, err := s.loadRecordLocked(executionID)
	if err != nil {
		return err
	}
	record := records[idx]
	if outcome.TransitionKey != "plan.execute" {
		return fmt.Errorf("execution %q is not a plan execution transition", executionID)
	}
	result, err := parsePhasedPlanOutcome(domainpb.WorkflowExecutionStatus_WORKFLOW_EXECUTION_STATUS_SUCCEEDED, outcome.Result)
	if err != nil {
		return err
	}
	if err := s.reconcilePlanManagerCompletion(ctx, &record, result.Outcome); err != nil {
		return err
	}
	if err := s.finishPhasedPlanClaim(&record, outcome, result); err != nil {
		return err
	}
	record.UpdatedAt = nowRFC3339()
	records[idx] = record
	if err := s.store.Save(records); err != nil {
		return err
	}
	s.dispatchStatusUpdate(record)
	return nil
}

// reconcilePlanManagerCompletion makes Plan Manager the final authority for a
// drain that claims completion. The agent can complete phases through the
// bound execution, but Swarm never infers the terminal plan state from a
// handoff alone.
func (s *Service) reconcilePlanManagerCompletion(ctx context.Context, record *Record, workflowOutcome string) error {
	if workflowOutcome != "complete" || strings.TrimSpace(record.PlanManagerExecutionID) == "" || strings.TrimSpace(record.PlanManagerReconciledAt) != "" {
		return nil
	}
	client, ok := s.planRenderer.(interface {
		GetStatus(context.Context, *executionv1.GetStatusRequest) (*executionv1.GetStatusResponse, error)
		Complete(context.Context, *executionv1.CompleteRequest) (*executionv1.CompleteResponse, error)
	})
	if !ok {
		return nil
	}
	status, err := client.GetStatus(ctx, &executionv1.GetStatusRequest{ExecutionId: record.PlanManagerExecutionID})
	if err != nil {
		return apierr.BadGateway("get plan-manager execution status: %s", err)
	}
	if status.GetExecution() == nil {
		return apierr.BadGateway("plan-manager status omitted execution")
	}
	if !status.GetExecution().GetComplete() {
		if _, err := client.Complete(ctx, &executionv1.CompleteRequest{ExecutionId: record.PlanManagerExecutionID}); err != nil {
			return apierr.BadGateway("reconcile plan-manager execution completion: %s", err)
		}
		status, err = client.GetStatus(ctx, &executionv1.GetStatusRequest{ExecutionId: record.PlanManagerExecutionID})
		if err != nil {
			return apierr.BadGateway("verify reconciled plan-manager execution: %s", err)
		}
		if status.GetExecution() == nil || !status.GetExecution().GetComplete() {
			return apierr.Conflict("drain claimed complete but bound plan-manager execution is unfinished")
		}
	}
	record.PlanManagerReconciledAt = nowRFC3339()
	return nil
}

func parsePhasedPlanOutcome(status domainpb.WorkflowExecutionStatus, raw json.RawMessage) (phasedPlanResult, error) {
	if status != domainpb.WorkflowExecutionStatus_WORKFLOW_EXECUTION_STATUS_SUCCEEDED {
		var result phasedPlanResult
		_ = json.Unmarshal(raw, &result)
		switch status {
		case domainpb.WorkflowExecutionStatus_WORKFLOW_EXECUTION_STATUS_BLOCKED:
			result.Outcome = "blocked"
			return result, nil
		case domainpb.WorkflowExecutionStatus_WORKFLOW_EXECUTION_STATUS_ABSTAINED:
			result.Outcome = "abstained"
			return result, nil
		case domainpb.WorkflowExecutionStatus_WORKFLOW_EXECUTION_STATUS_BUDGET_EXHAUSTED:
			return phasedPlanResult{Outcome: "budget_exhausted"}, nil
		case domainpb.WorkflowExecutionStatus_WORKFLOW_EXECUTION_STATUS_CANCELLED:
			return phasedPlanResult{Outcome: "cancelled"}, nil
		default:
			return phasedPlanResult{Outcome: "failed"}, nil
		}
	}
	var result phasedPlanResult
	if len(raw) == 0 || json.Unmarshal(raw, &result) != nil {
		return phasedPlanResult{}, apierr.BadGateway("workflow succeeded without a valid typed result")
	}
	switch result.Outcome {
	case "complete", "completed", "blocked", "needs_review", "needs_attention", "abstained", "failed", "budget_exhausted", "cancelled":
		if result.Outcome == "completed" {
			result.Outcome = "complete"
		}
		return result, nil
	default:
		return phasedPlanResult{}, apierr.BadGateway("workflow returned unsupported terminal outcome %q", result.Outcome)
	}
}

func (s *Service) finishPhasedPlanClaim(record *Record, outcome transitionrunner.Outcome, result phasedPlanResult) error {
	item, err := s.loadBacklogItemByRecord(record)
	if err != nil {
		return err
	}
	previous := record.Status
	record.FinishedAt = nowRFC3339()
	switch result.Outcome {
	case "complete":
		record.Status = StatusCompleted
		candidates := []string{}
		s.applyCompletedTransition(record, item, &candidates)
	case "cancelled":
		record.Status = StatusCanceled
		if err := s.updateBacklogStatus(item, restoreBacklogStatus(*record)); err != nil {
			return err
		}
	case "blocked", "needs_review", "needs_attention", "abstained":
		if result.Outcome == "abstained" {
			record.Status = StatusAbstained
		} else {
			record.Status = StatusNeedsReview
		}
		record.FailureReason = stringsx.FirstNonEmpty(result.Blocker.Summary, result.Reason, result.Summary, "workflow "+result.Outcome)
		if err := s.updateBacklogStatus(item, backlogStatusInReview); err != nil {
			return err
		}
	case "budget_exhausted":
		// A bounded drain is an expected, resumable terminal. Preserve the
		// reason and keep it in review so the operator can continue it; it is
		// not evidence that the agent or transition failed.
		record.Status = StatusBudgetExhausted
		record.FailureReason = stringsx.FirstNonEmpty(outcome.TerminalCode, result.Reason, result.Summary, "workflow budget exhausted")
		if err := s.updateBacklogStatus(item, backlogStatusInReview); err != nil {
			return err
		}
	default:
		record.Status = StatusFailed
		record.FailureReason = stringsx.FirstNonEmpty(outcome.TerminalCode, result.Outcome)
		if err := s.updateBacklogStatus(item, backlogStatusInReview); err != nil {
			return err
		}
	}
	s.logExecutionEvent(*record, previous)
	return nil
}

func isWorkflowConsumerTerminal(status Status) bool {
	switch status {
	case StatusCompleted, StatusFailed, StatusCanceled, StatusAbstained, StatusBudgetExhausted:
		return true
	default:
		return false
	}
}
