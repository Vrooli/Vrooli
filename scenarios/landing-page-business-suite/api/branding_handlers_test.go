package main

import (
	"bytes"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"testing"
)

func setupTestBrandingConfigStore(t *testing.T) (*ConfigStore, func()) {
	t.Helper()

	tmpDir := t.TempDir()
	brandingPath := filepath.Join(tmpDir, "branding.json")
	if err := os.WriteFile(brandingPath, []byte(`{
		"site_name": "Test Site",
		"robots_txt": "User-agent: *\nAllow: /"
	}`), 0o644); err != nil {
		t.Fatalf("failed to write branding file: %v", err)
	}

	cs := NewConfigStore("", brandingPath, nil)
	if err := cs.LoadAll(); err != nil {
		t.Fatalf("failed to load config: %v", err)
	}

	cleanup := func() {
		// tmpDir is cleaned up automatically by t.TempDir()
	}

	return cs, cleanup
}

func TestGetBranding(t *testing.T) {
	cs, cleanup := setupTestBrandingConfigStore(t)
	defer cleanup()

	handler := handleGetBranding(cs)

	req := httptest.NewRequest(http.MethodGet, "/api/v1/admin/branding", nil)
	rec := httptest.NewRecorder()

	handler.ServeHTTP(rec, req)

	if rec.Code != http.StatusOK {
		t.Errorf("expected status 200, got %d", rec.Code)
	}

	var branding SiteBranding
	if err := json.NewDecoder(rec.Body).Decode(&branding); err != nil {
		t.Fatalf("failed to decode response: %v", err)
	}

	if branding.SiteName == "" {
		t.Error("expected non-empty site name")
	}
}

func TestUpdateBranding(t *testing.T) {
	cs, cleanup := setupTestBrandingConfigStore(t)
	defer cleanup()

	handler := handleUpdateBranding(cs)

	update := BrandingUpdateRequest{
		SiteName: strPtr("Updated Site Name"),
		Tagline:  strPtr("New tagline"),
	}
	body, _ := json.Marshal(update)

	req := httptest.NewRequest(http.MethodPut, "/api/v1/admin/branding", bytes.NewReader(body))
	req.Header.Set("Content-Type", "application/json")
	rec := httptest.NewRecorder()

	handler.ServeHTTP(rec, req)

	if rec.Code != http.StatusOK {
		t.Errorf("expected status 200, got %d: %s", rec.Code, rec.Body.String())
	}

	// Verify the update
	branding := cs.GetBranding()

	if branding.SiteName != "Updated Site Name" {
		t.Errorf("expected site name 'Updated Site Name', got '%s'", branding.SiteName)
	}

	if branding.Tagline == nil || *branding.Tagline != "New tagline" {
		t.Error("expected tagline to be updated")
	}
}

func TestGetPublicBranding(t *testing.T) {
	cs, cleanup := setupTestBrandingConfigStore(t)
	defer cleanup()

	handler := handleGetPublicBranding(cs)

	req := httptest.NewRequest(http.MethodGet, "/api/v1/branding", nil)
	rec := httptest.NewRecorder()

	handler.ServeHTTP(rec, req)

	if rec.Code != http.StatusOK {
		t.Errorf("expected status 200, got %d", rec.Code)
	}

	var branding map[string]interface{}
	if err := json.NewDecoder(rec.Body).Decode(&branding); err != nil {
		t.Fatalf("failed to decode response: %v", err)
	}

	if branding["site_name"] == "" {
		t.Error("expected non-empty site name in public branding")
	}
}

func TestGetPublicBranding_ExposesOnlyPublicFields(t *testing.T) {
	cs, cleanup := setupTestBrandingConfigStore(t)
	defer cleanup()

	// Update branding with various fields
	logoIconURL := "https://example.com/icon.png"
	if _, err := cs.UpdateBranding(&BrandingUpdateRequest{
		Tagline:                strPtr("Visible tagline"),
		LogoIconURL:            strPtr(logoIconURL),
		DefaultDescription:     strPtr("Hidden description"),
		GoogleSiteVerification: strPtr("verify-me"),
	}); err != nil {
		t.Fatalf("failed to update branding: %v", err)
	}

	handler := handleGetPublicBranding(cs)

	req := httptest.NewRequest(http.MethodGet, "/api/v1/branding", nil)
	rec := httptest.NewRecorder()

	handler.ServeHTTP(rec, req)

	if rec.Code != http.StatusOK {
		t.Fatalf("expected status 200, got %d: %s", rec.Code, rec.Body.String())
	}

	var branding map[string]interface{}
	if err := json.NewDecoder(rec.Body).Decode(&branding); err != nil {
		t.Fatalf("failed to decode response: %v", err)
	}

	if branding["tagline"] != "Visible tagline" {
		t.Fatalf("expected tagline to be included, got %v", branding["tagline"])
	}
	if branding["logo_icon_url"] != logoIconURL {
		t.Fatalf("expected logo_icon_url to pass through, got %v", branding["logo_icon_url"])
	}
	for _, forbidden := range []string{"default_description", "google_site_verification", "robots_txt"} {
		if _, ok := branding[forbidden]; ok {
			t.Fatalf("expected %s to be omitted from public payload", forbidden)
		}
	}
}

