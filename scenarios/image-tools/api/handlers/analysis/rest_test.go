package analysis

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

	internalanalysis "image-tools/internal/analysis"
	internaljobs "image-tools/internal/jobs"
	"image-tools/internal/storage"

	"github.com/gorilla/mux"
	"google.golang.org/protobuf/encoding/protojson"

	analysisv1 "github.com/vrooli/vrooli/packages/proto/gen/go/image-tools/v1/analysis"
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

func newTestServer(t *testing.T, svc *internalanalysis.Service) (*mux.Router, *fakeRecorder) {
	t.Helper()
	rec := &fakeRecorder{}
	deps := &Deps{Service: svc, Jobs: rec, Guard: storage.DefaultGuard()}
	r := mux.NewRouter()
	r.HandleFunc("/api/v1/analysis/{operation}", deps.analyzeHandler).Methods(http.MethodPost)
	return r, rec
}

func pngBytes(t *testing.T, w, h int) []byte {
	t.Helper()
	img := image.NewRGBA(image.Rect(0, 0, w, h))
	for y := 0; y < h; y++ {
		for x := 0; x < w; x++ {
			img.Set(x, y, color.RGBA{R: 10, G: 120, B: 200, A: 255})
		}
	}
	var buf bytes.Buffer
	if err := png.Encode(&buf, img); err != nil {
		t.Fatalf("encode png: %v", err)
	}
	return buf.Bytes()
}

func postImage(t *testing.T, r *mux.Router, op string, img []byte) *httptest.ResponseRecorder {
	t.Helper()
	var body bytes.Buffer
	mw := multipart.NewWriter(&body)
	fw, _ := mw.CreateFormFile("file", "in.png")
	_, _ = fw.Write(img)
	_ = mw.Close()
	req := httptest.NewRequest(http.MethodPost, "/api/v1/analysis/"+op, &body)
	req.Header.Set("Content-Type", mw.FormDataContentType())
	w := httptest.NewRecorder()
	r.ServeHTTP(w, req)
	return w
}

func mustService(t *testing.T, cfg internalanalysis.Config) *internalanalysis.Service {
	t.Helper()
	svc, err := internalanalysis.NewService(cfg)
	if err != nil {
		t.Fatalf("new service: %v", err)
	}
	return svc
}

// TestAnalyze_Probe drives the always-available pure-Go probe end to end.
func TestAnalyze_Probe(t *testing.T) {
	svc := mustService(t, internalanalysis.Config{ModelInstalled: func(string) bool { return true }})
	r, rec := newTestServer(t, svc)
	w := postImage(t, r, "probe", pngBytes(t, 64, 48))
	if w.Code != http.StatusOK {
		t.Fatalf("status = %d, want 200; body=%s", w.Code, w.Body.String())
	}
	var resp analysisv1.AnalyzeResponse
	if err := protojson.Unmarshal(w.Body.Bytes(), &resp); err != nil {
		t.Fatalf("decode: %v", err)
	}
	probe := resp.GetProbe()
	if probe == nil {
		t.Fatal("expected a probe result")
	}
	if probe.Width != 64 || probe.Height != 48 {
		t.Errorf("dimensions = %dx%d, want 64x48", probe.Width, probe.Height)
	}
	if resp.JobId == "" || len(rec.jobs) != 1 {
		t.Errorf("expected a recorded job, got id=%q jobs=%d", resp.JobId, len(rec.jobs))
	}
}

// TestAnalyze_OCRUnavailable returns 503 when the backend is absent.
func TestAnalyze_OCRUnavailable(t *testing.T) {
	svc := mustService(t, internalanalysis.Config{
		ModelInstalled: func(string) bool { return true },
		LookPath:       func(string) (string, error) { return "", context.Canceled },
	})
	r, _ := newTestServer(t, svc)
	w := postImage(t, r, "ocr", pngBytes(t, 8, 8))
	if w.Code != http.StatusServiceUnavailable {
		t.Fatalf("status = %d, want 503; body=%s", w.Code, w.Body.String())
	}
}

// TestAnalyze_UnknownOperation returns 404.
func TestAnalyze_UnknownOperation(t *testing.T) {
	svc := mustService(t, internalanalysis.Config{ModelInstalled: func(string) bool { return true }})
	r, _ := newTestServer(t, svc)
	w := postImage(t, r, "bogus", pngBytes(t, 8, 8))
	if w.Code != http.StatusNotFound {
		t.Fatalf("status = %d, want 404", w.Code)
	}
}
