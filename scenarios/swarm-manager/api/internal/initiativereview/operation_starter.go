package initiativereview

import "context"

// OperationStarter starts the initiative-review operation through the generic
// operation runner (opsrunner.Runner via an api-layer adapter). It replaces the
// direct agent-manager SpawnInitiative the trigger used to call: the runner
// resolves the bound initiative-review-loop mode, spawns the agent through the
// operating-mode engine's chokepoint, records the durable workflow + provenance +
// run-owner attribution, and returns the live run association the review round
// carries for completion correlation.
type OperationStarter interface {
	StartInitiativeReviewOperation(ctx context.Context, req OperationStartRequest) (OperationStartResult, error)
}

// OperationStartRequest is the trigger's typed request to start one initiative
// review operation against the initiative target. It carries no caller inputs —
// the review context comes from the initiative folder + target adapters.
type OperationStartRequest struct {
	Operation      string
	TargetKind     string
	TargetID       string
	IdempotencyKey string
	RequestedBy    string
}

// OperationStartResult is the live run association the started operation returns.
type OperationStartResult struct {
	RunID       string
	WorkflowID  string
	ExecutionID string
}

// SetOperationStarter injects the operation runner adapter. When set, the trigger
// starts the initiative-review operation through it instead of a direct spawn.
func (s *Service) SetOperationStarter(starter OperationStarter) {
	s.operationStarter = starter
}
