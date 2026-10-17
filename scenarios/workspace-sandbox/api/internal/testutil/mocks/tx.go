package mocks

import (
	"database/sql"

	"workspace-sandbox/internal/repository"
)

// FakeTxRepository wraps a FakeRepository to satisfy the
// repository.TxRepository contract. All Repository methods route to
// the embedded FakeRepository so transaction tests share state with
// the parent fake. Commit/Rollback are no-ops by default; tests that
// need failure injection set CommitErr / RollbackErr.
type FakeTxRepository struct {
	*FakeRepository

	CommitErr   error
	RollbackErr error

	Committed  bool
	RolledBack bool
}

// NewFakeTxRepository wraps a FakeRepository with no errors set.
func NewFakeTxRepository(parent *FakeRepository) *FakeTxRepository {
	return &FakeTxRepository{FakeRepository: parent}
}

func (t *FakeTxRepository) Commit() error {
	if t.CommitErr != nil {
		return t.CommitErr
	}
	t.Committed = true
	return nil
}

func (t *FakeTxRepository) Rollback() error {
	if t.RollbackErr != nil {
		return t.RollbackErr
	}
	t.RolledBack = true
	return nil
}

// Tx returns nil for the in-memory fake. The archive repository's Insert
// handles a nil tx by falling back to its own *sql.DB; production tests
// for snapshotDiff use the real SQLite seam, not this fake.
func (t *FakeTxRepository) Tx() *sql.Tx { return nil }

var _ repository.TxRepository = (*FakeTxRepository)(nil)
