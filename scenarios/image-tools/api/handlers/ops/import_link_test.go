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
	"time"

	sharedcapacity "github.com/vrooli/vrooli/packages/capacity"
	"image-tools/internal/fetch"
	"image-tools/internal/storage"
)

type fakeLinkFetcher struct {
	data      []byte
	err       error
	url       string
	admission *fakeLinkAdmission
}

func (f *fakeLinkFetcher) Fetch(_ context.Context, raw string) ([]byte, string, error) {
	if f.admission != nil && !f.admission.reserved {
		return nil, "", errors.New("fetch ran before capacity admission")
	}
	f.url = raw
	return f.data, "image/png", f.err
}

type fakeLinkAdmission struct {
	reserved        bool
	completed       int
	request         sharedcapacity.DiskJobRequest
	err             error
	completeErr     error
	heartbeatErr    error
	heartbeatCalled chan struct{}
}

func (f *fakeLinkAdmission) ReserveDiskJob(_ context.Context, request sharedcapacity.DiskJobRequest) (sharedcapacity.DiskJobLease, error) {
	f.request = request
	if f.err != nil {
		return nil, f.err
	}
	f.reserved = true
	return &fakeLinkLease{admission: f}, nil
}

type fakeLinkLease struct{ admission *fakeLinkAdmission }

func (l *fakeLinkLease) Reservation() sharedcapacity.DiskJobReceipt {
	return sharedcapacity.DiskJobReceipt{OwnerID: l.admission.request.OwnerID, JobID: l.admission.request.JobID, Status: "active"}
}
func (l *fakeLinkLease) Heartbeat(context.Context) error {
	if l.admission.heartbeatCalled != nil {
		select {
		case l.admission.heartbeatCalled <- struct{}{}:
		default:
		}
	}
	return l.admission.heartbeatErr
}
func (*fakeLinkLease) Observe(context.Context) (sharedcapacity.DiskJobHeadroom, error) {
	return sharedcapacity.DiskJobHeadroom{}, nil
}
func (*fakeLinkLease) Extend(context.Context, []sharedcapacity.DiskJobPath) (sharedcapacity.DiskJobReceipt, error) {
	return sharedcapacity.DiskJobReceipt{}, nil
}
func (*fakeLinkLease) SamplePath(context.Context, string, int64, int64) (sharedcapacity.DiskJobReceipt, error) {
	return sharedcapacity.DiskJobReceipt{}, nil
}
func (l *fakeLinkLease) Complete(context.Context) (sharedcapacity.DiskJobReceipt, error) {
	l.admission.completed++
	return sharedcapacity.DiskJobReceipt{Status: "completed"}, l.admission.completeErr
}
func (l *fakeLinkLease) Release(ctx context.Context) error { _, err := l.Complete(ctx); return err }

func linkHandler(store ImportLinkStore, fetcher RemoteImageFetcher, disk sharedcapacity.DiskJobAdmitter) http.HandlerFunc {
	return importLinkHandler(store, fetcher, storage.DefaultGuard(), disk, func(context.Context) (string, error) { return "/tmp/image-tools-test-input", nil })
}

func TestImportLeaseHeartbeatFailureCancelsProducerContext(t *testing.T) {
	called := make(chan struct{}, 1)
	admission := &fakeLinkAdmission{heartbeatErr: errors.New("receipt renewal failed"), heartbeatCalled: called}
	ctx, cancel, done := maintainImportLease(context.Background(), &fakeLinkLease{admission: admission}, time.Millisecond)
	defer cancel()
	select {
	case <-called:
	case <-time.After(time.Second):
		t.Fatal("heartbeat was not attempted")
	}
	select {
	case <-ctx.Done():
	case <-time.After(time.Second):
		t.Fatal("heartbeat failure did not cancel producer context")
	}
	select {
	case <-done:
	case <-time.After(time.Second):
		t.Fatal("heartbeat worker did not stop")
	}
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
	admission := &fakeLinkAdmission{}
	fetcher.admission = admission
	h := linkHandler(store, fetcher, admission)
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
	if admission.completed != 1 || admission.request.OwnerID != "image-tools:link-import" || admission.request.FloorBytes != importLinkFreeFloorBytes || len(admission.request.Paths) != 1 || admission.request.Paths[0].PeakBytes != importLinkPeakBytes {
		t.Fatalf("capacity lifecycle/request mismatch: completed=%d request=%+v", admission.completed, admission.request)
	}
	if res.Header().Get("Cache-Control") != "no-store" {
		t.Fatalf("cache control=%q", res.Header().Get("Cache-Control"))
	}
}

