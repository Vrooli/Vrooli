// Package database provides SQLite persistence for agent-manager entities.
package database

import (
	"database/sql"
	"errors"

	"agent-manager/internal/domain"
)

func wrapDBError(operation, entityType, entityID string, err error) error {
	if err == nil {
		return nil
	}
	return &domain.DatabaseError{
		Operation:   operation,
		EntityType:  entityType,
		EntityID:    entityID,
		Cause:       err,
		IsTransient: isTransientDBError(err),
	}
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
