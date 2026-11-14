package orchestration

import (
	"errors"
	"time"
)

// processTerminationEffects keeps termination ordering testable without
// discovering, signalling, or waiting on any host process.
type processTerminationEffects struct {
	groupID          func(int) int
	graceful         func() error
	alive            func(int) bool
	wait             func(time.Duration)
	killGroup        func(int) error
	confirmGroupExit func(int) error
	killPID          func() error
}

func terminateReconcilerProcess(pid int, effects processTerminationEffects) error {
	// Retain the group before graceful shutdown can remove its leader.
	group := effects.groupID(pid)
	gracefulErr := effects.graceful()
	// A failed graceful request does not prove exit. Give a live leader its
	// grace period even when its native console is unavailable.
	if gracefulErr == nil || effects.alive(pid) {
		effects.wait(500 * time.Millisecond)
	}
	alive := effects.alive(pid)
	var groupErr error
	if group > 0 {
		groupErr = effects.killGroup(group)
		if groupErr == nil {
			return nil
		}
		// Signalling a group that already exited may itself fail. Only the
		// platform's group observation can prove that failure harmless; a dead
		// leader alone cannot establish that its children are gone.
		if confirmErr := effects.confirmGroupExit(group); confirmErr == nil {
			return nil
		} else {
			groupErr = errors.Join(groupErr, confirmErr)
		}
	}
	if !alive {
		// A dead leader with a failed group sweep can still leave children.
		return groupErr
	}
	if err := effects.killPID(); err != nil {
		return errors.Join(gracefulErr, groupErr, err)
	}
	// Fallback stops the leader but does not prove a failed group sweep was
	// successful. Retain that failure for the owning cleanup retry.
	return groupErr
}
