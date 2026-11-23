package handlers

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"mime/multipart"
	"net/http"
	"strings"
	"time"

	"agent-manager/internal/storage"

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

	// Accept a bounded batch so callers can upload a message's images atomically.
	const (
		maxBatchFiles = 16
		maxBatchBytes = 80 * 1024 * 1024
	)
	maxSize := h.storage.MaxFileSize()
	r.Body = http.MaxBytesReader(w, r.Body, maxBatchBytes+16*1024)

	if err := r.ParseMultipartForm(maxSize); err != nil {
		if err.Error() == "http: request body too large" {
			w.WriteHeader(http.StatusRequestEntityTooLarge)
			_ = json.NewEncoder(w).Encode(map[string]string{"error": "file too large"})
			return
		}
		writeSimpleError(w, r, "body", "invalid multipart form")
		return
	}

	if r.MultipartForm != nil {
		defer r.MultipartForm.RemoveAll()
	}
	files := r.MultipartForm.File["file"]
	if len(files) == 0 {
		writeSimpleError(w, r, "file", "missing file field")
		return
	}
	if len(files) > maxBatchFiles {
		writeSimpleError(w, r, "file", "too many files")
		return
	}

	metas := make([]*storage.AttachmentMeta, 0, len(files))
	for _, header := range files {
		file, err := header.Open()
		if err != nil {
			if rollbackAttachments(r.Context(), h.storage, metas) != nil {
				http.Error(w, "failed to clean up partial attachment batch", http.StatusInternalServerError)
				return
			}
			writeSimpleError(w, r, "file", "invalid uploaded file")
			return
		}
		meta, detectedType, err := h.storeAttachment(r.Context(), file, header)
		_ = file.Close()
		if err != nil {
			if rollbackAttachments(r.Context(), h.storage, metas) != nil {
				http.Error(w, "failed to clean up partial attachment batch", http.StatusInternalServerError)
				return
			}
			if detectedType != "" {
				w.WriteHeader(http.StatusUnsupportedMediaType)
				_ = json.NewEncoder(w).Encode(map[string]string{"error": "unsupported file type: " + detectedType})
				return
			}
			writeSimpleError(w, r, "upload", err.Error())
			return
		}
		metas = append(metas, meta)
	}

	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusCreated)
	if len(metas) == 1 {
		_ = json.NewEncoder(w).Encode(attachmentResponse(h.storage, metas[0]))
		return
	}
	ids := make([]string, len(metas))
	for i, meta := range metas {
		ids[i] = meta.ID
	}
	_ = json.NewEncoder(w).Encode(map[string][]string{"ids": ids})
}

func (h *Handler) storeAttachment(ctx context.Context, file multipart.File, header *multipart.FileHeader) (*storage.AttachmentMeta, string, error) {
	buf := make([]byte, 512)
	n, _ := file.Read(buf)
	detectedType := http.DetectContentType(buf[:n])
	if seeker, ok := file.(io.Seeker); ok {
		_, _ = seeker.Seek(0, io.SeekStart)
	}
	if !h.storage.IsAllowedType(detectedType) {
		return nil, detectedType, errors.New("unsupported file type")
	}
	header.Header.Set("Content-Type", detectedType)
	meta, err := h.storage.Upload(ctx, file, header)
	return meta, "", err
}

func rollbackAttachments(ctx context.Context, svc storage.Service, metas []*storage.AttachmentMeta) error {
	rollbackCtx, cancel := context.WithTimeout(context.WithoutCancel(ctx), 5*time.Second)
	defer cancel()
	var rollbackErr error
	for i := len(metas) - 1; i >= 0; i-- {
		if err := svc.Delete(rollbackCtx, metas[i].ID); err != nil {
			rollbackErr = errors.Join(rollbackErr, fmt.Errorf("deleting partial attachment: %w", err))
		}
	}
	return rollbackErr
}

func attachmentResponse(svc storage.Service, meta *storage.AttachmentMeta) map[string]interface{} {
	return map[string]interface{}{
		"id": meta.ID, "file_name": meta.FileName, "content_type": meta.ContentType,
		"file_size": meta.FileSize, "storage_path": meta.StoragePath,
		"url": svc.GetServingURL(meta.StoragePath),
	}
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
