package handlers_test

import (
	"bytes"
	"encoding/json"
	"mime/multipart"
	"net/http"

	"net/http/httptest"
	"net/textproto"
	"os"
	"path/filepath"
	"testing"

	"brand-manager/config"
	"brand-manager/domain"
)

// [REQ:BM-REQ-API-ASSETS] [REQ:BM-REQ-STORE-ASSETS]

func TestUploadAsset(t *testing.T) {
	cfg := config.Default()
	cfg.AssetBasePath = t.TempDir()

	_, router, brandRepo, _, _, _ := setupMockServerWithConfigAndRepos(t, cfg)

	// Seed a brand for the upload
	brandRepo.Seed(&domain.Brand{ID: "brand-1", Name: "Test Brand"})

	body := &bytes.Buffer{}
	writer := multipart.NewWriter(body)
	mh := make(textproto.MIMEHeader)
	mh.Set("Content-Disposition", `form-data; name="file"; filename="logo.png"`)
	mh.Set("Content-Type", "image/png")
	part, err := writer.CreatePart(mh)
	if err != nil {
		t.Fatal(err)
	}
	part.Write([]byte("fake-png-data"))
	writer.Close()

	req := httptest.NewRequest("POST", "/api/v1/brands/brand-1/assets", body)
	req.Header.Set("Content-Type", writer.FormDataContentType())
	w := httptest.NewRecorder()
	router.ServeHTTP(w, req)

	if w.Code != http.StatusCreated {
		t.Fatalf("status = %d, want %d; body: %s", w.Code, http.StatusCreated, w.Body.String())
	}

	var asset domain.Asset
	json.NewDecoder(w.Body).Decode(&asset)

	if asset.ID == "" {
		t.Error("expected non-empty asset ID")
	}
	if asset.BrandID != "brand-1" {
		t.Errorf("brand_id = %q, want %q", asset.BrandID, "brand-1")
	}
	if asset.Filename != "logo.png" {
		t.Errorf("filename = %q, want %q", asset.Filename, "logo.png")
	}
	if asset.Size != 13 {
		t.Errorf("size = %d, want 13", asset.Size)
	}

	// Verify file was written to disk
	if _, err := os.Stat(asset.FilePath); os.IsNotExist(err) {
		t.Errorf("asset file not found at %q", asset.FilePath)
	}
}

func TestUploadAsset_BrandNotFound(t *testing.T) {
	cfg := config.Default()
	cfg.AssetBasePath = t.TempDir()
	_, router, _, _, _, _ := setupMockServerWithConfigAndRepos(t, cfg)

	body := &bytes.Buffer{}
	writer := multipart.NewWriter(body)
	writer.CreateFormFile("file", "logo.png")
	writer.Close()

	req := httptest.NewRequest("POST", "/api/v1/brands/nonexistent/assets", body)
	req.Header.Set("Content-Type", writer.FormDataContentType())
	w := httptest.NewRecorder()
	router.ServeHTTP(w, req)

	if w.Code != http.StatusNotFound {
		t.Errorf("status = %d, want %d", w.Code, http.StatusNotFound)
	}
}

func TestUploadAsset_UnsupportedMimeType(t *testing.T) {
	cfg := config.Default()
	cfg.AssetBasePath = t.TempDir()
	_, router, brandRepo, _, _, _ := setupMockServerWithConfigAndRepos(t, cfg)
	brandRepo.Seed(&domain.Brand{ID: "brand-1", Name: "Test"})

	body := &bytes.Buffer{}
	writer := multipart.NewWriter(body)
	mh := make(textproto.MIMEHeader)
	mh.Set("Content-Disposition", `form-data; name="file"; filename="bad.txt"`)
	mh.Set("Content-Type", "text/plain")
	part, _ := writer.CreatePart(mh)
	part.Write([]byte("text"))
	writer.Close()

	req := httptest.NewRequest("POST", "/api/v1/brands/brand-1/assets", body)
	req.Header.Set("Content-Type", writer.FormDataContentType())
	w := httptest.NewRecorder()
	router.ServeHTTP(w, req)

	if w.Code != http.StatusBadRequest {
		t.Errorf("status = %d, want %d; body: %s", w.Code, http.StatusBadRequest, w.Body.String())
	}
}

func TestGetAsset(t *testing.T) {
	cfg := config.Default()
	_, router, _, _, _, assetRepo := setupMockServerWithConfigAndRepos(t, cfg)

	assetRepo.Seed(&domain.Asset{
		ID:       "asset-1",
		BrandID:  "brand-1",
		Filename: "icon.svg",
		MimeType: "image/svg+xml",
		Size:     42,
	})

	req := httptest.NewRequest("GET", "/api/v1/assets/asset-1", nil)
	w := httptest.NewRecorder()
	router.ServeHTTP(w, req)

	if w.Code != http.StatusOK {
		t.Fatalf("status = %d, want %d", w.Code, http.StatusOK)
	}

	var asset domain.Asset
	json.NewDecoder(w.Body).Decode(&asset)
	if asset.Filename != "icon.svg" {
		t.Errorf("filename = %q, want %q", asset.Filename, "icon.svg")
	}
}

func TestGetAsset_NotFound(t *testing.T) {
	cfg := config.Default()
	_, router, _, _, _, _ := setupMockServerWithConfigAndRepos(t, cfg)

	req := httptest.NewRequest("GET", "/api/v1/assets/nonexistent", nil)
	w := httptest.NewRecorder()
	router.ServeHTTP(w, req)

	if w.Code != http.StatusNotFound {
		t.Errorf("status = %d, want %d", w.Code, http.StatusNotFound)
	}
}

