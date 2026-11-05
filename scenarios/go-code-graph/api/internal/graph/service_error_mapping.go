package graph

import (
	"errors"

	"connectrpc.com/connect"
)

// ErrorToConnectCode translates a Service error to its canonical
// Connect error code. Unknown errors map to CodeInternal — callers
// never depend on raw implementation details.
//
// Mapping (per plan §8.2):
//
//	ExtractErrorNoGoMod              → InvalidArgument
//	ExtractErrorMultipleGoMod        → InvalidArgument
//	ExtractErrorWorkspaceUnsupported → Unimplemented
//	ExtractErrorPathUnreadable       → NotFound
//	ExtractErrorInvalidInput         → InvalidArgument
//	ExtractErrorInternal             → Internal
func ErrorToConnectCode(err error) connect.Code {
	if err == nil {
		return connect.Code(0)
	}
	var ex ExtractError
	if errors.As(err, &ex) {
		switch ex.Kind {
		case ExtractErrorNoGoMod, ExtractErrorMultipleGoMod, ExtractErrorInvalidInput:
			return connect.CodeInvalidArgument
		case ExtractErrorWorkspaceUnsupported:
			return connect.CodeUnimplemented
		case ExtractErrorPathUnreadable:
			return connect.CodeNotFound
		case ExtractErrorInternal:
			return connect.CodeInternal
		default:
			return connect.CodeInternal
		}
	}
	return connect.CodeInternal
}

// ToConnectError wraps err in a connect.Error tagged with the code
// returned by ErrorToConnectCode. nil in, nil out.
func ToConnectError(err error) error {
	if err == nil {
		return nil
	}
	return connect.NewError(ErrorToConnectCode(err), err)
}
