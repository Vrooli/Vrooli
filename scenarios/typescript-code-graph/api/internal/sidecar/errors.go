package sidecar

import (
	"errors"
	"fmt"
)

// Sentinel errors returned by the supervisor. Domain packages
// classify these into proto/Connect error codes via their own
// service_error_mapping.go — this package does not import proto.
var (
	// ErrSidecarUnavailable is returned when the child is not in a
	// state to accept requests (UNHEALTHY / RESTARTING) or when an
	// in-flight request was drained by a crash.
	ErrSidecarUnavailable = errors.New("sidecar unavailable")

	// ErrSidecarTimeout is returned when the supervisor's own per-call
	// deadline expires (distinct from caller-supplied ctx cancellation,
	// which surfaces as ctx.Err()).
	ErrSidecarTimeout = errors.New("sidecar timeout")

	// ErrSidecarPermanentlyUnhealthy is returned when the restart
	// budget has been exhausted and the supervisor has given up.
	ErrSidecarPermanentlyUnhealthy = errors.New("sidecar permanently unhealthy")
)

// SidecarVersionMismatch is returned when the handshake response
// reports a protocol_version the Go side does not support.
type SidecarVersionMismatch struct {
	Want int
	Got  int
}

func (e *SidecarVersionMismatch) Error() string {
	return fmt.Sprintf("sidecar protocol version mismatch: want %d, got %d", e.Want, e.Got)
}

// ExtractError carries a typed error from the sidecar's extract
// handler. Kind is the string the sidecar emits on the wire — see
// plan §8.4 for the closed set.
type ExtractError struct {
	Kind    string
	Message string
}

func (e *ExtractError) Error() string {
	if e.Message == "" {
		return fmt.Sprintf("extract error: %s", e.Kind)
	}
	return fmt.Sprintf("extract error: %s: %s", e.Kind, e.Message)
}

// RewriteError carries a typed error from the sidecar's
// rewrite_apply handler.
type RewriteError struct {
	Kind    string
	Message string
}

func (e *RewriteError) Error() string {
	if e.Message == "" {
		return fmt.Sprintf("rewrite error: %s", e.Kind)
	}
	return fmt.Sprintf("rewrite error: %s: %s", e.Kind, e.Message)
}

// errorEnvelope is the on-wire shape of a {type:"error",...} response.
// It is decoded inside the read loop before being routed to the
// originating request's channel.
type errorEnvelope struct {
	Kind    string `json:"kind"`
	Message string `json:"message"`
}

// toExtractError converts a wire envelope into an ExtractError.
func (e errorEnvelope) toExtractError() *ExtractError {
	return &ExtractError{Kind: e.Kind, Message: e.Message}
}

// toRewriteError converts a wire envelope into a RewriteError.
func (e errorEnvelope) toRewriteError() *RewriteError {
	return &RewriteError{Kind: e.Kind, Message: e.Message}
}
