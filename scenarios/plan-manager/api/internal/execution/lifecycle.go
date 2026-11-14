package execution

import flow "plan-manager/internal/execution/flow/generated"

// The flow contract owns the state/event table. Readiness means every phase is
// done; completion means its durable final handoff committed successfully.
type ExecutionLifecycleState = flow.ExecutionStatus

const (
	ExecutionLifecycleActive    = flow.ExecutionActive
	ExecutionLifecycleReady     = flow.ExecutionReady
	ExecutionLifecycleCompleted = flow.ExecutionCompleted
	ExecutionLifecycleAbandoned = flow.ExecutionAbandoned
)

func (e Execution) PhasesComplete() bool {
	return e.LifecycleState == ExecutionLifecycleReady || e.LifecycleState == ExecutionLifecycleCompleted
}
func (e Execution) IsActive() bool {
	return e.LifecycleState == ExecutionLifecycleActive || e.LifecycleState == ExecutionLifecycleReady
}

func (e *Execution) transition(event flow.ExecutionEvent) error {
	next, err := flow.TransitionExecutionStatus(e.LifecycleState, event)
	if err != nil {
		return ErrInvalidTransition{ExecutionID: e.ID, State: string(e.LifecycleState), Event: string(event)}
	}
	e.LifecycleState = next
	return nil
}

func (e *Execution) movePhase(phaseID string) error {
	event := flow.ExecutionPhasesOpen
	if phaseID == "" {
		event = flow.ExecutionPhasesReady
	}
	if err := e.transition(event); err != nil {
		return err
	}
	e.CurrentPhaseID = phaseID
	return nil
}
