package handlers

import (
	"context"
	"io"
	"net/http"
	"strings"

	"github.com/go-chi/chi/v5"
	"github.com/minio/minio-go/v7"
	"github.com/vrooli/browser-automation-studio/constants"
)

// storedObjectRoute preserves the small differences between the artifact and
// screenshot URLs while routing their shared storage read and HTTP stream
// through one owner.
type storedObjectRoute struct {
	MissingField string
	Unavailable  string
	ReadError    string
	StreamError  string
	NotFound     func(string) *APIError
	Read         func(context.Context, string) (io.ReadCloser, *minio.ObjectInfo, error)
}

// serveStoredObject owns the common object-path validation, storage read,
// response headers, reader close, and HTTP streaming contract for stored
// artifact endpoints. Route adapters supply only their storage getter and
// established API error vocabulary.
func (h *Handler) serveStoredObject(w http.ResponseWriter, r *http.Request, route storedObjectRoute) {
	objectName := chi.URLParam(r, "*")
	if objectName == "" {
		h.respondError(w, ErrMissingRequiredField.WithDetails(map[string]string{"field": route.MissingField}))
		return
	}
	objectName = strings.TrimPrefix(objectName, "/")

	if h.storage == nil {
		h.respondError(w, ErrServiceUnavailable.WithMessage(route.Unavailable))
		return
	}

	ctx, cancel := context.WithTimeout(r.Context(), constants.ExtendedRequestTimeout)
	defer cancel()

	object, info, err := route.Read(ctx, objectName)
	if err != nil {
		h.log.WithError(err).WithField("object_name", objectName).Error(route.ReadError)
		h.respondError(w, route.NotFound(objectName))
		return
	}
	streamHTTPResponse(w, object, info.ContentType, info.Size, "public, max-age=3600", func(err error) {
		h.log.WithError(err).Error(route.StreamError)
	})
}
