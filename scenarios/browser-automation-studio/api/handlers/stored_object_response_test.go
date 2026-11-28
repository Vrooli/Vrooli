package handlers

import (
	"bytes"
	"context"
	"io"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/go-chi/chi/v5"
	"github.com/minio/minio-go/v7"
	"github.com/sirupsen/logrus"
	"github.com/vrooli/browser-automation-studio/storage"
)

type storedObjectReader struct {
	*bytes.Reader
	closed int
}

func (r *storedObjectReader) Close() error {
	r.closed++
	return nil
}

type storedObjectRouteStorage struct {
	storage.StorageInterface
	artifactReads   int
	screenshotReads int
	reader          *storedObjectReader
}

func (s *storedObjectRouteStorage) GetArtifact(context.Context, string) (io.ReadCloser, *minio.ObjectInfo, error) {
	s.artifactReads++
	return s.reader, &minio.ObjectInfo{ContentType: "application/octet-stream", Size: 7}, nil
}

func (s *storedObjectRouteStorage) GetScreenshot(context.Context, string) (io.ReadCloser, *minio.ObjectInfo, error) {
	s.screenshotReads++
	return s.reader, &minio.ObjectInfo{ContentType: "image/png", Size: 7}, nil
}

func TestStoredObjectRoutesUseOneResponseOwnerAndKeepTheirStorageGetter(t *testing.T) {
	const objectName = "run-1/artifacts/item"
	const payload = "payload"

	for _, tc := range []struct {
		name               string
		contentType        string
		serve              func(*Handler, http.ResponseWriter, *http.Request)
		wantArtifactRead   int
		wantScreenshotRead int
	}{
		{
			name:             "artifact",
			contentType:      "application/octet-stream",
			serve:            func(h *Handler, w http.ResponseWriter, r *http.Request) { h.ServeArtifact(w, r) },
			wantArtifactRead: 1,
		},
		{
			name:               "screenshot",
			contentType:        "image/png",
			serve:              func(h *Handler, w http.ResponseWriter, r *http.Request) { h.ServeScreenshot(w, r) },
			wantScreenshotRead: 1,
		},
	} {
		t.Run(tc.name, func(t *testing.T) {
			reader := &storedObjectReader{Reader: bytes.NewReader([]byte(payload))}
			store := &storedObjectRouteStorage{reader: reader}
			h := &Handler{storage: store, log: logrus.New()}
			r := httptest.NewRequest(http.MethodGet, "/api/v1/objects/"+objectName, nil)
			rctx := chi.NewRouteContext()
			rctx.URLParams.Add("*", objectName)
			r = r.WithContext(context.WithValue(r.Context(), chi.RouteCtxKey, rctx))
			w := httptest.NewRecorder()

			tc.serve(h, w, r)

			if w.Code != http.StatusOK || w.Body.String() != payload {
				t.Fatalf("response = %d %q, want 200 %q", w.Code, w.Body.String(), payload)
			}
			if got := w.Header().Get("Content-Type"); got != tc.contentType {
				t.Fatalf("Content-Type = %q, want %q", got, tc.contentType)
			}
			if got := w.Header().Get("Content-Length"); got != "7" {
				t.Fatalf("Content-Length = %q, want 7", got)
			}
			if got := w.Header().Get("Cache-Control"); got != "public, max-age=3600" {
				t.Fatalf("Cache-Control = %q", got)
			}
			if store.artifactReads != tc.wantArtifactRead || store.screenshotReads != tc.wantScreenshotRead {
				t.Fatalf("getter calls artifact=%d screenshot=%d, want %d/%d", store.artifactReads, store.screenshotReads, tc.wantArtifactRead, tc.wantScreenshotRead)
			}
			if reader.closed != 1 {
				t.Fatalf("storage reader closed %d times, want exactly once", reader.closed)
			}
		})
	}
}
