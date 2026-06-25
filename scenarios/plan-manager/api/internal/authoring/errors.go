package authoring

import (
	"errors"
	"fmt"

	"connectrpc.com/connect"

	internalplans "plan-manager/internal/plans"
)

// ErrSessionNotFound is returned when a session id does not resolve to a stored
// authoring session.
type ErrSessionNotFound struct{ ID string }

func (e ErrSessionNotFound) Error() string {
	return fmt.Sprintf("authoring session %q not found", e.ID)
}

// ErrSectionNotFound is returned when a section key is not part of a session.
type ErrSectionNotFound struct {
	SessionID  string
	SectionKey string
}

func (e ErrSectionNotFound) Error() string {
	return fmt.Sprintf("section %q not found on authoring session %q", e.SectionKey, e.SessionID)
}

// ErrInvalidSession is returned when a session fails structural validation at the
// service boundary (e.g. an empty title at StartSession).
type ErrInvalidSession struct{ Reason string }

func (e ErrInvalidSession) Error() string {
	return fmt.Sprintf("invalid authoring session: %s", e.Reason)
}

// ErrStructureGate is returned when Finalize is attempted while the structure
// gate still has violations — the session cannot become a plan yet.
type ErrStructureGate struct{ Violations []StructureViolation }

func (e ErrStructureGate) Error() string {
	return fmt.Sprintf("structure gate not satisfied: %d violation(s)", len(e.Violations))
}

// ToConnectError translates authoring/plans sentinels into Connect's typed error
// model. Unknown errors map to internal.
func ToConnectError(err error) error {
	if err == nil {
		return nil
	}
	var sessionNotFound ErrSessionNotFound
	if errors.As(err, &sessionNotFound) {
		return connect.NewError(connect.CodeNotFound, sessionNotFound)
	}
	var sectionNotFound ErrSectionNotFound
	if errors.As(err, &sectionNotFound) {
		return connect.NewError(connect.CodeNotFound, sectionNotFound)
	}
	var invalid ErrInvalidSession
	if errors.As(err, &invalid) {
		return connect.NewError(connect.CodeInvalidArgument, invalid)
	}
	var gate ErrStructureGate
	if errors.As(err, &gate) {
		return connect.NewError(connect.CodeFailedPrecondition, gate)
	}
	var planInvalid internalplans.ErrInvalidPlan
	if errors.As(err, &planInvalid) {
		return connect.NewError(connect.CodeInvalidArgument, planInvalid)
	}
	return connect.NewError(connect.CodeInternal, err)
}
