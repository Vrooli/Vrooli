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
	db, err := sql.Open("postgres", "postgres://postgres:postgres@localhost:5432/test_landing?sslmode=disable")
	if err != nil {
		t.Skip("Test database not available:", err)
	}

	// Create tables
	_, err = db.Exec(`
		CREATE TABLE IF NOT EXISTS site_branding (
			id INTEGER PRIMARY KEY DEFAULT 1 CHECK (id = 1),
			site_name TEXT NOT NULL DEFAULT 'Test Site',
			tagline TEXT,
			logo_url TEXT,
			logo_icon_url TEXT,
			favicon_url TEXT,
			apple_touch_icon_url TEXT,
			default_title TEXT,
			default_description TEXT,
			default_og_image_url TEXT,
			theme_primary_color TEXT,
			theme_background_color TEXT,
			canonical_base_url TEXT,
			google_site_verification TEXT,
			robots_txt TEXT,
			created_at TIMESTAMP DEFAULT NOW(),
			updated_at TIMESTAMP DEFAULT NOW()
		);
	`)
	if err != nil {
		t.Skip("Failed to create test table:", err)
	}

	// Seed default data
	_, _ = db.Exec(`
		INSERT INTO site_branding (id, site_name, robots_txt)
		VALUES (1, 'Test Site', 'User-agent: *\nAllow: /')
		ON CONFLICT (id) DO NOTHING
	`)

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

	update := SiteBrandingUpdate{
		SiteName: stringPtr("Updated Site Name"),
		Tagline:  stringPtr("New tagline"),
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

	var branding PublicBranding
	if err := json.NewDecoder(rec.Body).Decode(&branding); err != nil {
		t.Fatalf("failed to decode response: %v", err)
	}

	if branding.SiteName == "" {
		t.Error("expected non-empty site name in public branding")
	}
}

func TestClearBrandingField(t *testing.T) {
	db := setupTestBrandingDB(t)
	defer db.Close()

	service := NewBrandingService(db)

	// First set a value
	update := SiteBrandingUpdate{Tagline: stringPtr("Test tagline")}
	if err := service.Update(&update); err != nil {
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

func stringPtr(s string) *string {
	return &s
}
