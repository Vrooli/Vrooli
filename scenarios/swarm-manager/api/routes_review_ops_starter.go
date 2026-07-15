package main

import (
	"context"
	"strings"

	"swarm-manager/internal/agentops"
	"swarm-manager/internal/initiativereview"
	"swarm-manager/internal/opsrunner"
	"swarm-manager/internal/review"
)

// pinnedReviewOperationVersion is the exact operation-contract version the review
// reroutes pin (an empty version resolves contract-latest but NOT the pinned
// system binding, which pins 1.0.0, so the reroute must pass a concrete version).
const pinnedReviewOperationVersion = "1.0.0"

// reviewOperationStarter adapts the generic operation runner to the review
// service's OperationStarter seam: it invokes the review-round / evidence-request
// operation against the backlog-item target and returns the live run association.
// It keeps the review package free of an agentops/opsrunner import.
type reviewOperationStarter struct {
	runner *opsrunner.Runner
}

// StartReviewOperation invokes the review operation and returns the live run
// association. A live (non-simulated) Invoke returns immediately with a
// StartHandle; the operation runs until its round is delivered to CommitResult by
// the completion bridge.
func (r *reviewOperationStarter) StartReviewOperation(ctx context.Context, req review.OperationStartRequest) (review.OperationStartResult, error) {
	res, err := r.runner.Invoke(ctx, opsrunner.InvokeRequest{
		Target:           opsrunner.TargetRef{Kind: agentops.TargetKind(req.TargetKind), ID: req.TargetID},
		Operation:        agentops.OperationID(req.Operation),
		OperationVersion: pinnedReviewOperationVersion,
		CallerInputs:     nonEmptyCallerInputs(req.CallerInputs),
		IdempotencyKey:   req.IdempotencyKey,
		RequestedBy:      req.RequestedBy,
	})
	if err != nil {
		return review.OperationStartResult{}, err
	}
	out := review.OperationStartResult{
		WorkflowID:  res.WorkflowInstanceID,
		ExecutionID: res.ExecutionID,
	}
	if res.StartHandle != nil {
		out.RunID = res.StartHandle.RunID
	} else {
		// Synchronous (simulation/test) path: no live handle; surface the execution
		// id as the correlation handle.
		out.RunID = res.ExecutionID
	}
	return out, nil
}

// initiativeReviewOperationStarter adapts the runner to the initiativereview
// service's OperationStarter seam (initiative-review against the initiative
// target).
type initiativeReviewOperationStarter struct {
	runner *opsrunner.Runner
}

// StartInitiativeReviewOperation invokes the initiative-review operation and
// returns the live run association.
func (r *initiativeReviewOperationStarter) StartInitiativeReviewOperation(ctx context.Context, req initiativereview.OperationStartRequest) (initiativereview.OperationStartResult, error) {
	res, err := r.runner.Invoke(ctx, opsrunner.InvokeRequest{
		Target:           opsrunner.TargetRef{Kind: agentops.TargetKind(req.TargetKind), ID: req.TargetID},
		Operation:        agentops.OperationID(req.Operation),
		OperationVersion: pinnedReviewOperationVersion,
		IdempotencyKey:   req.IdempotencyKey,
		RequestedBy:      req.RequestedBy,
	})
	if err != nil {
		return initiativereview.OperationStartResult{}, err
	}
	out := initiativereview.OperationStartResult{
		WorkflowID:  res.WorkflowInstanceID,
		ExecutionID: res.ExecutionID,
	}
	if res.StartHandle != nil {
		out.RunID = res.StartHandle.RunID
	} else {
		out.RunID = res.ExecutionID
	}
	return out, nil
}

// nonEmptyCallerInputs drops empty/blank caller-input strings so the pinned
// execution snapshot carries exactly the operator context supplied.
func nonEmptyCallerInputs(inputs map[string]any) map[string]any {
	var out map[string]any
	for k, v := range inputs {
		s, ok := v.(string)
		if !ok {
			if out == nil {
				out = map[string]any{}
			}
			out[k] = v
			continue
		}
		if strings.TrimSpace(s) == "" {
			continue
		}
		if out == nil {
			out = map[string]any{}
		}
		out[k] = strings.TrimSpace(s)
	}
	return out
}
