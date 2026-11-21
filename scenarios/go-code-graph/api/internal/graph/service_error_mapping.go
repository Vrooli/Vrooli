package graph

import (
	"errors"

	"connectrpc.com/connect"
	"github.com/vrooli/api-core/gograph"
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
	var ex gograph.ExtractError
	if errors.As(err, &ex) {
		switch ex.Kind {
		case gograph.ExtractErrorNoGoMod, gograph.ExtractErrorMultipleGoMod, gograph.ExtractErrorInvalidInput:
			return connect.CodeInvalidArgument
		case gograph.ExtractErrorWorkspaceUnsupported:
			return connect.CodeUnimplemented
		case gograph.ExtractErrorPathUnreadable:
			return connect.CodeNotFound
		case gograph.ExtractErrorInternal:
			return connect.CodeInternal
		default:
			return connect.CodeInternal
		}
	}
	return connect.CodeInternal
}
