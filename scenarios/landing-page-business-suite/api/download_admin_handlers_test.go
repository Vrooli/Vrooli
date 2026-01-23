package main

import (
	"database/sql"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/gorilla/mux"
)

// --- handleAdminListDownloadApps Tests ---

func TestHandleAdminListDownloadApps_Success(t *testing.T) {
	db := setupTestDB(t)
	defer db.Close()
	cleanupDownloadApps(t, db)

	// Insert test apps
	insertTestDownloadApp(t, db, "test_bundle", "app1", "App One")
	insertTestDownloadApp(t, db, "test_bundle", "app2", "App Two")

	downloads := NewDownloadService(db)
	plans := newTestPlanService(t, "test_bundle")

	handler := handleAdminListDownloadApps(downloads, plans)

	req := httptest.NewRequest(http.MethodGet, "/api/v1/admin/downloads/apps", nil)
	w := httptest.NewRecorder()

	handler(w, req)

	if w.Code != http.StatusOK {
		t.Errorf("Expected status %d, got %d: %s", http.StatusOK, w.Code, w.Body.String())
	}

	var resp map[string]interface{}
	if err := json.NewDecoder(w.Body).Decode(&resp); err != nil {
		t.Fatalf("Failed to decode response: %v", err)
	}

	apps, ok := resp["apps"].([]interface{})
	if !ok {
		t.Fatal("Expected 'apps' array in response")
	}
	if len(apps) != 2 {
		t.Errorf("Expected 2 apps, got %d", len(apps))
	}
}

func TestHandleAdminListDownloadApps_Empty(t *testing.T) {
	db := setupTestDB(t)
	defer db.Close()
	cleanupDownloadApps(t, db)

	downloads := NewDownloadService(db)
	plans := newTestPlanService(t, "empty_bundle")

	handler := handleAdminListDownloadApps(downloads, plans)

	req := httptest.NewRequest(http.MethodGet, "/api/v1/admin/downloads/apps", nil)
	w := httptest.NewRecorder()

	handler(w, req)

	if w.Code != http.StatusOK {
		t.Errorf("Expected status %d, got %d", http.StatusOK, w.Code)
	}

	var resp map[string]interface{}
	if err := json.NewDecoder(w.Body).Decode(&resp); err != nil {
		t.Fatalf("Failed to decode response: %v", err)
	}

	apps, ok := resp["apps"].([]interface{})
	if !ok {
		// apps should be an empty array, not null
		if resp["apps"] != nil {
			t.Fatal("Expected 'apps' to be an array or nil")
		}
		return
	}
	if len(apps) != 0 {
		t.Errorf("Expected empty apps array, got %d apps", len(apps))
	}
}

// --- handleAdminCreateDownloadApp Tests ---

func TestHandleAdminCreateDownloadApp_Success(t *testing.T) {
	db := setupTestDB(t)
	defer db.Close()
	cleanupDownloadApps(t, db)

	downloads := NewDownloadService(db)
	plans := newTestPlanService(t, "create_bundle")

	handler := handleAdminCreateDownloadApp(downloads, plans)

	body := `{
		"app_key": "new-app",
		"name": "New Application",
		"tagline": "A great app",
		"description": "This is a great app"
	}`
	req := httptest.NewRequest(http.MethodPost, "/api/v1/admin/downloads/apps", strings.NewReader(body))
	req.Header.Set("Content-Type", "application/json")
	w := httptest.NewRecorder()

	handler(w, req)

	if w.Code != http.StatusOK {
		t.Errorf("Expected status %d, got %d: %s", http.StatusOK, w.Code, w.Body.String())
	}

	var resp DownloadApp
	if err := json.NewDecoder(w.Body).Decode(&resp); err != nil {
		t.Fatalf("Failed to decode response: %v", err)
	}

	if resp.AppKey != "new-app" {
		t.Errorf("Expected app_key 'new-app', got '%s'", resp.AppKey)
	}
	if resp.Name != "New Application" {
		t.Errorf("Expected name 'New Application', got '%s'", resp.Name)
	}
}

