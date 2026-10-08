package artifacts

import (
	"errors"

	"connectrpc.com/connect"
)

// ToConnectError translates artifacts domain sentinels into Connect codes.
// Path-traversal maps to FailedPrecondition since the request was syntactically
// fine but the target resolved outside the safe root.
func ToConnectError(err error) error {
	if err == nil {
		return nil
	}
	if errors.Is(err, ErrFlowNotFound) {
		return connect.NewError(connect.CodeNotFound, err)
	}
	if errors.Is(err, ErrPathTraversal) {
		return connect.NewError(connect.CodeFailedPrecondition, err)
	}
	return connect.NewError(connect.CodeInternal, err)
}
