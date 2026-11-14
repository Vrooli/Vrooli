package heartbeat

import (
	"errors"
	"strconv"
	"strings"
)

// Dispatch-outcome classification for a run-creation attempt.
//
// A heartbeat dispatch creates a task and then a run at the owner. The unsafe
// case is a run-creation request whose outcome is unknown: the owner may have
// accepted the run even though the caller never saw the response. Releasing the
// member's slot on that error lets the next scheduled tick admit a duplicate
// heartbeat, recreating the stuck-running ambiguity the phase exists to remove.
//
// The seam distinguishes two outcomes:
//
//   - DispatchRejectedError: the owner gave a definitive response (validation,
//     conflict, auth, malformed request) or the request provably never left the
//     process. No run was accepted, so the slot may be released.
//   - DispatchUncertainError: the request may have reached the owner (transport
//     failure, exhausted retries, unreadable success body). The slot must be
//     retained as a visible, recoverable obligation until reconciled.
type DispatchRejectedError struct {
	Cause error
}

func (e *DispatchRejectedError) Error() string {
	if e == nil || e.Cause == nil {
		return "dispatch rejected"
	}
	return "dispatch rejected: " + e.Cause.Error()
}

func (e *DispatchRejectedError) Unwrap() error { return e.Cause }

// DispatchUncertainError marks a dispatch whose outcome the caller cannot prove.
type DispatchUncertainError struct {
	Cause error
}

// dispatchStageFailure retains a bounded owner substage and code across the
// existing dispatch error wrappers. Its text is never persisted or projected.
type dispatchStageFailure struct {
	phase string
	code  string
	cause error
}

func (e *dispatchStageFailure) Error() string {
	if e == nil || e.cause == nil {
		return "task dispatch stage failed"
	}
	return e.cause.Error()
}

func (e *dispatchStageFailure) Unwrap() error { return e.cause }

// AgentManagerHTTPError preserves the owner's typed HTTP status for downstream
// failure ownership decisions while retaining the response's safe message.
type AgentManagerHTTPError struct {
	StatusCode      int
	Code            string
	Message         string
	AdmissionReason string
}

// normalizeAgentManagerAdmissionReason preserves only Agent Manager's
// documented, allowlisted service-admission categories on refusal statuses.
func normalizeAgentManagerAdmissionReason(statusCode int, reason string) string {
	if statusCode != 403 && statusCode != 429 {
		return ""
	}
	switch reason {
	case "grant_limit_reached", "authority_state_unavailable", "authority_unavailable",
		"approval_raiser_unavailable", "approval_raise_failed", "approval_raise_limited",
		"operation_not_admitted":
		return reason
	default:
		return ""
	}
}

// admissionReasonFromError attaches only the safe owner category to retained
// diagnostics. It does not change dispatch classification or authorize retry.
func admissionReasonFromError(err error) string {
	var ownerErr *AgentManagerHTTPError
	if !errors.As(err, &ownerErr) || ownerErr == nil {
		return ""
	}
	return normalizeAgentManagerAdmissionReason(ownerErr.StatusCode, ownerErr.AdmissionReason)
}

func (e *AgentManagerHTTPError) Error() string {
	if e == nil {
		return "agent-manager request failed"
	}
	if e.Message != "" {
		return "agent-manager error: status " + strconv.Itoa(e.StatusCode) + ": " + e.Message
	}
	return "agent-manager error: status " + strconv.Itoa(e.StatusCode)
}

// AgentManagerAuthorizationStatus returns the exact owner authorization status
// without inferring it from an error string.
func AgentManagerAuthorizationStatus(err error) (int, bool) {
	var response *AgentManagerHTTPError
	if errors.As(err, &response) && response != nil && (response.StatusCode == 401 || response.StatusCode == 403) {
		return response.StatusCode, true
	}
	return 0, false
}

func (e *DispatchUncertainError) Error() string {
	if e == nil || e.Cause == nil {
		return "dispatch outcome uncertain"
	}
	return "dispatch outcome uncertain: " + e.Cause.Error()
}

func (e *DispatchUncertainError) Unwrap() error { return e.Cause }

// NewDispatchRejectedError wraps a definitive owner refusal.
func NewDispatchRejectedError(cause error) error { return &DispatchRejectedError{Cause: cause} }

// NewDispatchUncertainError wraps a request whose acceptance cannot be proven.
func NewDispatchUncertainError(cause error) error { return &DispatchUncertainError{Cause: cause} }

// ApprovalPendingError is the exact typed owner response with no side effect at
// the current operation: task-stage creates no task; run-stage creates no run.
// Stage identifies whether the next scheduled attempt restarts task creation or
// replays a persisted run intent.
type ApprovalPendingError struct {
	Stage string
	Cause error
}

func (e *ApprovalPendingError) Error() string {
	if e == nil || e.Cause == nil {
		return "approval pending"
	}
	return "approval pending: " + e.Cause.Error()
}

func (e *ApprovalPendingError) Unwrap() error { return e.Cause }

func NewApprovalPendingError(stage string, cause error) error {
	return &ApprovalPendingError{Stage: stage, Cause: cause}
}

func IsApprovalPending(err error) bool {
	var pending *ApprovalPendingError
	return errors.As(err, &pending)
}

func approvalPendingStage(err error) string {
	var pending *ApprovalPendingError
	if errors.As(err, &pending) && pending != nil {
		return pending.Stage
	}
	return ""
}

// IsDispatchRejected reports whether err is a definitive owner rejection.
func IsDispatchRejected(err error) bool {
	var rejected *DispatchRejectedError
	return errors.As(err, &rejected)
}

// IsDispatchUncertain reports whether err leaves the dispatch outcome unknown.
//
// It is deliberately conservative at the boundaries: a typed uncertain error is
// always uncertain, a typed rejection is always deterministic, and an
// unclassified error is treated as uncertain only when it carries a transport
// marker. Unclassified local errors (for example a request-building failure)
// stay deterministic so they do not strand a member's slot.
func IsDispatchUncertain(err error) bool {
	if err == nil {
		return false
	}
	var uncertain *DispatchUncertainError
	if errors.As(err, &uncertain) {
		return true
	}
	if IsApprovalPending(err) {
		return false
	}
	if IsDispatchRejected(err) {
		return false
	}
	return looksLikeTransportUncertainty(err.Error())
}

func looksLikeTransportUncertainty(message string) bool {
	lower := strings.ToLower(message)
	for _, marker := range []string{
		"connection refused",
		"connection reset",
		"broken pipe",
		"no such host",
		"network is unreachable",
		"i/o timeout",
		"timed out",
		"timeout",
		"deadline exceeded",
		"unexpected eof",
		"unreachable",
	} {
		if strings.Contains(lower, marker) {
			return true
		}
	}
	return false
}
