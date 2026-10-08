// Package database provides SQLite persistence for agent-manager entities.
package database

import (
	"database/sql"
	"errors"
	"strings"

	"agent-manager/internal/domain"
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
	if err == nil {
		return false
	}
	if errors.Is(err, sql.ErrConnDone) {
		return true
	}
	return false
}
