package recordings

import "errors"

// Sentinels reused across handlers — keep messages stable; UI tests assert on
// connect error codes, not strings, but the strings are still user-visible.
var (
	errProfileIDRequired = errors.New("profile_id is required")
	errDomainRequired    = errors.New("domain is required")
	errOriginRequired    = errors.New("origin is required")
	errNameRequired      = errors.New("name is required")
	errEntryIDRequired   = errors.New("entry_id is required")
	errScopeURLRequired  = errors.New("scope_url is required")
	errURLRequired       = errors.New("url is required")
	errProfileNotFound   = errors.New("session profile not found")
	errTabNotFound       = errors.New("tab not found")
	errNoActiveSession   = errors.New("no active session for this profile")
)