func TestHandleAdminCreateDownloadApp_InvalidJSON(t *testing.T) {
	db := setupTestDB(t)
	defer db.Close()

	downloads := NewDownloadService(db)
	plans := newTestPlanService(t, "create_invalid")

	handler := handleAdminCreateDownloadApp(downloads, plans)

	req := httptest.NewRequest(http.MethodPost, "/api/v1/admin/downloads/apps", strings.NewReader("{invalid"))
	req.Header.Set("Content-Type", "application/json")
	w := httptest.NewRecorder()

	handler(w, req)

	if w.Code != http.StatusBadRequest {
		t.Errorf("Expected status %d for invalid JSON, got %d", http.StatusBadRequest, w.Code)
	}
}

func TestHandleAdminCreateDownloadApp_ValidationError(t *testing.T) {
	db := setupTestDB(t)
	defer db.Close()

	downloads := NewDownloadService(db)
	plans := newTestPlanService(t, "create_validation")

	handler := handleAdminCreateDownloadApp(downloads, plans)

	// Missing name
	body := `{"app_key": "test-app"}`
	req := httptest.NewRequest(http.MethodPost, "/api/v1/admin/downloads/apps", strings.NewReader(body))
	req.Header.Set("Content-Type", "application/json")
	w := httptest.NewRecorder()

	handler(w, req)

	if w.Code != http.StatusBadRequest {
		t.Errorf("Expected status %d for validation error, got %d", http.StatusBadRequest, w.Code)
	}
}

func TestHandleAdminCreateDownloadApp_WithPlatforms(t *testing.T) {
	db := setupTestDB(t)
	defer db.Close()
	cleanupDownloadApps(t, db)

	downloads := NewDownloadService(db)
	plans := newTestPlanService(t, "create_platforms")

	handler := handleAdminCreateDownloadApp(downloads, plans)

	body := `{
		"app_key": "platform-app",
		"name": "Platform Application",
		"platforms": [
			{
				"platform": "windows",
				"artifact_source": "direct",
				"artifact_url": "https://example.com/app.exe",
				"release_version": "1.0.0"
			},
			{
				"platform": "mac",
				"artifact_source": "direct",
				"artifact_url": "https://example.com/app.dmg",
				"release_version": "1.0.0"
			}
		]
	}`
	req := httptest.NewRequest(http.MethodPost, "/api/v1/admin/downloads/apps", strings.NewReader(body))
	req.Header.Set("Content-Type", "application/json")
	w := httptest.NewRecorder()

	handler(w, req)

	if w.Code != http.StatusOK {
		t.Errorf("Expected status %d, got %d: %s", http.StatusOK, w.Code, w.Body.String())
	}

	var resp DownloadApp
	if err := json.NewDecoder(w.Body).Decode(&resp); err != nil {
		t.Fatalf("Failed to decode response: %v", err)
	}

	if len(resp.Platforms) != 2 {
		t.Errorf("Expected 2 platforms, got %d", len(resp.Platforms))
	}
}

// --- handleAdminSaveDownloadApp Tests ---

func TestHandleAdminSaveDownloadApp_Success(t *testing.T) {
	db := setupTestDB(t)
	defer db.Close()
	cleanupDownloadApps(t, db)

	// Create existing app
	insertTestDownloadApp(t, db, "save_bundle", "existing-app", "Original Name")

	downloads := NewDownloadService(db)
	plans := newTestPlanService(t, "save_bundle")

	handler := handleAdminSaveDownloadApp(downloads, plans)

	router := mux.NewRouter()
	router.HandleFunc("/api/v1/admin/downloads/apps/{app_key}", handler).Methods("PUT")

	body := `{
		"name": "Updated Name",
		"tagline": "Updated tagline"
	}`
	req := httptest.NewRequest(http.MethodPut, "/api/v1/admin/downloads/apps/existing-app", strings.NewReader(body))
	req.Header.Set("Content-Type", "application/json")
	w := httptest.NewRecorder()

	router.ServeHTTP(w, req)

	if w.Code != http.StatusOK {
		t.Errorf("Expected status %d, got %d: %s", http.StatusOK, w.Code, w.Body.String())
	}

	var resp DownloadApp
	if err := json.NewDecoder(w.Body).Decode(&resp); err != nil {
		t.Fatalf("Failed to decode response: %v", err)
	}

	if resp.Name != "Updated Name" {
		t.Errorf("Expected name 'Updated Name', got '%s'", resp.Name)
	}
}

