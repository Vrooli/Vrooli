// Package availability owns the closed vocabulary used when evidence is
// present, absent, degraded, or otherwise not safe to interpret as a result.
package availability

// State is deliberately closed by constants. New values require an explicit
// vocabulary decision instead of another ad-hoc string at a call site.
type State string

const (
	Available    State = "available"
	Unavailable  State = "unavailable"
	Degraded     State = "degraded"
	Unobserved   State = "unobserved"
	Unknown      State = "unknown"
	Resolved     State = "resolved"
	PolicyAbsent State = "policy_absent"
	Oversized    State = "oversized"
	NotCaptured  State = "not_captured"
	External     State = "external"
	Empty        State = "empty"
	Complete     State = "complete"
	Unreliable   State = "unreliable"
)

// Availability couples a closed state with the bounded reason that explains
// it. A blank reason is valid only when the state is self-explanatory.
type Availability struct {
	State  State  `json:"state"`
	Reason string `json:"reason,omitempty"`
}

func New(state State, reason string) Availability {
	return Availability{State: state, Reason: reason}
}
