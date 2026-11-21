package audiotools

import (
	"context"
	"errors"

	"connectrpc.com/connect"
)

// Common error envelope mapping. Consumers (audioports.Remote*) map these to
// canonical audioports errors so the audio-tools wire shape never leaks into
// web-console's domain code.

var (
	// ErrUnavailable is the canonical "audio-tools is not reachable / not
	// configured / required dependency missing" surface. Consumers should
	// degrade gracefully on this.
	ErrUnavailable = errors.New("audiotools: audio-tools service unavailable")

	// ErrTimeout means audio-tools was reachable but the requested operation
	// did not finish inside the caller's deadline.
	ErrTimeout = errors.New("audiotools: audio-tools operation timed out")

	// ErrInsufficientCredits propagates a Vrooli-tier billing failure.
	// Consumers MUST NOT silently retry or fall back to local — there's no
	// local fallback in web-console post-adoption.
	ErrInsufficientCredits = errors.New("audiotools: insufficient credits")

	// ErrInvalidArgument is a 4xx-class argument failure surfaced from
	// audio-tools (e.g., unknown BYOK provider).
	ErrInvalidArgument = errors.New("audiotools: invalid argument")

	// ErrFailedPrecondition means audio-tools was reachable but the selected
	// local dependency/configuration is not ready, such as a missing Ollama
	// summarizer model.
	ErrFailedPrecondition = errors.New("audiotools: failed precondition")
)

// NormalizeError maps a raw Connect error to one of the canonical audiotools
// error sentinels. Consumers can errors.Is(...) against these without
// importing connectrpc.com/connect.
func NormalizeError(err error) error {
	if err == nil {
		return nil
	}
	if errors.Is(err, context.DeadlineExceeded) {
		return ErrTimeout
	}
	var connectErr *connect.Error
	if errors.As(err, &connectErr) {
		switch connectErr.Code() {
		case connect.CodeDeadlineExceeded:
			return ErrTimeout
		case connect.CodeFailedPrecondition:
			return ErrFailedPrecondition
		case connect.CodeUnavailable:
			return ErrUnavailable
		case connect.CodeResourceExhausted:
			return ErrInsufficientCredits
		case connect.CodeInvalidArgument:
			return ErrInvalidArgument
		}
	}
	return err
}
