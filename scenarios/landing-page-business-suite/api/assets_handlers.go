package main

import (
	"net/http"
	"os"
	"path/filepath"
	"strings"
)

const maxUploadSize = 10 * 1024 * 1024 // 10MB

// handleAssetUpload handles POST /api/v1/admin/assets/upload
func handleAssetUpload(as *AssetsService) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		// Limit request body size
		r.Body = http.MaxBytesReader(w, r.Body, maxUploadSize)

		// Parse multipart form
		if err := r.ParseMultipartForm(maxUploadSize); err != nil {
			writeJSONError(w, http.StatusBadRequest, "File too large or invalid form data", ApiErrorTypeValidation)
			return
		}

		// Get the file from form
		file, header, err := r.FormFile("file")
		if err != nil {
			writeJSONError(w, http.StatusBadRequest, "No file provided in 'file' field", ApiErrorTypeValidation)
			return
		}
		defer file.Close()

		// Get optional form fields
		category := r.FormValue("category")
		altText := r.FormValue("alt_text")
		uploadedBy := r.FormValue("uploaded_by")

		// Upload the file
		asset, err := as.Upload(&AssetUploadRequest{
			File:       file,
			Header:     header,
			Category:   category,
			AltText:    altText,
			UploadedBy: uploadedBy,
		})
		if err != nil {
			status := http.StatusInternalServerError
			errType := ApiErrorTypeServerError
			if strings.Contains(err.Error(), "invalid file type") {
				status = http.StatusBadRequest
				errType = ApiErrorTypeValidation
			} else if strings.Contains(err.Error(), "file exceeds") {
				status = http.StatusRequestEntityTooLarge
				errType = ApiErrorTypeValidation
			}
			logStructuredError("asset_upload_failed", map[string]interface{}{
				"error":    err.Error(),
				"filename": header.Filename,
			})
			writeJSONError(w, status, err.Error(), errType)
			return
		}

		logStructured("asset_uploaded", map[string]interface{}{
			"id":       asset.ID,
			"filename": asset.Filename,
			"category": asset.Category,
			"size":     asset.SizeBytes,
		})

		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusCreated)
		writeJSONSuccessData(w, asset)
	}
}

// handleAssetsList handles GET /api/v1/admin/assets
func handleAssetsList(as *AssetsService) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		category := getQueryParam(r, "category")

		assets, err := as.List(category)
		if err != nil {
			logStructuredError("list_assets_failed", map[string]interface{}{"error": err.Error()})
			writeJSONError(w, http.StatusInternalServerError, "Failed to list assets", ApiErrorTypeServerError)
			return
		}

		if assets == nil {
			assets = []Asset{}
		}

		writeJSONSuccessData(w, map[string]interface{}{
			"assets": assets,
		})
	}
}

// handleAssetGet handles GET /api/v1/admin/assets/{id}
func handleAssetGet(as *AssetsService) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		id, ok := getPathParamInt(w, r, "id")
		if !ok {
			return
		}

		asset, err := as.Get(id)
		if err != nil {
			if err == ErrAssetNotFound {
				writeJSONError(w, http.StatusNotFound, "Asset not found", ApiErrorTypeNotFound)
				return
			}
			logStructuredError("get_asset_failed", map[string]interface{}{
				"id":    id,
				"error": err.Error(),
			})
			writeJSONError(w, http.StatusInternalServerError, "Failed to get asset", ApiErrorTypeServerError)
			return
		}

		writeJSONSuccessData(w, asset)
	}
}

// handleAssetDelete handles DELETE /api/v1/admin/assets/{id}
func handleAssetDelete(as *AssetsService) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		id, ok := getPathParamInt(w, r, "id")
		if !ok {
			return
		}

		if err := as.Delete(id); err != nil {
			if err == ErrAssetNotFound {
				writeJSONError(w, http.StatusNotFound, "Asset not found", ApiErrorTypeNotFound)
				return
			}
			logStructuredError("delete_asset_failed", map[string]interface{}{
				"id":    id,
				"error": err.Error(),
			})
			writeJSONError(w, http.StatusInternalServerError, "Failed to delete asset", ApiErrorTypeServerError)
			return
		}

		logStructured("asset_deleted", map[string]interface{}{"id": id})

		w.WriteHeader(http.StatusNoContent)
	}
}

// handleServeUpload handles GET /api/v1/uploads/{path...}
// This serves uploaded files publicly (no auth required)
//
//nolint:unused // reserved for debug-only asset preview handler
func handleServeUpload(as *AssetsService) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		// Get the path after /api/v1/uploads/
		storagePath, ok := getPathParam(r, "path")
		if !ok || storagePath == "" {
			writeJSONError(w, http.StatusBadRequest, "File path required", ApiErrorTypeValidation)
			return
		}

		// Security: prevent directory traversal
		cleanPath := filepath.Clean(storagePath)
		if strings.Contains(cleanPath, "..") {
			writeJSONError(w, http.StatusBadRequest, "Invalid path", ApiErrorTypeValidation)
			return
		}

		// Get full file path
		fullPath := as.GetFilePath(cleanPath)

		// Check if file exists
		stat, err := os.Stat(fullPath)
		if os.IsNotExist(err) {
			writeJSONError(w, http.StatusNotFound, "File not found", ApiErrorTypeNotFound)
			return
		}
		if err != nil {
			writeJSONError(w, http.StatusInternalServerError, "Failed to access file", ApiErrorTypeServerError)
			return
		}
		if stat.IsDir() {
			writeJSONError(w, http.StatusBadRequest, "Cannot serve directory", ApiErrorTypeValidation)
			return
		}

		// Detect content type
		contentType := detectMimeType(fullPath)
		w.Header().Set("Content-Type", contentType)

		// Set cache headers for static assets
		w.Header().Set("Cache-Control", "public, max-age=31536000, immutable")

		// Serve the file
		http.ServeFile(w, r, fullPath)
	}
}