func TestHandleAdminSaveDownloadApp_MissingPathParam(t *testing.T) {
	db := setupTestDB(t)
	defer db.Close()

	downloads := NewDownloadService(db)
	plans := newTestPlanService(t, "save_missing")

	handler := handleAdminSaveDownloadApp(downloads, plans)

	// Call handler directly without mux (no path param)
	body := `{"name": "Test"}`
	req := httptest.NewRequest(http.MethodPut, "/api/v1/admin/downloads/apps/", strings.NewReader(body))
	req.Header.Set("Content-Type", "application/json")
	w := httptest.NewRecorder()

	handler(w, req)

	if w.Code != http.StatusBadRequest {
		t.Errorf("Expected status %d for missing path param, got %d", http.StatusBadRequest, w.Code)
	}
}

func TestHandleAdminSaveDownloadApp_EmptyPathParam(t *testing.T) {
	db := setupTestDB(t)
	defer db.Close()

	downloads := NewDownloadService(db)
	plans := newTestPlanService(t, "save_empty")

	handler := handleAdminSaveDownloadApp(downloads, plans)

	router := mux.NewRouter()
	router.HandleFunc("/api/v1/admin/downloads/apps/{app_key}", handler).Methods("PUT")

	body := `{"name": "Test"}`
	// Use URL-encoded whitespace
	req := httptest.NewRequest(http.MethodPut, "/api/v1/admin/downloads/apps/%20%20%20", strings.NewReader(body))
	req.Header.Set("Content-Type", "application/json")
	w := httptest.NewRecorder()

	router.ServeHTTP(w, req)

	if w.Code != http.StatusBadRequest {
		t.Errorf("Expected status %d for empty path param, got %d", http.StatusBadRequest, w.Code)
	}
}

// --- handleAdminDeleteDownloadApp Tests ---

func TestHandleAdminDeleteDownloadApp_Success(t *testing.T) {
	db := setupTestDB(t)
	defer db.Close()
	cleanupDownloadApps(t, db)

	// Create app to delete
	insertTestDownloadApp(t, db, "delete_bundle", "to-delete", "To Delete")

	downloads := NewDownloadService(db)
	plans := newTestPlanService(t, "delete_bundle")

	handler := handleAdminDeleteDownloadApp(downloads, plans)

	router := mux.NewRouter()
	router.HandleFunc("/api/v1/admin/downloads/apps/{app_key}", handler).Methods("DELETE")

	req := httptest.NewRequest(http.MethodDelete, "/api/v1/admin/downloads/apps/to-delete", nil)
	w := httptest.NewRecorder()

	router.ServeHTTP(w, req)

	if w.Code != http.StatusOK {
		t.Errorf("Expected status %d, got %d: %s", http.StatusOK, w.Code, w.Body.String())
	}
}

func TestHandleAdminDeleteDownloadApp_NotFound(t *testing.T) {
	db := setupTestDB(t)
	defer db.Close()
	cleanupDownloadApps(t, db)

	downloads := NewDownloadService(db)
	plans := newTestPlanService(t, "delete_notfound")

	handler := handleAdminDeleteDownloadApp(downloads, plans)

	router := mux.NewRouter()
	router.HandleFunc("/api/v1/admin/downloads/apps/{app_key}", handler).Methods("DELETE")

	req := httptest.NewRequest(http.MethodDelete, "/api/v1/admin/downloads/apps/nonexistent", nil)
	w := httptest.NewRecorder()

	router.ServeHTTP(w, req)

	if w.Code != http.StatusNotFound {
		t.Errorf("Expected status %d for not found, got %d", http.StatusNotFound, w.Code)
	}
}

// --- buildDownloadAppFromPayload Validation Tests ---

func TestBuildDownloadApp_MissingAppKey(t *testing.T) {
	payload := downloadAppRequest{
		Name: "Test App",
	}

	_, err := buildDownloadAppFromPayload(payload, "test_bundle", "")
	if err == nil {
		t.Fatal("Expected error for missing app_key")
	}
	if !strings.Contains(err.Error(), "app_key is required") {
		t.Errorf("Expected 'app_key is required' error, got: %v", err)
	}
}

