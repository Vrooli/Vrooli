package main

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"testing"

	varianthttp "landing-page-business-suite-api/handlers/experimentation"
	"landing-page-business-suite-api/internal/experimentation"
	"landing-page-business-suite-api/internal/logx"
)

func TestHandleVariantSnapshotSync_RequiresAuth(t *testing.T) {
	db := setupTestDB(t)

	tmpDir := t.TempDir()
	variantsDir := filepath.Join(tmpDir, "variants")
	if err := os.MkdirAll(variantsDir, 0o755); err != nil {
		t.Fatalf("failed to create variants dir: %v", err)
	}
	brandingPath := filepath.Join(tmpDir, "branding.json")
	if err := os.WriteFile(brandingPath, []byte(`{"site_name": "Test"}`), 0o644); err != nil {
		t.Fatalf("failed to write branding file: %v", err)
	}

	cs := experimentation.NewConfigStore(variantsDir, brandingPath, experimentation.DefaultVariantSpace())
	if err := cs.LoadAll(); err != nil {
		t.Fatalf("failed to load config: %v", err)
	}

	sessionMgr := initSessionManager()
	server := &Server{db: db, configStore: cs, sessionManager: sessionMgr}

	req := httptest.NewRequest(http.MethodPost, "/api/v1/admin/variants/sync", nil)
	resp := httptest.NewRecorder()

	server.requireAdmin(varianthttp.Sync(varianthttp.WriteDependencies{Store: cs, WriteJSON: writeJSON, WriteError: writeJSONError, Log: logx.Info, LogError: logx.Error}))(resp, req)

	if resp.Code != http.StatusUnauthorized {
		t.Fatalf("expected 401, got %d", resp.Code)
	}
}

func TestHandleVariantSnapshotSync_SyncsSnapshots(t *testing.T) {
	db := setupTestDB(t)

	dir := t.TempDir()
	variantsDir := filepath.Join(dir, "variants")
	if err := os.MkdirAll(variantsDir, 0o755); err != nil {
		t.Fatalf("failed to create variants dir: %v", err)
	}

	// Write a test variant snapshot
	writeSnapshot(t, variantsDir, experimentation.VariantSnapshotInput{
		Variant: experimentation.VariantSnapshotMetaInput{
			Slug:        "sync-handler",
			Name:        "Sync Handler",
			Description: "Synced",
			Axes:        defaultAxesSelection(),
		},
		Sections: []experimentation.VariantSectionInput{
			{
				SectionType: "hero",
				Content:     json.RawMessage(`{"title": "Synced hero"}`),
				Order:       1,
				Enabled:     boolPtr(true),
			},
		},
	})

	brandingPath := filepath.Join(dir, "branding.json")
	if err := os.WriteFile(brandingPath, []byte(`{"site_name": "Test"}`), 0o644); err != nil {
		t.Fatalf("failed to write branding file: %v", err)
	}

	// Load the config store
	cs := experimentation.NewConfigStore(variantsDir, brandingPath, experimentation.DefaultVariantSpace())
	if err := cs.LoadAll(); err != nil {
		t.Fatalf("failed to load config: %v", err)
	}

	// Set env var for the handler
	t.Setenv("VARIANT_SNAPSHOT_DIR", variantsDir)

	sessionMgr := initSessionManager()
	server := &Server{db: db, configStore: cs, sessionManager: sessionMgr}

	req := httptest.NewRequest(http.MethodPost, "/api/v1/admin/variants/sync", nil)
	attachAdminSession(t, sessionMgr, req, defaultAdminEmail)
	resp := httptest.NewRecorder()

	server.requireAdmin(varianthttp.Sync(varianthttp.WriteDependencies{Store: cs, WriteJSON: writeJSON, WriteError: writeJSONError, Log: logx.Info, LogError: logx.Error}))(resp, req)

	if resp.Code != http.StatusOK {
		t.Fatalf("expected 200, got %d: %s", resp.Code, resp.Body.String())
	}

	// Verify the variant was loaded
	variant, err := cs.GetVariant("sync-handler")
	if err != nil {
		t.Fatalf("expected synced variant: %v", err)
	}
	if variant.Variant.Name != "Sync Handler" {
		t.Fatalf("expected synced variant name, got %s", variant.Variant.Name)
	}
}

func TestHandleVariantSnapshotSync_ReturnsErrorOnInvalidDir(t *testing.T) {
	db := setupTestDB(t)

	// Create a ConfigStore pointing to a file (not a directory)
	tempDir := t.TempDir()
	filePath := filepath.Join(tempDir, "not-a-dir.json")
	if err := os.WriteFile(filePath, []byte(`{}`), 0o644); err != nil {
		t.Fatalf("write temp file: %v", err)
	}

	brandingPath := filepath.Join(tempDir, "branding.json")
	if err := os.WriteFile(brandingPath, []byte(`{"site_name": "Test"}`), 0o644); err != nil {
		t.Fatalf("failed to write branding file: %v", err)
	}

	// Use the file path as the variants dir (which is invalid)
	cs := experimentation.NewConfigStore(filePath, brandingPath, experimentation.DefaultVariantSpace())

	sessionMgr := initSessionManager()
	server := &Server{db: db, configStore: cs, sessionManager: sessionMgr}

	t.Setenv("VARIANT_SNAPSHOT_DIR", filePath)

	req := httptest.NewRequest(http.MethodPost, "/api/v1/admin/variants/sync", nil)
	attachAdminSession(t, sessionMgr, req, defaultAdminEmail)
	resp := httptest.NewRecorder()

	server.requireAdmin(varianthttp.Sync(varianthttp.WriteDependencies{Store: cs, WriteJSON: writeJSON, WriteError: writeJSONError, Log: logx.Info, LogError: logx.Error}))(resp, req)

	// ConfigStore.LoadAll returns an error when variantsDir points to a file instead of a directory
	// So we expect 500 here
	if resp.Code != http.StatusInternalServerError {
		t.Fatalf("expected 500, got %d: %s", resp.Code, resp.Body.String())
	}
}
