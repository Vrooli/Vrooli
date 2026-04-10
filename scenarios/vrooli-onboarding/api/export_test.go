package main

import (
	"encoding/json"
	"net/http"
	"os"
	"path/filepath"
	"testing"
)

// TestConfigExport verifies POST /api/v1/config/export writes file and returns path.
// [REQ:REQ-P0-006] - Config Export
func TestConfigExport(t *testing.T) {
	srv := newTestServer(t, []map[string]string{testResPostgres})

	outputDir := t.TempDir()
	body := `{"resources": {"postgres": {"enabled": true, "name": "postgres"}}, "output_dir": "` + outputDir + `"}`
	w := doPost(t, srv, "/api/v1/config/export", body)
	requireStatus(t, w, http.StatusOK)

	var resp configExportResponse
	decodeJSON(t, w, &resp)

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
	srv := newTestServer(t, []map[string]string{testResPostgres})

	w := doPost(t, srv, "/api/v1/config/export", `{"resources": {}}`)
	requireStatus(t, w, http.StatusBadRequest)
}

// TestConfigExportBackup verifies that existing config is backed up before overwriting.
// [REQ:REQ-P0-006] - Config Export
func TestConfigExportBackup(t *testing.T) {
	srv := newTestServer(t, []map[string]string{testResPostgres})

	outputDir := t.TempDir()

	// Write an existing config file
	existingPath := filepath.Join(outputDir, "generated-service.json")
	if err := os.WriteFile(existingPath, []byte(`{"old": true}`), 0o644); err != nil {
		t.Fatalf("failed to create existing config: %v", err)
	}

	body := `{"resources": {"postgres": {"enabled": true, "name": "postgres"}}, "output_dir": "` + outputDir + `"}`
	w := doPost(t, srv, "/api/v1/config/export", body)
	requireStatus(t, w, http.StatusOK)

	var resp configExportResponse
	decodeJSON(t, w, &resp)

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

// TestConfigExportNoOutputDirNoRoot verifies 400 when VROOLI_ROOT is unset and no output_dir provided.
// [REQ:REQ-P0-006] - Config Export
func TestConfigExportNoOutputDirNoRoot(t *testing.T) {
	srv := newTestServer(t, []map[string]string{testResPostgres})
	t.Setenv("VROOLI_ROOT", "")

	w := doPost(t, srv, "/api/v1/config/export",
		`{"resources": {"postgres": {"enabled": true, "name": "postgres"}}}`)
	requireStatus(t, w, http.StatusBadRequest)

	var body map[string]string
	decodeJSON(t, w, &body)
	if body["error"] == "" {
		t.Error("expected error message about VROOLI_ROOT")
	}
}

// TestConfigExportInvalidJSON verifies 400 for malformed JSON body.
// [REQ:REQ-P0-006] - Config Export
func TestConfigExportInvalidJSON(t *testing.T) {
	srv := newTestServer(t, []map[string]string{testResPostgres})

	w := doPost(t, srv, "/api/v1/config/export", "{bad json")
	requireStatus(t, w, http.StatusBadRequest)
}

// TestBackupFileNoExisting verifies no backup when file doesn't exist.
// [REQ:REQ-P0-006] - Config Export
func TestBackupFileNoExisting(t *testing.T) {
	dir := t.TempDir()
	backupDir, err := backupFile(filepath.Join(dir, "nonexistent.json"), dir)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if backupDir != "" {
		t.Errorf("expected empty backup dir, got %q", backupDir)
	}
}

// TestBackupFileSuccess verifies backup creation for existing file.
// [REQ:REQ-P0-006] - Config Export
func TestBackupFileSuccess(t *testing.T) {
	dir := t.TempDir()
	existing := filepath.Join(dir, "config.json")
	if err := os.WriteFile(existing, []byte(`{"test": true}`), 0o644); err != nil {
		t.Fatal(err)
	}

	backupDir, err := backupFile(existing, dir)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if backupDir == "" {
		t.Fatal("expected non-empty backup dir")
	}

	entries, err := os.ReadDir(backupDir)
	if err != nil {
		t.Fatalf("failed to read backup dir: %v", err)
	}
	if len(entries) != 1 {
		t.Errorf("expected 1 backup file, got %d", len(entries))
	}

	// Verify backup content matches original
	backupData, err := os.ReadFile(filepath.Join(backupDir, entries[0].Name()))
	if err != nil {
		t.Fatalf("failed to read backup: %v", err)
	}
	if string(backupData) != `{"test": true}` {
		t.Errorf("backup content = %q, want original content", string(backupData))
	}
}

// TestConfigExportMultipleResources verifies export with multiple resources.
// [REQ:REQ-P0-006] - Config Export
func TestConfigExportMultipleResources(t *testing.T) {
	srv := newTestServer(t, []map[string]string{testResPostgres, testResRedis})

	outputDir := t.TempDir()
	body := `{"resources": {"postgres": {"enabled": true, "name": "postgres"}, "redis": {"enabled": true, "name": "redis"}}, "output_dir": "` + outputDir + `"}`
	w := doPost(t, srv, "/api/v1/config/export", body)
	requireStatus(t, w, http.StatusOK)

	var resp configExportResponse
	decodeJSON(t, w, &resp)

	// Verify file content has both resources
	data, err := os.ReadFile(resp.Path)
	if err != nil {
		t.Fatalf("failed to read exported file: %v", err)
	}

	var exported serviceJSONSnippet
	if err := json.Unmarshal(data, &exported); err != nil {
		t.Fatalf("failed to parse: %v", err)
	}
	if len(exported.Resources) != 2 {
		t.Errorf("expected 2 resources in exported config, got %d", len(exported.Resources))
	}
}

// TestConfigExportWriteToReadOnlyDir verifies 500 when output directory is not writable.
// [REQ:REQ-P0-006] - Config Export
func TestConfigExportWriteToReadOnlyDir(t *testing.T) {
	if os.Getuid() == 0 {
		t.Skip("skipping permission test when running as root")
	}

	srv := newTestServer(t, []map[string]string{testResPostgres})

	// Create a read+execute but not write directory that prevents file creation
	outputDir := t.TempDir()
	if err := os.Chmod(outputDir, 0o555); err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() {
		if err := os.Chmod(outputDir, 0o755); err != nil {
			t.Logf("cleanup chmod: %v", err)
		}
	})

	body := `{"resources": {"postgres": {"enabled": true, "name": "postgres"}}, "output_dir": "` + outputDir + `"}`
	w := doPost(t, srv, "/api/v1/config/export", body)
	if w.Code == http.StatusOK {
		t.Error("expected error when writing to read-only directory")
	}

	var resp map[string]string
	decodeJSON(t, w, &resp)
	if resp["error"] == "" {
		t.Error("expected error message in response")
	}
}

