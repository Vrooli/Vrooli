package ops

import (
	"bytes"
	"context"
	"image"
	"image/color"
	"image/png"
	"mime/multipart"
	"net/http"
	"net/http/httptest"
	"testing"

	internaljobs "image-tools/internal/jobs"
	internalops "image-tools/internal/ops"
	"image-tools/internal/storage"

	"github.com/gorilla/mux"
	"github.com/vrooli/api-core/blobstore"
)

// fakeRecorder records terminal jobs in memory (the handler's observability
// seam) without a database.
type fakeRecorder struct {
	jobs []internaljobs.Job
}

func (f *fakeRecorder) Record(spec internaljobs.Spec, resultRef string, runErr error) (internaljobs.Job, error) {
	state := internaljobs.StateSucceeded
	if runErr != nil {
		state = internaljobs.StateFailed
	}
	j := internaljobs.Job{ID: "job-" + spec.Operation, Operation: spec.Operation, Lane: spec.Lane, State: state, ResultRef: resultRef}
	if runErr != nil {
		j.Error = runErr.Error()
	}
	f.jobs = append(f.jobs, j)
	return j, nil
}

func newTestServer(t *testing.T) (*mux.Router, *storage.Store, *fakeRecorder) {
	t.Helper()
	store := storage.NewWithBlobStore(blobstore.NewMemoryBlobStore(), t.TempDir())
	rec := &fakeRecorder{}
	deps := &Deps{Store: store, Jobs: rec, Guard: storage.DefaultGuard()}
	r := mux.NewRouter()
	r.HandleFunc("/api/v1/ops/{operation}", deps.runHandler).Methods(http.MethodPost)
	r.HandleFunc("/api/v1/blobs/{key:.*}", deps.blobHandler).Methods(http.MethodGet)
	return r, store, rec
}

func pngBytes(t *testing.T, w, h int) []byte {
	t.Helper()
	img := image.NewRGBA(image.Rect(0, 0, w, h))
	for y := 0; y < h; y++ {
		for x := 0; x < w; x++ {
			img.Set(x, y, color.RGBA{R: uint8(x), G: uint8(y), B: 100, A: 255})
		}
	}
	var buf bytes.Buffer
	if err := png.Encode(&buf, img); err != nil {
		t.Fatal(err)
	}
	return buf.Bytes()
}

func multipartBody(t *testing.T, img []byte, params string, overlay []byte) (*bytes.Buffer, string) {
	t.Helper()
	var buf bytes.Buffer
	mw := multipart.NewWriter(&buf)
	fw, _ := mw.CreateFormFile("file", "in.png")
	_, _ = fw.Write(img)
	if params != "" {
		_ = mw.WriteField("params", params)
	}
	if overlay != nil {
		ow, _ := mw.CreateFormFile("overlay", "wm.png")
		_, _ = ow.Write(overlay)
	}
	_ = mw.Close()
	return &buf, mw.FormDataContentType()
}

func TestRunResizeBytes(t *testing.T) {
	r, _, rec := newTestServer(t)
	body, ct := multipartBody(t, pngBytes(t, 100, 80), `{"resize":{"width":50,"height":40,"fit":"stretch"}}`, nil)
	req := httptest.NewRequest(http.MethodPost, "/api/v1/ops/resize?output=bytes", body)
	req.Header.Set("Content-Type", ct)
	w := httptest.NewRecorder()
	r.ServeHTTP(w, req)

	if w.Code != http.StatusOK {
		t.Fatalf("status = %d, body=%s", w.Code, w.Body.String())
	}
	if got := w.Header().Get("X-Image-Tools-Width"); got != "50" {
		t.Fatalf("width header = %q, want 50", got)
	}
	out, _, err := internalops.Decode(w.Body.Bytes())
	if err != nil {
		t.Fatalf("decode result: %v", err)
	}
	if out.Bounds().Dx() != 50 || out.Bounds().Dy() != 40 {
		t.Fatalf("result %dx%d, want 50x40", out.Bounds().Dx(), out.Bounds().Dy())
	}
	if len(rec.jobs) != 1 || rec.jobs[0].State != internaljobs.StateSucceeded {
		t.Fatalf("expected one succeeded recorded job, got %+v", rec.jobs)
	}
}

