// Package handlers provides HTTP handlers for the Agent Inbox API.
// This file handles file upload and serving.
package handlers

import (
	"context"
	"log"
	"net/http"
	"strings"

	"agent-inbox/config"
	"agent-inbox/domain"
	"agent-inbox/services"

	"github.com/gorilla/mux"
)

// AttachmentRepository defines the interface for attachment persistence.
type AttachmentRepository interface {
	CreateAttachment(ctx context.Context, att *domain.Attachment) error
}

// UploadHandlers provides HTTP handlers for file uploads.
// Separated from main Handlers to allow independent storage configuration.
type UploadHandlers struct {
	Storage services.StorageService
	Repo    AttachmentRepository
	cfg     *config.StorageConfig
}

// NewUploadHandlers creates a new UploadHandlers with the given storage service.
func NewUploadHandlers(storage services.StorageService, cfg *config.StorageConfig) *UploadHandlers {
	return &UploadHandlers{
		Storage: storage,
		cfg:     cfg,
	}
}

// SetRepo sets the repository for attachment persistence.
// This is called after construction to break circular dependency.
func (h *UploadHandlers) SetRepo(repo AttachmentRepository) {
	h.Repo = repo
}

// UploadFile handles file uploads via multipart form.
// POST /api/v1/attachments/upload
//
// Request: multipart/form-data with "file" field
// Response: { "id": "uuid", "file_name": "...", "content_type": "...", ... }
func (h *UploadHandlers) UploadFile(w http.ResponseWriter, r *http.Request) {
	ctx := r.Context()

	// Parse multipart form with max file size limit
	maxSize := h.Storage.GetMaxFileSize()
	r.Body = http.MaxBytesReader(w, r.Body, maxSize+1024) // Extra for form overhead

	if err := r.ParseMultipartForm(maxSize); err != nil {
		if strings.Contains(err.Error(), "request body too large") {
			http.Error(w, "File too large", http.StatusRequestEntityTooLarge)
			return
		}
		http.Error(w, "Failed to parse form: "+err.Error(), http.StatusBadRequest)
		return
	}

	// Get the file from the form
	file, header, err := r.FormFile("file")
	if err != nil {
		http.Error(w, "No file provided", http.StatusBadRequest)
		return
	}
	defer file.Close()

	// Upload to storage
	attachment, err := h.Storage.Upload(ctx, file, header)
	if err != nil {
		log.Printf("[ERROR] Failed to upload file: %v", err)
		if strings.Contains(err.Error(), "not allowed") {
			http.Error(w, err.Error(), http.StatusUnsupportedMediaType)
			return
		}
		if strings.Contains(err.Error(), "exceeds maximum") {
			http.Error(w, err.Error(), http.StatusRequestEntityTooLarge)
			return
		}
		http.Error(w, "Failed to upload file", http.StatusInternalServerError)
		return
	}

	// Save to database if repo is available
	if h.Repo != nil {
		if err := h.Repo.CreateAttachment(ctx, attachment); err != nil {
			log.Printf("[ERROR] Failed to save attachment to database: %v", err)
			// Continue anyway - file is uploaded, just not tracked
		}
	}

	// Return attachment metadata
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusCreated)
	// Manual JSON to avoid import cycle
	w.Write([]byte(`{"id":"` + attachment.ID + `","file_name":"` + attachment.FileName + `","content_type":"` + attachment.ContentType + `","file_size":` + itoa(attachment.FileSize) + `,"storage_path":"` + attachment.StoragePath + `","url":"` + h.Storage.GetFileURL(attachment.StoragePath) + `"}`))
}

// ServeFile serves uploaded files.
// GET /api/v1/uploads/{path:.*}
func (h *UploadHandlers) ServeFile(w http.ResponseWriter, r *http.Request) {
	vars := mux.Vars(r)
	path := vars["path"]

	// Security: Prevent path traversal
	if strings.Contains(path, "..") {
		http.Error(w, "Invalid path", http.StatusBadRequest)
		return
	}

	// Get full filesystem path
	fullPath := h.Storage.GetPath(path)

	// Serve the file
	http.ServeFile(w, r, fullPath)
}

// RegisterRoutes registers upload-related routes.
func (h *UploadHandlers) RegisterRoutes(r *mux.Router) {
	r.HandleFunc("/api/v1/attachments/upload", h.UploadFile).Methods("POST", "OPTIONS")
	r.HandleFunc("/api/v1/uploads/{path:.*}", h.ServeFile).Methods("GET", "OPTIONS")
}

// Helper to convert int64 to string without importing strconv
func itoa(n int64) string {
	if n == 0 {
		return "0"
	}

	var result []byte
	negative := n < 0
	if negative {
		n = -n
	}

	for n > 0 {
		result = append([]byte{byte('0' + n%10)}, result...)
		n /= 10
	}

	if negative {
		result = append([]byte{'-'}, result...)
	}
	return string(result)
}
