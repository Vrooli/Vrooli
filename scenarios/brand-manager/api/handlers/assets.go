// Package handlers - asset upload, retrieval, listing, and deletion.
// [REQ:BM-REQ-API-ASSETS] [REQ:BM-REQ-STORE-ASSETS]
package handlers

import (
	"database/sql"
	"errors"
	"io"
	"net/http"
	"os"
	"path/filepath"
	"strings"

	"brand-manager/apierr"
	"brand-manager/domain"

	"github.com/gorilla/mux"
)

// allowedMimeTypes is the whitelist of acceptable asset mime types.
var allowedMimeTypes = map[string]bool{
	"image/png":                true,
	"image/jpeg":               true,
	"image/svg+xml":            true,
	"image/webp":               true,
	"image/x-icon":             true,
	"image/vnd.microsoft.icon": true,
}

// UploadAsset handles POST /api/v1/brands/{id}/assets. [REQ:BM-REQ-API-ASSETS]
func (h *Handlers) UploadAsset(w http.ResponseWriter, r *http.Request) {
	brandID := mux.Vars(r)["id"]

	// Verify brand exists
	if _, done := getOrNotFound(w, func() (*domain.Brand, error) {
		return h.brands.GetByID(r.Context(), brandID)
	}, "brand"); done {
		return
	}

	// Parse multipart form (max 32MB)
	if err := r.ParseMultipartForm(32 << 20); err != nil {
		apierr.Write(w, apierr.Validation("invalid multipart form"))
		return
	}

	file, header, err := r.FormFile("file")
	if err != nil {
		apierr.Write(w, apierr.Validation("file field is required"))
		return
	}
	defer file.Close()

	// Validate mime type
	mimeType := header.Header.Get("Content-Type")
	if mimeType == "" {
		mimeType = "application/octet-stream"
	}
	if !allowedMimeTypes[mimeType] {
		apierr.Write(w, apierr.Validation("unsupported file type: "+mimeType+"; allowed: png, jpeg, svg, webp, ico"))
		return
	}

	// Sanitize filename
	filename := filepath.Base(header.Filename)
	if filename == "" || filename == "." || filename == ".." {
		apierr.Write(w, apierr.Validation("invalid filename"))
		return
	}

	// Create asset directory
	assetDir := h.assetDir(brandID)
	if err := os.MkdirAll(assetDir, 0o755); err != nil {
		apierr.Write(w, apierr.Internal("create asset directory", err))
		return
	}

	// Read upload into memory then write atomically
	destPath := filepath.Join(assetDir, filename)
	data, err := io.ReadAll(file)
	if err != nil {
		apierr.Write(w, apierr.Internal("read upload", err))
		return
	}
	size := int64(len(data))

	if err := writeFileAtomic(destPath, data); err != nil {
		apierr.Write(w, apierr.Internal("write asset file", err))
		return
	}

	// Create database record
	asset := &domain.Asset{
		ID:       h.newID(),
		BrandID:  brandID,
		Filename: filename,
		MimeType: mimeType,
		FilePath: destPath,
		Size:     size,
	}

	if err := h.assets.Create(r.Context(), asset); err != nil {
		apierr.Write(w, apierr.Internal("create asset record", err))
		return
	}

	writeJSON(w, http.StatusCreated, asset)
}

// GetAsset handles GET /api/v1/assets/{id}. [REQ:BM-REQ-API-ASSETS]
func (h *Handlers) GetAsset(w http.ResponseWriter, r *http.Request) {
	id := mux.Vars(r)["id"]
	asset, done := getOrNotFound(w, func() (*domain.Asset, error) {
		return h.assets.GetByID(r.Context(), id)
	}, "asset")
	if done {
		return
	}
	writeJSON(w, http.StatusOK, asset)
}

// ListAssets handles GET /api/v1/brands/{id}/assets. [REQ:BM-REQ-API-ASSETS]
func (h *Handlers) ListAssets(w http.ResponseWriter, r *http.Request) {
	brandID := mux.Vars(r)["id"]
	assets, err := h.assets.ListByBrandID(r.Context(), brandID)
	if err != nil {
		apierr.Write(w, apierr.Internal("list assets", err))
		return
	}
	if assets == nil {
		assets = []*domain.Asset{}
	}
	writeJSON(w, http.StatusOK, assets)
}

// DeleteAsset handles DELETE /api/v1/assets/{id}. [REQ:BM-REQ-API-ASSETS]
// Idempotent: returns 204 whether the asset existed or was already deleted.
func (h *Handlers) DeleteAsset(w http.ResponseWriter, r *http.Request) {
	id := mux.Vars(r)["id"]

	// Try to get the asset to delete the file
	asset, err := h.assets.GetByID(r.Context(), id)
	if err == nil {
		os.Remove(asset.FilePath)
	}

	err = h.assets.Delete(r.Context(), id)
	if err != nil && !errors.Is(err, sql.ErrNoRows) {
		apierr.Write(w, apierr.Internal("delete asset", err))
		return
	}
	w.WriteHeader(http.StatusNoContent)
}

// ServeAssetFile handles GET /api/v1/assets/{id}/file. [REQ:BM-REQ-API-ASSETS]
func (h *Handlers) ServeAssetFile(w http.ResponseWriter, r *http.Request) {
	id := mux.Vars(r)["id"]
	asset, done := getOrNotFound(w, func() (*domain.Asset, error) {
		return h.assets.GetByID(r.Context(), id)
	}, "asset")
	if done {
		return
	}

	// Validate path doesn't escape asset directory
	cleanPath := filepath.Clean(asset.FilePath)
	if !strings.HasPrefix(cleanPath, h.assetBaseDir()) {
		apierr.Write(w, apierr.Validation("invalid asset path"))
		return
	}

	w.Header().Set("Content-Type", asset.MimeType)
	http.ServeFile(w, r, cleanPath)
}

// assetDir returns the filesystem path for a brand's assets.
func (h *Handlers) assetDir(brandID string) string {
	return filepath.Join(h.assetBaseDir(), brandID)
}

// assetBaseDir returns the root assets directory.
func (h *Handlers) assetBaseDir() string {
	return h.cfg.AssetBasePath
}
