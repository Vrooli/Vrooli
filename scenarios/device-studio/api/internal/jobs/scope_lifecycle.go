package jobs

// RunScopeState is the lifecycle of a durable render-workflow admission scope.
// Child admission stops at closing; cancellation or completion is terminal.
func (state RunScopeState) Valid() bool {
	switch state {
	case RunScopeActive, RunScopeClosing, RunScopeCancelled, RunScopeCompleted:
		return true
	default:
		return false
	}
}

func (state RunScopeState) Terminal() bool {
	return state == RunScopeCancelled || state == RunScopeCompleted
}

func (state RunScopeState) CanTransition(next RunScopeState) bool {
	if !state.Valid() || !next.Valid() {
		return false
	}
	if state == next {
		return true
	}
	switch state {
	case RunScopeActive:
		return next == RunScopeClosing || next == RunScopeCancelled || next == RunScopeCompleted
	case RunScopeClosing:
		return next == RunScopeCancelled
	default:
		return false
	}
}

// Transition applies the single legal-edge policy for a workflow admission scope.
func (scope *RunScope) Transition(next RunScopeState) error {
	if scope == nil || !scope.State.CanTransition(next) {
		return ErrInvalidTransition
	}
	scope.State = next
	return nil
}
