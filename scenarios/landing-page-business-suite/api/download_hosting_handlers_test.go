package main

import (
	"context"
	"database/sql"
	"encoding/json"
	"errors"
	"net/http"
	"net/http/httptest"
	"strconv"
	"strings"
	"testing"

	"github.com/gorilla/mux"
)

// --- handleAdminGetDownloadStorage Tests ---

func TestHandleAdminGetDownloadStorage_Success(t *testing.T) {
	db := setupTestDB(t)
	defer db.Close()
	cleanupDownloadStorageSettings(t, db)

	// Insert test settings
	_, err := db.Exec(`
		INSERT INTO download_storage_settings (bundle_key, provider, bucket, region, signed_url_ttl_seconds)
		VALUES ('test_bundle', 's3', 'test-bucket', 'us-east-1', 900)
	`)
	if err != nil {
		t.Fatalf("Failed to insert settings: %v", err)
	}

	hosting := NewDownloadHostingService(db)
	plans := newTestPlanService(t, "test_bundle")

	handler := handleAdminGetDownloadStorage(hosting, plans)

	req := httptest.NewRequest(http.MethodGet, "/api/v1/admin/downloads/storage", nil)
	w := httptest.NewRecorder()

	handler(w, req)

	if w.Code != http.StatusOK {
		t.Errorf("Expected status %d, got %d: %s", http.StatusOK, w.Code, w.Body.String())
	}

	var resp map[string]interface{}
	if err := json.NewDecoder(w.Body).Decode(&resp); err != nil {
		t.Fatalf("Failed to decode response: %v", err)
	}

	settings, ok := resp["settings"].(map[string]interface{})
	if !ok {
		t.Fatal("Expected 'settings' in response")
	}
	if settings["bucket"] != "test-bucket" {
		t.Errorf("Expected bucket 'test-bucket', got '%v'", settings["bucket"])
	}
}

func TestHandleAdminGetDownloadStorage_NotConfigured(t *testing.T) {
	db := setupTestDB(t)
	defer db.Close()
	cleanupDownloadStorageSettings(t, db)

	hosting := NewDownloadHostingService(db)
	plans := newTestPlanService(t, "unconfigured_bundle")

	handler := handleAdminGetDownloadStorage(hosting, plans)

	req := httptest.NewRequest(http.MethodGet, "/api/v1/admin/downloads/storage", nil)
	w := httptest.NewRecorder()

	handler(w, req)

	if w.Code != http.StatusOK {
		t.Errorf("Expected status %d, got %d: %s", http.StatusOK, w.Code, w.Body.String())
	}

	var resp map[string]interface{}
	if err := json.NewDecoder(w.Body).Decode(&resp); err != nil {
		t.Fatalf("Failed to decode response: %v", err)
	}

	settings, ok := resp["settings"].(map[string]interface{})
	if !ok {
		t.Fatal("Expected 'settings' in response")
	}
	// Not configured should have SettingsRowAvailable = false
	if settings["settings_row_available"] != false {
		t.Error("Expected settings_row_available to be false")
	}
}

// --- handleAdminUpdateDownloadStorage Tests ---

func TestHandleAdminUpdateDownloadStorage_Success(t *testing.T) {
	db := setupTestDB(t)
	defer db.Close()
	cleanupDownloadStorageSettings(t, db)

	hosting := NewDownloadHostingService(db)
	plans := newTestPlanService(t, "update_test")

	handler := handleAdminUpdateDownloadStorage(hosting, plans)

	body := `{"bucket": "new-bucket", "region": "eu-west-1", "signed_url_ttl_seconds": 1800}`
	req := httptest.NewRequest(http.MethodPut, "/api/v1/admin/downloads/storage", strings.NewReader(body))
	req.Header.Set("Content-Type", "application/json")
	w := httptest.NewRecorder()

	handler(w, req)

	if w.Code != http.StatusOK {
		t.Errorf("Expected status %d, got %d: %s", http.StatusOK, w.Code, w.Body.String())
	}

	var resp map[string]interface{}
	if err := json.NewDecoder(w.Body).Decode(&resp); err != nil {
		t.Fatalf("Failed to decode response: %v", err)
	}

	settings, ok := resp["settings"].(map[string]interface{})
	if !ok {
		t.Fatal("Expected 'settings' in response")
	}
	if settings["bucket"] != "new-bucket" {
		t.Errorf("Expected bucket 'new-bucket', got '%v'", settings["bucket"])
	}
}

