package dbx

import (
	"context"
	"database/sql"
	"path/filepath"
	"strings"
	"testing"
	"time"

	_ "modernc.org/sqlite"
)

// ReadWrite sends standalone queries to the read side and statements and
// transactions to the writer. A query-only reader proves the split: a write
// that reached it would be refused.
func TestReadWriteRoutesQueriesToReadsAndStatementsToWrites(t *testing.T) {
	path := filepath.Join(t.TempDir(), "split.db")
	writer, err := sql.Open("sqlite", "file:"+path+"?_pragma=journal_mode(WAL)")
	if err != nil {
		t.Fatal(err)
	}
	defer writer.Close()
	writer.SetMaxOpenConns(1)
	reader, err := sql.Open("sqlite", "file:"+path+"?_pragma=query_only(1)")
	if err != nil {
		t.Fatal(err)
	}
	defer reader.Close()
	ctx := context.Background()
	handle := ReadWrite{Writes: writer, Reads: reader}
	if _, err := handle.ExecContext(ctx, `CREATE TABLE t (v TEXT NOT NULL); INSERT INTO t VALUES ('one')`); err != nil {
		t.Fatalf("statement on the writer: %v", err)
	}
	tx, err := handle.BeginTx(ctx, nil)
	if err != nil {
		t.Fatal(err)
	}
	if _, err := tx.ExecContext(ctx, `INSERT INTO t VALUES ('two')`); err != nil {
		t.Fatalf("transaction on the writer: %v", err)
	}
	if err := tx.Commit(); err != nil {
		t.Fatal(err)
	}
	var n int
	if err := handle.QueryRowContext(ctx, `SELECT COUNT(*) FROM t`).Scan(&n); err != nil || n != 2 {
		t.Fatalf("query on the reader = %d, %v; want both committed rows", n, err)
	}
	err = handle.QueryRowContext(ctx, `INSERT INTO t VALUES ('three') RETURNING v`).Scan(new(string))
	if err == nil || !strings.Contains(strings.ToLower(err.Error()), "readonly") {
		t.Fatalf("a write issued as a query reached the writer: %v", err)
	}
}

// openSplit opens one WAL file as a single-connection writer and a query-only
// read pool, the shape production runs.
func openSplit(t *testing.T) (ReadWrite, *sql.DB) {
	t.Helper()
	path := filepath.Join(t.TempDir(), "snapshot.db")
	writer, err := sql.Open("sqlite", "file:"+path+"?_pragma=journal_mode(WAL)")
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() { _ = writer.Close() })
	writer.SetMaxOpenConns(1)
	reader, err := sql.Open("sqlite", "file:"+path+"?_pragma=query_only(1)")
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() { _ = reader.Close() })
	if _, err := writer.Exec(`CREATE TABLE t (v TEXT NOT NULL); INSERT INTO t VALUES ('one')`); err != nil {
		t.Fatal(err)
	}
	return ReadWrite{Writes: writer, Reads: reader}, writer
}

// Every read inside ReadSnapshot sees the state at its first read, even after
// the writer commits, and the snapshot never holds the single writer.
func TestReadSnapshotHoldsOneStateOnTheReadPool(t *testing.T) {
	handle, writer := openSplit(t)
	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()
	count := func(q Querier) int {
		t.Helper()
		var n int
		if err := q.QueryRowContext(ctx, `SELECT COUNT(*) FROM t`).Scan(&n); err != nil {
			t.Fatal(err)
		}
		return n
	}
	err := ReadSnapshot(ctx, handle, func(q Querier) error {
		before := count(q)
		// Served by the writer, this commit would wait on its only connection
		// until the deadline.
		if _, err := writer.ExecContext(ctx, `INSERT INTO t VALUES ('two')`); err != nil {
			return err
		}
		if after := count(q); after != before {
			t.Fatalf("snapshot moved from %d to %d rows mid-read", before, after)
		}
		return nil
	})
	if err != nil {
		t.Fatal(err)
	}
	if n := count(handle); n != 2 {
		t.Fatalf("a read after the snapshot saw %d rows, want the committed 2", n)
	}
}

// A handle without a read side still gets one snapshot, from its own pool.
func TestReadSnapshotFallsBackToTheHandle(t *testing.T) {
	_, writer := openSplit(t)
	err := ReadSnapshot(context.Background(), writer, func(q Querier) error {
		var n int
		return q.QueryRowContext(context.Background(), `SELECT COUNT(*) FROM t`).Scan(&n)
	})
	if err != nil {
		t.Fatal(err)
	}
}
