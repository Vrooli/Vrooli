package main

import (
	"bytes"
	"database/sql"
	"encoding/json"
	"fmt"
	"mime/multipart"
	"net/http"
	"net/http/httptest"
	"net/textproto"
	"strconv"
	"testing"

	"github.com/gorilla/mux"
)

// setupTestAssetsService creates an AssetsService with a temp upload directory
// to prevent tests from polluting the real uploads folder.
func setupTestAssetsService(t *testing.T, db *sql.DB) *AssetsService {
	t.Helper()
	tmpDir := t.TempDir()
	t.Setenv("UPLOAD_DIR", tmpDir)
	return NewAssetsService(db)
}

// --- handleAssetUpload Tests ---

func TestHandleAssetUpload_Success(t *testing.T) {
	db := setupTestDB(t)
	defer db.Close()
	cleanupAssets(t, db)

	assetsService := setupTestAssetsService(t, db)
	handler := handleAssetUpload(assetsService)

	body, contentType := createMultipartRequest(t, "file", "test.png", createTestPNG(), "image/png", nil)
	req := httptest.NewRequest(http.MethodPost, "/api/v1/admin/assets/upload", body)
	req.Header.Set("Content-Type", contentType)
	w := httptest.NewRecorder()

	handler(w, req)

	if w.Code != http.StatusCreated {
		t.Errorf("Expected status %d, got %d: %s", http.StatusCreated, w.Code, w.Body.String())
	}

	var resp Asset
	if err := json.NewDecoder(w.Body).Decode(&resp); err != nil {
		t.Fatalf("Failed to decode response: %v", err)
	}

	if resp.ID == 0 {
		t.Error("Expected asset ID to be set")
	}
	if resp.OriginalFilename != "test.png" {
		t.Errorf("Expected original_filename 'test.png', got '%s'", resp.OriginalFilename)
	}
}

func TestHandleAssetUpload_InvalidFileType(t *testing.T) {
	db := setupTestDB(t)
	defer db.Close()

	assetsService := setupTestAssetsService(t, db)
	handler := handleAssetUpload(assetsService)

	// Upload a text file (not allowed)
	body, contentType := createMultipartRequest(t, "file", "test.txt", []byte("hello world"), "text/plain", nil)
	req := httptest.NewRequest(http.MethodPost, "/api/v1/admin/assets/upload", body)
	req.Header.Set("Content-Type", contentType)
	w := httptest.NewRecorder()

	handler(w, req)

	if w.Code != http.StatusBadRequest {
		t.Errorf("Expected status %d for invalid file type, got %d", http.StatusBadRequest, w.Code)
	}
}

func TestHandleAssetUpload_NoFile(t *testing.T) {
	db := setupTestDB(t)
	defer db.Close()

	assetsService := setupTestAssetsService(t, db)
	handler := handleAssetUpload(assetsService)

	// Create multipart request without file
	body := &bytes.Buffer{}
	writer := multipart.NewWriter(body)
	writer.WriteField("category", "general")
	writer.Close()

	req := httptest.NewRequest(http.MethodPost, "/api/v1/admin/assets/upload", body)
	req.Header.Set("Content-Type", writer.FormDataContentType())
	w := httptest.NewRecorder()

	handler(w, req)

	if w.Code != http.StatusBadRequest {
		t.Errorf("Expected status %d for missing file, got %d", http.StatusBadRequest, w.Code)
	}
}

func TestHandleAssetUpload_InvalidMultipart(t *testing.T) {
	db := setupTestDB(t)
	defer db.Close()

	assetsService := setupTestAssetsService(t, db)
	handler := handleAssetUpload(assetsService)

	// Send non-multipart request
	req := httptest.NewRequest(http.MethodPost, "/api/v1/admin/assets/upload", bytes.NewReader([]byte("invalid")))
	req.Header.Set("Content-Type", "application/json")
	w := httptest.NewRecorder()

	handler(w, req)

	if w.Code != http.StatusBadRequest {
		t.Errorf("Expected status %d for invalid multipart, got %d", http.StatusBadRequest, w.Code)
	}
}

