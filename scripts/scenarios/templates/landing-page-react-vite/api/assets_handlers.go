package main

import (
	"encoding/json"
	"net/http"
	"os"
	"path/filepath"
	"strconv"
	"strings"

	"github.com/gorilla/mux"
)

const maxUploadSize = 10 * 1024 * 1024 // 10MB

// handleAssetUpload handles POST /api/v1/admin/assets/upload
func handleAssetUpload(as *AssetsService) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		// Limit request body size
		r.Body = http.MaxBytesReader(w, r.Body, maxUploadSize)

		// Parse multipart form
		if err := r.ParseMultipartForm(maxUploadSize); err != nil {
			http.Error(w, "File too large or invalid form data", http.StatusBadRequest)
			return
		}

		// Get the file from form
		file, header, err := r.FormFile("file")
		if err != nil {
			http.Error(w, "No file provided in 'file' field", http.StatusBadRequest)
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
			if strings.Contains(err.Error(), "invalid file type") {
				status = http.StatusBadRequest
			} else if strings.Contains(err.Error(), "file exceeds") {
				status = http.StatusRequestEntityTooLarge
			}
			logStructuredError("asset_upload_failed", map[string]interface{}{
				"error":    err.Error(),
				"filename": header.Filename,
			})
			http.Error(w, err.Error(), status)
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
		json.NewEncoder(w).Encode(asset)
	}
}

// handleAssetsList handles GET /api/v1/admin/assets
func handleAssetsList(as *AssetsService) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		category := r.URL.Query().Get("category")

		assets, err := as.List(category)
		if err != nil {
			logStructuredError("list_assets_failed", map[string]interface{}{"error": err.Error()})
			http.Error(w, "Failed to list assets", http.StatusInternalServerError)
			return
		}

		if assets == nil {
			assets = []Asset{}
		}

		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(map[string]interface{}{
			"assets": assets,
		})
	}
}

// handleAssetGet handles GET /api/v1/admin/assets/{id}
func handleAssetGet(as *AssetsService) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		vars := mux.Vars(r)
		idStr := vars["id"]

		id, err := strconv.Atoi(idStr)
		if err != nil {
			http.Error(w, "Invalid asset ID", http.StatusBadRequest)
			return
		}

		asset, err := as.Get(id)
		if err != nil {
			if err == ErrAssetNotFound {
				http.Error(w, "Asset not found", http.StatusNotFound)
				return
			}
			logStructuredError("get_asset_failed", map[string]interface{}{
				"id":    id,
				"error": err.Error(),
			})
			http.Error(w, "Failed to get asset", http.StatusInternalServerError)
			return
		}

		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(asset)
	}
}

// handleAssetDelete handles DELETE /api/v1/admin/assets/{id}
func handleAssetDelete(as *AssetsService) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		vars := mux.Vars(r)
		idStr := vars["id"]

		id, err := strconv.Atoi(idStr)
		if err != nil {
			http.Error(w, "Invalid asset ID", http.StatusBadRequest)
			return
		}

		if err := as.Delete(id); err != nil {
			if err == ErrAssetNotFound {
				http.Error(w, "Asset not found", http.StatusNotFound)
				return
			}
			logStructuredError("delete_asset_failed", map[string]interface{}{
				"id":    id,
				"error": err.Error(),
			})
			http.Error(w, "Failed to delete asset", http.StatusInternalServerError)
			return
		}

		logStructured("asset_deleted", map[string]interface{}{"id": id})

		w.WriteHeader(http.StatusNoContent)
	}
}

// handleServeUpload handles GET /api/v1/uploads/{path...}
// This serves uploaded files publicly (no auth required)
func handleServeUpload(as *AssetsService) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		// Get the path after /api/v1/uploads/
		vars := mux.Vars(r)
		storagePath := vars["path"]

		if storagePath == "" {
			http.Error(w, "File path required", http.StatusBadRequest)
			return
		}

		// Security: prevent directory traversal
		cleanPath := filepath.Clean(storagePath)
		if strings.Contains(cleanPath, "..") {
			http.Error(w, "Invalid path", http.StatusBadRequest)
			return
		}

		// Get full file path
		fullPath := as.GetFilePath(cleanPath)

		// Check if file exists
		stat, err := os.Stat(fullPath)
		if os.IsNotExist(err) {
			http.Error(w, "File not found", http.StatusNotFound)
			return
		}
		if err != nil {
			http.Error(w, "Failed to access file", http.StatusInternalServerError)
			return
		}
		if stat.IsDir() {
			http.Error(w, "Cannot serve directory", http.StatusBadRequest)
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