func TestHandleAdminUpdateDownloadStorage_ValidationError(t *testing.T) {
	db := setupTestDB(t)
	defer db.Close()
	cleanupDownloadStorageSettings(t, db)

	hosting := NewDownloadHostingService(db)
	plans := newTestPlanService(t, "validation_test")

	handler := handleAdminUpdateDownloadStorage(hosting, plans)

	// Invalid endpoint URL
	body := `{"bucket": "test-bucket", "endpoint": "not-a-valid-url"}`
	req := httptest.NewRequest(http.MethodPut, "/api/v1/admin/downloads/storage", strings.NewReader(body))
	req.Header.Set("Content-Type", "application/json")
	w := httptest.NewRecorder()

	handler(w, req)

	if w.Code != http.StatusBadRequest {
		t.Errorf("Expected status %d for validation error, got %d: %s", http.StatusBadRequest, w.Code, w.Body.String())
	}
}

func TestHandleAdminUpdateDownloadStorage_InvalidJSON(t *testing.T) {
	db := setupTestDB(t)
	defer db.Close()

	hosting := NewDownloadHostingService(db)
	plans := newTestPlanService(t, "invalid_json")

	handler := handleAdminUpdateDownloadStorage(hosting, plans)

	req := httptest.NewRequest(http.MethodPut, "/api/v1/admin/downloads/storage", strings.NewReader("{invalid"))
	req.Header.Set("Content-Type", "application/json")
	w := httptest.NewRecorder()

	handler(w, req)

	if w.Code != http.StatusBadRequest {
		t.Errorf("Expected status %d for invalid JSON, got %d", http.StatusBadRequest, w.Code)
	}
}

// --- handleAdminTestDownloadStorage Tests ---

func TestHandleAdminTestDownloadStorage_Success(t *testing.T) {
	db := setupTestDB(t)
	defer db.Close()
	cleanupDownloadStorageSettings(t, db)

	mockStorage := &mockDownloadStorage{testConnectionErr: nil}
	mockProvider := &mockStorageProvider{storage: mockStorage}

	hosting := NewDownloadHostingService(db, mockProvider)
	plans := newTestPlanService(t, "test_connection")

	// Insert settings
	_, err := db.Exec(`
		INSERT INTO download_storage_settings (bundle_key, provider, bucket, region, signed_url_ttl_seconds)
		VALUES ('test_connection', 's3', 'test-bucket', 'us-east-1', 900)
	`)
	if err != nil {
		t.Fatalf("Failed to insert settings: %v", err)
	}

	handler := handleAdminTestDownloadStorage(hosting, plans)

	req := httptest.NewRequest(http.MethodPost, "/api/v1/admin/downloads/storage/test", nil)
	w := httptest.NewRecorder()

	handler(w, req)

	if w.Code != http.StatusOK {
		t.Errorf("Expected status %d, got %d: %s", http.StatusOK, w.Code, w.Body.String())
	}
}

func TestHandleAdminTestDownloadStorage_Failure(t *testing.T) {
	db := setupTestDB(t)
	defer db.Close()
	cleanupDownloadStorageSettings(t, db)

	mockStorage := &mockDownloadStorage{testConnectionErr: errors.New("connection failed")}
	mockProvider := &mockStorageProvider{storage: mockStorage}

	hosting := NewDownloadHostingService(db, mockProvider)
	plans := newTestPlanService(t, "test_fail")

	// Insert settings
	_, err := db.Exec(`
		INSERT INTO download_storage_settings (bundle_key, provider, bucket, region, signed_url_ttl_seconds)
		VALUES ('test_fail', 's3', 'test-bucket', 'us-east-1', 900)
	`)
	if err != nil {
		t.Fatalf("Failed to insert settings: %v", err)
	}

	handler := handleAdminTestDownloadStorage(hosting, plans)

	req := httptest.NewRequest(http.MethodPost, "/api/v1/admin/downloads/storage/test", nil)
	w := httptest.NewRecorder()

	handler(w, req)

	if w.Code != http.StatusBadRequest {
		t.Errorf("Expected status %d for connection failure, got %d", http.StatusBadRequest, w.Code)
	}
}