func TestBuildDownloadApp_MissingName(t *testing.T) {
	payload := downloadAppRequest{
		AppKey: "test-app",
	}

	_, err := buildDownloadAppFromPayload(payload, "test_bundle", "")
	if err == nil {
		t.Fatal("Expected error for missing name")
	}
	if !strings.Contains(err.Error(), "name is required") {
		t.Errorf("Expected 'name is required' error, got: %v", err)
	}
}

func TestBuildDownloadApp_InvalidArtifactSource(t *testing.T) {
	payload := downloadAppRequest{
		AppKey: "test-app",
		Name:   "Test App",
		Platforms: []downloadAssetRequest{
			{
				Platform:       "windows",
				ArtifactSource: "invalid",
				ReleaseVersion: "1.0.0",
			},
		},
	}

	_, err := buildDownloadAppFromPayload(payload, "test_bundle", "")
	if err == nil {
		t.Fatal("Expected error for invalid artifact_source")
	}
	if !strings.Contains(err.Error(), "artifact_source must be 'direct' or 'managed'") {
		t.Errorf("Expected artifact_source validation error, got: %v", err)
	}
}

func TestBuildDownloadApp_DirectMissingURL(t *testing.T) {
	payload := downloadAppRequest{
		AppKey: "test-app",
		Name:   "Test App",
		Platforms: []downloadAssetRequest{
			{
				Platform:       "windows",
				ArtifactSource: "direct",
				ArtifactURL:    "", // Missing URL
				ReleaseVersion: "1.0.0",
			},
		},
	}

	_, err := buildDownloadAppFromPayload(payload, "test_bundle", "")
	if err == nil {
		t.Fatal("Expected error for missing artifact_url")
	}
}

func TestBuildDownloadApp_ManagedMissingArtifactID(t *testing.T) {
	payload := downloadAppRequest{
		AppKey: "test-app",
		Name:   "Test App",
		Platforms: []downloadAssetRequest{
			{
				Platform:       "windows",
				ArtifactSource: "managed",
				ArtifactID:     nil, // Missing artifact_id
				ReleaseVersion: "1.0.0",
			},
		},
	}

	_, err := buildDownloadAppFromPayload(payload, "test_bundle", "")
	if err == nil {
		t.Fatal("Expected error for missing artifact_id")
	}
	if !strings.Contains(err.Error(), "artifact_id is required for managed") {
		t.Errorf("Expected artifact_id validation error, got: %v", err)
	}
}

func TestBuildDownloadApp_MissingPlatform(t *testing.T) {
	payload := downloadAppRequest{
		AppKey: "test-app",
		Name:   "Test App",
		Platforms: []downloadAssetRequest{
			{
				Platform:       "", // Missing platform
				ArtifactSource: "direct",
				ArtifactURL:    "https://example.com/app.exe",
				ReleaseVersion: "1.0.0",
			},
		},
	}

	_, err := buildDownloadAppFromPayload(payload, "test_bundle", "")
	if err == nil {
		t.Fatal("Expected error for missing platform")
	}
	if !strings.Contains(err.Error(), "platform is required") {
		t.Errorf("Expected platform validation error, got: %v", err)
	}
}

func TestBuildDownloadApp_MissingReleaseVersion(t *testing.T) {
	payload := downloadAppRequest{
		AppKey: "test-app",
		Name:   "Test App",
		Platforms: []downloadAssetRequest{
			{
				Platform:       "windows",
				ArtifactSource: "direct",
				ArtifactURL:    "https://example.com/app.exe",
				ReleaseVersion: "", // Missing release_version
			},
		},
	}

	_, err := buildDownloadAppFromPayload(payload, "test_bundle", "")
	if err == nil {
		t.Fatal("Expected error for missing release_version")
	}
	if !strings.Contains(err.Error(), "release_version is required") {
		t.Errorf("Expected release_version validation error, got: %v", err)
	}
}

func TestBuildDownloadApp_StorefrontEmptyURL(t *testing.T) {
	payload := downloadAppRequest{
		AppKey: "test-app",
		Name:   "Test App",
		Storefronts: []DownloadStorefront{
			{
				Store: "app_store",
				Label: "App Store",
				URL:   "", // Empty URL
			},
		},
	}

	_, err := buildDownloadAppFromPayload(payload, "test_bundle", "")
	if err == nil {
		t.Fatal("Expected error for empty storefront URL")
	}
	if !strings.Contains(err.Error(), "storefront url is required") {
		t.Errorf("Expected storefront URL validation error, got: %v", err)
	}
}

