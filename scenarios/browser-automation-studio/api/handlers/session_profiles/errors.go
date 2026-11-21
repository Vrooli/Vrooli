package session_profiles

import "errors"

var (
	errInvalidProfileID = errors.New("invalid profile id")
	errProfileNotFound  = errors.New("session profile not found")
	errNothingToUpdate  = errors.New("at least one of 'name' or 'browser_profile' must be provided")
	errInvalidBrowser   = errors.New("invalid browser profile")
)