func TestHandleAdminTestDownloadStorage_NotConfigured(t *testing.T) {
	db := setupTestDB(t)
	defer db.Close()
	cleanupDownloadStorageSettings(t, db)

	hosting := NewDownloadHostingService(db)
	plans := newTestPlanService(t, "not_configured")

	handler := handleAdminTestDownloadStorage(hosting, plans)

	req := httptest.NewRequest(http.MethodPost, "/api/v1/admin/downloads/storage/test", nil)
	w := httptest.NewRecorder()

	handler(w, req)

	if w.Code != http.StatusConflict {
		t.Errorf("Expected status %d for not configured, got %d", http.StatusConflict, w.Code)
	}
}

// --- handleAdminListDownloadArtifacts Tests ---

func TestHandleAdminListDownloadArtifacts_Success(t *testing.T) {
	db := setupTestDB(t)
	defer db.Close()
	cleanupDownloadStorageSettings(t, db)
	cleanupDownloadArtifacts(t, db)

	mockStorage := &mockDownloadStorage{
		headEtag:        "abc123",
		headSize:        1000,
		headContentType: "application/zip",
	}
	mockProvider := &mockStorageProvider{storage: mockStorage}

	hosting := NewDownloadHostingService(db, mockProvider)
	plans := newTestPlanService(t, "list_artifacts")

	// Insert settings
	_, err := db.Exec(`
		INSERT INTO download_storage_settings (bundle_key, provider, bucket, region, signed_url_ttl_seconds)
		VALUES ('list_artifacts', 's3', 'test-bucket', 'us-east-1', 900)
	`)
	if err != nil {
		t.Fatalf("Failed to insert settings: %v", err)
	}

	// Create some artifacts
	for i := 0; i < 3; i++ {
		req := CommitArtifactRequest{
			Bucket:    "test-bucket",
			ObjectKey: "list/" + string(rune('a'+i)) + ".zip",
		}
		_, err := hosting.CommitArtifact(testContext(t), "list_artifacts", req)
		if err != nil {
			t.Fatalf("CommitArtifact failed: %v", err)
		}
	}

	handler := handleAdminListDownloadArtifacts(hosting, plans)

	req := httptest.NewRequest(http.MethodGet, "/api/v1/admin/downloads/artifacts", nil)
	w := httptest.NewRecorder()

	handler(w, req)

	if w.Code != http.StatusOK {
		t.Errorf("Expected status %d, got %d: %s", http.StatusOK, w.Code, w.Body.String())
	}

	var resp ListArtifactsResult
	if err := json.NewDecoder(w.Body).Decode(&resp); err != nil {
		t.Fatalf("Failed to decode response: %v", err)
	}

	if resp.Total != 3 {
		t.Errorf("Expected 3 artifacts, got %d", resp.Total)
	}
}

