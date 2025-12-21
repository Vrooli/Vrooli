package handlers

import (
	"context"
	"fmt"
	"io"
	"net/http"
	"strings"

	"github.com/go-chi/chi/v5"
	"github.com/vrooli/browser-automation-studio/constants"
)

// ServeArtifact handles GET /api/v1/artifacts/*
func (h *Handler) ServeArtifact(w http.ResponseWriter, r *http.Request) {
	objectName := chi.URLParam(r, "*")
	if objectName == "" {
		h.respondError(w, ErrMissingRequiredField.WithDetails(map[string]string{"field": "artifact_path"}))
		return
	}

	objectName = strings.TrimPrefix(objectName, "/")

	if h.storage == nil {
		h.respondError(w, ErrServiceUnavailable.WithMessage("Artifact storage not available"))
		return
	}

	ctx, cancel := context.WithTimeout(r.Context(), constants.ExtendedRequestTimeout)
	defer cancel()

	object, info, err := h.storage.GetArtifact(ctx, objectName)
	if err != nil {
		h.log.WithError(err).WithField("object_name", objectName).Error("Failed to get artifact")
		h.respondError(w, ErrArtifactNotFound.WithDetails(map[string]string{"artifact": objectName}))
		return
	}
	defer object.Close()

	w.Header().Set("Content-Type", info.ContentType)
	w.Header().Set("Content-Length", fmt.Sprintf("%d", info.Size))
	w.Header().Set("Cache-Control", "public, max-age=3600")

	if _, err := io.Copy(w, object); err != nil {
		h.log.WithError(err).Error("Failed to stream artifact")
	}
}
