package ai

import (
	"bytes"
	"context"
	"encoding/json"
	"mime/multipart"
	"net/http"
	"net/http/httptest"
	"os"
	"testing"

	internalai "image-tools/internal/ai"
	"image-tools/internal/backends"
	"image-tools/internal/capabilities"
	internaljobs "image-tools/internal/jobs"
	"image-tools/internal/models"
	"image-tools/internal/storage"

	"github.com/gorilla/mux"
	"github.com/vrooli/api-core/blobstore"
	"google.golang.org/protobuf/encoding/protojson"

	aiv1 "github.com/vrooli/vrooli/packages/proto/gen/go/image-tools/v1/ai"
)

// fakeProvider is a standalone provider that writes a fixed output.
type fakeProvider struct {
	name string
	ops  []string
}

func (p *fakeProvider) Name() string                   { return p.name }
func (p *fakeProvider) Operations() []string           { return p.ops }
func (p *fakeProvider) Standalone() bool               { return true }
func (p *fakeProvider) IsCloud() bool                  { return false }
func (p *fakeProvider) Available(context.Context) bool { return true }
func (p *fakeProvider) Execute(_ context.Context, req backends.Request) (backends.Result, error) {
	_ = os.WriteFile(req.Output.LocalPath, []byte("out"), 0o600)
	return backends.Result{OutputRef: req.Output.LocalPath}, nil
}

// fakeSubmitter records submitted specs and returns a synthetic job.
type fakeSubmitter struct {
	last internaljobs.Spec
}

func (f *fakeSubmitter) Submit(_ context.Context, spec internaljobs.Spec) (internaljobs.Job, error) {
	f.last = spec
	return internaljobs.Job{ID: "job-1", Operation: spec.Operation, State: internaljobs.StateQueued}, nil
}

func newTestServer(t *testing.T, installed bool) (*mux.Router, *fakeSubmitter) {
	t.Helper()
	registry, err := models.Load()
	if err != nil {
		t.Fatalf("load registry: %v", err)
	}
	def, _ := registry.DefaultFor("text_to_image")
	backendReg := backends.New()
	if err := backendReg.Register(&fakeProvider{name: def.Backend, ops: []string{"text_to_image"}}); err != nil {
		t.Fatalf("register provider: %v", err)
	}
	store := storage.NewWithBlobStore(blobstore.NewMemoryBlobStore(), t.TempDir())
	eng, err := internalai.NewEngine(internalai.Deps{
		Registry:       registry,
		Backends:       backendReg,
		Probe:          capabilities.FakeProbe{Host: capabilities.Host{OS: "linux", Arch: "amd64", Cores: 8}},
		Store:          store,
		ModelInstalled: func(string) bool { return installed },
		ModelsRoot:     t.TempDir(),
	})
	if err != nil {
		t.Fatalf("new engine: %v", err)
	}
	sub := &fakeSubmitter{}
	deps := &Deps{Engine: eng, Store: store, Jobs: sub, Guard: storage.DefaultGuard()}
	r := mux.NewRouter()
	r.HandleFunc("/api/v1/ai/{operation}", deps.submitHandler).Methods(http.MethodPost)
	return r, sub
}

func TestSubmit_TextToImage_Accepted(t *testing.T) {
	r, sub := newTestServer(t, true)
	req := newMultipartSubmit(t, "text_to_image", `{"prompt":"a cat","width":512}`)
	w := httptest.NewRecorder()
	r.ServeHTTP(w, req)

	if w.Code != http.StatusAccepted {
		t.Fatalf("status = %d, want 202; body=%s", w.Code, w.Body.String())
	}
	var resp aiv1.SubmitAIResponse
	if err := protojson.Unmarshal(w.Body.Bytes(), &resp); err != nil {
		t.Fatalf("decode response: %v", err)
	}
	if resp.JobId != "job-1" {
		t.Errorf("job id = %q, want job-1", resp.JobId)
	}
	if resp.ModelId == "" || resp.Tier == "" {
		t.Errorf("expected model/tier in response, got %+v", &resp)
	}
	if sub.last.Operation != "text_to_image" {
		t.Errorf("submitted op = %q", sub.last.Operation)
	}
	if sub.last.Lane != internaljobs.LaneGPU {
		t.Errorf("AI ops should submit on the GPU lane, got %q", sub.last.Lane)
	}
	var payload internalai.Payload
	if err := json.Unmarshal(sub.last.Payload, &payload); err != nil {
		t.Fatalf("decode submitted payload: %v", err)
	}
	if payload.Backend == "" || payload.Tier == "" {
		t.Fatalf("payload should carry trace backend/tier metadata: %+v", payload)
	}
}

func TestSubmit_CarriesOpenRouterRoleParam(t *testing.T) {
	r, sub := newTestServer(t, true)
	req := newMultipartSubmit(t, "text_to_image", `{"prompt":"a cat","openrouterRole":"image.generate.logo"}`)
	w := httptest.NewRecorder()
	r.ServeHTTP(w, req)

	if w.Code != http.StatusAccepted {
		t.Fatalf("status = %d, want 202; body=%s", w.Code, w.Body.String())
	}
	var payload internalai.Payload
	if err := json.Unmarshal(sub.last.Payload, &payload); err != nil {
		t.Fatalf("decode submitted payload: %v", err)
	}
	if got := payload.Params["openrouter_role"]; got != "image.generate.logo" {
		t.Fatalf("openrouter_role = %q", got)
	}
}

func TestSubmit_ModelNotInstalled_Conflict(t *testing.T) {
	r, _ := newTestServer(t, false)
	req := newMultipartSubmit(t, "text_to_image", `{"prompt":"x"}`)
	w := httptest.NewRecorder()
	r.ServeHTTP(w, req)
	if w.Code != http.StatusConflict {
		t.Fatalf("status = %d, want 409; body=%s", w.Code, w.Body.String())
	}
}

func TestSubmit_UnknownOperation_NotFound(t *testing.T) {
	r, _ := newTestServer(t, true)
	req := newMultipartSubmit(t, "nope", `{}`)
	w := httptest.NewRecorder()
	r.ServeHTTP(w, req)
	if w.Code != http.StatusNotFound {
		t.Fatalf("status = %d, want 404", w.Code)
	}
}

// newMultipartSubmit builds a POST to /api/v1/ai/{op} carrying only the params
// part (the generation ops under test need no input image).
func newMultipartSubmit(t *testing.T, op, params string) *http.Request {
	t.Helper()
	var body bytes.Buffer
	mw := multipart.NewWriter(&body)
	if err := mw.WriteField("params", params); err != nil {
		t.Fatalf("write params: %v", err)
	}
	if err := mw.Close(); err != nil {
		t.Fatalf("close multipart: %v", err)
	}
	req := httptest.NewRequest(http.MethodPost, "/api/v1/ai/"+op, &body)
	req.Header.Set("Content-Type", mw.FormDataContentType())
	return req
}
