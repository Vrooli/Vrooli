package objectives

import "errors"

// Sentinel errors returned by the service. Callers use errors.Is to branch.
var (
	ErrNotFound         = errors.New("objectives: not found")
	ErrConflict         = errors.New("objectives: revision conflict")
	ErrDuplicateLink    = errors.New("objectives: duplicate attachment")
	ErrUnknownObjective = errors.New("objectives: unknown objective")
	ErrUnknownTeam      = errors.New("objectives: unknown team")
	ErrInvalidClass     = errors.New("objectives: invalid class")
	ErrInvalidRole      = errors.New("objectives: invalid role")
	ErrInvalidCoverage  = errors.New("objectives: invalid coverage")
	ErrCycle            = errors.New("objectives: relation cycle")
	ErrActiveReferences = errors.New("objectives: objective has active references")
	ErrOrderMismatch    = errors.New("objectives: order does not match attachments")
	ErrImportConflict   = errors.New("objectives: import conflict")
)

// ValidationError wraps a sentinel with the offending field and a human detail.
type ValidationError struct {
	Err   error
	Field string
	Msg   string
}

func (e *ValidationError) Error() string {
	if e.Field == "" {
		return e.Err.Error() + ": " + e.Msg
	}
	return e.Err.Error() + ": " + e.Field + " " + e.Msg
}

// Unwrap exposes the sentinel to errors.Is.
func (e *ValidationError) Unwrap() error { return e.Err }