func TestHandleAssetUpload_WithOptionalFields(t *testing.T) {
	db := setupTestDB(t)
	defer db.Close()
	cleanupAssets(t, db)

	assetsService := setupTestAssetsService(t, db)
	handler := handleAssetUpload(assetsService)

	extraFields := map[string]string{
		"category":    "logo",
		"alt_text":    "Company Logo",
		"uploaded_by": "test@example.com",
	}
	body, contentType := createMultipartRequest(t, "file", "logo.png", createTestPNG(), "image/png", extraFields)
	req := httptest.NewRequest(http.MethodPost, "/api/v1/admin/assets/upload", body)
	req.Header.Set("Content-Type", contentType)
	w := httptest.NewRecorder()

	handler(w, req)

	if w.Code != http.StatusCreated {
		t.Errorf("Expected status %d, got %d: %s", http.StatusCreated, w.Code, w.Body.String())
	}

	var resp Asset
	if err := json.NewDecoder(w.Body).Decode(&resp); err != nil {
		t.Fatalf("Failed to decode response: %v", err)
	}

	if resp.Category != "logo" {
		t.Errorf("Expected category 'logo', got '%s'", resp.Category)
	}
	if resp.AltText == nil || *resp.AltText != "Company Logo" {
		t.Errorf("Expected alt_text 'Company Logo', got %v", resp.AltText)
	}
	if resp.UploadedBy == nil || *resp.UploadedBy != "test@example.com" {
		t.Errorf("Expected uploaded_by 'test@example.com', got %v", resp.UploadedBy)
	}
}

func TestHandleAssetUpload_DefaultCategory(t *testing.T) {
	db := setupTestDB(t)
	defer db.Close()
	cleanupAssets(t, db)

	assetsService := setupTestAssetsService(t, db)
	handler := handleAssetUpload(assetsService)

	// Upload without specifying category
	body, contentType := createMultipartRequest(t, "file", "test.png", createTestPNG(), "image/png", nil)
	req := httptest.NewRequest(http.MethodPost, "/api/v1/admin/assets/upload", body)
	req.Header.Set("Content-Type", contentType)
	w := httptest.NewRecorder()

	handler(w, req)

	if w.Code != http.StatusCreated {
		t.Errorf("Expected status %d, got %d: %s", http.StatusCreated, w.Code, w.Body.String())
	}

	var resp Asset
	if err := json.NewDecoder(w.Body).Decode(&resp); err != nil {
		t.Fatalf("Failed to decode response: %v", err)
	}

	if resp.Category != "general" {
		t.Errorf("Expected default category 'general', got '%s'", resp.Category)
	}
}

// --- handleAssetsList Tests ---

func TestHandleAssetsList_Success(t *testing.T) {
	db := setupTestDB(t)
	defer db.Close()
	cleanupAssets(t, db)

	assetsService := setupTestAssetsService(t, db)

	// Create test assets
	for i := 0; i < 3; i++ {
		insertTestAsset(t, db, "test"+strconv.Itoa(i)+".png", "general")
	}

	handler := handleAssetsList(assetsService)

	req := httptest.NewRequest(http.MethodGet, "/api/v1/admin/assets", nil)
	w := httptest.NewRecorder()

	handler(w, req)

	if w.Code != http.StatusOK {
		t.Errorf("Expected status %d, got %d: %s", http.StatusOK, w.Code, w.Body.String())
	}

	var resp map[string]interface{}
	if err := json.NewDecoder(w.Body).Decode(&resp); err != nil {
		t.Fatalf("Failed to decode response: %v", err)
	}

	assets, ok := resp["assets"].([]interface{})
	if !ok {
		t.Fatal("Expected 'assets' array in response")
	}
	if len(assets) != 3 {
		t.Errorf("Expected 3 assets, got %d", len(assets))
	}
}

func TestHandleAssetsList_WithCategory(t *testing.T) {
	db := setupTestDB(t)
	defer db.Close()
	cleanupAssets(t, db)

	assetsService := setupTestAssetsService(t, db)

	// Create assets in different categories
	insertTestAsset(t, db, "logo1.png", "logo")
	insertTestAsset(t, db, "logo2.png", "logo")
	insertTestAsset(t, db, "general1.png", "general")

	handler := handleAssetsList(assetsService)

	req := httptest.NewRequest(http.MethodGet, "/api/v1/admin/assets?category=logo", nil)
	w := httptest.NewRecorder()

	handler(w, req)

	if w.Code != http.StatusOK {
		t.Errorf("Expected status %d, got %d: %s", http.StatusOK, w.Code, w.Body.String())
	}

	var resp map[string]interface{}
	if err := json.NewDecoder(w.Body).Decode(&resp); err != nil {
		t.Fatalf("Failed to decode response: %v", err)
	}

	assets := resp["assets"].([]interface{})
	if len(assets) != 2 {
		t.Errorf("Expected 2 logo assets, got %d", len(assets))
	}
}

