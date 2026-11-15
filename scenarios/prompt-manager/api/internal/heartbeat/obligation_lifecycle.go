package heartbeat

import "fmt"

var obligationTransitions = map[string]map[string]struct{}{
	ObligationRunning: {
		ObligationDispatchUncertain: {}, ObligationOwnerUnreachable: {},
		ObligationPaused: {}, ObligationApprovalPending: {},
		ObligationAdmissionDeferred: {}, ObligationNeedsReview: {},
		ObligationWatcherUnavailable: {}, ObligationPreparationInterrupted: {},
	},
	ObligationDispatchUncertain: {
		ObligationDispatchCanceled: {}, ObligationOwnerUnreachable: {},
		ObligationPaused: {}, ObligationApprovalPending: {},
		ObligationAdmissionDeferred: {}, ObligationNeedsReview: {},
		ObligationWatcherUnavailable: {}, ObligationRunning: {},
	},
	ObligationOwnerUnreachable: {
		ObligationDispatchUncertain: {}, ObligationPaused: {},
		ObligationNeedsReview: {}, ObligationWatcherUnavailable: {},
		ObligationRunning: {},
	},
	ObligationPaused: {
		ObligationOwnerUnreachable: {}, ObligationNeedsReview: {},
		ObligationWatcherUnavailable: {}, ObligationRunning: {},
	},
	ObligationApprovalPending: {
		ObligationDispatchUncertain: {}, ObligationAdmissionDeferred: {},
		ObligationOwnerUnreachable: {}, ObligationPaused: {},
		ObligationNeedsReview: {}, ObligationWatcherUnavailable: {},
		ObligationRunning: {},
	},
	ObligationAdmissionDeferred: {
		ObligationDispatchUncertain: {}, ObligationOwnerUnreachable: {},
		ObligationPaused: {}, ObligationNeedsReview: {},
		ObligationWatcherUnavailable: {}, ObligationRunning: {},
	},
	ObligationNeedsReview: {
		ObligationOwnerUnreachable: {}, ObligationPaused: {},
		ObligationWatcherUnavailable: {}, ObligationRunning: {},
	},
	ObligationWatcherUnavailable: {
		ObligationDispatchUncertain: {}, ObligationOwnerUnreachable: {},
		ObligationPaused: {}, ObligationNeedsReview: {},
		ObligationRunning: {},
	},
	ObligationPreparationInterrupted: {ObligationRunning: {}},
}

// transitionObligationState owns legal changes to the persisted heartbeat slot
// obligation. Owner observations may resolve uncertainty directly to a known
// live/paused/review state; dispatch tombstones remain immutable.
func transitionObligationState(from, to string) error {
	if !validObligationState(from) {
		return fmt.Errorf("unknown heartbeat obligation state %q", from)
	}
	if !validObligationState(to) {
		return fmt.Errorf("unknown heartbeat obligation state %q", to)
	}
	if from == to {
		return nil
	}
	if _, ok := obligationTransitions[from][to]; !ok {
		return fmt.Errorf("invalid heartbeat obligation transition %q -> %q", from, to)
	}
	return nil
}

func validObligationState(state string) bool {
	switch state {
	case ObligationRunning, ObligationDispatchUncertain, ObligationDispatchCanceled,
		ObligationOwnerUnreachable, ObligationPaused, ObligationApprovalPending,
		ObligationAdmissionDeferred, ObligationNeedsReview,
		ObligationWatcherUnavailable, ObligationPreparationInterrupted:
		return true
	default:
		return false
	}
}

func (entry *runningEntry) transition(to string) error {
	if entry == nil {
		return fmt.Errorf("heartbeat running entry is nil")
	}
	if err := transitionObligationState(entry.State, to); err != nil {
		return err
	}
	entry.State = to
	return nil
}

func (run *ActiveRun) transition(to string) error {
	if run == nil {
		return fmt.Errorf("heartbeat active run is nil")
	}
	if err := transitionObligationState(run.State, to); err != nil {
		return err
	}
	run.State = to
	return nil
}

func (run *persistedRun) transition(to string) error {
	if run == nil {
		return fmt.Errorf("persisted heartbeat run is nil")
	}
	if err := transitionObligationState(run.State, to); err != nil {
		return err
	}
	run.State = to
	return nil
}
