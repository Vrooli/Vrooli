package development

import (
	"context"
	"errors"
	"fmt"
	"sort"
	"strings"

	"swarm-manager/internal/workflowcontract"
)

// Coordinator is the only adapter that may turn a reviewed development
// engagement into an Agent Manager workflow. Reserve happens before transport
// and bind happens after the owner returns an execution identity. A lost
// response therefore leaves a durable reservation for reconciliation instead
// of creating an unaccounted retry.
type Coordinator struct {
	engagement *Service
	workflows  workflowcontract.Invoker
}

func NewCoordinator(engagement *Service, workflows workflowcontract.Invoker) *Coordinator {
	return &Coordinator{engagement: engagement, workflows: workflows}
}

type StartRequest struct {
	WorkItem   string
	Digest     string
	AttemptKey string
	Mode       string
	Upper      Usage
	Invocation workflowcontract.Invocation
	Effects    []Effect
}

type StartResult struct {
	Engagement Engagement
	Execution  workflowcontract.Start
}

// Cancel closes local dispatch authority before contacting the execution owner.
// A failed owner call leaves a durable requested intent for retry; settlement
// remains possible afterwards so late terminal usage is still charged.
func (c *Coordinator) Cancel(ctx context.Context, workItem, attemptKey, executionID string, expected int64, reason string) (Engagement, error) {
	if c == nil || c.engagement == nil {
		return Engagement{}, fmt.Errorf("development execution owner is unavailable: %w", ErrDenied)
	}
	state, err := c.engagement.Get(ctx, workItem)
	if err != nil {
		return state, err
	}
	active := activeAttempt(state)
	if active == nil || active.Key != attemptKey || active.ExecutionID != executionID {
		return state, ErrConflict
	}
	state, err = c.engagement.Revoke(ctx, workItem, expected, reason)
	if err != nil {
		return state, err
	}
	if c.workflows == nil {
		return state, fmt.Errorf("local revocation committed; workflow cancellation owner is unavailable: %w", ErrDenied)
	}
	canceller, ok := c.workflows.(workflowcontract.Canceller)
	if !ok {
		return state, fmt.Errorf("local revocation committed; workflow cancellation is not supported: %w", ErrDenied)
	}
	operationID := ""
	if state.Cancellation != nil {
		operationID = state.Cancellation.OperationID
	}
	if err := canceller.Cancel(ctx, executionID, operationID, reason); err != nil {
		return state, err
	}
	return c.engagement.AcknowledgeCancellation(ctx, workItem, state.Version, operationID)
}