func TestClearBrandingField(t *testing.T) {
	cs, cleanup := setupTestBrandingConfigStore(t)
	defer cleanup()

	// First set a value
	if _, err := cs.UpdateBranding(&BrandingUpdateRequest{Tagline: strPtr("Test tagline")}); err != nil {
		t.Fatalf("failed to update branding: %v", err)
	}

	// Clear the field
	handler := handleClearBrandingField(cs)
	body, _ := json.Marshal(map[string]string{"field": "tagline"})
	req := httptest.NewRequest(http.MethodPost, "/api/v1/admin/branding/clear-field", bytes.NewReader(body))
	req.Header.Set("Content-Type", "application/json")
	rec := httptest.NewRecorder()

	handler.ServeHTTP(rec, req)

	if rec.Code != http.StatusOK {
		t.Errorf("expected status 200, got %d: %s", rec.Code, rec.Body.String())
	}

	// Verify the field is cleared
	branding := cs.GetBranding()

	if branding.Tagline != nil && *branding.Tagline != "" {
		t.Error("expected tagline to be cleared")
	}
}

func TestUpdateBranding_InvalidBody(t *testing.T) {
	cs, cleanup := setupTestBrandingConfigStore(t)
	defer cleanup()

	handler := handleUpdateBranding(cs)

	req := httptest.NewRequest(http.MethodPut, "/api/v1/admin/branding", bytes.NewBufferString("{bad json"))
	req.Header.Set("Content-Type", "application/json")
	rec := httptest.NewRecorder()

	handler.ServeHTTP(rec, req)

	if rec.Code != http.StatusBadRequest {
		t.Fatalf("expected 400 for invalid body, got %d", rec.Code)
	}
}

func TestClearBrandingField_RequiresFieldName(t *testing.T) {
	cs, cleanup := setupTestBrandingConfigStore(t)
	defer cleanup()

	handler := handleClearBrandingField(cs)

	req := httptest.NewRequest(http.MethodPost, "/api/v1/admin/branding/clear-field", bytes.NewBufferString(`{}`))
	req.Header.Set("Content-Type", "application/json")
	rec := httptest.NewRecorder()

	handler.ServeHTTP(rec, req)

	if rec.Code != http.StatusBadRequest {
		t.Fatalf("expected 400 for missing field name, got %d", rec.Code)
	}
}

func TestClearBrandingField_IgnoresUnsupportedFieldWithoutMutation(t *testing.T) {
	cs, cleanup := setupTestBrandingConfigStore(t)
	defer cleanup()

	original := "Keep me"
	if _, err := cs.UpdateBranding(&BrandingUpdateRequest{Tagline: strPtr(original)}); err != nil {
		t.Fatalf("failed to seed tagline: %v", err)
	}

	handler := handleClearBrandingField(cs)
	req := httptest.NewRequest(http.MethodPost, "/api/v1/admin/branding/clear-field", bytes.NewBufferString(`{"field":"nonexistent"}`))
	req.Header.Set("Content-Type", "application/json")
	rec := httptest.NewRecorder()

	handler.ServeHTTP(rec, req)

	if rec.Code != http.StatusOK {
		t.Fatalf("expected status 200 when field is ignored, got %d: %s", rec.Code, rec.Body.String())
	}

	var branding SiteBranding
	if err := json.NewDecoder(rec.Body).Decode(&branding); err != nil {
		t.Fatalf("failed to decode branding: %v", err)
	}
	if branding.Tagline == nil || *branding.Tagline != original {
		t.Fatalf("expected tagline to remain unchanged, got %v", branding.Tagline)
	}

	stored := cs.GetBranding()
	if stored.Tagline == nil || *stored.Tagline != original {
		t.Fatalf("expected stored branding to preserve tagline after ignored field, got %v", stored.Tagline)
	}
}

func strPtr(s string) *string {
	return &s
}
