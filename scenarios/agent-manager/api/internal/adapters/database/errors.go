// Package database provides SQLite persistence for agent-manager entities.
package database

import (
	"strings"

	"agent-manager/internal/domain"
	"agent-manager/internal/sqlcompat"
)

func wrapDBError(operation, entityType, entityID string, err error) error {
	if err == nil {
		return nil
	}
	if isUniqueConstraintViolation(err) {
		return domain.NewStateError(
			entityType,
			"exists",
			operation,
			entityType+" already exists with that name or key",
		)
	}
	return &domain.DatabaseError{
		Operation:   operation,
		EntityType:  entityType,
		EntityID:    entityID,
		Cause:       err,
		IsTransient: isTransientDBError(err),
	}
}

// isUniqueConstraintViolation detects SQLite UNIQUE constraint failures.
// modernc.org/sqlite reports these with "UNIQUE constraint failed" in the message.
func isUniqueConstraintViolation(err error) bool {
	return err != nil && strings.Contains(err.Error(), "UNIQUE constraint failed")
}

func isTransientDBError(err error) bool {
	return sqlcompat.TransientError(err)
}
