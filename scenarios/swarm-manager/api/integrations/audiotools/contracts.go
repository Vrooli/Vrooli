package audiotools

import (
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

	// ErrInsufficientCredits propagates a Vrooli-tier billing failure.
	// Consumers MUST NOT silently retry or fall back to local — there's no
	// local fallback in web-console post-adoption.
	ErrInsufficientCredits = errors.New("audiotools: insufficient credits")

	// ErrInvalidArgument is a 4xx-class argument failure surfaced from
	// audio-tools (e.g., unknown BYOK provider).
	ErrInvalidArgument = errors.New("audiotools: invalid argument")
)

// NormalizeError maps a raw Connect error to one of the canonical audiotools
// error sentinels. Consumers can errors.Is(...) against these without
// importing connectrpc.com/connect.
func NormalizeError(err error) error {
	if err == nil {
		return nil
	}
	var connectErr *connect.Error
	if errors.As(err, &connectErr) {
		switch connectErr.Code() {
		case connect.CodeUnavailable, connect.CodeDeadlineExceeded, connect.CodeFailedPrecondition:
			return ErrUnavailable
		case connect.CodeResourceExhausted:
			return ErrInsufficientCredits
		case connect.CodeInvalidArgument:
			return ErrInvalidArgument
		}
	}
	return err
}