func (c *Coordinator) Start(ctx context.Context, req StartRequest) (StartResult, error) {
	if c == nil || c.engagement == nil || c.workflows == nil {
		return StartResult{}, fmt.Errorf("development execution owner is unavailable: %w", ErrDenied)
	}
	if strings.TrimSpace(req.WorkItem) == "" || strings.TrimSpace(req.Digest) == "" || strings.TrimSpace(req.AttemptKey) == "" {
		return StartResult{}, ErrInvalid
	}
	selection, err := SelectLane(req.Mode, CapabilityRequirements{Binding: true, Continuation: true, Metering: true, Cancellation: true, Containment: true}, []LaneCapability{QualifiedFallbackCapability()})
	if err != nil {
		return StartResult{}, err
	}
	req.Mode = selection.Lane
	grant, err := grantForUsage(req.Upper)
	if err != nil {
		return StartResult{}, err
	}
	stateSnapshot, snapshotErr := c.engagement.Snapshot(ctx, req.Digest)
	if snapshotErr != nil {
		return StartResult{}, snapshotErr
	}
	if err := validateRequestedEffects(req.Effects, stateSnapshot.Proposal.AllowedEffects); err != nil {
		return StartResult{}, err
	}
	// The approved set is the owner ceiling. A worker may request a narrower
	// set, but omission cannot erase declared workflow effects.
	grant.AllowedEffects = append([]string(nil), stateSnapshot.Proposal.AllowedEffects...)
	sort.Strings(grant.AllowedEffects)
	prior, priorErr := c.engagement.Get(ctx, req.WorkItem)
	if priorErr != nil && !errors.Is(priorErr, ErrNotFound) {
		return StartResult{}, priorErr
	}
	var priorAttempt *Attempt
	if priorErr == nil {
		for i := range prior.Attempts {
			if prior.Attempts[i].Key == req.AttemptKey {
				priorAttempt = &prior.Attempts[i]
				break
			}
		}
	}
	state, created, err := c.engagement.ReserveOwned(ctx, req.WorkItem, req.Digest, req.AttemptKey, req.Mode, req.Upper)
	if err != nil {
		return StartResult{Engagement: state}, fmt.Errorf("reserve development allowance: %w", err)
	}
	// Reserve is idempotent and may have observed a reservation created by a
	// concurrent dispatcher after the initial read above. Only the caller that
	// advanced the version may invoke the owner; every other caller reconciles
	// the existing attempt instead of issuing a second dispatch.
	if priorAttempt == nil {
		if !created {
			for i := range state.Attempts {
				if state.Attempts[i].Key != req.AttemptKey {
					continue
				}
				if state.Attempts[i].ExecutionID == "" {
					if reconciled, handled, reconcileErr := c.reconcileUnboundStart(ctx, state, req); handled {
						return reconciled, reconcileErr
					}
					return StartResult{Engagement: state}, fmt.Errorf("attempt %q is owned by another dispatcher and has no bound owner execution: %w", req.AttemptKey, ErrConflict)
				}
				return StartResult{Engagement: state, Execution: workflowcontract.Start{ExecutionID: state.Attempts[i].ExecutionID, WorkflowDigest: state.Attempts[i].WorkflowDigest, DefinitionDigest: state.Attempts[i].WorkflowDigest, ApprovalDigest: state.Attempts[i].Digest, GrantDigest: state.Attempts[i].GrantDigest, CapabilityRevision: state.Attempts[i].CapabilityRevision, SelectionReason: state.Attempts[i].SelectionReason}}, nil
			}
			return StartResult{Engagement: state}, ErrConflict
		}
	}
	if priorAttempt != nil {
		if priorAttempt.ExecutionID == "" {
			if reconciled, handled, reconcileErr := c.reconcileUnboundStart(ctx, state, req); handled {
				return reconciled, reconcileErr
			}
			return StartResult{Engagement: state}, fmt.Errorf("attempt %q has no bound owner execution; reconcile before retrying: %w", req.AttemptKey, ErrConflict)
		}
		return StartResult{Engagement: state, Execution: workflowcontract.Start{ExecutionID: priorAttempt.ExecutionID, WorkflowDigest: priorAttempt.WorkflowDigest, DefinitionDigest: priorAttempt.WorkflowDigest, ApprovalDigest: priorAttempt.Digest, GrantDigest: priorAttempt.GrantDigest, CapabilityRevision: priorAttempt.CapabilityRevision, SelectionReason: priorAttempt.SelectionReason}}, nil
	}
	invocation := req.Invocation
	invocation.ApprovalDigest = req.Digest
	invocation.GrantDigest = workflowcontract.GrantDigest(grant)
	invocation.IdempotencyKey = strings.TrimSpace(invocation.IdempotencyKey)
	if invocation.IdempotencyKey == "" {
		invocation.IdempotencyKey = req.AttemptKey
	}
	if invocation.IdempotencyKey != req.AttemptKey {
		return StartResult{Engagement: state}, fmt.Errorf("owner idempotency key must equal attempt key: %w", ErrConflict)
	}
	invocation.Grant = grant
	started, err := c.workflows.Start(ctx, invocation)
	if err != nil {
		// Do not release the reservation. The owner may have accepted the
		// request before the transport failed; reconciliation owns this state.
		return StartResult{Engagement: state}, err
	}
	if strings.TrimSpace(started.ExecutionID) == "" {
		return StartResult{Engagement: state}, errors.New("agent-manager returned no execution identity")
	}
	workflowDigest := strings.TrimSpace(started.WorkflowDigest)
	if workflowDigest == "" {
		workflowDigest = strings.TrimSpace(started.DefinitionDigest)
	}
	expectedWorkflowDigest := strings.TrimSpace(invocation.WorkflowDigest)
	if expectedWorkflowDigest != "" && workflowDigest == "" {
		return StartResult{Engagement: state, Execution: started}, fmt.Errorf("owner returned no workflow definition digest for expected %q: %w", expectedWorkflowDigest, ErrConflict)
	}
	if expectedWorkflowDigest != "" && workflowDigest != expectedWorkflowDigest {
		if canceller, ok := c.workflows.(workflowcontract.Canceller); ok {
			_ = canceller.Cancel(ctx, started.ExecutionID, req.AttemptKey+"-digest-fence", "development workflow revision does not match approved digest")
		}
		return StartResult{Engagement: state, Execution: started}, fmt.Errorf("workflow digest %q does not match expected workflow digest %q: %w", workflowDigest, expectedWorkflowDigest, ErrConflict)
	}
	if started.ApprovalDigest == "" {
		return StartResult{Engagement: state, Execution: started}, fmt.Errorf("owner returned no approval digest for engagement %q: %w", invocation.ApprovalDigest, ErrConflict)
	}
	if started.ApprovalDigest != invocation.ApprovalDigest {
		return StartResult{Engagement: state, Execution: started}, fmt.Errorf("owner returned approval digest %q, expected %q: %w", started.ApprovalDigest, invocation.ApprovalDigest, ErrConflict)
	}
	if started.GrantDigest == "" {
		return StartResult{Engagement: state, Execution: started}, fmt.Errorf("owner returned no grant digest for engagement grant: %w", ErrConflict)
	}
	if started.GrantDigest != invocation.GrantDigest {
		return StartResult{Engagement: state, Execution: started}, fmt.Errorf("owner returned grant digest %q, expected %q: %w", started.GrantDigest, invocation.GrantDigest, ErrConflict)
	}
	started.WorkflowDigest, started.DefinitionDigest, started.ApprovalDigest, started.GrantDigest = workflowDigest, workflowDigest, invocation.ApprovalDigest, invocation.GrantDigest
	started.CapabilityRevision, started.SelectionReason = selection.Revision, selection.Reason
	bound, err := c.engagement.BindExecution(ctx, req.WorkItem, req.AttemptKey, started.ExecutionID, workflowDigest, invocation.GrantDigest, selection.Revision, selection.Reason)
	if err != nil {
		// A mismatched binding is a safety failure. Best-effort cancellation is
		// allowed, but the durable engagement remains unresolved for review.
		if canceller, ok := c.workflows.(workflowcontract.Canceller); ok {
			_ = canceller.Cancel(ctx, started.ExecutionID, req.AttemptKey+"-bind-fence", "development engagement binding failed")
		}
		return StartResult{Engagement: bound, Execution: started}, err
	}
	return StartResult{Engagement: bound, Execution: started}, nil
}

