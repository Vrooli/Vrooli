package targets

import "data-backup-manager/internal/connecterrors"

// ToConnectError translates target sentinels into Connect's typed error model.
func ToConnectError(err error) error {
	return connecterrors.ToConnectError[ErrInvalidTarget, ErrTargetNotFound](err)
}