func TestHandleAdminListDownloadArtifacts_Pagination(t *testing.T) {
	db := setupTestDB(t)
	defer db.Close()
	cleanupDownloadStorageSettings(t, db)
	cleanupDownloadArtifacts(t, db)

	mockStorage := &mockDownloadStorage{
		headEtag:        "abc123",
		headSize:        1000,
		headContentType: "application/zip",
	}
	mockProvider := &mockStorageProvider{storage: mockStorage}

	hosting := NewDownloadHostingService(db, mockProvider)
	plans := newTestPlanService(t, "pagination_test")

	// Insert settings
	_, err := db.Exec(`
		INSERT INTO download_storage_settings (bundle_key, provider, bucket, region, signed_url_ttl_seconds)
		VALUES ('pagination_test', 's3', 'test-bucket', 'us-east-1', 900)
	`)
	if err != nil {
		t.Fatalf("Failed to insert settings: %v", err)
	}

	// Create many artifacts
	for i := 0; i < 15; i++ {
		req := CommitArtifactRequest{
			Bucket:    "test-bucket",
			ObjectKey: "page/" + string(rune('a'+i)) + ".zip",
		}
		_, err := hosting.CommitArtifact(testContext(t), "pagination_test", req)
		if err != nil {
			t.Fatalf("CommitArtifact failed: %v", err)
		}
	}

	handler := handleAdminListDownloadArtifacts(hosting, plans)

	// Test with page and page_size params
	req := httptest.NewRequest(http.MethodGet, "/api/v1/admin/downloads/artifacts?page=2&page_size=5", nil)
	w := httptest.NewRecorder()

	handler(w, req)

	if w.Code != http.StatusOK {
		t.Errorf("Expected status %d, got %d: %s", http.StatusOK, w.Code, w.Body.String())
	}

	var resp ListArtifactsResult
	if err := json.NewDecoder(w.Body).Decode(&resp); err != nil {
		t.Fatalf("Failed to decode response: %v", err)
	}

	if resp.Total != 15 {
		t.Errorf("Expected total 15, got %d", resp.Total)
	}
	if resp.Page != 2 {
		t.Errorf("Expected page 2, got %d", resp.Page)
	}
	if resp.PageSize != 5 {
		t.Errorf("Expected page_size 5, got %d", resp.PageSize)
	}
	if len(resp.Artifacts) != 5 {
		t.Errorf("Expected 5 artifacts on page, got %d", len(resp.Artifacts))
	}
}

// --- handleAdminPresignUploadDownloadArtifact Tests ---

func TestHandleAdminPresignUpload_Success(t *testing.T) {
	db := setupTestDB(t)
	defer db.Close()
	cleanupDownloadStorageSettings(t, db)

	mockStorage := &mockDownloadStorage{
		presignPutURL:     "https://s3.test.com/upload?signed=yes",
		presignPutHeaders: map[string]string{"x-amz-meta-test": "value"},
	}
	mockProvider := &mockStorageProvider{storage: mockStorage}

	hosting := NewDownloadHostingService(db, mockProvider)
	plans := newTestPlanService(t, "presign_upload")

	// Insert settings
	_, err := db.Exec(`
		INSERT INTO download_storage_settings (bundle_key, provider, bucket, region, signed_url_ttl_seconds)
		VALUES ('presign_upload', 's3', 'test-bucket', 'us-east-1', 900)
	`)
	if err != nil {
		t.Fatalf("Failed to insert settings: %v", err)
	}

	handler := handleAdminPresignUploadDownloadArtifact(hosting, plans)

	body := `{"filename": "test-app.zip", "content_type": "application/zip", "platform": "windows"}`
	req := httptest.NewRequest(http.MethodPost, "/api/v1/admin/downloads/artifacts/presign-upload", strings.NewReader(body))
	req.Header.Set("Content-Type", "application/json")
	w := httptest.NewRecorder()

	handler(w, req)

	if w.Code != http.StatusOK {
		t.Errorf("Expected status %d, got %d: %s", http.StatusOK, w.Code, w.Body.String())
	}

	var resp PresignUploadResponse
	if err := json.NewDecoder(w.Body).Decode(&resp); err != nil {
		t.Fatalf("Failed to decode response: %v", err)
	}

	if resp.UploadURL != "https://s3.test.com/upload?signed=yes" {
		t.Errorf("Expected upload URL, got '%s'", resp.UploadURL)
	}
	if resp.Bucket != "test-bucket" {
		t.Errorf("Expected bucket 'test-bucket', got '%s'", resp.Bucket)
	}
}

