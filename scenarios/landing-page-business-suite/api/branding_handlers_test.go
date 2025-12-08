package main

import (
	"bytes"
	"database/sql"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"
)

func setupTestBrandingDB(t *testing.T) *sql.DB {
	db := setupTestDB(t)
	if _, err := db.Exec(`DELETE FROM site_branding`); err != nil {
		t.Fatalf("failed to clean site_branding: %v", err)
	}
	if _, err := db.Exec(`
		INSERT INTO site_branding (id, site_name, robots_txt)
		VALUES (1, 'Test Site', 'User-agent: *\nAllow: /')
	`); err != nil {
		t.Fatalf("failed to seed site branding: %v", err)
	}

	return db
}

func TestGetBranding(t *testing.T) {
	db := setupTestBrandingDB(t)
	defer db.Close()

	service := NewBrandingService(db)
	handler := handleGetBranding(service)

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
	db := setupTestBrandingDB(t)
	defer db.Close()

	service := NewBrandingService(db)
	handler := handleUpdateBranding(service)

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
	branding, err := service.Get()
	if err != nil {
		t.Fatalf("failed to get branding: %v", err)
	}

	if branding.SiteName != "Updated Site Name" {
		t.Errorf("expected site name 'Updated Site Name', got '%s'", branding.SiteName)
	}

	if branding.Tagline == nil || *branding.Tagline != "New tagline" {
		t.Error("expected tagline to be updated")
	}
}

func TestGetPublicBranding(t *testing.T) {
	db := setupTestBrandingDB(t)
	defer db.Close()

	service := NewBrandingService(db)
	handler := handleGetPublicBranding(service)

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

func TestGetBranding_ServiceFailure(t *testing.T) {
	db := setupTestBrandingDB(t)
	service := NewBrandingService(db)
	db.Close()

	handler := handleGetBranding(service)

	req := httptest.NewRequest(http.MethodGet, "/api/v1/admin/branding", nil)
	rec := httptest.NewRecorder()

	handler.ServeHTTP(rec, req)

	if rec.Code != http.StatusInternalServerError {
		t.Fatalf("expected 500 when branding service fails, got %d", rec.Code)
	}
}

func TestGetPublicBranding_ExposesOnlyPublicFields(t *testing.T) {
	db := setupTestBrandingDB(t)
	defer db.Close()

	service := NewBrandingService(db)
	secretDescription := "Hidden description"
	verifyToken := "verify-me"
	logoIconURL := "https://example.com/icon.png"
	if _, err := service.Update(&BrandingUpdateRequest{
		Tagline:                strPtr("Visible tagline"),
		LogoIconURL:            strPtr(logoIconURL),
		DefaultDescription:     strPtr(secretDescription),
		GoogleSiteVerification: strPtr(verifyToken),
	}); err != nil {
		t.Fatalf("failed to seed branding: %v", err)
	}

	handler := handleGetPublicBranding(service)

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
	db := setupTestBrandingDB(t)
	defer db.Close()

	service := NewBrandingService(db)

	// First set a value
	update := BrandingUpdateRequest{Tagline: strPtr("Test tagline")}
	if _, err := service.Update(&update); err != nil {
		t.Fatalf("failed to update branding: %v", err)
	}

	// Clear the field
	handler := handleClearBrandingField(service)
	body, _ := json.Marshal(map[string]string{"field": "tagline"})
	req := httptest.NewRequest(http.MethodPost, "/api/v1/admin/branding/clear-field", bytes.NewReader(body))
	req.Header.Set("Content-Type", "application/json")
	rec := httptest.NewRecorder()

	handler.ServeHTTP(rec, req)

	if rec.Code != http.StatusOK {
		t.Errorf("expected status 200, got %d: %s", rec.Code, rec.Body.String())
	}

	// Verify the field is cleared
	branding, err := service.Get()
	if err != nil {
		t.Fatalf("failed to get branding: %v", err)
	}

	if branding.Tagline != nil && *branding.Tagline != "" {
		t.Error("expected tagline to be cleared")
	}
}

func TestUpdateBranding_InvalidBody(t *testing.T) {
	db := setupTestBrandingDB(t)
	defer db.Close()

	service := NewBrandingService(db)
	handler := handleUpdateBranding(service)

	req := httptest.NewRequest(http.MethodPut, "/api/v1/admin/branding", bytes.NewBufferString("{bad json"))
	req.Header.Set("Content-Type", "application/json")
	rec := httptest.NewRecorder()

	handler.ServeHTTP(rec, req)

	if rec.Code != http.StatusBadRequest {
		t.Fatalf("expected 400 for invalid body, got %d", rec.Code)
	}
}

func TestClearBrandingField_RequiresFieldName(t *testing.T) {
	db := setupTestBrandingDB(t)
	defer db.Close()

	service := NewBrandingService(db)
	handler := handleClearBrandingField(service)

	req := httptest.NewRequest(http.MethodPost, "/api/v1/admin/branding/clear-field", bytes.NewBufferString(`{}`))
	req.Header.Set("Content-Type", "application/json")
	rec := httptest.NewRecorder()

	handler.ServeHTTP(rec, req)

	if rec.Code != http.StatusBadRequest {
		t.Fatalf("expected 400 for missing field name, got %d", rec.Code)
	}
}

func TestClearBrandingField_FailureReturns500(t *testing.T) {
	db := setupTestBrandingDB(t)
	service := NewBrandingService(db)
	handler := handleClearBrandingField(service)
	db.Close()

	req := httptest.NewRequest(http.MethodPost, "/api/v1/admin/branding/clear-field", bytes.NewBufferString(`{"field":"tagline"}`))
	req.Header.Set("Content-Type", "application/json")
	rec := httptest.NewRecorder()

	handler.ServeHTTP(rec, req)

	if rec.Code != http.StatusInternalServerError {
		t.Fatalf("expected 500 when clear operation fails, got %d", rec.Code)
	}
}

func TestClearBrandingField_IgnoresUnsupportedFieldWithoutMutation(t *testing.T) {
	db := setupTestBrandingDB(t)
	defer db.Close()

	service := NewBrandingService(db)
	original := "Keep me"
	if _, err := service.Update(&BrandingUpdateRequest{Tagline: strPtr(original)}); err != nil {
		t.Fatalf("failed to seed tagline: %v", err)
	}

	handler := handleClearBrandingField(service)
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

	stored, err := service.Get()
	if err != nil {
		t.Fatalf("failed to fetch stored branding: %v", err)
	}
	if stored.Tagline == nil || *stored.Tagline != original {
		t.Fatalf("expected stored branding to preserve tagline after ignored field, got %v", stored.Tagline)
	}
}

func strPtr(s string) *string {
	return &s
}
