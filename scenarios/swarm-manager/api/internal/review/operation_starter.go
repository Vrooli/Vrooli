package review

import "context"

// OperationStarter starts a review operation through the generic operation runner
// (opsrunner.Runner via an api-layer adapter). It replaces the direct
// agent-manager SpawnBacklog the review flow used to call: the runner resolves the
// bound operating mode, spawns the agent through the operating-mode engine's
// chokepoint, records the durable workflow + provenance + run-owner attribution,
// and returns the live run association the review round carries for completion
// correlation.
type OperationStarter interface {
	StartReviewOperation(ctx context.Context, req OperationStartRequest) (OperationStartResult, error)
}

// OperationStartRequest is the review flow's typed request to start one review
// operation. CallerInputs carry the operator's typed request context keyed by the
// OPERATION contract's caller-input names (e.g. EVIDENCE_REQUEST); review-round
// carries none (its context comes from the item folder + target adapters).
type OperationStartRequest struct {
	Operation      string
	TargetKind     string
	TargetID       string
	IdempotencyKey string
	CallerInputs   map[string]any
	RequestedBy    string
}

// OperationStartResult is the live run association the started operation returns.
// RunID is the agent-manager run id (stamped on the review round for completion
// correlation); WorkflowID + ExecutionID identify the durable operation execution.
type OperationStartResult struct {
	RunID       string
	WorkflowID  string
	ExecutionID string
}

// SetOperationStarter injects the operation runner adapter. When set, the review
// flow starts review operations through it instead of a direct agent spawn.
func (s *Service) SetOperationStarter(starter OperationStarter) {
	s.operationStarter = starter
}