func TestHandleAdminPresignUpload_NotConfigured(t *testing.T) {
	db := setupTestDB(t)
	defer db.Close()
	cleanupDownloadStorageSettings(t, db)

	hosting := NewDownloadHostingService(db)
	plans := newTestPlanService(t, "not_configured")

	handler := handleAdminPresignUploadDownloadArtifact(hosting, plans)

	body := `{"filename": "test.zip"}`
	req := httptest.NewRequest(http.MethodPost, "/api/v1/admin/downloads/artifacts/presign-upload", strings.NewReader(body))
	req.Header.Set("Content-Type", "application/json")
	w := httptest.NewRecorder()

	handler(w, req)

	if w.Code != http.StatusConflict {
		t.Errorf("Expected status %d for not configured, got %d", http.StatusConflict, w.Code)
	}
}

func TestHandleAdminPresignUpload_ValidationError(t *testing.T) {
	db := setupTestDB(t)
	defer db.Close()
	cleanupDownloadStorageSettings(t, db)

	mockStorage := &mockDownloadStorage{}
	mockProvider := &mockStorageProvider{storage: mockStorage}

	hosting := NewDownloadHostingService(db, mockProvider)
	plans := newTestPlanService(t, "validation_upload")

	// Insert settings
	_, err := db.Exec(`
		INSERT INTO download_storage_settings (bundle_key, provider, bucket, region, signed_url_ttl_seconds)
		VALUES ('validation_upload', 's3', 'test-bucket', 'us-east-1', 900)
	`)
	if err != nil {
		t.Fatalf("Failed to insert settings: %v", err)
	}

	handler := handleAdminPresignUploadDownloadArtifact(hosting, plans)

	// Missing filename
	body := `{"content_type": "application/zip"}`
	req := httptest.NewRequest(http.MethodPost, "/api/v1/admin/downloads/artifacts/presign-upload", strings.NewReader(body))
	req.Header.Set("Content-Type", "application/json")
	w := httptest.NewRecorder()

	handler(w, req)

	if w.Code != http.StatusBadRequest {
		t.Errorf("Expected status %d for missing filename, got %d: %s", http.StatusBadRequest, w.Code, w.Body.String())
	}
}

// --- handleAdminCommitDownloadArtifact Tests ---

func TestHandleAdminCommitArtifact_Success(t *testing.T) {
	db := setupTestDB(t)
	defer db.Close()
	cleanupDownloadStorageSettings(t, db)
	cleanupDownloadArtifacts(t, db)

	mockStorage := &mockDownloadStorage{
		headEtag:        "abc123",
		headSize:        1024000,
		headContentType: "application/zip",
	}
	mockProvider := &mockStorageProvider{storage: mockStorage}

	hosting := NewDownloadHostingService(db, mockProvider)
	plans := newTestPlanService(t, "commit_handler")

	// Insert settings
	_, err := db.Exec(`
		INSERT INTO download_storage_settings (bundle_key, provider, bucket, region, signed_url_ttl_seconds)
		VALUES ('commit_handler', 's3', 'test-bucket', 'us-east-1', 900)
	`)
	if err != nil {
		t.Fatalf("Failed to insert settings: %v", err)
	}

	handler := handleAdminCommitDownloadArtifact(hosting, plans)

	body := `{
		"bucket": "test-bucket",
		"object_key": "artifacts/test/app.zip",
		"original_filename": "app.zip",
		"platform": "windows",
		"release_version": "1.0.0"
	}`
	req := httptest.NewRequest(http.MethodPost, "/api/v1/admin/downloads/artifacts/commit", strings.NewReader(body))
	req.Header.Set("Content-Type", "application/json")
	w := httptest.NewRecorder()

	handler(w, req)

	if w.Code != http.StatusOK {
		t.Errorf("Expected status %d, got %d: %s", http.StatusOK, w.Code, w.Body.String())
	}

	var resp DownloadArtifact
	if err := json.NewDecoder(w.Body).Decode(&resp); err != nil {
		t.Fatalf("Failed to decode response: %v", err)
	}

	if resp.ID == 0 {
		t.Error("Expected artifact ID to be set")
	}
	if resp.ETag != "abc123" {
		t.Errorf("Expected ETag 'abc123', got '%s'", resp.ETag)
	}
}

