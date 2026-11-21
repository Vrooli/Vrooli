package jobs

// AttemptEvent describes a fact that advances an attempt. Repository methods
// own persistence and timestamps; Attempt.Transition owns the legal state graph.
type AttemptEvent string

const (
	AttemptStarted           AttemptEvent = "started"
	AttemptCheckpointedEvent AttemptEvent = "checkpointed"
	AttemptSucceededEvent    AttemptEvent = "succeeded"
	AttemptFailedEvent       AttemptEvent = "failed"
	AttemptCancelledEvent    AttemptEvent = "cancelled"
)

func (state AttemptState) Valid() bool {
	switch state {
	case AttemptQueued, AttemptRunning, AttemptCheckpointed, AttemptSucceeded, AttemptFailed, AttemptCancelled:
		return true
	default:
		return false
	}
}

func (state AttemptState) Terminal() bool {
	return state == AttemptSucceeded || state == AttemptFailed || state == AttemptCancelled
}

// Transition applies the attempt lifecycle's single legal-edge policy. Invalid
// events and unknown persisted states leave the attempt unchanged.
func (attempt *Attempt) Transition(event AttemptEvent) error {
	if attempt == nil || !attempt.State.Valid() {
		return ErrInvalidTransition
	}
	var next AttemptState
	switch event {
	case AttemptStarted:
		if attempt.State != AttemptQueued {
			return ErrInvalidTransition
		}
		next = AttemptRunning
	case AttemptCheckpointedEvent:
		if attempt.State != AttemptRunning {
			return ErrInvalidTransition
		}
		next = AttemptCheckpointed
	case AttemptSucceededEvent:
		if attempt.State != AttemptRunning {
			return ErrInvalidTransition
		}
		next = AttemptSucceeded
	case AttemptFailedEvent:
		if attempt.State != AttemptQueued && attempt.State != AttemptRunning {
			return ErrInvalidTransition
		}
		next = AttemptFailed
	case AttemptCancelledEvent:
		if attempt.State != AttemptQueued && attempt.State != AttemptRunning && attempt.State != AttemptCheckpointed {
			return ErrInvalidTransition
		}
		next = AttemptCancelled
	default:
		return ErrInvalidTransition
	}
	attempt.State = next
	return nil
}
