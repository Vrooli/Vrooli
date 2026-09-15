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

func TestOpenRecoveryPreservesDurableBindingAndCursor(t *testing.T) {
	root := t.TempDir()
	runID := uuid.New()
	first, err := Open(runID, OpenOptions{RootDir: root, RunnerType: domain.RunnerTypeCodex, WorkingDir: "/work", RunnerIdentity: "attempt-1", OwnerEpoch: 7})
	if err != nil {
		t.Fatalf("initial Open: %v", err)
	}
	if err := first.PersistProcess(111, 222); err != nil {
		t.Fatalf("PersistProcess: %v", err)
	}
	if err := first.PersistCursor(42, 9); err != nil {
		t.Fatalf("PersistCursor: %v", err)
	}
	if err := first.Close(); err != nil {
		t.Fatalf("first Close: %v", err)
	}

	second, err := Open(runID, OpenOptions{RootDir: root, RunnerType: domain.RunnerTypeCodex, WorkingDir: "/work"})
	if err != nil {
		t.Fatalf("recovery Open: %v", err)
	}
	defer func() { _ = second.Close() }()
	snap := second.Snapshot()
	if snap.Meta.RunnerIdentity != "attempt-1" || snap.Meta.OwnerEpoch != 7 || snap.Meta.RunnerPID != 111 || snap.Meta.RunnerPGID != 222 {
		t.Fatalf("recovery binding = %+v", snap.Meta)
	}
	if snap.Cursor.TranscriptCursor != 42 || snap.Cursor.TranscriptLastSeq != 9 {
		t.Fatalf("recovery cursor = %+v", snap.Cursor)
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

func TestValidateRootRejectsProcessEphemeralStorage(t *testing.T) {
	for _, root := range []string{"/tmp/agent-manager-runs", "/var/tmp/agent-manager-runs", "/run/agent-manager-runs", "/dev/shm/agent-manager-runs"} {
		t.Run(filepath.Base(root), func(t *testing.T) {
			if err := ValidateRoot(root); err == nil {
				t.Fatalf("ValidateRoot(%q) succeeded; want ephemeral-root rejection", root)
			}
		})
	}
}

func TestValidateRootCreatesAndSyncsDurableRoot(t *testing.T) {
	root, err := os.MkdirTemp("/home/matthalloran8", "agent-manager-runstate-test-")
	if err != nil {
		t.Fatalf("MkdirTemp: %v", err)
	}
	t.Cleanup(func() { _ = os.RemoveAll(root) })
	root = filepath.Join(root, "runs")
	if err := ValidateRoot(root); err != nil {
		t.Fatalf("ValidateRoot: %v", err)
	}
	if info, err := os.Stat(root); err != nil || !info.IsDir() {
		t.Fatalf("validated root = %q, stat=%v info=%v", root, err, info)
	}
}
