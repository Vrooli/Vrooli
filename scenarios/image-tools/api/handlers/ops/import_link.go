package ops

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"io"
	"net/http"
	"strings"

	"image-tools/internal/fetch"
	"image-tools/internal/httpx"
	"image-tools/internal/storage"

	"github.com/google/uuid"
)

type RemoteImageFetcher interface {
	Fetch(context.Context, string) ([]byte, string, error)
}

type ImportLinkStore interface {
	Put(context.Context, string, io.Reader, string) error
}

type importLinkRequest struct {
	URL string `json:"url"`
}

type importLinkResponse struct {
	InputRef string `json:"input_ref"`
	Format   string `json:"format"`
	MIME     string `json:"mime_type"`
	Width    int    `json:"width"`
	Height   int    `json:"height"`
}

// importLinkHandler fetches one public image URL into Image Tools-owned blob
// storage. The URL is fetched server-side only after the fetcher's public-IP
// checks; callers receive an opaque ref usable by existing image operations.
func importLinkHandler(store ImportLinkStore, fetcher RemoteImageFetcher, guard storage.Guard) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodPost {
			w.Header().Set("Allow", http.MethodPost)
			httpx.WriteError(w, http.StatusMethodNotAllowed, httpx.CodeInvalidRequest, "method not allowed")
			return
		}
		r.Body = http.MaxBytesReader(w, r.Body, 8<<10)
		var req importLinkRequest
		dec := json.NewDecoder(r.Body)
		dec.DisallowUnknownFields()
		if err := dec.Decode(&req); err != nil || strings.TrimSpace(req.URL) == "" {
			httpx.WriteError(w, http.StatusBadRequest, httpx.CodeInvalidRequest, "provide a valid image URL")
			return
		}
		var extra any
		if err := dec.Decode(&extra); !errors.Is(err, io.EOF) {
			httpx.WriteError(w, http.StatusBadRequest, httpx.CodeInvalidRequest, "request must contain one JSON object")
			return
		}
		data, _, err := fetcher.Fetch(r.Context(), req.URL)
		if err != nil {
			status := http.StatusUnprocessableEntity
			if errors.Is(err, fetch.ErrUnsafeRemoteURL) {
				status = http.StatusBadRequest
			}
			if errors.Is(err, fetch.ErrRemoteImageTooLarge) {
				status = http.StatusRequestEntityTooLarge
			}
			httpx.WriteError(w, status, httpx.CodeInvalidRequest, err.Error())
			return
		}
		inspected, err := guard.Inspect(bytes.NewReader(data))
		if err != nil {
			status := http.StatusUnprocessableEntity
			if errors.Is(err, storage.ErrTooLarge) {
				status = http.StatusRequestEntityTooLarge
			}
			httpx.WriteError(w, status, httpx.CodeInvalidRequest, err.Error())
			return
		}
		if inspected.Format == "" {
			httpx.WriteError(w, http.StatusUnprocessableEntity, httpx.CodeInvalidRequest, "URL did not return a supported raster image")
			return
		}
		mime := "image/" + inspected.Format
		if inspected.Format == "jpeg" {
			mime = "image/jpeg"
		}
		ref := "input/" + uuid.NewString() + "." + inspected.Format
		if err := store.Put(r.Context(), ref, bytes.NewReader(inspected.Bytes), mime); err != nil {
			httpx.WriteError(w, http.StatusInternalServerError, httpx.CodeInternal, "failed to store imported image")
			return
		}
		w.Header().Set("Content-Type", "application/json")
		w.Header().Set("Cache-Control", "no-store")
		w.WriteHeader(http.StatusCreated)
		_ = json.NewEncoder(w).Encode(importLinkResponse{InputRef: ref, Format: inspected.Format, MIME: mime, Width: inspected.Width, Height: inspected.Height})
	}
}
