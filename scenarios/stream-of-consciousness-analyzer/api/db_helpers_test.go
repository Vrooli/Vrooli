package main

import (
	"database/sql"
	"testing"

	"github.com/DATA-DOG/go-sqlmock"
)

// [REQ:P0-001] [REQ:P0-002] Test deleteByID helper returns nil on successful delete
func TestDeleteByID_Success(t *testing.T) {
	db, mock, err := sqlmock.New()
	if err != nil {
		t.Fatalf("sqlmock: %v", err)
	}
	defer db.Close()

	mock.ExpectExec("DELETE FROM schemes").
		WithArgs("abc-123").
		WillReturnResult(sqlmock.NewResult(0, 1))

	if err := deleteByID(db, "DELETE FROM schemes WHERE id = $1", "abc-123"); err != nil {
		t.Errorf("expected nil, got %v", err)
	}
	if err := mock.ExpectationsWereMet(); err != nil {
		t.Errorf("unmet expectations: %v", err)
	}
}

// [REQ:P0-001] [REQ:P0-002] Test deleteByID returns sql.ErrNoRows when no row is affected
func TestDeleteByID_NotFound(t *testing.T) {
	db, mock, err := sqlmock.New()
	if err != nil {
		t.Fatalf("sqlmock: %v", err)
	}
	defer db.Close()

	mock.ExpectExec("DELETE FROM schemes").
		WithArgs("missing-id").
		WillReturnResult(sqlmock.NewResult(0, 0))

	if err := deleteByID(db, "DELETE FROM schemes WHERE id = $1", "missing-id"); err != sql.ErrNoRows {
		t.Errorf("expected sql.ErrNoRows, got %v", err)
	}
}

// [REQ:P0-001] Test deleteByID propagates DB errors
func TestDeleteByID_ExecError(t *testing.T) {
	db, mock, err := sqlmock.New()
	if err != nil {
		t.Fatalf("sqlmock: %v", err)
	}
	defer db.Close()

	mock.ExpectExec("DELETE FROM schemes").
		WithArgs("abc-123").
		WillReturnError(sql.ErrConnDone)

	if err := deleteByID(db, "DELETE FROM schemes WHERE id = $1", "abc-123"); err != sql.ErrConnDone {
		t.Errorf("expected ErrConnDone, got %v", err)
	}
}
