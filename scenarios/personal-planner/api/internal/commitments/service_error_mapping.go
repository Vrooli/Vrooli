package commitments

import (
	"errors"

	"connectrpc.com/connect"
)

func ToConnectError(err error) error {
	var inv ErrInvalidCommitment
	if errors.As(err, &inv) {
		return connect.NewError(connect.CodeInvalidArgument, err)
	}
	var nf ErrCommitmentNotFound
	if errors.As(err, &nf) {
		return connect.NewError(connect.CodeNotFound, err)
	}
	var rc ErrRevisionConflict
	if errors.As(err, &rc) {
		return connect.NewError(connect.CodeAborted, err)
	}
	return connect.NewError(connect.CodeInternal, err)
}
