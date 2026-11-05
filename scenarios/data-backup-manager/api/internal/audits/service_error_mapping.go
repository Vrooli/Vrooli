package audits

import "data-backup-manager/internal/connecterrors"

// ToConnectError translates audit sentinels into Connect's typed error model.
func ToConnectError(err error) error {
	return connecterrors.ToConnectError[ErrInvalidAudit, ErrAuditNotFound](err)
}
