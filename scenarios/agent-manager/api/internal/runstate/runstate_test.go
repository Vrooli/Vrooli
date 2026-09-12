package runstate

import (
	"context"
	"os"
	"path/filepath"
	"testing"
	"time"

	"agent-manager/internal/domain"

	"github.com/google/uuid"
	coredb "github.com/vrooli/api-core/database"
	"github.com/vrooli/api-core/filerouting"
	"github.com/vrooli/api-core/storage"
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

func TestRoutedRootResolvesPerTestContextAndAccountsWrites(t *testing.T) {
	primary := storage.Paths{StateDir: filepath.Join(t.TempDir(), "primary-state")}
	roots := filerouting.New(primary)
	testPaths := storage.Paths{StateDir: filepath.Join(t.TempDir(), "test-state")}
	if err := roots.InstallTestRoots(testPaths, "lease-1", time.Minute); err != nil {
		t.Fatalf("InstallTestRoots: %v", err)
	}
	resolver := RoutedRoot{Roots: roots}

	prod, err := resolver.Resolve(context.Background())
	if err != nil || prod != filepath.Join(primary.StateDir, "runs") {
		t.Fatalf("production root = %q, %v", prod, err)
	}
	testCtx := coredb.WithTestMode(context.Background())
	isolated, err := resolver.Resolve(testCtx)
	if err != nil || isolated != filepath.Join(testPaths.StateDir, "runs") {
		t.Fatalf("test root = %q, %v", isolated, err)
	}
	resolver.RecordWrite(testCtx)
	if got := roots.LeaseStats(); got.TestRootWrites != 1 || got.PrimaryWritesDuringTestMode != 0 {
		t.Fatalf("lease write stats = %+v", got)
	}
}

func TestOpenNotifiesOnlySuccessfulStateWrites(t *testing.T) {
	writes := 0
	state, err := Open(uuid.New(), OpenOptions{RootDir: t.TempDir(), OnWrite: func() { writes++ }})
	if err != nil {
		t.Fatalf("Open: %v", err)
	}
	defer func() { _ = state.Close() }()
	if writes != 1 {
		t.Fatalf("writes after Open = %d, want 1", writes)
	}
	if err := state.PersistSessionID("session"); err != nil {
		t.Fatalf("PersistSessionID: %v", err)
	}
	if writes != 2 {
		t.Fatalf("writes after PersistSessionID = %d, want 2", writes)
	}
}
