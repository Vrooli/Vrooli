package handlers

import (
	"context"
	"fmt"
	"io"
	"net/http"
	"strings"
	"time"

	"github.com/go-chi/chi/v5"
)

// ServeScreenshot handles GET /api/v1/screenshots/*
// Serves a screenshot from MinIO storage
func (h *Handler) ServeScreenshot(w http.ResponseWriter, r *http.Request) {
	// Extract object name from URL path
	objectName := chi.URLParam(r, "*")
	if objectName == "" {
		http.Error(w, "Screenshot path required", http.StatusBadRequest)
		return
	}

	// Remove any leading slash
	objectName = strings.TrimPrefix(objectName, "/")

	if h.storage == nil {
		http.Error(w, "Screenshot storage not available", http.StatusServiceUnavailable)
		return
	}

	ctx, cancel := context.WithTimeout(r.Context(), 10*time.Second)
	defer cancel()

	// Get screenshot from MinIO
	object, info, err := h.storage.GetScreenshot(ctx, objectName)
	if err != nil {
		h.log.WithError(err).WithField("object_name", objectName).Error("Failed to get screenshot")
		http.Error(w, "Screenshot not found", http.StatusNotFound)
		return
	}
	defer object.Close()

	// Set appropriate headers
	w.Header().Set("Content-Type", info.ContentType)
	w.Header().Set("Content-Length", fmt.Sprintf("%d", info.Size))
	w.Header().Set("Cache-Control", "public, max-age=3600") // Cache for 1 hour

	// Stream the file to response
	if _, err := io.Copy(w, object); err != nil {
		h.log.WithError(err).Error("Failed to stream screenshot")
	}
}

// ServeThumbnail handles GET /api/v1/screenshots/thumbnail/*
// Serves a thumbnail version of a screenshot
func (h *Handler) ServeThumbnail(w http.ResponseWriter, r *http.Request) {
	// For now, serve the same image as thumbnail
	// In a real implementation, you might generate actual thumbnails
	h.ServeScreenshot(w, r)
}
