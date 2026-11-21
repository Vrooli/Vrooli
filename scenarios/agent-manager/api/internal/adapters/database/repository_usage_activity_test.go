package database

import (
	"testing"
	"time"
)

func TestLastProviderActivityIsNowWhileLiveAndOtherwiseTheNewestIndexedUsage(t *testing.T) {
	db, cleanup := setupTestDB(t)
	defer cleanup()
	repo := NewUsageActivityRepository(db)
	ctx := t.Context()

	if last, err := repo.LastProviderActivity(ctx, "openai"); err != nil || !last.IsZero() {
		t.Fatalf("no work ever: last=%v err=%v", last, err)
	}
	exec := func(query string, args ...any) {
		t.Helper()
		if _, err := db.ExecContext(ctx, query, args...); err != nil {
			t.Fatal(err)
		}
	}
	exec(`INSERT INTO transcript_files (file_key, path, root_kind, root_path, harness, first_seen_at, active) VALUES ('k1', '/p', 'codex', '/r', 'codex', '2026-10-29T00:00:00Z', 0)`)
	exec(`INSERT INTO transcript_usage (file_key, line_offset, ordinal, run_id, observed_at) VALUES ('k1', 0, 0, '', '2026-10-29T03:00:00Z'), ('k1', 1, 0, '', '2026-10-29T05:30:00Z')`)
	want := time.Date(2026, 10, 29, 5, 30, 0, 0, time.UTC)
	if last, err := repo.LastProviderActivity(ctx, "openai"); err != nil || !last.Equal(want) {
		t.Fatalf("newest indexed usage: last=%v err=%v want %v", last, err, want)
	}
	if last, _ := repo.LastProviderActivity(ctx, "anthropic"); !last.IsZero() {
		t.Fatalf("another provider's usage must not count: %v", last)
	}

	exec(`UPDATE transcript_files SET active = 1 WHERE file_key = 'k1'`)
	if last, err := repo.LastProviderActivity(ctx, "openai"); err != nil || time.Since(last) > time.Minute {
		t.Fatalf("an active session is activity now: last=%v err=%v", last, err)
	}
	if last, err := repo.LastProviderActivity(ctx, "unrecognised"); err != nil || time.Since(last) > time.Minute {
		t.Fatalf("an unattributable provider cannot be ruled idle: last=%v err=%v", last, err)
	}
}