// TestBackupFileReadOnlyBackupDir verifies error when backup directory cannot be created.
// [REQ:REQ-P0-006] - Config Export
func TestBackupFileReadOnlyBackupDir(t *testing.T) {
	if os.Getuid() == 0 {
		t.Skip("skipping permission test when running as root")
	}

	dir := t.TempDir()
	existing := filepath.Join(dir, "config.json")
	if err := os.WriteFile(existing, []byte(`{"test": true}`), 0o644); err != nil {
		t.Fatal(err)
	}

	// Make the directory read+execute but not write so we can stat files
	// but MkdirAll("backups") fails
	if err := os.Chmod(dir, 0o555); err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() {
		if err := os.Chmod(dir, 0o755); err != nil {
			t.Logf("cleanup chmod: %v", err)
		}
	})

	_, err := backupFile(existing, dir)
	if err == nil {
		t.Fatal("expected error when backup directory cannot be created")
	}
}

// TestConfigExportWithVrooliRoot verifies export uses VROOLI_ROOT when no output_dir.
// [REQ:REQ-P0-006] - Config Export
func TestConfigExportWithVrooliRoot(t *testing.T) {
	srv := newTestServer(t, []map[string]string{testResPostgres})

	// VROOLI_ROOT is already set by newTestServer; export without output_dir
	w := doPost(t, srv, "/api/v1/config/export",
		`{"resources": {"postgres": {"enabled": true, "name": "postgres"}}}`)
	requireStatus(t, w, http.StatusOK)

	var resp configExportResponse
	decodeJSON(t, w, &resp)
	if !resp.Success {
		t.Error("expected success=true when using VROOLI_ROOT default path")
	}
	if resp.Path == "" {
		t.Error("expected non-empty path")
	}

	// Verify file was written
	if _, err := os.Stat(resp.Path); err != nil {
		t.Errorf("exported file does not exist at %q: %v", resp.Path, err)
	}
}

