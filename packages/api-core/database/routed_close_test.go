package database_test

import (
	"context"
	"path/filepath"
	"testing"
)

// TestRoutedDBCloseKeepsClosedHandleSafe pins the shutdown contract: work that
// races pool close must receive a database error, never a nil-pointer panic.
// RoutedDB used to clear its primary handle on Close, so a best-effort
// transaction starting concurrently with shutdown dereferenced nil inside
// database/sql and crashed the process.
func TestRoutedDBCloseKeepsClosedHandleSafe(t *testing.T) {
	path := filepath.Join(t.TempDir(), "closed.db")
	seedPool(t, path, "PRIMARY")

	r, err := openSQLitePool(t, path)
	if err != nil {
		t.Fatalf("open routed: %v", err)
	}
	if err := r.Close(); err != nil {
		t.Fatalf("Close: %v", err)
	}
	if err := r.Close(); err != nil {
		t.Fatalf("second Close should be idempotent: %v", err)
	}

	if _, err := r.BeginTx(context.Background(), nil); err == nil {
		t.Fatal("BeginTx after Close should return an error")
	}
	if _, err := r.QueryContext(context.Background(), `SELECT 1`); err == nil {
		t.Fatal("QueryContext after Close should return an error")
	}
	if err := r.PingContext(context.Background()); err == nil {
		t.Fatal("PingContext after Close should return an error")
	}

	var value int
	if err := r.QueryRowContext(context.Background(), `SELECT 1`).Scan(&value); err == nil {
		t.Fatal("QueryRowContext after Close should return an error on Scan")
	}
}