func (c *Coordinator) reconcileUnboundStart(ctx context.Context, state Engagement, req StartRequest) (StartResult, bool, error) {
	if c == nil || c.workflows == nil {
		return StartResult{Engagement: state}, false, nil
	}
	selection, err := SelectLane(req.Mode, CapabilityRequirements{Binding: true, Continuation: true, Metering: true, Cancellation: true, Containment: true}, []LaneCapability{QualifiedFallbackCapability()})
	if err != nil {
		return StartResult{Engagement: state}, true, err
	}
	reconciler, ok := c.workflows.(workflowcontract.StartReconciler)
	if !ok {
		return StartResult{Engagement: state}, false, nil
	}
	started, err := reconciler.ReconcileStart(ctx, req.AttemptKey)
	if err != nil {
		if errors.Is(err, workflowcontract.ErrStartNotFound) {
			return StartResult{Engagement: state}, false, nil
		}
		return StartResult{Engagement: state}, true, err
	}
	if strings.TrimSpace(started.ExecutionID) == "" {
		return StartResult{Engagement: state}, true, errors.New("owner reconciliation returned no execution identity")
	}
	workflowDigest := strings.TrimSpace(started.WorkflowDigest)
	if workflowDigest == "" {
		workflowDigest = strings.TrimSpace(started.DefinitionDigest)
	}
	expectedWorkflowDigest := strings.TrimSpace(req.Invocation.WorkflowDigest)
	grant, grantErr := grantForUsage(req.Upper)
	if grantErr != nil {
		return StartResult{Engagement: state}, true, grantErr
	}
	snapshot, snapshotErr := c.engagement.Snapshot(ctx, req.Digest)
	if snapshotErr != nil {
		return StartResult{Engagement: state}, true, snapshotErr
	}
	grant.AllowedEffects = append([]string(nil), snapshot.Proposal.AllowedEffects...)
	sort.Strings(grant.AllowedEffects)
	expectedGrantDigest := strings.TrimSpace(req.Invocation.GrantDigest)
	if expectedGrantDigest == "" {
		expectedGrantDigest = workflowcontract.GrantDigest(grant)
	}
	if expectedWorkflowDigest != "" && workflowDigest == "" {
		return StartResult{Engagement: state, Execution: started}, true, fmt.Errorf("reconciled owner returned no workflow definition digest for expected %q: %w", expectedWorkflowDigest, ErrConflict)
	}
	if expectedWorkflowDigest != "" && workflowDigest != expectedWorkflowDigest {
		if canceller, ok := c.workflows.(workflowcontract.Canceller); ok {
			_ = canceller.Cancel(ctx, started.ExecutionID, req.AttemptKey+"-digest-fence", "reconciled workflow revision does not match approved digest")
		}
		return StartResult{Engagement: state, Execution: started}, true, fmt.Errorf("reconciled workflow digest %q does not match expected workflow digest %q: %w", workflowDigest, expectedWorkflowDigest, ErrConflict)
	}
	if started.ApprovalDigest == "" {
		return StartResult{Engagement: state, Execution: started}, true, fmt.Errorf("reconciled owner returned no approval digest: %w", ErrConflict)
	}
	if started.ApprovalDigest != req.Digest {
		return StartResult{Engagement: state, Execution: started}, true, fmt.Errorf("reconciled owner returned approval digest %q, expected %q: %w", started.ApprovalDigest, req.Digest, ErrConflict)
	}
	if started.GrantDigest == "" || started.GrantDigest != expectedGrantDigest {
		return StartResult{Engagement: state, Execution: started}, true, fmt.Errorf("reconciled owner returned grant digest %q, expected %q: %w", started.GrantDigest, expectedGrantDigest, ErrConflict)
	}
	started.WorkflowDigest, started.DefinitionDigest, started.ApprovalDigest = workflowDigest, workflowDigest, req.Digest
	started.GrantDigest = expectedGrantDigest
	started.CapabilityRevision, started.SelectionReason = selection.Revision, selection.Reason
	bound, err := c.engagement.BindExecution(ctx, req.WorkItem, req.AttemptKey, started.ExecutionID, workflowDigest, expectedGrantDigest, selection.Revision, selection.Reason)
	if err != nil {
		return StartResult{Engagement: bound, Execution: started}, true, err
	}
	return StartResult{Engagement: bound, Execution: started}, true, nil
}

