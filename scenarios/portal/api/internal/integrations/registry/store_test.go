package registry

import (
	"context"
	"testing"
	"time"

	"portal/internal/testutil/db"
)

func TestStorePersistsOverride(t *testing.T) {
	t.Parallel()

	handle := db.NewSQLite(t)
	if _, err := handle.ExecContext(context.Background(), `
CREATE TABLE integration_settings (
    key TEXT PRIMARY KEY,
    value TEXT NOT NULL,
    updated_at TEXT NOT NULL
)`); err != nil {
		t.Fatalf("create schema: %v", err)
	}

	store := NewStore(handle)
	got, err := store.Override(context.Background())
	if err != nil {
		t.Fatalf("read default override: %v", err)
	}
	if got != OverrideAuto {
		t.Fatalf("expected auto default, got %q", got)
	}
	if err := store.SetOverride(context.Background(), OverrideForceOff, time.Date(2026, 7, 6, 12, 0, 0, 0, time.UTC)); err != nil {
		t.Fatalf("set override: %v", err)
	}
	got, err = store.Override(context.Background())
	if err != nil {
		t.Fatalf("read override: %v", err)
	}
	if got != OverrideForceOff {
		t.Fatalf("expected force-off, got %q", got)
	}
}
