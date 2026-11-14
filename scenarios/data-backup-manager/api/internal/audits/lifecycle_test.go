package audits

import "testing"

type auditTransitionPair struct {
	state AuditStatus
	event AuditEvent
}

func TestAuditTransitionMatrixAndInvalidStateImmutability(t *testing.T) {
	states := []AuditStatus{AuditRequested, AuditRunning, AuditCompleted, AuditFailed, "future"}
	events := []AuditEvent{AuditStart, AuditComplete, AuditFail, "future"}
	want := map[auditTransitionPair]AuditStatus{
		{AuditRequested, AuditStart}:  AuditRunning,
		{AuditRequested, AuditFail}:   AuditFailed,
		{AuditRunning, AuditComplete}: AuditCompleted,
		{AuditRunning, AuditFail}:     AuditFailed,
	}
	for _, state := range states {
		for _, event := range events {
			t.Run(string(state)+"/"+string(event), func(t *testing.T) {
				got, ok := Transition(state, event)
				expected, valid := want[auditTransitionPair{state, event}]
				if ok != valid || (valid && got != expected) {
					t.Fatalf("Transition(%q, %q) = %q, %v; want %q, %v", state, event, got, ok, expected, valid)
				}
				a := Audit{Status: state}
				err := a.TransitionStatus(event)
				if valid && err != nil {
					t.Fatalf("valid transition: %v", err)
				}
				if !valid && (err == nil || a.Status != state) {
					t.Fatalf("invalid transition err=%v status=%q; want error and unchanged state", err, a.Status)
				}
			})
		}
	}
}
