package drills

import "fmt"

type Event string

const (
	EventStart  Event = "start"
	EventVerify Event = "verify"
	EventFail   Event = "fail"
)

func Transition(from Status, event Event) (Status, bool) {
	switch from {
	case StatusRequested, StatusRunning, StatusVerified, StatusFailed:
	default:
		return from, false
	}
	switch event {
	case EventStart:
		if from == StatusRequested {
			return StatusRunning, true
		}
	case EventVerify:
		if from == StatusRunning {
			return StatusVerified, true
		}
	case EventFail:
		if from == StatusRequested || from == StatusRunning {
			return StatusFailed, true
		}
	}
	return from, false
}

// TransitionStatus is the sole in-memory owner for recovery-drill lifecycle state.
func (d *Drill) TransitionStatus(event Event) error {
	if d == nil {
		return fmt.Errorf("cannot transition a nil recovery drill")
	}
	next, ok := Transition(d.Status, event)
	if !ok {
		return fmt.Errorf("invalid recovery-drill transition %q + %q", d.Status, event)
	}
	d.Status = next
	return nil
}

func IsActive(status Status) bool {
	return status == StatusRequested || status == StatusRunning
}

func IsTerminal(status Status) bool {
	return status == StatusVerified || status == StatusFailed
}

func ValidStatus(status Status) bool {
	switch status {
	case StatusRequested, StatusRunning, StatusVerified, StatusFailed:
		return true
	default:
		return false
	}
}