func TestHandleAdminCommitArtifact_HeadError(t *testing.T) {
	db := setupTestDB(t)
	defer db.Close()
	cleanupDownloadStorageSettings(t, db)

	mockStorage := &mockDownloadStorage{
		headErr: errors.New("object not found"),
	}
	mockProvider := &mockStorageProvider{storage: mockStorage}

	hosting := NewDownloadHostingService(db, mockProvider)
	plans := newTestPlanService(t, "commit_error")

	// Insert settings
	_, err := db.Exec(`
		INSERT INTO download_storage_settings (bundle_key, provider, bucket, region, signed_url_ttl_seconds)
		VALUES ('commit_error', 's3', 'test-bucket', 'us-east-1', 900)
	`)
	if err != nil {
		t.Fatalf("Failed to insert settings: %v", err)
	}

	handler := handleAdminCommitDownloadArtifact(hosting, plans)

	body := `{"bucket": "test-bucket", "object_key": "nonexistent/file.zip"}`
	req := httptest.NewRequest(http.MethodPost, "/api/v1/admin/downloads/artifacts/commit", strings.NewReader(body))
	req.Header.Set("Content-Type", "application/json")
	w := httptest.NewRecorder()

	handler(w, req)

	if w.Code != http.StatusBadRequest {
		t.Errorf("Expected status %d for HeadObject error, got %d", http.StatusBadRequest, w.Code)
	}
}

// --- handleAdminPresignGetDownloadArtifact Tests ---

func TestHandleAdminPresignGet_Success(t *testing.T) {
	db := setupTestDB(t)
	defer db.Close()
	cleanupDownloadStorageSettings(t, db)
	cleanupDownloadArtifacts(t, db)

	mockStorage := &mockDownloadStorage{
		headEtag:        "abc123",
		headSize:        512000,
		headContentType: "application/zip",
		presignGetURL:   "https://s3.test.com/download?signed=yes",
	}
	mockProvider := &mockStorageProvider{storage: mockStorage}

	hosting := NewDownloadHostingService(db, mockProvider)
	plans := newTestPlanService(t, "presign_get")

	// Insert settings
	_, err := db.Exec(`
		INSERT INTO download_storage_settings (bundle_key, provider, bucket, region, signed_url_ttl_seconds)
		VALUES ('presign_get', 's3', 'test-bucket', 'us-east-1', 900)
	`)
	if err != nil {
		t.Fatalf("Failed to insert settings: %v", err)
	}

	// Create artifact first
	commitReq := CommitArtifactRequest{
		Bucket:    "test-bucket",
		ObjectKey: "download/test.zip",
	}
	artifact, err := hosting.CommitArtifact(testContext(t), "presign_get", commitReq)
	if err != nil {
		t.Fatalf("CommitArtifact failed: %v", err)
	}

	handler := handleAdminPresignGetDownloadArtifact(hosting, plans)

	router := mux.NewRouter()
	router.HandleFunc("/api/v1/admin/downloads/artifacts/{artifact_id}/presign", handler).Methods("GET")

	reqPath := "/api/v1/admin/downloads/artifacts/" + itoa(artifact.ID) + "/presign"
	req := httptest.NewRequest(http.MethodGet, reqPath, nil)
	w := httptest.NewRecorder()

	router.ServeHTTP(w, req)

	if w.Code != http.StatusOK {
		t.Errorf("Expected status %d, got %d: %s", http.StatusOK, w.Code, w.Body.String())
	}

	var resp map[string]string
	if err := json.NewDecoder(w.Body).Decode(&resp); err != nil {
		t.Fatalf("Failed to decode response: %v", err)
	}

	if resp["url"] != "https://s3.test.com/download?signed=yes" {
		t.Errorf("Expected signed URL, got '%s'", resp["url"])
	}
}

