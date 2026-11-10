package ops

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"image"
	"image/color"
	"image/png"
	"io"
	"net/http"
	"net/http/httptest"
	"testing"

	"image-tools/internal/fetch"
	"image-tools/internal/storage"
)

type fakeLinkFetcher struct {
	data []byte
	err  error
	url  string
}

func (f *fakeLinkFetcher) Fetch(_ context.Context, raw string) ([]byte, string, error) {
	f.url = raw
	return f.data, "image/png", f.err
}

type fakeImportStore struct {
	key  string
	mime string
	data []byte
	err  error
}

func (s *fakeImportStore) Put(_ context.Context, key string, r io.Reader, mime string) error {
	s.key, s.mime, s.err = key, mime, s.err
	if s.err != nil {
		return s.err
	}
	s.data, s.err = io.ReadAll(r)
	return s.err
}

func testPNG(t *testing.T) []byte {
	t.Helper()
	img := image.NewRGBA(image.Rect(0, 0, 2, 3))
	img.Set(0, 0, color.White)
	var buf bytes.Buffer
	if err := png.Encode(&buf, img); err != nil {
		t.Fatal(err)
	}
	return buf.Bytes()
}

func TestImportLinkStoresVerifiedImageAndReturnsOwnedReference(t *testing.T) {
	data := testPNG(t)
	fetcher := &fakeLinkFetcher{data: data}
	store := &fakeImportStore{}
	h := importLinkHandler(store, fetcher, storage.DefaultGuard())
	req := httptest.NewRequest(http.MethodPost, "/api/v1/import/link", bytes.NewBufferString(`{"url":"https://images.example/image.png"}`))
	req.Header.Set("Content-Type", "application/json")
	res := httptest.NewRecorder()
	h.ServeHTTP(res, req)
	if res.Code != http.StatusCreated {
		t.Fatalf("status=%d body=%s", res.Code, res.Body.String())
	}
	var got importLinkResponse
	if err := json.Unmarshal(res.Body.Bytes(), &got); err != nil {
		t.Fatal(err)
	}
	if got.InputRef == "" || got.Format != "png" || got.Width != 2 || got.Height != 3 {
		t.Fatalf("unexpected import response: %+v", got)
	}
	if store.key != got.InputRef || store.mime != "image/png" || !bytes.Equal(store.data, data) {
		t.Fatalf("stored key/mime/data mismatch: key=%q mime=%q bytes=%d", store.key, store.mime, len(store.data))
	}
	if fetcher.url != "https://images.example/image.png" {
		t.Fatalf("fetcher URL=%q", fetcher.url)
	}
	if res.Header().Get("Cache-Control") != "no-store" {
		t.Fatalf("cache control=%q", res.Header().Get("Cache-Control"))
	}
}

func TestImportLinkRejectsUnsafeURLAndDoesNotFetch(t *testing.T) {
	fetcher := &fakeLinkFetcher{err: fetch.ErrUnsafeRemoteURL}
	store := &fakeImportStore{}
	h := importLinkHandler(store, fetcher, storage.DefaultGuard())
	res := httptest.NewRecorder()
	h.ServeHTTP(res, httptest.NewRequest(http.MethodPost, "/api/v1/import/link", bytes.NewBufferString(`{"url":"http://127.0.0.1/private.png"}`)))
	if res.Code != http.StatusBadRequest {
		t.Fatalf("status=%d body=%s", res.Code, res.Body.String())
	}
	if store.key != "" {
		t.Fatalf("unsafe image was persisted at %q", store.key)
	}
}

func TestImportLinkRejectsNonImageAndMalformedRequests(t *testing.T) {
	t.Run("non image", func(t *testing.T) {
		store := &fakeImportStore{}
		h := importLinkHandler(store, &fakeLinkFetcher{data: []byte("not an image")}, storage.DefaultGuard())
		res := httptest.NewRecorder()
		h.ServeHTTP(res, httptest.NewRequest(http.MethodPost, "/api/v1/import/link", bytes.NewBufferString(`{"url":"https://images.example/file"}`)))
		if res.Code != http.StatusUnprocessableEntity || store.key != "" {
			t.Fatalf("status=%d stored=%q", res.Code, store.key)
		}
	})
	t.Run("ingest size limit", func(t *testing.T) {
		data := testPNG(t)
		store := &fakeImportStore{}
		guard := storage.Guard{MaxBytes: int64(len(data) - 1), MaxMegapixels: 128, MaxDimension: 30000}
		h := importLinkHandler(store, &fakeLinkFetcher{data: data}, guard)
		res := httptest.NewRecorder()
		h.ServeHTTP(res, httptest.NewRequest(http.MethodPost, "/api/v1/import/link", bytes.NewBufferString(`{"url":"https://images.example/file"}`)))
		if res.Code != http.StatusRequestEntityTooLarge || store.key != "" {
			t.Fatalf("status=%d stored=%q", res.Code, store.key)
		}
	})
	t.Run("extra field", func(t *testing.T) {
		fetcher := &fakeLinkFetcher{data: testPNG(t)}
		h := importLinkHandler(&fakeImportStore{}, fetcher, storage.DefaultGuard())
		res := httptest.NewRecorder()
		h.ServeHTTP(res, httptest.NewRequest(http.MethodPost, "/api/v1/import/link", bytes.NewBufferString(`{"url":"https://images.example/file","path":"/tmp/out"}`)))
		if res.Code != http.StatusBadRequest || fetcher.url != "" {
			t.Fatalf("status=%d fetched=%q", res.Code, fetcher.url)
		}
	})
}

func TestImportLinkMapsTooLargeAndStorageFailure(t *testing.T) {
	t.Run("too large", func(t *testing.T) {
		fetcher := &fakeLinkFetcher{err: fetch.ErrRemoteImageTooLarge}
		h := importLinkHandler(&fakeImportStore{}, fetcher, storage.DefaultGuard())
		res := httptest.NewRecorder()
		h.ServeHTTP(res, httptest.NewRequest(http.MethodPost, "/api/v1/import/link", bytes.NewBufferString(`{"url":"https://images.example/file"}`)))
		if res.Code != http.StatusRequestEntityTooLarge {
			t.Fatalf("status=%d body=%s", res.Code, res.Body.String())
		}
	})
	t.Run("store failure", func(t *testing.T) {
		store := &fakeImportStore{err: errors.New("disk full")}
		h := importLinkHandler(store, &fakeLinkFetcher{data: testPNG(t)}, storage.DefaultGuard())
		res := httptest.NewRecorder()
		h.ServeHTTP(res, httptest.NewRequest(http.MethodPost, "/api/v1/import/link", bytes.NewBufferString(`{"url":"https://images.example/file"}`)))
		if res.Code != http.StatusInternalServerError {
			t.Fatalf("status=%d body=%s", res.Code, res.Body.String())
		}
	})
}
