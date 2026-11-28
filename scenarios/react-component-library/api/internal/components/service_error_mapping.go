package components

import (
	"errors"

	"connectrpc.com/connect"
)

type serviceErrorMapping struct {
	code  connect.Code
	match func(error) (error, bool)
}

var serviceErrorMappings = [...]serviceErrorMapping{
	{code: connect.CodeNotFound, match: typedServiceErrorTarget[ErrComponentNotFound]},
	{code: connect.CodeInvalidArgument, match: typedServiceErrorTarget[ErrInvalidHeader]},
	{code: connect.CodeAlreadyExists, match: typedServiceErrorTarget[ErrComponentAlreadyExists]},
	{code: connect.CodeInvalidArgument, match: typedServiceErrorTarget[ErrPathEscape]},
	{code: connect.CodeFailedPrecondition, match: typedServiceErrorTarget[ErrContentConflict]},
	{code: connect.CodeFailedPrecondition, match: typedServiceErrorTarget[ErrReleasedVersionMutated]},
	{code: connect.CodeFailedPrecondition, match: typedServiceErrorTarget[ErrParityWaiverRequired]},
	{code: connect.CodeFailedPrecondition, match: typedServiceErrorTarget[ErrHarvestBehaviorLoss]},
	{code: connect.CodeFailedPrecondition, match: typedServiceErrorTarget[ErrVersionCheckFailed]},
}

func typedServiceErrorTarget[T error](err error) (error, bool) {
	var target T
	if errors.As(err, &target) {
		return target, true
	}
	return nil, false
}

// ToConnectError translates components-domain sentinels into Connect's typed
// error model. Unknown errors map to internal so callers never depend on raw
// storage details.
func ToConnectError(err error) error {
	if err == nil {
		return nil
	}
	for _, mapping := range serviceErrorMappings {
		if target, ok := mapping.match(err); ok {
			return connect.NewError(mapping.code, target)
		}
	}
	return connect.NewError(connect.CodeInternal, err)
}
