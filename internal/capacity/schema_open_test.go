package capacity

import (
	"context"
	"database/sql"
	"path/filepath"
	"testing"
	"time"

	"github.com/vrooli/vrooli/internal/clock"
)

// Opening a current ledger is a read. It must not need the writer lock: every
// companion poll and every pre-flight admission question opens a store, and
// re-applying the DDL on each open is what turned normal traffic into
// SQLITE_BUSY ("ledger unavailable") on a live host.
func TestOpeningACurrentLedgerDoesNotNeedTheWriterLock(t *testing.T) {
	ctx := context.Background()
	dbPath := filepath.Join(t.TempDir(), "capacity.db")
	first, err := NewSQLiteStore(ctx, Config{DBPath: dbPath, Clock: clock.Real{}})
	if err != nil {
		t.Fatalf("create ledger: %v", err)
	}
	if err := first.Close(); err != nil {
		t.Fatalf("close first store: %v", err)
	}

	// Hold the only writer slot: an uncommitted IMMEDIATE transaction.
	writer, err := sql.Open("sqlite", buildDSN(dbPath))
	if err != nil {
		t.Fatalf("open writer: %v", err)
	}
	defer writer.Close()
	writer.SetMaxOpenConns(1)
	tx, err := writer.BeginTx(ctx, nil)
	if err != nil {
		t.Fatalf("begin writer transaction: %v", err)
	}
	defer func() { _ = tx.Rollback() }()
	if _, err := tx.ExecContext(ctx, `INSERT INTO capacity_policy (key, value, updated_at) VALUES ('test-writer-lock', '1', '2026-01-01T00:00:00Z')`); err != nil {
		t.Fatalf("take writer lock: %v", err)
	}

	// A read-only open must complete while the writer is held. Bound it well
	// below the DSN busy_timeout so a regression fails fast instead of hanging.
	openCtx, cancel := context.WithTimeout(ctx, 2*time.Second)
	defer cancel()
	second, err := NewSQLiteStore(openCtx, Config{DBPath: dbPath, Clock: clock.Real{}})
	if err != nil {
		t.Fatalf("opening a current ledger blocked on the writer lock: %v", err)
	}
	defer second.Close()

	claims, err := second.ListClaims(openCtx, ClaimFilter{})
	if err != nil {
		t.Fatalf("read while a writer is held: %v", err)
	}
	if len(claims) != 0 {
		t.Fatalf("claims = %v, want none", claims)
	}
}

func TestSchemaObjectsCoversEveryDeclaredObject(t *testing.T) {
	ctx := context.Background()
	path := filepath.Join(t.TempDir(), "capacity.db")
	store, err := NewSQLiteStore(ctx, Config{DBPath: path, Clock: clock.Real{}})
	if err != nil {
		t.Fatal(err)
	}
	defer store.Close()
	complete, err := schemaObjectsPresent(ctx, store.db)
	if err != nil {
		t.Fatal(err)
	}
	if !complete {
		t.Fatal("a freshly created ledger is missing a declared schema object")
	}
	if len(schemaObjects) < 10 {
		t.Fatalf("schemaObjects = %v, want every table and index", schemaObjects)
	}
}
