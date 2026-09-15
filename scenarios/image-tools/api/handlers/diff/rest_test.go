package diff

import (
	"bytes"
	"image"
	"image/color"
	"image/png"
	"mime/multipart"
	"net/http"
	"net/http/httptest"
	"testing"

	internaljobs "image-tools/internal/jobs"
	"image-tools/internal/storage"

	"github.com/gorilla/mux"
	"github.com/vrooli/api-core/blobstore"
	"google.golang.org/protobuf/encoding/protojson"

	diffv1 "github.com/vrooli/vrooli/packages/proto/gen/go/image-tools/v1/diff"
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
	deps := &Deps{Store: store, Jobs: rec, Guard: storage.DefaultGuard()}
	r := mux.NewRouter()
	r.HandleFunc("/api/v1/diff/compare", deps.compareHandler).Methods(http.MethodPost)
	return r, rec, store
}

func solidPNG(t *testing.T, c color.NRGBA) []byte {
	t.Helper()
	img := image.NewNRGBA(image.Rect(0, 0, 40, 40))
	for y := 0; y < 40; y++ {
		for x := 0; x < 40; x++ {
			img.SetNRGBA(x, y, c)
		}
	}
	var buf bytes.Buffer
	if err := png.Encode(&buf, img); err != nil {
		t.Fatalf("encode png: %v", err)
	}
	return buf.Bytes()
}

func postCompare(t *testing.T, r *mux.Router, base, cmp []byte, params string, omitCompare bool) *httptest.ResponseRecorder {
	t.Helper()
	var body bytes.Buffer
	mw := multipart.NewWriter(&body)
	if base != nil {
		fw, _ := mw.CreateFormFile("base", "base.png")
		_, _ = fw.Write(base)
	}
	if !omitCompare {
		fw, _ := mw.CreateFormFile("compare", "compare.png")
		_, _ = fw.Write(cmp)
	}
	if params != "" {
		_ = mw.WriteField("params", params)
	}
	_ = mw.Close()
	req := httptest.NewRequest(http.MethodPost, "/api/v1/diff/compare", &body)
	req.Header.Set("Content-Type", mw.FormDataContentType())
	w := httptest.NewRecorder()
	r.ServeHTTP(w, req)
	return w
}

func TestCompareEndToEnd(t *testing.T) {
	r, rec, store := newTestServer(t)
	base := solidPNG(t, color.NRGBA{R: 10, G: 10, B: 10, A: 255})
	other := solidPNG(t, color.NRGBA{R: 250, G: 250, B: 250, A: 255})
	w := postCompare(t, r, base, other, "", false)
	if w.Code != http.StatusOK {
		t.Fatalf("status = %d, want 200; body=%s", w.Code, w.Body.String())
	}
	var resp diffv1.DiffResult
	if err := protojson.Unmarshal(w.Body.Bytes(), &resp); err != nil {
		t.Fatalf("unmarshal: %v", err)
	}
	if resp.GetVerdict() != "different" {
		t.Errorf("verdict = %q, want different", resp.GetVerdict())
	}
	if resp.GetJobId() == "" {
		t.Error("expected a job id")
	}
	if resp.GetHeatmapRef() == "" {
		t.Error("expected a heat-map ref (default on)")
	}
	// The heat-map blob should have been stored.
	rc, _, err := store.Get(t.Context(), resp.GetHeatmapRef())
	if err != nil {
		t.Errorf("heat-map blob not stored: %v", err)
	} else {
		_ = rc.Close()
	}
	if len(rec.jobs) != 1 || rec.jobs[0].Operation != compareOp {
		t.Errorf("expected one recorded %q job, got %+v", compareOp, rec.jobs)
	}
}

func TestCompareMetricsOnlyOptOut(t *testing.T) {
	r, _, _ := newTestServer(t)
	img := solidPNG(t, color.NRGBA{R: 100, G: 100, B: 100, A: 255})
	w := postCompare(t, r, img, img, `{"includeHeatmap":false}`, false)
	if w.Code != http.StatusOK {
		t.Fatalf("status = %d, want 200; body=%s", w.Code, w.Body.String())
	}
	var resp diffv1.DiffResult
	if err := protojson.Unmarshal(w.Body.Bytes(), &resp); err != nil {
		t.Fatalf("unmarshal: %v", err)
	}
	if resp.GetHeatmapRef() != "" {
		t.Errorf("heatmap_ref = %q, want empty when opted out", resp.GetHeatmapRef())
	}
	if resp.GetVerdict() != "identical" {
		t.Errorf("verdict = %q, want identical", resp.GetVerdict())
	}
}

func TestCompareMissingPart(t *testing.T) {
	r, _, _ := newTestServer(t)
	img := solidPNG(t, color.NRGBA{R: 100, G: 100, B: 100, A: 255})
	w := postCompare(t, r, img, nil, "", true)
	if w.Code != http.StatusBadRequest {
		t.Fatalf("status = %d, want 400 for missing compare part; body=%s", w.Code, w.Body.String())
	}
}

func TestCompareUndecodable(t *testing.T) {
	r, _, _ := newTestServer(t)
	img := solidPNG(t, color.NRGBA{R: 100, G: 100, B: 100, A: 255})
	w := postCompare(t, r, img, []byte("not-an-image-at-all"), "", false)
	if w.Code != http.StatusUnprocessableEntity {
		t.Fatalf("status = %d, want 422 for undecodable compare; body=%s", w.Code, w.Body.String())
	}
}

func TestCompareBadParams(t *testing.T) {
	r, _, _ := newTestServer(t)
	img := solidPNG(t, color.NRGBA{R: 100, G: 100, B: 100, A: 255})
	w := postCompare(t, r, img, img, `{bad json`, false)
	if w.Code != http.StatusBadRequest {
		t.Fatalf("status = %d, want 400 for bad params; body=%s", w.Code, w.Body.String())
	}
}
