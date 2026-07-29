package main

import (
	"context"
	"net/http"
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"

	"github.com/vrooli/api-core/database"
	"github.com/vrooli/api-core/filerouting"
	"github.com/vrooli/api-core/storage"
)

func TestOperatorStateRoundTripIsAtomicAndDoesNotNeedDatabase(t *testing.T) {
	oldPath, oldNow := operatorStatePath, operatorStateNow
	t.Cleanup(func() { operatorStatePath, operatorStateNow = oldPath, oldNow })
	path := filepath.Join(t.TempDir(), "operator-state.json")
	operatorStatePath = func() (string, error) { return path, nil }
	operatorStateNow = func() time.Time { return time.Date(2026, 7, 29, 0, 0, 0, 0, time.UTC) }
	srv := NewServer()
	w := doGet(t, srv, "/api/v1/operator-state")
	if w.Code != http.StatusOK {
		t.Fatalf("GET = %d: %s", w.Code, w.Body.String())
	}
	w = doRequest(t, srv, http.MethodPut, "/api/v1/operator-state", `{"scenarios":{"example":{"enabled":true,"auto_restart":false}}}`)
	if w.Code != http.StatusOK {
		t.Fatalf("PUT = %d: %s", w.Code, w.Body.String())
	}
	w = doGet(t, srv, "/api/v1/operator-state")
	if w.Code != http.StatusOK || !strings.Contains(w.Body.String(), `"example"`) {
		t.Fatalf("round trip = %d: %s", w.Code, w.Body.String())
	}
}

func TestOperatorStateUsesDesktopStorageRatherThanBundleCatalog(t *testing.T) {
	storage := t.TempDir()
	t.Setenv("VROOLI_ROOT", "")
	t.Setenv("VROOLI_STORAGE_ROOT", storage)
	path, err := operatorStatePath()
	if err != nil {
		t.Fatalf("operatorStatePath: %v", err)
	}
	want := filepath.Join(storage, "operator-state.json")
	if path != want {
		t.Fatalf("operator state path = %q, want %q", path, want)
	}
}

func TestOperatorStateUsesLeasedFileRootsInTestMode(t *testing.T) {
	primary := t.TempDir()
	oldRoots, oldPath := operatorStateRoots, operatorStatePath
	t.Cleanup(func() { operatorStateRoots, operatorStatePath = oldRoots, oldPath })
	roots := filerouting.New(storage.Paths{ConfigDir: primary, DataDir: primary, CacheDir: primary, LogsDir: primary, StateDir: primary})
	operatorStateRoots = roots
	operatorStatePath = func() (string, error) { return filepath.Join(primary, "operator-state.json"), nil }
	if _, err := roots.InstallLeasedTestRoots("operator-state-test", time.Minute, true); err != nil {
		t.Fatalf("install test roots: %v", err)
	}
	ctx := database.WithTestMode(context.Background())
	if err := saveOperatorStateFor(ctx, defaultOperatorState()); err != nil {
		t.Fatalf("save operator state: %v", err)
	}
	testRoot, err := roots.Pick(ctx, storage.ClassConfig)
	if err != nil {
		t.Fatalf("pick test root: %v", err)
	}
	if _, err := os.Stat(filepath.Join(testRoot, "operator-state.json")); err != nil {
		t.Fatalf("state was not written to leased root: %v", err)
	}
	if _, err := os.Stat(filepath.Join(primary, "operator-state.json")); !os.IsNotExist(err) {
		t.Fatalf("primary state should remain untouched, got %v", err)
	}
	if got := roots.LeaseStats().TestRootWrites; got != 1 {
		t.Fatalf("test root writes = %d, want 1", got)
	}
}