func TestHandleAssetsList_Empty(t *testing.T) {
	db := setupTestDB(t)
	defer db.Close()
	cleanupAssets(t, db)

	assetsService := setupTestAssetsService(t, db)
	handler := handleAssetsList(assetsService)

	req := httptest.NewRequest(http.MethodGet, "/api/v1/admin/assets", nil)
	w := httptest.NewRecorder()

	handler(w, req)

	if w.Code != http.StatusOK {
		t.Errorf("Expected status %d, got %d", http.StatusOK, w.Code)
	}

	var resp map[string]interface{}
	if err := json.NewDecoder(w.Body).Decode(&resp); err != nil {
		t.Fatalf("Failed to decode response: %v", err)
	}

	assets, ok := resp["assets"].([]interface{})
	if !ok {
		t.Fatal("Expected 'assets' array in response")
	}
	if len(assets) != 0 {
		t.Errorf("Expected empty assets array, got %d assets", len(assets))
	}
}

// --- handleAssetGet Tests ---

func TestHandleAssetGet_Success(t *testing.T) {
	db := setupTestDB(t)
	defer db.Close()
	cleanupAssets(t, db)

	assetsService := setupTestAssetsService(t, db)

	// Create test asset
	assetID := insertTestAsset(t, db, "test.png", "general")

	handler := handleAssetGet(assetsService)

	router := mux.NewRouter()
	router.HandleFunc("/api/v1/admin/assets/{id}", handler).Methods("GET")

	req := httptest.NewRequest(http.MethodGet, "/api/v1/admin/assets/"+strconv.FormatInt(assetID, 10), nil)
	w := httptest.NewRecorder()

	router.ServeHTTP(w, req)

	if w.Code != http.StatusOK {
		t.Errorf("Expected status %d, got %d: %s", http.StatusOK, w.Code, w.Body.String())
	}

	var resp Asset
	if err := json.NewDecoder(w.Body).Decode(&resp); err != nil {
		t.Fatalf("Failed to decode response: %v", err)
	}

	if resp.ID != int(assetID) {
		t.Errorf("Expected asset ID %d, got %d", assetID, resp.ID)
	}
}

func TestHandleAssetGet_NotFound(t *testing.T) {
	db := setupTestDB(t)
	defer db.Close()
	cleanupAssets(t, db)

	assetsService := setupTestAssetsService(t, db)
	handler := handleAssetGet(assetsService)

	router := mux.NewRouter()
	router.HandleFunc("/api/v1/admin/assets/{id}", handler).Methods("GET")

	req := httptest.NewRequest(http.MethodGet, "/api/v1/admin/assets/99999", nil)
	w := httptest.NewRecorder()

	router.ServeHTTP(w, req)

	if w.Code != http.StatusNotFound {
		t.Errorf("Expected status %d for not found, got %d", http.StatusNotFound, w.Code)
	}
}

func TestHandleAssetGet_InvalidID(t *testing.T) {
	db := setupTestDB(t)
	defer db.Close()

	assetsService := setupTestAssetsService(t, db)
	handler := handleAssetGet(assetsService)

	router := mux.NewRouter()
	router.HandleFunc("/api/v1/admin/assets/{id}", handler).Methods("GET")

	req := httptest.NewRequest(http.MethodGet, "/api/v1/admin/assets/invalid", nil)
	w := httptest.NewRecorder()

	router.ServeHTTP(w, req)

	if w.Code != http.StatusBadRequest {
		t.Errorf("Expected status %d for invalid ID, got %d", http.StatusBadRequest, w.Code)
	}
}

// --- handleAssetDelete Tests ---