func TestBuildDownloadApp_Success_Minimal(t *testing.T) {
	payload := downloadAppRequest{
		AppKey: "test-app",
		Name:   "Test App",
	}

	app, err := buildDownloadAppFromPayload(payload, "test_bundle", "")
	if err != nil {
		t.Fatalf("Unexpected error: %v", err)
	}

	if app.AppKey != "test-app" {
		t.Errorf("Expected app_key 'test-app', got '%s'", app.AppKey)
	}
	if app.Name != "Test App" {
		t.Errorf("Expected name 'Test App', got '%s'", app.Name)
	}
	if app.BundleKey != "test_bundle" {
		t.Errorf("Expected bundle_key 'test_bundle', got '%s'", app.BundleKey)
	}
}

func TestBuildDownloadApp_Success_Full(t *testing.T) {
	displayOrder := 5
	artifactID := int64(123)
	requiresEntitlement := true

	payload := downloadAppRequest{
		AppKey:          "full-app",
		Name:            "Full Application",
		Tagline:         "The best app",
		Description:     "A comprehensive description",
		IconURL:         "https://example.com/icon.png",
		ScreenshotURL:   "https://example.com/screenshot.png",
		InstallOverview: "Simple installation process",
		InstallSteps:    []string{"Step 1", "Step 2", ""},
		DisplayOrder:    &displayOrder,
		Storefronts: []DownloadStorefront{
			{Store: "app_store", Label: "App Store", URL: "https://appstore.com/app"},
		},
		Metadata: map[string]interface{}{"version": "1.0"},
		Platforms: []downloadAssetRequest{
			{
				Platform:            "windows",
				ArtifactSource:      "managed",
				ArtifactID:          &artifactID,
				ReleaseVersion:      "1.0.0",
				ReleaseNotes:        "Initial release",
				Checksum:            "abc123",
				RequiresEntitlement: &requiresEntitlement,
				Metadata:            map[string]interface{}{"arch": "x64"},
			},
		},
	}

	app, err := buildDownloadAppFromPayload(payload, "test_bundle", "")
	if err != nil {
		t.Fatalf("Unexpected error: %v", err)
	}

	if app.DisplayOrder != 5 {
		t.Errorf("Expected display_order 5, got %d", app.DisplayOrder)
	}
	if len(app.InstallSteps) != 2 {
		t.Errorf("Expected 2 install steps (empty filtered), got %d", len(app.InstallSteps))
	}
	if len(app.Platforms) != 1 {
		t.Errorf("Expected 1 platform, got %d", len(app.Platforms))
	}
	if app.Platforms[0].RequiresEntitlement != true {
		t.Error("Expected requires_entitlement to be true")
	}
}

func TestBuildDownloadApp_OverrideAppKey(t *testing.T) {
	payload := downloadAppRequest{
		AppKey: "payload-key",
		Name:   "Test App",
	}

	app, err := buildDownloadAppFromPayload(payload, "test_bundle", "override-key")
	if err != nil {
		t.Fatalf("Unexpected error: %v", err)
	}

	// Override key should take precedence
	if app.AppKey != "override-key" {
		t.Errorf("Expected app_key 'override-key', got '%s'", app.AppKey)
	}
}

func TestFilterStrings_RemovesEmpty(t *testing.T) {
	input := []string{"one", "", "two", "   ", "three", ""}
	result := filterStrings(input)

	if len(result) != 3 {
		t.Errorf("Expected 3 items, got %d", len(result))
	}

	expected := []string{"one", "two", "three"}
	for i, v := range expected {
		if result[i] != v {
			t.Errorf("Expected result[%d] to be '%s', got '%s'", i, v, result[i])
		}
	}
}

// Helper functions

func insertTestDownloadApp(t *testing.T, db *sql.DB, bundleKey, appKey, name string) {
	t.Helper()
	_, err := db.Exec(`
		INSERT INTO download_apps (bundle_key, app_key, name, tagline, description, icon_url, screenshot_url, install_overview)
		VALUES ($1, $2, $3, '', '', '', '', '')
	`, bundleKey, appKey, name)
	if err != nil {
		t.Fatalf("Failed to insert test download app: %v", err)
	}
}
