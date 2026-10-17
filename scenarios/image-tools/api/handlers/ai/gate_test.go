package ai

import (
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

// newGatedServer builds the AI submit edge wired with a Responsible-Use gate for
// the given tier. Models are reported installed so the gate (not a missing
// model) is what decides the outcome.
func newGatedServer(t *testing.T, tier safety.Tier) (*mux.Router, *fakeSubmitter) {
	t.Helper()
	registry, err := models.Load()
	if err != nil {
		t.Fatalf("load registry: %v", err)
	}
	backendReg := backends.New()
	// Register a provider for the ops the gate tests exercise.
	if err := backendReg.Register(&fakeProvider{name: "stable-diffusion.cpp", ops: []string{"text_to_image", "inpaint", "image_to_image"}}); err != nil {
		t.Fatalf("register provider: %v", err)
	}
	store := storage.NewWithBlobStore(blobstore.NewMemoryBlobStore(), t.TempDir())
	eng, err := internalai.NewEngine(internalai.Deps{
		Registry:       registry,
		Backends:       backendReg,
		Probe:          capabilities.FakeProbe{Host: capabilities.Host{OS: "linux", Arch: "amd64", Cores: 8}},
		Store:          store,
		ModelInstalled: func(string) bool { return true },
		ModelsRoot:     t.TempDir(),
	})
	if err != nil {
		t.Fatalf("new engine: %v", err)
	}
	sub := &fakeSubmitter{}
	deps := &Deps{Engine: eng, Store: store, Jobs: sub, Guard: storage.DefaultGuard(), Gate: safety.NewGate(tier, nil)}
	r := mux.NewRouter()
	r.HandleFunc("/api/v1/ai/{operation}", deps.submitHandler).Methods(http.MethodPost)
	return r, sub
}

func TestGate_PublicBlocksHighWeightWithoutConsent(t *testing.T) {
	r, _ := newGatedServer(t, safety.TierPublic)
	// inpaint is a high-weight op; no consent_affirmed → blocked at the gate.
	req := newMultipartSubmit(t, "inpaint", `{"prompt":"change the shirt"}`)
	w := httptest.NewRecorder()
	r.ServeHTTP(w, req)
	if w.Code != http.StatusForbidden {
		t.Fatalf("status = %d, want 403; body=%s", w.Code, w.Body.String())
	}
	if body := w.Body.String(); body == "" || !contains(body, "consent") {
		t.Errorf("403 body should mention consent; got %s", body)
	}
}

func TestGate_PublicAllowsLowWeight(t *testing.T) {
	r, _ := newGatedServer(t, safety.TierPublic)
	// text_to_image is none-weight; the gate never blocks it. It needs no image,
	// so it should reach the accepted path.
	req := newMultipartSubmit(t, "text_to_image", `{"prompt":"a landscape"}`)
	w := httptest.NewRecorder()
	r.ServeHTTP(w, req)
	if w.Code != http.StatusAccepted {
		t.Fatalf("status = %d, want 202 (gate must not block none-weight); body=%s", w.Code, w.Body.String())
	}
}

func TestGate_LocalNeverBlocks(t *testing.T) {
	r, _ := newGatedServer(t, safety.TierLocal)
	// On local, a high-weight op without consent is NOT blocked by the gate; it
	// proceeds to the normal pipeline (here inpaint needs image+mask, so it fails
	// validation — but NOT with a 403 from the gate).
	req := newMultipartSubmit(t, "inpaint", `{"prompt":"x"}`)
	w := httptest.NewRecorder()
	r.ServeHTTP(w, req)
	if w.Code == http.StatusForbidden {
		t.Fatalf("local tier must not gate identity-altering ops; got 403: %s", w.Body.String())
	}
}

func contains(s, sub string) bool {
	for i := 0; i+len(sub) <= len(s); i++ {
		if s[i:i+len(sub)] == sub {
			return true
		}
	}
	return false
}