func TestHandleAdminPresignGet_NotFound(t *testing.T) {
	db := setupTestDB(t)
	defer db.Close()
	cleanupDownloadStorageSettings(t, db)
	cleanupDownloadArtifacts(t, db)

	hosting := NewDownloadHostingService(db)
	plans := newTestPlanService(t, "presign_notfound")

	// Insert settings (needed to pass storage check)
	_, err := db.Exec(`
		INSERT INTO download_storage_settings (bundle_key, provider, bucket, region, signed_url_ttl_seconds)
		VALUES ('presign_notfound', 's3', 'test-bucket', 'us-east-1', 900)
	`)
	if err != nil {
		t.Fatalf("Failed to insert settings: %v", err)
	}

	handler := handleAdminPresignGetDownloadArtifact(hosting, plans)

	router := mux.NewRouter()
	router.HandleFunc("/api/v1/admin/downloads/artifacts/{artifact_id}/presign", handler).Methods("GET")

	req := httptest.NewRequest(http.MethodGet, "/api/v1/admin/downloads/artifacts/99999/presign", nil)
	w := httptest.NewRecorder()

	router.ServeHTTP(w, req)

	if w.Code != http.StatusNotFound {
		t.Errorf("Expected status %d for not found, got %d", http.StatusNotFound, w.Code)
	}
}

func TestHandleAdminPresignGet_InvalidID(t *testing.T) {
	db := setupTestDB(t)
	defer db.Close()

	hosting := NewDownloadHostingService(db)
	plans := newTestPlanService(t, "presign_invalid")

	handler := handleAdminPresignGetDownloadArtifact(hosting, plans)

	router := mux.NewRouter()
	router.HandleFunc("/api/v1/admin/downloads/artifacts/{artifact_id}/presign", handler).Methods("GET")

	req := httptest.NewRequest(http.MethodGet, "/api/v1/admin/downloads/artifacts/invalid/presign", nil)
	w := httptest.NewRecorder()

	router.ServeHTTP(w, req)

	if w.Code != http.StatusBadRequest {
		t.Errorf("Expected status %d for invalid ID, got %d", http.StatusBadRequest, w.Code)
	}
}

// --- handleAdminApplyDownloadArtifact Tests ---

func TestHandleAdminApplyArtifact_Success(t *testing.T) {
	db := setupTestDB(t)
	defer db.Close()
	cleanupDownloadStorageSettings(t, db)
	cleanupDownloadArtifacts(t, db)
	cleanupDownloadAssets(t, db)
	cleanupDownloadApps(t, db)

	mockStorage := &mockDownloadStorage{
		headEtag:        "abc123",
		headSize:        512000,
		headContentType: "application/zip",
	}
	mockProvider := &mockStorageProvider{storage: mockStorage}

	hosting := NewDownloadHostingService(db, mockProvider)
	downloads := NewDownloadService(db)
	plans := newTestPlanService(t, "apply_test")

	// Insert settings
	_, err := db.Exec(`
		INSERT INTO download_storage_settings (bundle_key, provider, bucket, region, signed_url_ttl_seconds)
		VALUES ('apply_test', 's3', 'test-bucket', 'us-east-1', 900)
	`)
	if err != nil {
		t.Fatalf("Failed to insert settings: %v", err)
	}

	// Create download_apps entry (required for FK constraint)
	_, err = db.Exec(`
		INSERT INTO download_apps (bundle_key, app_key, name)
		VALUES ('apply_test', 'my-app', 'My Test App')
	`)
	if err != nil {
		t.Fatalf("Failed to insert download_apps: %v", err)
	}

	// Create artifact first
	commitReq := CommitArtifactRequest{
		Bucket:         "test-bucket",
		ObjectKey:      "apply/test.zip",
		ReleaseVersion: "1.0.0",
	}
	artifact, err := hosting.CommitArtifact(testContext(t), "apply_test", commitReq)
	if err != nil {
		t.Fatalf("CommitArtifact failed: %v", err)
	}

	handler := handleAdminApplyDownloadArtifact(downloads, hosting, plans)

	body := `{
		"app_key": "my-app",
		"platform": "windows",
		"artifact_id": ` + itoa(artifact.ID) + `,
		"release_version": "1.0.0"
	}`
	req := httptest.NewRequest(http.MethodPost, "/api/v1/admin/downloads/artifacts/apply", strings.NewReader(body))
	req.Header.Set("Content-Type", "application/json")
	w := httptest.NewRecorder()

	handler(w, req)

	if w.Code != http.StatusOK {
		t.Errorf("Expected status %d, got %d: %s", http.StatusOK, w.Code, w.Body.String())
	}

	var resp DownloadAsset
	if err := json.NewDecoder(w.Body).Decode(&resp); err != nil {
		t.Fatalf("Failed to decode response: %v", err)
	}

	if resp.AppKey != "my-app" {
		t.Errorf("Expected app_key 'my-app', got '%s'", resp.AppKey)
	}
	if resp.Platform != "windows" {
		t.Errorf("Expected platform 'windows', got '%s'", resp.Platform)
	}
}

