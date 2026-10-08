package signals

import (
	"errors"

	"architecture-cartographer/internal/graph"

	"connectrpc.com/connect"
)

// ErrorToConnectCode translates signals typed sentinels into Connect codes.
func ErrorToConnectCode(err error) connect.Code {
	if err == nil {
		return 0
	}
	var (
		inv             ErrInvalidScoreRequest
		snapshotMissing graph.ErrSnapshotNotFound
	)
	switch {
	case errors.As(err, &inv):
		return connect.CodeInvalidArgument
	case errors.As(err, &snapshotMissing):
		return connect.CodeNotFound
	default:
		return connect.CodeInternal
	}
}
