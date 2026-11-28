package ai

import (
	"bytes"
	"context"
	"errors"
	"mime/multipart"
	"net/http"
	"net/http/httptest"
	"testing"

	internalai "image-tools/internal/ai"
	"image-tools/internal/backends"
	"image-tools/internal/capabilities"
	"image-tools/internal/models"
	"image-tools/internal/safety"
	"image-tools/internal/storage"

	"github.com/gorilla/mux"
	"github.com/vrooli/api-core/blobstore"
)

type cloudProvider struct{ fakeProvider }

func (cloudProvider) IsCloud() bool { return true }

type stubFaces struct {
	n     int
	err   error
	calls int
}

func (s *stubFaces) CountFaces(context.Context, [][]byte) (int, error) {
	s.calls++
	return s.n, s.err
}

// newCloudCaptionServer wires the submit edge so the only caption route is a hosted
// vision row, with the given face counter behind the consent gate.
func newCloudCaptionServer(t *testing.T, counter safety.FaceCounter) (*mux.Router, *fakeSubmitter) {
	t.Helper()
	registry, err := models.Load()
	if err != nil {
		t.Fatalf("load registry: %v", err)
	}
	backendReg := backends.New()
	if err := backendReg.Register(&cloudProvider{fakeProvider{name: "openrouter", ops: []string{"caption"}}}); err != nil {
		t.Fatal(err)
	}
	store := storage.NewWithBlobStore(blobstore.NewMemoryBlobStore(), t.TempDir())
	eng, err := internalai.NewEngineWithRightsEvaluator(internalai.Deps{
		Registry:       registry,
		Backends:       backendReg,
		Probe:          capabilities.FakeProbe{Host: capabilities.Host{OS: "linux", Arch: "amd64", Cores: 8}},
		Store:          store,
		ModelInstalled: func(string) bool { return true },
		ModelsRoot:     t.TempDir(),
	}, allowModelRightsForHandlerTest)
	if err != nil {
		t.Fatalf("new engine: %v", err)
	}
	sub := &fakeSubmitter{}
	gate := safety.NewGate(safety.TierLocal, nil)
	gate.SetFaceCounter(counter)
	deps := &Deps{Engine: eng, Store: store, Jobs: sub, Guard: storage.DefaultGuard(), Gate: gate}
	r := mux.NewRouter()
	r.HandleFunc("/api/v1/ai/{operation}", deps.submitHandler).Methods(http.MethodPost)
	return r, sub
}

func cloudCaptionSubmit(t *testing.T, consent bool) *http.Request {
	t.Helper()
	var body bytes.Buffer
	mw := multipart.NewWriter(&body)
	part, err := mw.CreateFormFile("file", "photo.png")
	if err != nil {
		t.Fatal(err)
	}
	_, _ = part.Write([]byte("synthetic image bytes"))
	params := `{"model_override":"gemini-flash-lite-vision","allow_byok":true}`
	if consent {
		params = `{"model_override":"gemini-flash-lite-vision","allow_byok":true,"consent_affirmed":true}`
	}
	if err := mw.WriteField("params", params); err != nil {
		t.Fatal(err)
	}
	if err := mw.Close(); err != nil {
		t.Fatal(err)
	}
	req := httptest.NewRequest(http.MethodPost, "/api/v1/ai/caption", &body)
	req.Header.Set("Content-Type", mw.FormDataContentType())
	return req
}

// A caption sent to a provider asks for consent only when the image shows people:
// the picture of a guinea pig goes through, the portrait does not, and the job
// records which it was.
func TestCloudCaptionAsksForConsentOnlyWhenTheImageShowsPeople(t *testing.T) {
	t.Run("nobody shown: admitted without consent and the basis is recorded", func(t *testing.T) {
		faces := &stubFaces{}
		r, sub := newCloudCaptionServer(t, faces)
		w := httptest.NewRecorder()
		r.ServeHTTP(w, cloudCaptionSubmit(t, false))
		if w.Code != http.StatusAccepted {
			t.Fatalf("status = %d, want 202; body=%s", w.Code, w.Body.String())
		}
		if !contains(string(sub.last.Payload), `"consent_basis":"faces_detected=0"`) || !contains(string(sub.last.Payload), `"consent_waived":true`) {
			t.Errorf("payload must record the people check; got %s", sub.last.Payload)
		}
		if faces.calls != 1 {
			t.Errorf("the people check ran %d times for one request, want once", faces.calls)
		}
	})
	t.Run("a person is shown: refused without consent", func(t *testing.T) {
		r, sub := newCloudCaptionServer(t, &stubFaces{n: 1})
		w := httptest.NewRecorder()
		r.ServeHTTP(w, cloudCaptionSubmit(t, false))
		if w.Code != http.StatusForbidden || !contains(w.Body.String(), "consent") {
			t.Fatalf("status = %d body=%s, want a 403 naming consent", w.Code, w.Body.String())
		}
		if sub.calls != 0 {
			t.Error("a refused request must not enqueue a job")
		}
	})
	t.Run("a person is shown: admitted with consent", func(t *testing.T) {
		r, sub := newCloudCaptionServer(t, &stubFaces{n: 2})
		w := httptest.NewRecorder()
		r.ServeHTTP(w, cloudCaptionSubmit(t, true))
		if w.Code != http.StatusAccepted {
			t.Fatalf("status = %d, want 202; body=%s", w.Code, w.Body.String())
		}
		if !contains(string(sub.last.Payload), `"consent_basis":"faces_detected=2"`) || contains(string(sub.last.Payload), `"consent_waived"`) {
			t.Errorf("an affirmed request records the faces and is not waived; got %s", sub.last.Payload)
		}
	})
	t.Run("the check fails: refused without consent (fail closed)", func(t *testing.T) {
		r, _ := newCloudCaptionServer(t, &stubFaces{err: errors.New("detector crashed")})
		w := httptest.NewRecorder()
		r.ServeHTTP(w, cloudCaptionSubmit(t, false))
		if w.Code != http.StatusForbidden || !contains(w.Body.String(), "could not run") {
			t.Fatalf("status = %d body=%s, want a 403 saying the check could not run", w.Code, w.Body.String())
		}
	})
	t.Run("no detector configured: refused without consent", func(t *testing.T) {
		r, _ := newCloudCaptionServer(t, nil)
		w := httptest.NewRecorder()
		r.ServeHTTP(w, cloudCaptionSubmit(t, false))
		if w.Code != http.StatusForbidden {
			t.Fatalf("status = %d, want 403; body=%s", w.Code, w.Body.String())
		}
	})
}
