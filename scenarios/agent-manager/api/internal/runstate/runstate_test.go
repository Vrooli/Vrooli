package runstate

import (
	"os"
	"path/filepath"
	"testing"
	"time"

	"agent-manager/internal/domain"

	"github.com/google/uuid"
)

func TestOpenCreatesStateFilesAndPersistsUpdates(t *testing.T) {
	root := t.TempDir()
	runID := uuid.New()

	state, err := Open(runID, OpenOptions{
		RootDir:    root,
		RunnerType: domain.RunnerTypeClaudeCode,
		WorkingDir: "/tmp/work",
		StartedAt:  time.Unix(123, 0).UTC(),
	})
	if err != nil {
		t.Fatalf("Open: %v", err)
	}
	defer func() { _ = state.Close() }()

	snap := state.Snapshot()
	for _, path := range []string{snap.MetaPath, snap.TranscriptPath, snap.StderrPath, snap.CursorPath} {
		if _, err := os.Stat(path); err != nil {
			t.Fatalf("expected %s to exist: %v", filepath.Base(path), err)
		}
	}

	if err := state.PersistProcess(111, 222); err != nil {
		t.Fatalf("PersistProcess: %v", err)
	}
	if err := state.PersistSessionID("session-1"); err != nil {
		t.Fatalf("PersistSessionID: %v", err)
	}
	if err := state.PersistCursor(99, 7); err != nil {
		t.Fatalf("PersistCursor: %v", err)
	}

	loaded, err := Load(runID, root)
	if err != nil {
		t.Fatalf("Load: %v", err)
	}
	if loaded.Meta.RunnerPID != 111 || loaded.Meta.RunnerPGID != 222 {
		t.Fatalf("loaded meta pid/pgid = %d/%d", loaded.Meta.RunnerPID, loaded.Meta.RunnerPGID)
	}
	if loaded.Meta.SessionID != "session-1" {
		t.Fatalf("loaded session = %q", loaded.Meta.SessionID)
	}
	if loaded.Cursor.TranscriptCursor != 99 || loaded.Cursor.TranscriptLastSeq != 7 {
		t.Fatalf("loaded cursor = %+v", loaded.Cursor)
	}
}
