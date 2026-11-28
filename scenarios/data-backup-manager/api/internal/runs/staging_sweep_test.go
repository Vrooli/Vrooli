package runs

import (
	"context"
	"os"
	"path/filepath"
	"testing"
	"time"

	"github.com/vrooli/api-core/scheduletest"
)

func TestSweepStaleStagingRemovesOnlyOldRunDirs(t *testing.T) {
	base := t.TempDir()
	now := time.Date(2026, 11, 18, 12, 0, 0, 0, time.UTC)
	s := &service{deps: Deps{StagingRoot: base, Clock: scheduletest.New(now)}}

	oldDir := filepath.Join(base, "dbm-run-old")
	freshDir := filepath.Join(base, "dbm-run-fresh")
	unrelated := filepath.Join(base, "operator-notes")
	for _, dir := range []string{oldDir, freshDir, unrelated} {
		if err := os.MkdirAll(dir, 0o755); err != nil {
			t.Fatal(err)
		}
	}
	old := now.Add(-48 * time.Hour)
	if err := os.Chtimes(oldDir, old, old); err != nil {
		t.Fatal(err)
	}

	removed, err := s.SweepStaleStaging(context.Background())
	if err != nil {
		t.Fatal(err)
	}
	if removed != 1 {
		t.Fatalf("removed = %d, want only the stale run directory", removed)
	}
	if _, err := os.Stat(oldDir); !os.IsNotExist(err) {
		t.Fatalf("stale staging dir survived: %v", err)
	}
	for _, dir := range []string{freshDir, unrelated} {
		if _, err := os.Stat(dir); err != nil {
			t.Fatalf("%s should be preserved: %v", dir, err)
		}
	}
}
