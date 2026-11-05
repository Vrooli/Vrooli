package databasetest

import (
	"context"
	"database/sql"
	"database/sql/driver"
	"sync"
	"sync/atomic"

	apidb "github.com/vrooli/api-core/database"
)

// FakeExecer is the canonical test double for database.SchemaExecer.
//
// It records successful queries in order, supports all-call and ordered error
// injection, and exposes synchronized accessors so callers can use it in
// race-enabled tests.
type FakeExecer struct {
	mu sync.Mutex

	queries []string
	execErr error

	// FailOnCall, if greater than zero, makes that 1-based ExecContext call
	// fail before recording the query.
	FailOnCall int64

	// FailErr is returned when FailOnCall matches. If FailErr is nil,
	// sql.ErrConnDone is returned so a configured failure is never silent.
	FailErr error

	// ExecCalls counts all ExecContext invocations, including calls that
	// returned an injected error.
	ExecCalls atomic.Int64
}

// ExecContext implements database.SchemaExecer.
func (f *FakeExecer) ExecContext(_ context.Context, query string, _ ...any) (sql.Result, error) {
	call := f.ExecCalls.Add(1)

	f.mu.Lock()
	defer f.mu.Unlock()

	if f.execErr != nil {
		return nil, f.execErr
	}
	if f.FailOnCall > 0 && call == f.FailOnCall {
		if f.FailErr != nil {
			return nil, f.FailErr
		}
		return nil, sql.ErrConnDone
	}

	f.queries = append(f.queries, query)
	return driver.RowsAffected(0), nil
}

// SetExecErr configures an error returned by every ExecContext call.
//
// The all-call error takes precedence over FailOnCall and prevents query
// recording.
func (f *FakeExecer) SetExecErr(err error) {
	f.mu.Lock()
	defer f.mu.Unlock()
	f.execErr = err
}

// SnapshotQueries returns a copy of the successfully recorded queries.
func (f *FakeExecer) SnapshotQueries() []string {
	f.mu.Lock()
	defer f.mu.Unlock()
	return append([]string(nil), f.queries...)
}

var _ apidb.SchemaExecer = (*FakeExecer)(nil)
