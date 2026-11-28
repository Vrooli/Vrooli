package selection

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
	internalselection "image-tools/internal/selection"
	"image-tools/internal/storage"

	"github.com/gorilla/mux"
	"github.com/vrooli/api-core/blobstore"
	"google.golang.org/protobuf/encoding/protojson"

	selectionv1 "github.com/vrooli/vrooli/packages/proto/gen/go/image-tools/v1/selection"
)

type fakeRecorder struct{ jobs []internaljobs.Job }

func (f *fakeRecorder) Record(spec internaljobs.Spec, ref string, runErr error) (internaljobs.Job, error) {
	state := internaljobs.StateSucceeded
	if runErr != nil {
		state = internaljobs.StateFailed
	}
	j := internaljobs.Job{ID: "job-" + spec.Operation, Operation: spec.Operation, State: state, ResultRef: ref}
	f.jobs = append(f.jobs, j)
	return j, nil
}

func newTestServer(t *testing.T) (*mux.Router, *fakeRecorder, *storage.Store) {
	t.Helper()
	rec := &fakeRecorder{}
	store := storage.NewWithBlobStore(blobstore.NewMemoryBlobStore(), t.TempDir())
	deps := &Deps{Service: internalselection.NewService(), Store: store, Jobs: rec, Guard: storage.DefaultGuard()}
	r := mux.NewRouter()
	r.HandleFunc("/api/v1/selection/segment", deps.segmentHandler).Methods(http.MethodPost)
	return r, rec, store
}

// rectPNG paints a red square on a blue canvas and PNG-encodes it.
func rectPNG(t *testing.T) []byte {
	t.Helper()
	img := image.NewNRGBA(image.Rect(0, 0, 100, 100))
	for y := 0; y < 100; y++ {
		for x := 0; x < 100; x++ {
			c := color.NRGBA{R: 30, G: 60, B: 220, A: 255}
			if x >= 30 && x < 70 && y >= 30 && y < 70 {
				c = color.NRGBA{R: 220, G: 20, B: 20, A: 255}
			}
			img.SetNRGBA(x, y, c)
		}
	}
	var buf bytes.Buffer
	if err := png.Encode(&buf, img); err != nil {
		t.Fatalf("encode png: %v", err)
	}
	return buf.Bytes()
}

func postSegment(t *testing.T, r *mux.Router, img []byte, params string) *httptest.ResponseRecorder {
	t.Helper()
	var body bytes.Buffer
	mw := multipart.NewWriter(&body)
	fw, _ := mw.CreateFormFile("file", "in.png")
	_, _ = fw.Write(img)
	if params != "" {
		_ = mw.WriteField("params", params)
	}
	_ = mw.Close()
	req := httptest.NewRequest(http.MethodPost, "/api/v1/selection/segment", &body)
	req.Header.Set("Content-Type", mw.FormDataContentType())
	w := httptest.NewRecorder()
	r.ServeHTTP(w, req)
	return w
}

func TestSegmentPointEndToEnd(t *testing.T) {
	r, rec, store := newTestServer(t)
	w := postSegment(t, r, rectPNG(t), `{"mode":"SEGMENT_MODE_POINT","points":[{"x":0.5,"y":0.5}]}`)
	if w.Code != http.StatusOK {
		t.Fatalf("status = %d, want 200; body=%s", w.Code, w.Body.String())
	}
	var resp selectionv1.SegmentResult
	if err := protojson.Unmarshal(w.Body.Bytes(), &resp); err != nil {
		t.Fatalf("decode: %v", err)
	}
	if resp.MaskRef == "" {
		t.Fatal("no mask_ref returned")
	}
	if resp.Tier != internalselection.TierBuiltinCPU {
		t.Errorf("tier = %q, want builtin-cpu", resp.Tier)
	}
	if resp.RegionClass == "" || len(resp.SuggestedEdits) == 0 {
		t.Errorf("expected class + edits, got class=%q edits=%d", resp.RegionClass, len(resp.SuggestedEdits))
	}
	if resp.JobId == "" || len(rec.jobs) != 1 {
		t.Errorf("expected a recorded job, got id=%q jobs=%d", resp.JobId, len(rec.jobs))
	}
	// The mask blob must be fetchable and be a valid PNG.
	rc, _, err := store.Get(context.Background(), resp.MaskRef)
	if err != nil {
		t.Fatalf("fetch mask blob: %v", err)
	}
	defer func() { _ = rc.Close() }()
	if _, err := png.Decode(rc); err != nil {
		t.Fatalf("mask blob is not a valid PNG: %v", err)
	}
}

func TestSegmentAutoNoParams(t *testing.T) {
	r, _, _ := newTestServer(t)
	// No params at all → defaults to AUTO mode (no seed required).
	w := postSegment(t, r, rectPNG(t), "")
	if w.Code != http.StatusOK {
		t.Fatalf("status = %d, want 200; body=%s", w.Code, w.Body.String())
	}
}

func TestSegmentMissingImage(t *testing.T) {
	r, _, _ := newTestServer(t)
	var body bytes.Buffer
	mw := multipart.NewWriter(&body)
	_ = mw.WriteField("params", `{"mode":"SEGMENT_MODE_AUTO"}`)
	_ = mw.Close()
	req := httptest.NewRequest(http.MethodPost, "/api/v1/selection/segment", &body)
	req.Header.Set("Content-Type", mw.FormDataContentType())
	w := httptest.NewRecorder()
	r.ServeHTTP(w, req)
	if w.Code != http.StatusBadRequest {
		t.Fatalf("status = %d, want 400", w.Code)
	}
}

func TestSegmentInvalidParams(t *testing.T) {
	r, _, _ := newTestServer(t)
	w := postSegment(t, r, rectPNG(t), `{not json}`)
	if w.Code != http.StatusBadRequest {
		t.Fatalf("status = %d, want 400; body=%s", w.Code, w.Body.String())
	}
}

func TestSegmentPointModeMissingSeed(t *testing.T) {
	r, _, _ := newTestServer(t)
	// Explicit point mode but no points → 422 (invalid mode/seed).
	w := postSegment(t, r, rectPNG(t), `{"mode":"SEGMENT_MODE_POINT"}`)
	if w.Code != http.StatusUnprocessableEntity {
		t.Fatalf("status = %d, want 422; body=%s", w.Code, w.Body.String())
	}
}
