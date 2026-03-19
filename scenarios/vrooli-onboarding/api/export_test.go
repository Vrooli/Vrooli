package main

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"strings"
	"testing"
)

// TestConfigExport verifies POST /api/v1/config/export writes file and returns path.
// [REQ:REQ-P0-006] - Config Export
func TestConfigExport(t *testing.T) {
	srv := newTestServer(t, []map[string]string{
		{"name": "postgres", "status": "running", "installed": "true", "last_updated": "2026-01-01T00:00:00Z"},
	})

	outputDir := t.TempDir()
	body := `{"resources": {"postgres": {"enabled": true, "name": "postgres"}}, "output_dir": "` + outputDir + `"}`
	req := httptest.NewRequest(http.MethodPost, "/api/v1/config/export", strings.NewReader(body))
	req.Header.Set("Content-Type", "application/json")
	w := httptest.NewRecorder()
	srv.router.ServeHTTP(w, req)

	if w.Code != http.StatusOK {
		t.Fatalf("status = %d, want %d; body: %s", w.Code, http.StatusOK, w.Body.String())
	}

	var resp configExportResponse
	if err := json.Unmarshal(w.Body.Bytes(), &resp); err != nil {
		t.Fatalf("failed to decode response: %v", err)
	}

	if !resp.Success {
		t.Error("expected success=true")
	}

	expectedPath := filepath.Join(outputDir, "generated-service.json")
	if resp.Path != expectedPath {
		t.Errorf("path = %q, want %q", resp.Path, expectedPath)
	}

	// Verify file was actually written
	data, err := os.ReadFile(expectedPath)
	if err != nil {
		t.Fatalf("failed to read exported file: %v", err)
	}

	var exported serviceJSONSnippet
	if err := json.Unmarshal(data, &exported); err != nil {
		t.Fatalf("failed to parse exported file: %v", err)
	}

	if _, ok := exported.Resources["postgres"]; !ok {
		t.Error("exported config missing postgres resource")
	}
}

// TestConfigExportEmptyResources verifies 400 when resources are empty.
// [REQ:REQ-P0-006] - Config Export
func TestConfigExportEmptyResources(t *testing.T) {
	srv := newTestServer(t, []map[string]string{
		{"name": "postgres", "status": "running", "installed": "true", "last_updated": "2026-01-01T00:00:00Z"},
	})

	body := `{"resources": {}}`
	req := httptest.NewRequest(http.MethodPost, "/api/v1/config/export", strings.NewReader(body))
	req.Header.Set("Content-Type", "application/json")
	w := httptest.NewRecorder()
	srv.router.ServeHTTP(w, req)

	if w.Code != http.StatusBadRequest {
		t.Fatalf("status = %d, want %d; body: %s", w.Code, http.StatusBadRequest, w.Body.String())
	}
}

// TestConfigExportBackup verifies that existing config is backed up before overwriting.
// [REQ:REQ-P0-006] - Config Export
func TestConfigExportBackup(t *testing.T) {
	srv := newTestServer(t, []map[string]string{
		{"name": "postgres", "status": "running", "installed": "true", "last_updated": "2026-01-01T00:00:00Z"},
	})

	outputDir := t.TempDir()

	// Write an existing config file
	existingPath := filepath.Join(outputDir, "generated-service.json")
	if err := os.WriteFile(existingPath, []byte(`{"old": true}`), 0o644); err != nil {
		t.Fatalf("failed to create existing config: %v", err)
	}

	body := `{"resources": {"postgres": {"enabled": true, "name": "postgres"}}, "output_dir": "` + outputDir + `"}`
	req := httptest.NewRequest(http.MethodPost, "/api/v1/config/export", strings.NewReader(body))
	req.Header.Set("Content-Type", "application/json")
	w := httptest.NewRecorder()
	srv.router.ServeHTTP(w, req)

	if w.Code != http.StatusOK {
		t.Fatalf("status = %d, want %d; body: %s", w.Code, http.StatusOK, w.Body.String())
	}

	var resp configExportResponse
	if err := json.Unmarshal(w.Body.Bytes(), &resp); err != nil {
		t.Fatalf("failed to decode response: %v", err)
	}

	if resp.BackupDir == "" {
		t.Fatal("expected backup_dir to be set when overwriting existing config")
	}

	// Verify backup exists
	backups, err := os.ReadDir(resp.BackupDir)
	if err != nil {
		t.Fatalf("failed to read backup dir: %v", err)
	}
	if len(backups) == 0 {
		t.Error("expected at least one backup file")
	}
}
