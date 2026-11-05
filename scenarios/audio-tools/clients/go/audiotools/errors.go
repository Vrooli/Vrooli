package audiotools

import (
	"errors"

	"connectrpc.com/connect"
)

// Sentinel errors returned by NormalizeError. Adopters compare via
// errors.Is to avoid coupling to Connect codes directly.
var (
	ErrUnavailable         = errors.New("audiotools: unavailable")
	ErrInsufficientCredits = errors.New("audiotools: insufficient credits")
	ErrInvalidArgument     = errors.New("audiotools: invalid argument")
	ErrInternal            = errors.New("audiotools: internal")
	ErrNotFound            = errors.New("audiotools: not found")
	ErrFailedPrecondition  = errors.New("audiotools: failed precondition")
)

// NormalizeError maps a Connect-RPC error to one of the audiotools
// sentinel errors. Non-Connect errors and unknown codes are wrapped
// under ErrInternal so adopters can branch on errors.Is(err, ErrX)
// uniformly. The original error is preserved as the wrap target so
// errors.Unwrap still returns the underlying cause for logging.
func NormalizeError(err error) error {
	if err == nil {
		return nil
	}
	var connectErr *connect.Error
	if !errors.As(err, &connectErr) {
		return errors.Join(ErrInternal, err)
	}
	switch connectErr.Code() {
	case connect.CodeUnavailable:
		return errors.Join(ErrUnavailable, err)
	case connect.CodeResourceExhausted:
		return errors.Join(ErrInsufficientCredits, err)
	case connect.CodeInvalidArgument:
		return errors.Join(ErrInvalidArgument, err)
	case connect.CodeNotFound:
		return errors.Join(ErrNotFound, err)
	case connect.CodeFailedPrecondition:
		return errors.Join(ErrFailedPrecondition, err)
	default:
		return errors.Join(ErrInternal, err)
	}
}
