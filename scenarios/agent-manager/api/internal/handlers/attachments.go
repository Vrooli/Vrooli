package handlers

import (
	"encoding/json"
	"io"
	"net/http"
	"strings"

	"github.com/gorilla/mux"
)

// =============================================================================
// ATTACHMENT ENDPOINTS
// =============================================================================

// UploadAttachment handles multipart file uploads.
// POST /api/v1/attachments/upload
func (h *Handler) UploadAttachment(w http.ResponseWriter, r *http.Request) {
	if h.storage == nil {
		writeSimpleError(w, r, "storage", "file storage not configured")
		return
	}

	// Enforce max size at the HTTP level
	maxSize := h.storage.MaxFileSize()
	r.Body = http.MaxBytesReader(w, r.Body, maxSize+1024) // +1KB for multipart overhead

	if err := r.ParseMultipartForm(maxSize); err != nil {
		if err.Error() == "http: request body too large" {
			w.WriteHeader(http.StatusRequestEntityTooLarge)
			_ = json.NewEncoder(w).Encode(map[string]string{"error": "file too large"})
			return
		}
		writeSimpleError(w, r, "body", "invalid multipart form")
		return
	}

	file, header, err := r.FormFile("file")
	if err != nil {
		writeSimpleError(w, r, "file", "missing file field")
		return
	}
	defer file.Close()

	// Detect content type from file bytes (don't trust Content-Type header)
	buf := make([]byte, 512)
	n, _ := file.Read(buf)
	detectedType := http.DetectContentType(buf[:n])
	// Reset the file reader
	if seeker, ok := file.(io.ReadSeeker); ok {
		_, _ = seeker.Seek(0, io.SeekStart)
	}

	if !h.storage.IsAllowedType(detectedType) {
		w.WriteHeader(http.StatusUnsupportedMediaType)
		_ = json.NewEncoder(w).Encode(map[string]string{"error": "unsupported file type: " + detectedType})
		return
	}

	// Override the header content-type with detected type
	header.Header.Set("Content-Type", detectedType)

	meta, err := h.storage.Upload(r.Context(), file, header)
	if err != nil {
		writeSimpleError(w, r, "upload", err.Error())
		return
	}

	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusCreated)
	_ = json.NewEncoder(w).Encode(map[string]interface{}{
		"id":           meta.ID,
		"file_name":    meta.FileName,
		"content_type": meta.ContentType,
		"file_size":    meta.FileSize,
		"storage_path": meta.StoragePath,
		"url":          h.storage.GetServingURL(meta.StoragePath),
	})
}

// ServeUpload serves uploaded files.
// GET /api/v1/uploads/{path:.*}
func (h *Handler) ServeUpload(w http.ResponseWriter, r *http.Request) {
	if h.storage == nil {
		http.NotFound(w, r)
		return
	}

	filePath := mux.Vars(r)["path"]
	if filePath == "" || strings.Contains(filePath, "..") {
		http.Error(w, "invalid path", http.StatusBadRequest)
		return
	}

	fullPath := h.storage.GetFilePath(filePath)
	http.ServeFile(w, r, fullPath)
}

// GetRunByTag retrieves a run by its custom tag.