func TestHandleAssetDelete_Success(t *testing.T) {
	db := setupTestDB(t)
	defer db.Close()
	cleanupAssets(t, db)

	assetsService := setupTestAssetsService(t, db)

	// Create test asset
	assetID := insertTestAsset(t, db, "to-delete.png", "general")

	handler := handleAssetDelete(assetsService)

	router := mux.NewRouter()
	router.HandleFunc("/api/v1/admin/assets/{id}", handler).Methods("DELETE")

	req := httptest.NewRequest(http.MethodDelete, "/api/v1/admin/assets/"+strconv.FormatInt(assetID, 10), nil)
	w := httptest.NewRecorder()

	router.ServeHTTP(w, req)

	if w.Code != http.StatusNoContent {
		t.Errorf("Expected status %d, got %d: %s", http.StatusNoContent, w.Code, w.Body.String())
	}

	// Verify asset is deleted
	var count int
	err := db.QueryRow("SELECT COUNT(*) FROM assets WHERE id = $1", assetID).Scan(&count)
	if err != nil {
		t.Fatalf("Failed to check asset: %v", err)
	}
	if count != 0 {
		t.Error("Expected asset to be deleted")
	}
}

func TestHandleAssetDelete_NotFound(t *testing.T) {
	db := setupTestDB(t)
	defer db.Close()
	cleanupAssets(t, db)

	assetsService := setupTestAssetsService(t, db)
	handler := handleAssetDelete(assetsService)

	router := mux.NewRouter()
	router.HandleFunc("/api/v1/admin/assets/{id}", handler).Methods("DELETE")

	req := httptest.NewRequest(http.MethodDelete, "/api/v1/admin/assets/99999", nil)
	w := httptest.NewRecorder()

	router.ServeHTTP(w, req)

	if w.Code != http.StatusNotFound {
		t.Errorf("Expected status %d for not found, got %d", http.StatusNotFound, w.Code)
	}
}

// Helper functions

func createMultipartRequest(t *testing.T, fieldName, filename string, content []byte, contentType string, extraFields map[string]string) (*bytes.Buffer, string) {
	t.Helper()

	body := &bytes.Buffer{}
	writer := multipart.NewWriter(body)

	// Add file with Content-Type header
	h := make(textproto.MIMEHeader)
	h.Set("Content-Disposition", fmt.Sprintf(`form-data; name="%s"; filename="%s"`, fieldName, filename))
	h.Set("Content-Type", contentType)

	part, err := writer.CreatePart(h)
	if err != nil {
		t.Fatalf("Failed to create form file: %v", err)
	}
	if _, err := part.Write(content); err != nil {
		t.Fatalf("Failed to write file content: %v", err)
	}

	// Add extra fields
	for key, value := range extraFields {
		if err := writer.WriteField(key, value); err != nil {
			t.Fatalf("Failed to write field %s: %v", key, err)
		}
	}

	if err := writer.Close(); err != nil {
		t.Fatalf("Failed to close multipart writer: %v", err)
	}

	return body, writer.FormDataContentType()
}

// createTestPNG creates a minimal valid PNG image for testing
func createTestPNG() []byte {
	// Minimal 1x1 transparent PNG
	return []byte{
		0x89, 0x50, 0x4E, 0x47, 0x0D, 0x0A, 0x1A, 0x0A, 0x00, 0x00, 0x00, 0x0D,
		0x49, 0x48, 0x44, 0x52, 0x00, 0x00, 0x00, 0x01, 0x00, 0x00, 0x00, 0x01,
		0x08, 0x06, 0x00, 0x00, 0x00, 0x1F, 0x15, 0xC4, 0x89, 0x00, 0x00, 0x00,
		0x0A, 0x49, 0x44, 0x41, 0x54, 0x78, 0x9C, 0x63, 0x00, 0x01, 0x00, 0x00,
		0x05, 0x00, 0x01, 0x0D, 0x0A, 0x2D, 0xB4, 0x00, 0x00, 0x00, 0x00, 0x49,
		0x45, 0x4E, 0x44, 0xAE, 0x42, 0x60, 0x82,
	}
}

func insertTestAsset(t *testing.T, db *sql.DB, filename, category string) int64 {
	t.Helper()

	var id int64
	query := `
		INSERT INTO assets (filename, original_filename, mime_type, size_bytes, storage_path, category)
		VALUES ($1, $2, 'image/png', 1000, $3, $4)
		RETURNING id
	`
	storagePath := "test/" + filename
	err := db.QueryRow(query, filename, filename, storagePath, category).Scan(&id)
	if err != nil {
		t.Fatalf("Failed to insert test asset: %v", err)
	}
	return id
}

func cleanupAssets(t *testing.T, db *sql.DB) {
	t.Helper()
	if _, err := db.Exec("DELETE FROM assets"); err != nil {
		t.Fatalf("Failed to cleanup assets table: %v", err)
	}
}
