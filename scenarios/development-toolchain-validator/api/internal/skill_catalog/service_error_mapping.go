package skill_catalog

import (
	"errors"

	"connectrpc.com/connect"
)

// ToConnectError translates domain sentinels into Connect's typed error
// model. Unknown errors intentionally map to internal so callers never
// depend on raw storage or implementation details.
func ToConnectError(err error) error {
	if err == nil {
		return nil
	}
	var invalid ErrInvalidSkill
	if errors.As(err, &invalid) {
		return connect.NewError(connect.CodeInvalidArgument, invalid)
	}
	var notFound ErrSkillNotFound
	if errors.As(err, &notFound) {
		return connect.NewError(connect.CodeNotFound, notFound)
	}
	var syncErr ErrSyncFailed
	if errors.As(err, &syncErr) {
		if syncErr.NotReady {
			return connect.NewError(connect.CodeUnavailable, syncErr)
		}
		return connect.NewError(connect.CodeInternal, syncErr)
	}
	return connect.NewError(connect.CodeInternal, err)
}
