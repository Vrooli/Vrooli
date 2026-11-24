package handlers

import (
	"context"
	"io"
	"net/http"

	"github.com/minio/minio-go/v7"
)

// ServeArtifact handles GET /api/v1/artifacts/*
func (h *Handler) ServeArtifact(w http.ResponseWriter, r *http.Request) {
	h.serveStoredObject(w, r, storedObjectRoute{
		MissingField: "artifact_path",
		Unavailable:  "Artifact storage not available",
		NotFound: func(objectName string) *APIError {
			return ErrArtifactNotFound.WithDetails(map[string]string{"artifact": objectName})
		},
		ReadError:   "Failed to get artifact",
		StreamError: "Failed to stream artifact",
		Read: func(ctx context.Context, objectName string) (io.ReadCloser, *minio.ObjectInfo, error) {
			return h.storage.GetArtifact(ctx, objectName)
		},
	})
}
