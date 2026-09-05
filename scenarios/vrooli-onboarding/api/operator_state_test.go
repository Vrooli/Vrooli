package main

import (
	"context"
	"encoding/json"
	"net/http"
	"os"
	"path/filepath"
	"reflect"
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
	w = doRequest(t, srv, http.MethodPatch, "/api/v2/operator-state", `{"scenarios":{"example":{"enabled":true,"auto_restart":false}}}`)
	if w.Code != http.StatusOK {
		t.Fatalf("PATCH = %d: %s", w.Code, w.Body.String())
	}
	w = doGet(t, srv, "/api/v1/operator-state")
	if w.Code != http.StatusOK || !strings.Contains(w.Body.String(), `"example"`) {
		t.Fatalf("round trip = %d: %s", w.Code, w.Body.String())
	}
}

func TestOperatorStateRoundTripPreservesOpaqueSafeguardConfig(t *testing.T) {
	oldPath, oldNow := operatorStatePath, operatorStateNow
	t.Cleanup(func() { operatorStatePath, operatorStateNow = oldPath, oldNow })
	path := filepath.Join(t.TempDir(), "operator-state.json")
	operatorStatePath = func() (string, error) { return path, nil }
	operatorStateNow = func() time.Time { return time.Date(2026, 7, 29, 0, 0, 0, 0, time.UTC) }

	input := `{"version":"1.0.0","updated_at":"2026-07-29T00:00:00Z","host_safeguards":{"remote_desktop_access":{"opted_in":true,"config":{"experience":"direct-desktop","future_permission":true}}}}`
	if err := os.WriteFile(path, []byte(input), 0o600); err != nil {
		t.Fatalf("write fixture: %v", err)
	}

	state, err := loadOperatorStateFor(context.Background())
	if err != nil {
		t.Fatalf("load operator state: %v", err)
	}
	if err := saveOperatorStateFor(context.Background(), state); err != nil {
		t.Fatalf("save operator state: %v", err)
	}

	var got struct {
		HostSafeguards map[string]struct {
			Config map[string]any `json:"config"`
		} `json:"host_safeguards"`
	}
	data, err := os.ReadFile(path)
	if err != nil {
		t.Fatalf("read round-trip state: %v", err)
	}
	if err := json.Unmarshal(data, &got); err != nil {
		t.Fatalf("decode round-trip state: %v", err)
	}
	want := map[string]any{"experience": "direct-desktop", "future_permission": true}
	if !reflect.DeepEqual(got.HostSafeguards["remote_desktop_access"].Config, want) {
		t.Fatalf("config = %#v, want %#v", got.HostSafeguards["remote_desktop_access"].Config, want)
	}
}

func TestOperatorStateRejectsInvalidSafeguardConfigBeforeWrite(t *testing.T) {
	oldPath := operatorStatePath
	t.Cleanup(func() { operatorStatePath = oldPath })
	operatorStatePath = func() (string, error) { return filepath.Join(t.TempDir(), "operator-state.json"), nil }
	srv := NewServer()
	w := doRequest(t, srv, http.MethodPatch, "/api/v2/operator-state", `{"host_safeguards":{"remote_desktop_access":{"opted_in":true,"config":{"experience":"not-a-real-experience"}}}}`)
	if w.Code != http.StatusBadRequest {
		t.Fatalf("invalid config status = %d, want %d: %s", w.Code, http.StatusBadRequest, w.Body.String())
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
