package plans

import (
	"errors"

	"connectrpc.com/connect"
)

// ToConnectError translates plans domain sentinels into Connect's typed error
// model. Unknown errors map to internal so callers never depend on raw storage
// or implementation details.
func ToConnectError(err error) error {
	if err == nil {
		return nil
	}
	var invalid ErrInvalidPlan
	if errors.As(err, &invalid) {
		return connect.NewError(connect.CodeInvalidArgument, invalid)
	}
	var notFound ErrPlanNotFound
	if errors.As(err, &notFound) {
		return connect.NewError(connect.CodeNotFound, notFound)
	}
	var phaseNotFound ErrPhaseNotFound
	if errors.As(err, &phaseNotFound) {
		return connect.NewError(connect.CodeNotFound, phaseNotFound)
	}
	var tmplNotFound ErrTemplateNotFound
	if errors.As(err, &tmplNotFound) {
		return connect.NewError(connect.CodeNotFound, tmplNotFound)
	}
	var candidateNotFound ErrCandidateNotFound
	if errors.As(err, &candidateNotFound) {
		return connect.NewError(connect.CodeNotFound, candidateNotFound)
	}
	var stale ErrCandidateStaleBase
	if errors.As(err, &stale) {
		return connect.NewError(connect.CodeAborted, stale)
	}
	var active ErrCandidateExecutionActive
	if errors.As(err, &active) {
		return connect.NewError(connect.CodeFailedPrecondition, active)
	}
	var candidateState ErrCandidateState
	if errors.As(err, &candidateState) {
		return connect.NewError(connect.CodeFailedPrecondition, candidateState)
	}
	return connect.NewError(connect.CodeInternal, err)
}