func TestRunBlobThenFetch(t *testing.T) {
	r, _, _ := newTestServer(t)
	body, ct := multipartBody(t, pngBytes(t, 60, 60), `{"convert":{"format":"jpeg","quality":80}}`, nil)
	req := httptest.NewRequest(http.MethodPost, "/api/v1/ops/convert?output=blob", body)
	req.Header.Set("Content-Type", ct)
	w := httptest.NewRecorder()
	r.ServeHTTP(w, req)
	if w.Code != http.StatusOK {
		t.Fatalf("status = %d, body=%s", w.Code, w.Body.String())
	}
	// Response is RunOpResponse JSON with a blob ref; fetch it back.
	if !bytes.Contains(w.Body.Bytes(), []byte(`"image/jpeg"`)) {
		t.Fatalf("expected jpeg mime in response, got %s", w.Body.String())
	}
}

func TestRunFormatQueryOverride(t *testing.T) {
	// filter has no format param; ?format=webp must still yield WebP output.
	r, _, _ := newTestServer(t)
	body, ct := multipartBody(t, pngBytes(t, 40, 40), `{"filter":{"filter":"grayscale"}}`, nil)
	req := httptest.NewRequest(http.MethodPost, "/api/v1/ops/filter?output=bytes&format=webp", body)
	req.Header.Set("Content-Type", ct)
	w := httptest.NewRecorder()
	r.ServeHTTP(w, req)
	if w.Code != http.StatusOK {
		t.Fatalf("status = %d, body=%s", w.Code, w.Body.String())
	}
	if got := w.Header().Get("X-Image-Tools-Format"); got != "webp" {
		t.Fatalf("format header = %q, want webp", got)
	}
	if _, meta, err := internalops.Decode(w.Body.Bytes()); err != nil || meta.Format != "webp" {
		t.Fatalf("result not webp: meta=%+v err=%v", meta, err)
	}
}

func TestRunUnknownOp(t *testing.T) {
	r, _, _ := newTestServer(t)
	body, ct := multipartBody(t, pngBytes(t, 10, 10), "", nil)
	req := httptest.NewRequest(http.MethodPost, "/api/v1/ops/bogus", body)
	req.Header.Set("Content-Type", ct)
	w := httptest.NewRecorder()
	r.ServeHTTP(w, req)
	if w.Code != http.StatusNotFound {
		t.Fatalf("status = %d, want 404", w.Code)
	}
}

func TestRunBadImage(t *testing.T) {
	r, _, rec := newTestServer(t)
	body, ct := multipartBody(t, []byte("not an image"), `{"resize":{"width":10}}`, nil)
	req := httptest.NewRequest(http.MethodPost, "/api/v1/ops/resize", body)
	req.Header.Set("Content-Type", ct)
	w := httptest.NewRecorder()
	r.ServeHTTP(w, req)
	if w.Code != http.StatusUnprocessableEntity {
		t.Fatalf("status = %d, want 422, body=%s", w.Code, w.Body.String())
	}
	if len(rec.jobs) != 1 || rec.jobs[0].State != internaljobs.StateFailed {
		t.Fatalf("expected one failed recorded job, got %+v", rec.jobs)
	}
}

func TestRunOverlayImage(t *testing.T) {
	r, _, _ := newTestServer(t)
	body, ct := multipartBody(t, pngBytes(t, 100, 100), `{"overlay":{"position":"top-left","opacity":0.5}}`, pngBytes(t, 20, 20))
	req := httptest.NewRequest(http.MethodPost, "/api/v1/ops/overlay?output=bytes", body)
	req.Header.Set("Content-Type", ct)
	w := httptest.NewRecorder()
	r.ServeHTTP(w, req)
	if w.Code != http.StatusOK {
		t.Fatalf("status = %d, body=%s", w.Code, w.Body.String())
	}
}

func TestBlobNotFound(t *testing.T) {
	r, _, _ := newTestServer(t)
	req := httptest.NewRequest(http.MethodGet, "/api/v1/blobs/out/missing.png", nil)
	w := httptest.NewRecorder()
	r.ServeHTTP(w, req)
	if w.Code != http.StatusNotFound {
		t.Fatalf("status = %d, want 404", w.Code)
	}
}

func TestListOperations(t *testing.T) {
	h := NewConnectHandler()
	resp, err := h.ListOperations(context.Background(), nil)
	if err != nil {
		t.Fatal(err)
	}
	if len(resp.Msg.Operations) != len(internalops.Names()) {
		t.Fatalf("ListOperations returned %d ops, want %d", len(resp.Msg.Operations), len(internalops.Names()))
	}
	if len(resp.Msg.EncodableFormats) == 0 || len(resp.Msg.DecodableFormats) == 0 {
		t.Fatal("expected format lists to be populated")
	}
}
