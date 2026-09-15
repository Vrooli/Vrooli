package codesigning

import "errors"

// ErrProfileNotFound is returned when a signing request references no profile.
var ErrProfileNotFound = errors.New("profile not found")
