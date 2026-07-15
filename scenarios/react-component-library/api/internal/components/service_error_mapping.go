package components

import (
	"errors"

	"connectrpc.com/connect"
)

// ToConnectError translates components-domain sentinels into Connect's typed
// error model. Unknown errors map to internal so callers never depend on raw
// storage details.
func ToConnectError(err error) error {
	if err == nil {
		return nil
	}
	var notFound ErrComponentNotFound
	if errors.As(err, &notFound) {
		return connect.NewError(connect.CodeNotFound, notFound)
	}
	var invalidHeader ErrInvalidHeader
	if errors.As(err, &invalidHeader) {
		return connect.NewError(connect.CodeInvalidArgument, invalidHeader)
	}
	var alreadyExists ErrComponentAlreadyExists
	if errors.As(err, &alreadyExists) {
		return connect.NewError(connect.CodeAlreadyExists, alreadyExists)
	}
	var pathEscape ErrPathEscape
	if errors.As(err, &pathEscape) {
		return connect.NewError(connect.CodeInvalidArgument, pathEscape)
	}
	var conflict ErrContentConflict
	if errors.As(err, &conflict) {
		return connect.NewError(connect.CodeFailedPrecondition, conflict)
	}
	var parity ErrParityWaiverRequired
	if errors.As(err, &parity) {
		return connect.NewError(connect.CodeFailedPrecondition, parity)
	}
	var behaviorLoss ErrHarvestBehaviorLoss
	if errors.As(err, &behaviorLoss) {
		return connect.NewError(connect.CodeFailedPrecondition, behaviorLoss)
	}
	return connect.NewError(connect.CodeInternal, err)
}