// TestBackupFileUnreadableFile verifies error when existing file cannot be read.
// [REQ:REQ-P0-006] - Config Export
func TestBackupFileUnreadableFile(t *testing.T) {
	if os.Getuid() == 0 {
		t.Skip("skipping permission test when running as root")
	}

	dir := t.TempDir()
	existing := filepath.Join(dir, "config.json")
	if err := os.WriteFile(existing, []byte(`{"test": true}`), 0o644); err != nil {
		t.Fatal(err)
	}

	// Remove read permission from the file so Stat succeeds but ReadFile fails
	if err := os.Chmod(existing, 0o000); err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() {
		if err := os.Chmod(existing, 0o644); err != nil {
			t.Logf("cleanup chmod: %v", err)
		}
	})

	_, err := backupFile(existing, dir)
	if err == nil {
		t.Fatal("expected error when file cannot be read")
	}
}

// TestConfigExportMkdirAllFailure verifies 500 when output parent directory cannot be created.
// [REQ:REQ-P0-006] - Config Export
func TestConfigExportMkdirAllFailure(t *testing.T) {
	if os.Getuid() == 0 {
		t.Skip("skipping permission test when running as root")
	}

	srv := newTestServer(t, []map[string]string{testResPostgres})

	// Use a path where the parent exists but is read-only, so MkdirAll for nested path fails
	readOnlyDir := t.TempDir()
	nestedOutput := filepath.Join(readOnlyDir, "deep", "nested")
	if err := os.Chmod(readOnlyDir, 0o555); err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() {
		if err := os.Chmod(readOnlyDir, 0o755); err != nil {
			t.Logf("cleanup chmod: %v", err)
		}
	})

	body := `{"resources": {"postgres": {"enabled": true, "name": "postgres"}}, "output_dir": "` + nestedOutput + `"}`
	w := doPost(t, srv, "/api/v1/config/export", body)
	if w.Code == http.StatusOK {
		t.Error("expected error when output directory parent is not writable")
	}

	var resp map[string]string
	decodeJSON(t, w, &resp)
	if resp["error"] == "" {
		t.Error("expected error message in response")
	}
}

// TestConfigExportOverwritePreservesContent verifies the new export replaces old content.
// [REQ:REQ-P0-006] - Config Export
func TestConfigExportOverwritePreservesContent(t *testing.T) {
	srv := newTestServer(t, []map[string]string{testResPostgres, testResRedis})

	outputDir := t.TempDir()
	existingPath := filepath.Join(outputDir, "generated-service.json")

	// Write initial config with only postgres
	if err := os.WriteFile(existingPath, []byte(`{"resources":{"postgres":{"enabled":true,"name":"postgres"}}}`), 0o644); err != nil {
		t.Fatal(err)
	}

	// Export with redis - should overwrite
	body := `{"resources": {"redis": {"enabled": true, "name": "redis"}}, "output_dir": "` + outputDir + `"}`
	w := doPost(t, srv, "/api/v1/config/export", body)
	requireStatus(t, w, http.StatusOK)

	data, err := os.ReadFile(existingPath)
	if err != nil {
		t.Fatal(err)
	}

	var exported serviceJSONSnippet
	if err := json.Unmarshal(data, &exported); err != nil {
		t.Fatal(err)
	}
	if _, ok := exported.Resources["redis"]; !ok {
		t.Error("exported config should contain redis after overwrite")
	}
	if _, ok := exported.Resources["postgres"]; ok {
		t.Error("exported config should not contain old postgres after overwrite")
	}
}