func TestHandleAdminApplyArtifact_MissingFields(t *testing.T) {
	db := setupTestDB(t)
	defer db.Close()

	hosting := NewDownloadHostingService(db)
	downloads := NewDownloadService(db)
	plans := newTestPlanService(t, "apply_missing")

	handler := handleAdminApplyDownloadArtifact(downloads, hosting, plans)

	// Missing app_key
	body := `{"platform": "windows", "artifact_id": 1}`
	req := httptest.NewRequest(http.MethodPost, "/api/v1/admin/downloads/artifacts/apply", strings.NewReader(body))
	req.Header.Set("Content-Type", "application/json")
	w := httptest.NewRecorder()

	handler(w, req)

	if w.Code != http.StatusBadRequest {
		t.Errorf("Expected status %d for missing app_key, got %d", http.StatusBadRequest, w.Code)
	}
}

func TestHandleAdminApplyArtifact_ArtifactNotFound(t *testing.T) {
	db := setupTestDB(t)
	defer db.Close()
	cleanupDownloadStorageSettings(t, db)
	cleanupDownloadArtifacts(t, db)

	hosting := NewDownloadHostingService(db)
	downloads := NewDownloadService(db)
	plans := newTestPlanService(t, "apply_notfound")

	// Insert settings (needed to pass storage check)
	_, err := db.Exec(`
		INSERT INTO download_storage_settings (bundle_key, provider, bucket, region, signed_url_ttl_seconds)
		VALUES ('apply_notfound', 's3', 'test-bucket', 'us-east-1', 900)
	`)
	if err != nil {
		t.Fatalf("Failed to insert settings: %v", err)
	}

	handler := handleAdminApplyDownloadArtifact(downloads, hosting, plans)

	body := `{
		"app_key": "my-app",
		"platform": "windows",
		"artifact_id": 99999,
		"release_version": "1.0.0"
	}`
	req := httptest.NewRequest(http.MethodPost, "/api/v1/admin/downloads/artifacts/apply", strings.NewReader(body))
	req.Header.Set("Content-Type", "application/json")
	w := httptest.NewRecorder()

	handler(w, req)

	if w.Code != http.StatusNotFound {
		t.Errorf("Expected status %d for artifact not found, got %d", http.StatusNotFound, w.Code)
	}
}

// Helper functions

func itoa(i int64) string {
	return strconv.FormatInt(i, 10)
}

func testContext(t *testing.T) context.Context {
	t.Helper()
	return context.Background()
}

// newTestPlanService creates a PlanService with a specific bundle key for testing
func newTestPlanService(t *testing.T, bundleKey string) *PlanService {
	t.Helper()
	t.Setenv("BUNDLE_KEY", bundleKey)
	return NewPlanService(nil) // db is not used for BundleKey()
}

// cleanupDownloadAssets removes all entries from download_assets table
func cleanupDownloadAssets(t *testing.T, db *sql.DB) {
	t.Helper()
	if _, err := db.Exec("DELETE FROM download_assets"); err != nil {
		t.Fatalf("Failed to cleanup download_assets table: %v", err)
	}
}

// cleanupDownloadApps removes all entries from download_apps table
func cleanupDownloadApps(t *testing.T, db *sql.DB) {
	t.Helper()
	// Must delete assets first due to FK
	if _, err := db.Exec("DELETE FROM download_assets"); err != nil {
		t.Fatalf("Failed to cleanup download_assets table: %v", err)
	}
	if _, err := db.Exec("DELETE FROM download_apps"); err != nil {
		t.Fatalf("Failed to cleanup download_apps table: %v", err)
	}
}
