package destinations

import "data-backup-manager/internal/connecterrors"

// ToConnectError translates destination sentinels into Connect's typed error model.
func ToConnectError(err error) error {
	return connecterrors.ToConnectError[ErrInvalidDestination, ErrDestinationNotFound](err)
}