func TestImportLinkRejectsUnsafeURLAndDoesNotFetch(t *testing.T) {
	fetcher := &fakeLinkFetcher{err: fetch.ErrUnsafeRemoteURL}
	store := &fakeImportStore{}
	h := linkHandler(store, fetcher, &fakeLinkAdmission{})
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
		admission := &fakeLinkAdmission{}
		h := linkHandler(store, &fakeLinkFetcher{data: []byte("not an image")}, admission)
		res := httptest.NewRecorder()
		h.ServeHTTP(res, httptest.NewRequest(http.MethodPost, "/api/v1/import/link", bytes.NewBufferString(`{"url":"https://images.example/file"}`)))
		if res.Code != http.StatusUnprocessableEntity || store.key != "" || admission.completed != 1 {
			t.Fatalf("status=%d stored=%q", res.Code, store.key)
		}
	})
	t.Run("ingest size limit", func(t *testing.T) {
		data := testPNG(t)
		store := &fakeImportStore{}
		guard := storage.Guard{MaxBytes: int64(len(data) - 1), MaxMegapixels: 128, MaxDimension: 30000}
		admission := &fakeLinkAdmission{}
		h := importLinkHandler(store, &fakeLinkFetcher{data: data}, guard, admission, func(context.Context) (string, error) { return "/tmp/test", nil })
		res := httptest.NewRecorder()
		h.ServeHTTP(res, httptest.NewRequest(http.MethodPost, "/api/v1/import/link", bytes.NewBufferString(`{"url":"https://images.example/file"}`)))
		if res.Code != http.StatusRequestEntityTooLarge || store.key != "" {
			t.Fatalf("status=%d stored=%q", res.Code, store.key)
		}
	})
	t.Run("extra field", func(t *testing.T) {
		fetcher := &fakeLinkFetcher{data: testPNG(t)}
		admission := &fakeLinkAdmission{}
		h := linkHandler(&fakeImportStore{}, fetcher, admission)
		res := httptest.NewRecorder()
		h.ServeHTTP(res, httptest.NewRequest(http.MethodPost, "/api/v1/import/link", bytes.NewBufferString(`{"url":"https://images.example/file","path":"/tmp/out"}`)))
		if res.Code != http.StatusBadRequest || fetcher.url != "" || admission.request.OwnerID != "" {
			t.Fatalf("status=%d fetched=%q admitted=%q", res.Code, fetcher.url, admission.request.OwnerID)
		}
	})
	t.Run("storage route failure before admission", func(t *testing.T) {
		fetcher := &fakeLinkFetcher{data: testPNG(t)}
		admission := &fakeLinkAdmission{}
		h := importLinkHandler(&fakeImportStore{}, fetcher, storage.DefaultGuard(), admission, func(context.Context) (string, error) { return "", errors.New("route unavailable") })
		res := httptest.NewRecorder()
		h.ServeHTTP(res, httptest.NewRequest(http.MethodPost, "/api/v1/import/link", bytes.NewBufferString(`{"url":"https://images.example/file"}`)))
		if res.Code != http.StatusServiceUnavailable || fetcher.url != "" || admission.request.OwnerID != "" {
			t.Fatalf("status=%d fetched=%q admitted=%q", res.Code, fetcher.url, admission.request.OwnerID)
		}
	})
}

func TestImportLinkMapsTooLargeAndStorageFailure(t *testing.T) {
	t.Run("too large", func(t *testing.T) {
		fetcher := &fakeLinkFetcher{err: fetch.ErrRemoteImageTooLarge}
		admission := &fakeLinkAdmission{}
		h := linkHandler(&fakeImportStore{}, fetcher, admission)
		res := httptest.NewRecorder()
		h.ServeHTTP(res, httptest.NewRequest(http.MethodPost, "/api/v1/import/link", bytes.NewBufferString(`{"url":"https://images.example/file"}`)))
		if res.Code != http.StatusRequestEntityTooLarge {
			t.Fatalf("status=%d body=%s", res.Code, res.Body.String())
		}
		if admission.completed != 1 {
			t.Fatalf("lease completed=%d", admission.completed)
		}
	})
	t.Run("store failure", func(t *testing.T) {
		store := &fakeImportStore{err: errors.New("disk full")}
		admission := &fakeLinkAdmission{}
		h := linkHandler(store, &fakeLinkFetcher{data: testPNG(t)}, admission)
		res := httptest.NewRecorder()
		h.ServeHTTP(res, httptest.NewRequest(http.MethodPost, "/api/v1/import/link", bytes.NewBufferString(`{"url":"https://images.example/file"}`)))
		if res.Code != http.StatusInternalServerError {
			t.Fatalf("status=%d body=%s", res.Code, res.Body.String())
		}
		if admission.completed != 0 {
			t.Fatalf("lease completed=%d after unverified store cleanup; want retained", admission.completed)
		}
	})
	t.Run("admission denied before fetch", func(t *testing.T) {
		admission := &fakeLinkAdmission{err: errors.New("capacity denied")}
		fetcher := &fakeLinkFetcher{data: testPNG(t)}
		store := &fakeImportStore{}
		res := httptest.NewRecorder()
		linkHandler(store, fetcher, admission).ServeHTTP(res, httptest.NewRequest(http.MethodPost, "/api/v1/import/link", bytes.NewBufferString(`{"url":"https://images.example/file"}`)))
		if res.Code != http.StatusServiceUnavailable || fetcher.url != "" || store.key != "" {
			t.Fatalf("status=%d fetched=%q stored=%q", res.Code, fetcher.url, store.key)
		}
	})
	t.Run("completion failure does not claim success", func(t *testing.T) {
		admission := &fakeLinkAdmission{completeErr: errors.New("receipt store unavailable")}
		fetcher := &fakeLinkFetcher{data: testPNG(t)}
		res := httptest.NewRecorder()
		linkHandler(&fakeImportStore{}, fetcher, admission).ServeHTTP(res, httptest.NewRequest(http.MethodPost, "/api/v1/import/link", bytes.NewBufferString(`{"url":"https://images.example/file"}`)))
		if res.Code != http.StatusInternalServerError || admission.completed != 1 {
			t.Fatalf("status=%d completed=%d", res.Code, admission.completed)
		}
	})
}