func TestListAssets(t *testing.T) {
	cfg := config.Default()
	_, router, _, _, _, assetRepo := setupMockServerWithConfigAndRepos(t, cfg)

	assetRepo.Seed(&domain.Asset{ID: "a1", BrandID: "brand-1", Filename: "logo.png"})
	assetRepo.Seed(&domain.Asset{ID: "a2", BrandID: "brand-1", Filename: "icon.svg"})
	assetRepo.Seed(&domain.Asset{ID: "a3", BrandID: "brand-2", Filename: "other.png"})

	req := httptest.NewRequest("GET", "/api/v1/brands/brand-1/assets", nil)
	w := httptest.NewRecorder()
	router.ServeHTTP(w, req)

	if w.Code != http.StatusOK {
		t.Fatalf("status = %d, want %d", w.Code, http.StatusOK)
	}

	var assets []domain.Asset
	json.NewDecoder(w.Body).Decode(&assets)
	if len(assets) != 2 {
		t.Errorf("got %d assets, want 2", len(assets))
	}
}

func TestListAssets_EmptyList(t *testing.T) {
	cfg := config.Default()
	_, router, _, _, _, _ := setupMockServerWithConfigAndRepos(t, cfg)

	req := httptest.NewRequest("GET", "/api/v1/brands/brand-1/assets", nil)
	w := httptest.NewRecorder()
	router.ServeHTTP(w, req)

	if w.Code != http.StatusOK {
		t.Fatalf("status = %d, want %d", w.Code, http.StatusOK)
	}

	var assets []domain.Asset
	json.NewDecoder(w.Body).Decode(&assets)
	if len(assets) != 0 {
		t.Errorf("got %d assets, want 0", len(assets))
	}
}

func TestDeleteAsset(t *testing.T) {
	cfg := config.Default()
	cfg.AssetBasePath = t.TempDir()
	_, router, _, _, _, assetRepo := setupMockServerWithConfigAndRepos(t, cfg)

	// Create a file on disk so delete can remove it
	filePath := filepath.Join(cfg.AssetBasePath, "brand-1", "logo.png")
	os.MkdirAll(filepath.Dir(filePath), 0o755)
	os.WriteFile(filePath, []byte("data"), 0o644)

	assetRepo.Seed(&domain.Asset{
		ID:       "asset-1",
		BrandID:  "brand-1",
		Filename: "logo.png",
		FilePath: filePath,
	})

	req := httptest.NewRequest("DELETE", "/api/v1/assets/asset-1", nil)
	w := httptest.NewRecorder()
	router.ServeHTTP(w, req)

	if w.Code != http.StatusNoContent {
		t.Errorf("status = %d, want %d", w.Code, http.StatusNoContent)
	}

	// Verify file removed from disk
	if _, err := os.Stat(filePath); !os.IsNotExist(err) {
		t.Error("expected asset file to be deleted from disk")
	}
}

func TestDeleteAsset_Idempotent(t *testing.T) {
	cfg := config.Default()
	_, router, _, _, _, _ := setupMockServerWithConfigAndRepos(t, cfg)

	// Delete nonexistent asset — should return 204 (idempotent)
	req := httptest.NewRequest("DELETE", "/api/v1/assets/nonexistent", nil)
	w := httptest.NewRecorder()
	router.ServeHTTP(w, req)

	if w.Code != http.StatusNoContent {
		t.Errorf("status = %d, want %d (idempotent delete)", w.Code, http.StatusNoContent)
	}
}

func TestServeAssetFile(t *testing.T) {
	cfg := config.Default()
	cfg.AssetBasePath = t.TempDir()
	_, router, _, _, _, assetRepo := setupMockServerWithConfigAndRepos(t, cfg)

	// Write a test file
	filePath := filepath.Join(cfg.AssetBasePath, "brand-1", "logo.png")
	os.MkdirAll(filepath.Dir(filePath), 0o755)
	os.WriteFile(filePath, []byte("PNG-DATA"), 0o644)

	assetRepo.Seed(&domain.Asset{
		ID:       "asset-1",
		BrandID:  "brand-1",
		Filename: "logo.png",
		MimeType: "image/png",
		FilePath: filePath,
	})

	req := httptest.NewRequest("GET", "/api/v1/assets/asset-1/file", nil)
	w := httptest.NewRecorder()
	router.ServeHTTP(w, req)

	if w.Code != http.StatusOK {
		t.Fatalf("status = %d, want %d; body: %s", w.Code, http.StatusOK, w.Body.String())
	}

	if ct := w.Header().Get("Content-Type"); ct != "image/png" {
		t.Errorf("content-type = %q, want %q", ct, "image/png")
	}
}

func TestServeAssetFile_NotFound(t *testing.T) {
	cfg := config.Default()
	_, router, _, _, _, _ := setupMockServerWithConfigAndRepos(t, cfg)

	req := httptest.NewRequest("GET", "/api/v1/assets/nonexistent/file", nil)
	w := httptest.NewRecorder()
	router.ServeHTTP(w, req)

	if w.Code != http.StatusNotFound {
		t.Errorf("status = %d, want %d", w.Code, http.StatusNotFound)
	}
}
