package pipeline

import (
	"context"
	"errors"
	"fmt"
	"net"
	"os"
	"syscall"

	"audio-tools/internal/sttbackend"
)

// EnvAutoEnsure is the operator toggle for on-demand STT backend recovery (plan
// L1, default ON). Set STT_AUTO_ENSURE=0/false/off to disable auto-start so a
// down backend returns the typed error immediately without a `resource start`.
const EnvAutoEnsure = "STT_AUTO_ENSURE"

// STTBackendError is a typed, USER-SAFE backend-down error. Its Message never
// contains transport detail (no raw dial string), so the streaming-error event
// and the Connect handler can surface it verbatim. Unwrap exposes the
// ErrSTTBackendUnavailable sentinel so callers classify via errors.Is/As.
//
// Transient distinguishes the Connect mapping (plan L2): a backend that is
// starting (auto-ensure ran, the retry may yet succeed) is transient →
// CodeUnavailable ("try again shortly"); a backend that could not be started or
// where auto-ensure is off requires operator action → CodeFailedPrecondition.
type STTBackendState string

const (
	STTBackendStateUnknown     STTBackendState = ""
	STTBackendStateStarting    STTBackendState = "starting"
	STTBackendStateUnavailable STTBackendState = "unavailable"
	STTBackendStateDegraded    STTBackendState = "degraded"
)

type STTBackendError struct {
	Resource  string
	Transient bool
	State     STTBackendState
	Message   string
}

func (e *STTBackendError) Error() string { return e.Message }

// Unwrap lets errors.Is(err, ErrSTTBackendUnavailable) match through the chain.
func (e *STTBackendError) Unwrap() error { return ErrSTTBackendUnavailable }

// newBackendStarting is the transient case: auto-ensure attempted recovery but
// the backend is not serving yet. Retrying shortly is the right action.
func newBackendStarting(resource string) *STTBackendError {
	return &STTBackendError{
		Resource:  resource,
		Transient: true,
		State:     STTBackendStateStarting,
		Message:   fmt.Sprintf("Speech backend (%s) is starting — please try again in a moment.", resource),
	}
}

// newBackendNeedsOperator is the non-transient case: auto-ensure is disabled or
// the start failed. The message carries the exact remediation command.
func newBackendNeedsOperator(resource string) *STTBackendError {
	return &STTBackendError{
		Resource:  resource,
		Transient: false,
		State:     STTBackendStateUnavailable,
		Message:   fmt.Sprintf("Speech backend (%s) is unavailable — run `vrooli resource start %s` and try again.", resource, resource),
	}
}

// newBackendDegraded is the slow/wedged case: the backend is not proven down
// (so do not start/restart it here), but ASR did not complete within its
// bounded request deadline. Clients should report degraded/retrying rather
// than "resource stopped" or a raw timeout.
func newBackendDegraded(resource string) *STTBackendError {
	return &STTBackendError{
		Resource:  resource,
		Transient: true,
		State:     STTBackendStateDegraded,
		Message:   fmt.Sprintf("Speech backend (%s) is slow or degraded — please try again shortly.", resource),
	}
}

func (e *STTBackendError) EffectiveState() STTBackendState {
	if e == nil {
		return STTBackendStateUnknown
	}
	if e.State != STTBackendStateUnknown {
		return e.State
	}
	if e.Transient {
		return STTBackendStateStarting
	}
	return STTBackendStateUnavailable
}

func isBackendTimeout(err error) bool {
	if err == nil || errors.Is(err, context.Canceled) {
		return false
	}
	if errors.Is(err, context.DeadlineExceeded) || errors.Is(err, os.ErrDeadlineExceeded) {
		return true
	}
	var netErr net.Error
	return errors.As(err, &netErr) && netErr.Timeout()
}

// isBackendDown reports whether err is a transport-level "backend is not
// listening" failure — connection refused, no route, or a dial error — as
// opposed to a timeout/cancellation (which is NOT a down backend). Classification
// is typed (errors.Is / net.OpError / syscall errno), never a string compare
// (plan §12 prohibited pattern; mirrors the ErrFfmpegExec sentinel precedent).
func isBackendDown(err error) bool {
	if err == nil {
		return false
	}
	// Timeouts and cancellations are a different failure mode — never treat them
	// as a down backend (the backend may be up but slow / the client went away).
	if errors.Is(err, os.ErrDeadlineExceeded) {
		return false
	}
	var netErr net.Error
	if errors.As(err, &netErr) && netErr.Timeout() {
		return false
	}
	if errors.Is(err, syscall.ECONNREFUSED) ||
		errors.Is(err, syscall.ECONNRESET) ||
		errors.Is(err, syscall.EHOSTUNREACH) ||
		errors.Is(err, syscall.ENETUNREACH) {
		return true
	}
	// A dial-phase *net.OpError (e.g. "dial tcp 127.0.0.1:8090: connect:
	// connection refused") is the canonical incident signature.
	var opErr *net.OpError
	if errors.As(err, &opErr) && opErr.Op == "dial" {
		return true
	}
	return false
}

// AutoEnsureEnabledFromEnv resolves the STT_AUTO_ENSURE toggle from the process
// environment (default ON). Bootstrap passes the result to Service.SetAutoEnsure.
func AutoEnsureEnabledFromEnv() bool { return autoEnsureEnabled(os.Getenv) }

// autoEnsureEnabled resolves the STT_AUTO_ENSURE toggle (default ON; only an
// explicit 0/false/off/no disables it).
func autoEnsureEnabled(get func(string) string) bool {
	switch get(EnvAutoEnsure) {
	case "0", "false", "FALSE", "off", "OFF", "no", "NO":
		return false
	default:
		return true
	}
}

// ensure the sttbackend.Ensurer interface is the seam the Service depends on.
var _ sttbackend.Ensurer = (*sttbackend.CLIEnsurer)(nil)
