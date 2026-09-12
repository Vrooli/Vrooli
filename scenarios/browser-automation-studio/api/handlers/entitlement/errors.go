package entitlement

import "errors"

var (
	errInvalidEmail         = errors.New("invalid email format")
	errCreditsUnavailable   = errors.New("usage tracking is not available")
	errUserRequired         = errors.New("user identity is required")
	errLocalControlsRemoved = errors.New("local entitlement controls are removed; use a signed LPBS lease")
	errAPISourceRemoved     = errors.New("local entitlement API sources are removed; use LPBS")
)
