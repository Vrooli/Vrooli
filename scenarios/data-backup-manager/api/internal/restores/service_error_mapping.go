package restores

import "data-backup-manager/internal/connecterrors"

// ToConnectError translates restore sentinels into Connect's typed error model.
func ToConnectError(err error) error {
	return connecterrors.ToConnectError[ErrInvalidRestore, ErrRestoreNotFound](err)
}
