package main

import (
	"context"
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"scenario-to-desktop-api/state"
	"testing"
	"time"

	"github.com/vrooli/api-core/database"
	"github.com/vrooli/api-core/filerouting"
	"github.com/vrooli/api-core/storage"
)

func TestDesktopEvidenceFixtureWritesOnlyToLeasedRoots(t *testing.T) {
	primaryRoot := t.TempDir()
	testRoot := t.TempDir()
	roots := filerouting.New(storage.Paths{DataDir: primaryRoot, StateDir: primaryRoot})
	if err := roots.InstallTestRoots(storage.Paths{DataDir: testRoot, StateDir: testRoot}, "fixture-lease", time.Minute); err != nil {
		t.Fatalf("install test roots: %v", err)
	}
	t.Cleanup(func() { _ = roots.ClearTestRoots("fixture-lease") })
	store, err := state.NewRoutedStore(roots)
	if err != nil {
		t.Fatalf("new routed store: %v", err)
	}
	server := &Server{fileRoots: roots, stateService: state.NewService(store, nil)}
	req := httptest.NewRequest(http.MethodPost, "/api/v1/test-fixtures/desktop-evidence", nil).WithContext(database.WithTestMode(context.Background()))
	res := httptest.NewRecorder()

	server.desktopEvidenceFixtureHandler(res, req)

	if res.Code != http.StatusOK {
		t.Fatalf("status = %d, want %d: %s", res.Code, http.StatusOK, res.Body.String())
	}
	if _, err := os.Stat(filepath.Join(testRoot, "desktop-evidence-fixture", "fixture-desktop.AppImage")); err != nil {
		t.Fatalf("leased artifact missing: %v", err)
	}
	if _, err := os.Stat(filepath.Join(primaryRoot, "desktop-evidence-fixture", "fixture-desktop.AppImage")); !os.IsNotExist(err) {
		t.Fatalf("fixture leaked to primary data root: %v", err)
	}
	loaded, err := server.stateService.LoadState(req.Context(), "scenario-to-desktop", state.LoadStateRequest{})
	if err != nil || !loaded.Found || len(loaded.State.BuildArtifacts) != 1 {
		t.Fatalf("leased fixture state = %#v, err = %v", loaded, err)
	}
	stats := roots.LeaseStats()
	if stats.TestRootWrites < 3 || stats.PrimaryWritesDuringTestMode != 0 {
		t.Fatalf("lease stats = %+v, want leased writes and no primary leak", stats)
	}
}

func TestDesktopEvidenceFixtureRefusesNonTestRequest(t *testing.T) {
	server := &Server{}
	res := httptest.NewRecorder()
	server.desktopEvidenceFixtureHandler(res, httptest.NewRequest(http.MethodPost, "/api/v1/test-fixtures/desktop-evidence", nil))
	if res.Code != http.StatusNotFound {
		t.Fatalf("status = %d, want %d", res.Code, http.StatusNotFound)
	}
}
