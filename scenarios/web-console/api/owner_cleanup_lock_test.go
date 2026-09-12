package main

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"net/http/httptest"
	"os"
	"path/filepath"
	"testing"
	"time"

	platform "github.com/vrooli/platform-go"
	"web-console/internal/backend"
	"web-console/internal/sessionstore"
)

type failingSessionMetadataStore struct {
	sessionstore.Store
	err error
}

func (s failingSessionMetadataStore) Save(context.Context, sessionstore.Metadata) error {
	return s.err
}

func TestWebConsoleCleanupRecoveryLockHonorsStorageRecoveryHolder(t *testing.T) {
	path := filepath.Join(t.TempDir(), "state", "storage-manager", "recovery.lock")
	if err := os.MkdirAll(filepath.Dir(path), 0o755); err != nil {
		t.Fatal(err)
	}
	heldFile, err := os.OpenFile(path, os.O_CREATE|os.O_RDWR, 0o600)
	if err != nil {
		t.Fatal(err)
	}
	heldRelease, err := platform.LockFile(heldFile, true)
	if err != nil {
		_ = heldFile.Close()
		t.Fatal(err)
	}
	defer func() {
		heldRelease()
		_ = heldFile.Close()
	}()

	h := &webConsoleCleanup{recoveryLockPath: path}
	if _, err := h.acquireRecoveryLock(); err == nil {
		t.Fatal("owner cleanup acquired a lock held by storage recovery")
	}
}

func TestWebConsoleAutomaticRetentionPreservesOnlyNativeHistoryCopy(t *testing.T) {
	root := t.TempDir()
	t.Setenv("WC_SESSION_STATE_ROOT", root)

	srv := newFakeTestServer()
	srv.sessionStore = sessionstore.NewInMemory()
	meta := sessionstore.Metadata{
		ID:         "native-only",
		AgentType:  sessionstore.AgentCodex,
		ArchivedAt: time.Now().UTC().Add(-90 * 24 * time.Hour),
	}
	if err := srv.sessionStore.Save(context.Background(), meta); err != nil {
		t.Fatal(err)
	}
	history := filepath.Join(root, "codex", meta.ID, "rollout.jsonl")
	if err := os.MkdirAll(filepath.Dir(history), 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(history, []byte("native recovery history"), 0o600); err != nil {
		t.Fatal(err)
	}

	cleanup := &webConsoleCleanup{server: srv, recoveryLockPath: filepath.Join(t.TempDir(), "recovery.lock"), done: make(map[string]ownerCleanupResult)}
	cleanup.autoSweep(context.Background(), 1, 0, 0)

	if _, err := os.Stat(history); err != nil {
		t.Fatalf("automatic retention removed the only native-history copy: %v", err)
	}
}

func TestWebConsoleCleanupApplyPreservesNativeHistoryWhenLifecycleDeleteFails(t *testing.T) {
	root := t.TempDir()
	t.Setenv("WC_SESSION_STATE_ROOT", root)

	srv := newFakeTestServer()
	srv.sessionStore = sessionstore.NewInMemory()
	meta := sessionstore.Metadata{
		ID:         "delete-failure",
		AgentType:  sessionstore.AgentCodex,
		ArchivedAt: time.Now().UTC().Add(-90 * 24 * time.Hour),
	}
	if err := srv.sessionStore.Save(context.Background(), meta); err != nil {
		t.Fatal(err)
	}
	history := filepath.Join(root, "codex", meta.ID, "rollout.jsonl")
	if err := os.MkdirAll(filepath.Dir(history), 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(history, []byte("native recovery history"), 0o600); err != nil {
		t.Fatal(err)
	}
	srv.lifecycleDelete = func(context.Context, string) error { return os.ErrPermission }

	cleanup := &webConsoleCleanup{server: srv, recoveryLockPath: filepath.Join(t.TempDir(), "recovery.lock"), done: make(map[string]ownerCleanupResult)}
	rows, _, _, _ := cleanup.candidates(httptest.NewRequest("GET", "/?min_age_seconds=1", nil))
	items := cleanup.items(rows, 0, 0)
	if len(items) != 1 {
		t.Fatalf("cleanup items = %+v, want one candidate", items)
	}
	body, err := json.Marshal(ownerCleanupApply{
		Preview:        ownerCleanupPreview{ProviderID: "web-console-sessions", Items: items},
		IdempotencyKey: "delete-failure",
		ApprovalMode:   "owner",
	})
	if err != nil {
		t.Fatal(err)
	}
	req := httptest.NewRequest("POST", "/", bytes.NewReader(body))
	res := httptest.NewRecorder()
	cleanup.apply(res, req)
	if res.Code != 200 {
		t.Fatalf("cleanup apply status = %d, body = %s", res.Code, res.Body.String())
	}
	if _, err := os.Stat(history); err != nil {
		t.Fatalf("cleanup removed native history before lifecycle delete succeeded: %v", err)
	}
}

func TestSessionCreateFailsClosedWhenMetadataPersistenceFails(t *testing.T) {
	srv := newFakeTestServer()
	srv.sessions.SetStore(failingSessionMetadataStore{
		Store: sessionstore.NewInMemory(),
		err:   errors.New("metadata store unavailable"),
	})

	if _, err := srv.sessions.Create(context.Background(), "", 80, 24, backend.Standard, nil); err == nil {
		t.Fatal("session creation succeeded without durable metadata")
	}
	if got := len(srv.sessions.List()); got != 0 {
		t.Fatalf("untracked sessions after persistence failure = %d, want 0", got)
	}
}

func TestWebConsoleAutomaticRetentionRecordsNativeHistoryPrune(t *testing.T) {
	root := t.TempDir()
	t.Setenv("WC_SESSION_STATE_ROOT", root)

	srv := newFakeTestServer()
	srv.sessionStore = sessionstore.NewInMemory()
	meta := sessionstore.Metadata{
		ID:         "native-with-transcript",
		AgentType:  sessionstore.AgentCodex,
		ArchivedAt: time.Now().UTC().Add(-90 * 24 * time.Hour),
	}
	if err := srv.sessionStore.Save(context.Background(), meta); err != nil {
		t.Fatal(err)
	}
	if _, result := srv.conversations.AppendAssistantEvent(context.Background(), meta.ID, "test", "durable transcript"); !result.Appended {
		t.Fatalf("append conversation: %+v", result)
	}
	history := filepath.Join(root, "codex", meta.ID, "rollout.jsonl")
	if err := os.MkdirAll(filepath.Dir(history), 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(history, []byte("native recovery history"), 0o600); err != nil {
		t.Fatal(err)
	}
	ledger := &retentionLedgerStub{}
	cleanup := &webConsoleCleanup{
		server:           srv,
		recoveryLockPath: filepath.Join(t.TempDir(), "recovery.lock"),
		retentionLedger:  ledger,
	}

	cleanup.autoSweep(context.Background(), 1, 0, 0)
	if _, err := os.Stat(history); !os.IsNotExist(err) {
		t.Fatalf("native history still exists after receipted prune, stat error=%v", err)
	}
	if ledger.put != 1 || ledger.completed != 1 {
		t.Fatalf("retention receipt calls = put:%d complete:%d, want 1/1", ledger.put, ledger.completed)
	}
}
