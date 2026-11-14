package audits

import "fmt"

// AuditEvent names a transition in the persisted snapshot-audit lifecycle.
type AuditEvent string

const (
	AuditStart    AuditEvent = "start"
	AuditComplete AuditEvent = "complete"
	AuditFail     AuditEvent = "fail"
)

// Transition returns the legal next state for an audit event.
func Transition(from AuditStatus, event AuditEvent) (AuditStatus, bool) {
	switch from {
	case AuditRequested, AuditRunning:
	case AuditCompleted, AuditFailed:
		return from, false
	default:
		return from, false
	}
	switch event {
	case AuditStart:
		if from == AuditRequested {
			return AuditRunning, true
		}
	case AuditComplete:
		if from == AuditRunning {
			return AuditCompleted, true
		}
	case AuditFail:
		if from == AuditRequested || from == AuditRunning {
			return AuditFailed, true
		}
	}
	return from, false
}

// TransitionStatus is the only in-memory Audit.Status mutation owner.
func (a *Audit) TransitionStatus(event AuditEvent) error {
	if a == nil {
		return fmt.Errorf("cannot transition a nil audit")
	}
	next, ok := Transition(a.Status, event)
	if !ok {
		return fmt.Errorf("invalid audit transition %q + %q", a.Status, event)
	}
	a.Status = next
	return nil
}

// IsTerminal reports whether an audit has reached a final result.
func IsTerminal(status AuditStatus) bool {
	return status == AuditCompleted || status == AuditFailed
}
