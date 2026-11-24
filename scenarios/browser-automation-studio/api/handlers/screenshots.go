package handlers

import (
	"context"
	"io"
	"net/http"

	"github.com/minio/minio-go/v7"
)

// ServeScreenshot handles GET /api/v1/screenshots/*.
func (h *Handler) ServeScreenshot(w http.ResponseWriter, r *http.Request) {
	h.serveStoredObject(w, r, storedObjectRoute{
		MissingField: "screenshot_path",
		Unavailable:  "Screenshot storage not available",
		NotFound: func(objectName string) *APIError {
			return ErrScreenshotNotFound.WithDetails(map[string]string{"screenshot": objectName})
		},
		ReadError:   "Failed to get screenshot",
		StreamError: "Failed to stream screenshot",
		Read: func(ctx context.Context, objectName string) (io.ReadCloser, *minio.ObjectInfo, error) {
			return h.storage.GetScreenshot(ctx, objectName)
		},
	})
}

// ServeThumbnail handles GET /api/v1/screenshots/thumbnail/*.
// For now, it serves the same image as the screenshot route.
func (h *Handler) ServeThumbnail(w http.ResponseWriter, r *http.Request) {
	h.ServeScreenshot(w, r)
}
