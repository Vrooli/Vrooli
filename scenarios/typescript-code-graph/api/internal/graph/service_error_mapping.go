package graph

import (
	"context"
	"errors"

	"connectrpc.com/connect"

	"typescript-code-graph/internal/sidecar"
)

// ErrorToConnectCode translates a Service error to its canonical
// Connect error code. Unknown errors map to CodeInternal — callers
// never depend on raw implementation details.
//
// Mapping (per plan §8.2):
//
//	ExtractErrorNoTsConfig            → InvalidArgument
//	ExtractErrorMultipleTsConfig      → InvalidArgument
//	ExtractErrorWorkspaceUnsupported  → Unimplemented
//	ExtractErrorPathUnreadable        → NotFound
//	ExtractErrorInvalidInput          → InvalidArgument
//	ExtractErrorSidecarUnavailable    → Unavailable
//	ExtractErrorSidecarTimeout        → DeadlineExceeded
//	ExtractErrorInternal              → Internal
func ErrorToConnectCode(err error) connect.Code {
	if err == nil {
		return connect.Code(0)
	}
	var ex ExtractError
	if errors.As(err, &ex) {
		switch ex.Kind {
		case ExtractErrorNoTsConfig,
			ExtractErrorMultipleTsConfig,
			ExtractErrorInvalidInput:
			return connect.CodeInvalidArgument
		case ExtractErrorWorkspaceUnsupported:
			return connect.CodeUnimplemented
		case ExtractErrorPathUnreadable:
			return connect.CodeNotFound
		case ExtractErrorSidecarUnavailable:
			return connect.CodeUnavailable
		case ExtractErrorSidecarTimeout:
			return connect.CodeDeadlineExceeded
		case ExtractErrorInternal:
			return connect.CodeInternal
		default:
			return connect.CodeInternal
		}
	}
	if errors.Is(err, context.DeadlineExceeded) {
		return connect.CodeDeadlineExceeded
	}
	if errors.Is(err, context.Canceled) {
		return connect.CodeCanceled
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

// fromSidecarError translates a sidecar.* sentinel or typed error into
// the domain's ExtractError. Used by Service.Extract so the handler
// only ever sees domain errors.
func fromSidecarError(path string, err error) error {
	if err == nil {
		return nil
	}
	if errors.Is(err, sidecar.ErrSidecarUnavailable) ||
		errors.Is(err, sidecar.ErrSidecarPermanentlyUnhealthy) {
		return ExtractError{Kind: ExtractErrorSidecarUnavailable, Path: path, Cause: err}
	}
	if errors.Is(err, sidecar.ErrSidecarTimeout) {
		return ExtractError{Kind: ExtractErrorSidecarTimeout, Path: path, Cause: err}
	}
	if errors.Is(err, context.DeadlineExceeded) {
		return ExtractError{Kind: ExtractErrorSidecarTimeout, Path: path, Cause: err}
	}
	if errors.Is(err, context.Canceled) {
		return ExtractError{Kind: ExtractErrorInternal, Path: path, Cause: err}
	}
	var ee *sidecar.ExtractError
	if errors.As(err, &ee) {
		return mapExtractErrorKind(path, ee)
	}
	return ExtractError{Kind: ExtractErrorInternal, Path: path, Cause: err}
}

func mapExtractErrorKind(path string, ee *sidecar.ExtractError) ExtractError {
	switch ee.Kind {
	case "no_tsconfig_found":
		return ExtractError{Kind: ExtractErrorNoTsConfig, Path: path, Message: ee.Message, Cause: ee}
	case "multiple_tsconfig_files":
		return ExtractError{Kind: ExtractErrorMultipleTsConfig, Path: path, Message: ee.Message, Cause: ee}
	case "workspace_unsupported":
		return ExtractError{Kind: ExtractErrorWorkspaceUnsupported, Path: path, Message: ee.Message, Cause: ee}
	case "path_unreadable":
		return ExtractError{Kind: ExtractErrorPathUnreadable, Path: path, Message: ee.Message, Cause: ee}
	case "cancelled":
		return ExtractError{Kind: ExtractErrorInternal, Path: path, Message: ee.Message, Cause: ee}
	case "parse_failure":
		// Parse failures are normally surfaced as warnings; if one
		// escapes as a top-level error, treat as Internal so the
		// caller sees CodeInternal (not InvalidArgument — the input
		// isn't malformed, the sidecar just couldn't make sense of
		// the project).
		return ExtractError{Kind: ExtractErrorInternal, Path: path, Message: ee.Message, Cause: ee}
	default:
		return ExtractError{Kind: ExtractErrorInternal, Path: path, Message: ee.Message, Cause: ee}
	}
}