// Collect reconciles only terminal, measured owner state. Wall time is the
// owner's conservative elapsed created-to-ended measurement until an active
// provider meter exists. A timeout or missing receipt leaves the reservation
// held and can be retried after restart.
func (c *Coordinator) Collect(ctx context.Context, workItem, attemptKey, executionID string) (Engagement, error) {
	if c == nil || c.engagement == nil || c.workflows == nil {
		return Engagement{}, fmt.Errorf("development execution owner is unavailable: %w", ErrDenied)
	}
	completion, err := c.workflows.Collect(ctx, executionID)
	if err != nil {
		return Engagement{}, err
	}
	if completion.ExecutionID != "" && completion.ExecutionID != executionID {
		return Engagement{}, fmt.Errorf("owner returned execution %q for requested %q: %w", completion.ExecutionID, executionID, ErrConflict)
	}
	state, err := c.engagement.Get(ctx, workItem)
	if err != nil {
		return state, err
	}
	var attempt *Attempt
	for i := range state.Attempts {
		if state.Attempts[i].Key == attemptKey {
			attempt = &state.Attempts[i]
			break
		}
	}
	if attempt == nil || attempt.ExecutionID != executionID {
		return state, ErrConflict
	}
	workflowDigest := strings.TrimSpace(completion.WorkflowDigest)
	if workflowDigest == "" {
		workflowDigest = strings.TrimSpace(completion.DefinitionDigest)
	}
	if attempt.WorkflowDigest != "" && workflowDigest == "" {
		return state, fmt.Errorf("owner returned no workflow definition digest for attempt %q: %w", attemptKey, ErrConflict)
	}
	if attempt.WorkflowDigest != "" && workflowDigest != attempt.WorkflowDigest {
		return state, fmt.Errorf("owner returned workflow digest %q for attempt %q, expected %q: %w", workflowDigest, attemptKey, attempt.WorkflowDigest, ErrConflict)
	}
	if completion.Usage == nil || !completion.Usage.TokensKnown {
		return state, fmt.Errorf("agent-manager terminal usage is missing: %w", ErrDenied)
	}
	usage := Usage{Tokens: completion.Usage.Tokens, WallSeconds: completion.Usage.WallSeconds}
	if usage.WallSeconds <= 0 {
		return state, fmt.Errorf("agent-manager terminal wall-time usage is missing: %w", ErrDenied)
	}
	checkpoint := "terminal"
	if completion.TerminalCode != "" {
		checkpoint = completion.TerminalCode
	}
	return c.engagement.Settle(ctx, workItem, attemptKey, executionID, &usage, checkpoint)
}

func grantForUsage(usage Usage) (*workflowcontract.Grant, error) {
	if usage.Tokens <= 0 || usage.WallSeconds <= 0 {
		return nil, ErrInvalid
	}
	return &workflowcontract.Grant{MaxTokens: usage.Tokens, MaxWallTimeSeconds: usage.WallSeconds}, nil
}
