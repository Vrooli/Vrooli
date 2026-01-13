package main

import (
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func TestHandleVariantSnapshotSync_RequiresAuth(t *testing.T) {
	db := setupTestDB(t)
	defer db.Close()

	initSessionStore()
	server := &Server{db: db}

	req := httptest.NewRequest(http.MethodPost, "/api/v1/admin/variants/sync", nil)
	resp := httptest.NewRecorder()

	server.requireAdmin(handleVariantSnapshotSync(NewVariantService(db, testVariantSpace()), NewContentService(db)))(resp, req)

	if resp.Code != http.StatusUnauthorized {
		t.Fatalf("expected 401, got %d", resp.Code)
	}
}

func TestHandleVariantSnapshotSync_SyncsSnapshots(t *testing.T) {
	db := setupTestDB(t)
	defer db.Close()

	initSessionStore()
	server := &Server{db: db}

	dir := t.TempDir()
	writeSnapshot(t, dir, VariantSnapshotInput{
		Variant: VariantSnapshotMetaInput{
			Slug:        "sync-handler",
			Name:        "Sync Handler",
			Description: "Synced",
			Axes:        defaultAxesSelection(),
		},
		Sections: []VariantSectionInput{
			{
				SectionType: "hero",
				Content: map[string]interface{}{
					"title": "Synced hero",
				},
				Order:   1,
				Enabled: boolPtr(true),
			},
		},
	})

	t.Setenv("VARIANT_SNAPSHOT_DIR", dir)

	req := httptest.NewRequest(http.MethodPost, "/api/v1/admin/variants/sync", nil)
	attachAdminSession(t, req, defaultAdminEmail)
	resp := httptest.NewRecorder()

	vs := NewVariantService(db, testVariantSpace())
	cs := NewContentService(db)

	server.requireAdmin(handleVariantSnapshotSync(vs, cs))(resp, req)

	if resp.Code != http.StatusOK {
		t.Fatalf("expected 200, got %d: %s", resp.Code, resp.Body.String())
	}

	updated, err := vs.GetVariantBySlug("sync-handler")
	if err != nil {
		t.Fatalf("expected synced variant: %v", err)
	}
	if updated.Name != "Sync Handler" {
		t.Fatalf("expected synced variant name, got %s", updated.Name)
	}
}

func TestHandleVariantSnapshotSync_ReturnsErrorOnInvalidDir(t *testing.T) {
	db := setupTestDB(t)
	defer db.Close()

	initSessionStore()
	server := &Server{db: db}

	tempDir := t.TempDir()
	filePath := filepath.Join(tempDir, "not-a-dir.json")
	if err := os.WriteFile(filePath, []byte(`{}`), 0o644); err != nil {
		t.Fatalf("write temp file: %v", err)
	}

	t.Setenv("VARIANT_SNAPSHOT_DIR", filePath)

	req := httptest.NewRequest(http.MethodPost, "/api/v1/admin/variants/sync", nil)
	attachAdminSession(t, req, defaultAdminEmail)
	resp := httptest.NewRecorder()

	server.requireAdmin(handleVariantSnapshotSync(NewVariantService(db, testVariantSpace()), NewContentService(db)))(resp, req)

	if resp.Code != http.StatusInternalServerError {
		t.Fatalf("expected 500, got %d", resp.Code)
	}
	if !strings.Contains(resp.Body.String(), "Failed to sync variant snapshots") {
		t.Fatalf("expected error message in response, got %q", resp.Body.String())
	}
}
