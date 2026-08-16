package entitlement

import "errors"

var (
	errInvalidEmail         = errors.New("invalid email format")
	errInvalidTier          = errors.New("unknown subscription tier")
	errInvalidApiSource     = errors.New("source must be one of: production, local, disabled")
	errSettingsUnavailable  = errors.New("settings storage unavailable")
	errCreditsUnavailable   = errors.New("usage tracking is not available")
	errUserRequired         = errors.New("user identity is required")
	errLocalControlsRemoved = errors.New("local entitlement controls are removed; use a signed LPBS lease")
	errApiSourceRemoved     = errors.New("local entitlement API sources are removed; use LPBS")
)
