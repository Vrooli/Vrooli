package database

import (
	"database/sql"
	"errors"
	"testing"

	"agent-manager/internal/domain"
)

func TestDatabaseErrorClassification(t *testing.T) {
	if wrapDBError("read", "Profile", "id", nil) != nil {
		t.Fatal("nil database error was wrapped")
	}
	unique := errors.New("UNIQUE constraint failed: profiles.name")
	if !isUniqueConstraintViolation(unique) || isUniqueConstraintViolation(nil) {
		t.Fatal("unique constraint classification mismatch")
	}
	if !isTransientDBError(sql.ErrConnDone) || isTransientDBError(errors.New("permanent")) {
		t.Fatal("transient classification mismatch")
	}
	if got := wrapDBError("create", "Profile", "id", unique); got == nil {
		t.Fatal("unique error was not wrapped")
	} else if state, ok := got.(*domain.StateError); !ok || state.Code().Category() != "STATE" {
		t.Fatalf("unique wrap=%T %v", got, got)
	}
	got := wrapDBError("read", "Profile", "id", sql.ErrConnDone)
	dbErr, ok := got.(*domain.DatabaseError)
	if !ok || !dbErr.IsTransient || dbErr.Operation != "read" {
		t.Fatalf("database wrap=%+v", got)
	}
}
