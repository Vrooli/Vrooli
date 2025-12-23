// Package database provides PostgreSQL persistence for agent-manager entities.
package database

import (
	"database/sql"
	"errors"

	"agent-manager/internal/domain"
	"github.com/lib/pq"
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
	var pqErr *pq.Error
	if errors.As(err, &pqErr) {
		switch pqErr.Code.Class() {
		case "08": // connection exception
			return true
		}
	}
	return false
}
