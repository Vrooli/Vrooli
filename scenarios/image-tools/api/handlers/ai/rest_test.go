package ai

import (
	"bytes"
	"context"
	"encoding/json"
	"io"
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

	"github.com/google/uuid"
	"github.com/gorilla/mux"
	"github.com/vrooli/api-core/blobstore"
	"google.golang.org/protobuf/encoding/protojson"

	aiv1 "github.com/vrooli/vrooli/packages/proto/gen/go/image-tools/v1/ai"
	aiconnect "github.com/vrooli/vrooli/packages/proto/gen/go/image-tools/v1/ai/ai_v1connect"
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
	last  internaljobs.Spec
	calls int
}

func (f *fakeSubmitter) Submit(_ context.Context, spec internaljobs.Spec) (internaljobs.Job, error) {
	f.last = spec
	f.calls++
	return internaljobs.Job{ID: "job-1", Operation: spec.Operation, State: internaljobs.StateQueued, Payload: spec.Payload, EstimatedSeconds: spec.EstimatedSeconds, ClientOperationID: spec.ClientOperationID, RequestFingerprint: spec.RequestFingerprint}, nil
}

func (f *fakeSubmitter) GetByClientOperation(_ context.Context, id string) (internaljobs.Job, error) {
	if id != f.last.ClientOperationID || id == "" {
		return internaljobs.Job{}, internaljobs.ErrNotFound
	}
	return internaljobs.Job{ID: "job-1", Operation: f.last.Operation, State: internaljobs.StateQueued, Payload: f.last.Payload, EstimatedSeconds: f.last.EstimatedSeconds, ClientOperationID: id, RequestFingerprint: f.last.RequestFingerprint}, nil
}

func TestSubmitImageOperationRetryUsesSourceBytesAndSurvivesModelUnload(t *testing.T) {
	r, sub, store := newTestServerWithStore(t, true)
	for _, key := range []string{"input/original", "input/same-bytes-other-key"} {
		if err := store.Put(context.Background(), key, bytes.NewReader([]byte("same oriented image bytes")), "image/png"); err != nil {
			t.Fatal(err)
		}
	}
	if err := store.Put(context.Background(), "input/changed", bytes.NewReader([]byte("different image")), "image/png"); err != nil {
		t.Fatal(err)
	}
	operationID := uuid.NewString()
	request := func(input, prompt string) *http.Request {
		var body bytes.Buffer
		mw := multipart.NewWriter(&body)
		if err := mw.WriteField("input_ref", input); err != nil {
			t.Fatal(err)
		}
		if err := mw.WriteField("params", prompt); err != nil {
			t.Fatal(err)
		}
		if err := mw.Close(); err != nil {
			t.Fatal(err)
		}
		req := httptest.NewRequest(http.MethodPost, "/api/v1/ai/segment", &body)
		req.Header.Set("Content-Type", mw.FormDataContentType())
		req.Header.Set("Idempotency-Key", operationID)
		return req
	}
	const params = `{"modelOverride":"mobilesam","segmentation":{"points":[{"x":0.5,"y":0.4}]}}`
	for _, input := range []string{"input/original", "input/same-bytes-other-key"} {
		w := httptest.NewRecorder()
		r.ServeHTTP(w, request(input, params))
		if w.Code != http.StatusAccepted {
			t.Fatalf("retry=%d %s", w.Code, w.Body.String())
		}
	}
	if sub.calls != 1 {
		t.Fatalf("same input submitted %d inferences", sub.calls)
	}
	for _, req := range []*http.Request{
		request("input/changed", params),
		request("input/original", `{"modelOverride":"mobilesam","segmentation":{"points":[{"x":0.7,"y":0.4}]}}`),
	} {
		w := httptest.NewRecorder()
		r.ServeHTTP(w, req)
		if w.Code != http.StatusConflict {
			t.Fatalf("changed closure reused operation: %d %s", w.Code, w.Body.String())
		}
	}
	offline, retained, offlineStore := newTestServerWithStore(t, false)
	retained.last = sub.last
	if err := offlineStore.Put(context.Background(), "input/original", bytes.NewReader([]byte("same oriented image bytes")), "image/png"); err != nil {
		t.Fatal(err)
	}
	w := httptest.NewRecorder()
	offline.ServeHTTP(w, request("input/original", params))
	if w.Code != http.StatusAccepted || retained.calls != 0 || !bytes.Contains(w.Body.Bytes(), []byte("mobilesam")) {
		t.Fatalf("model unload blocked recovery: %d %s, new inferences=%d", w.Code, w.Body.String(), retained.calls)
	}
}

func newTestServer(t *testing.T, installed bool) (*mux.Router, *fakeSubmitter) {
	r, sub, _ := newTestServerWithStore(t, installed)
	return r, sub
}

func newTestServerWithStore(t *testing.T, installed bool) (*mux.Router, *fakeSubmitter, *storage.Store) {
	t.Helper()
	registry, err := models.Load()
	if err != nil {
		t.Fatalf("load registry: %v", err)
	}
	def, _ := registry.DefaultFor("text_to_image")
	backendReg := backends.New()
	if err := backendReg.Register(&fakeProvider{name: def.Backend, ops: []string{"text_to_image", "edit_instruct"}}); err != nil {
		t.Fatalf("register provider: %v", err)
	}
	if err := backendReg.Register(&fakeProvider{name: "onnxruntime", ops: []string{"segment"}}); err != nil {
		t.Fatal(err)
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
	path, handler := aiconnect.NewAIServiceHandler(NewConnectHandler(registry, deps))
	r.PathPrefix(path).Handler(handler)
	r.HandleFunc("/api/v1/ai/{operation}", deps.submitHandler).Methods(http.MethodPost)
	r.HandleFunc("/api/v1/ai/operations/{operation_id}", deps.operationHandler).Methods(http.MethodGet)
	return r, sub, store
}

func TestSubmit_ClientOperationKeyCanBeReadBack(t *testing.T) {
	r, sub := newTestServer(t, true)
	key := uuid.NewString()
	req := newMultipartSubmit(t, "text_to_image", `{"prompt":"a cat","width":512,"height":512,"allowByok":false,"fallbackPolicy":"local_only"}`)
	req.Header.Set("Idempotency-Key", key)
	w := httptest.NewRecorder()
	r.ServeHTTP(w, req)
	if w.Code != http.StatusAccepted {
		t.Fatalf("submit status=%d body=%s", w.Code, w.Body.String())
	}
	if sub.last.ClientOperationID != key || sub.last.RequestFingerprint == "" || sub.last.ExecutionTimeoutSeconds != 30*60 {
		t.Fatalf("idempotency metadata missing: %+v", sub.last)
	}
	read := httptest.NewRecorder()
	r.ServeHTTP(read, httptest.NewRequest(http.MethodGet, "/api/v1/ai/operations/"+key, nil))
	if read.Code != http.StatusOK || !bytes.Contains(read.Body.Bytes(), []byte(`"job_id":"job-1"`)) {
		t.Fatalf("readback status=%d body=%s", read.Code, read.Body.String())
	}
}

func TestSubmit_PromptedSegmentationRetainsTypedCoordinates(t *testing.T) {
	r, sub, store := newTestServerWithStore(t, true)
	if err := store.Put(context.Background(), "input/device.png", bytes.NewReader([]byte("image")), "image/png"); err != nil {
		t.Fatal(err)
	}
	var body bytes.Buffer
	mw := multipart.NewWriter(&body)
	mw.WriteField("input_ref", "input/device.png")
	mw.WriteField("params", `{"modelOverride":"mobilesam","segmentation":{"points":[{"x":0.72,"y":0.4},{"x":0.1,"y":0.2,"negative":true}],"box":{"x":0.2,"y":0.1,"width":0.6,"height":0.8}}}`)
	mw.Close()
	req := httptest.NewRequest(http.MethodPost, "/api/v1/ai/segment", &body)
	req.Header.Set("Content-Type", mw.FormDataContentType())
	w := httptest.NewRecorder()
	r.ServeHTTP(w, req)
	if w.Code != http.StatusAccepted {
		t.Fatalf("submit=%d %s", w.Code, w.Body.String())
	}
	var payload internalai.Payload
	if err := json.Unmarshal(sub.last.Payload, &payload); err != nil {
		t.Fatal(err)
	}
	var prompt aiv1.SegmentationPrompt
	if err := protojson.Unmarshal([]byte(payload.Params["segmentation"]), &prompt); err != nil {
		t.Fatal(err)
	}
	if len(prompt.Points) != 2 || !prompt.Points[1].Negative || prompt.Points[0].X != 0.72 || prompt.Box.Width != 0.6 || payload.ModelID != "mobilesam" {
		t.Fatalf("prompt/model lost: %+v %+v", prompt, payload)
	}
	// All local model jobs serialize to bound resident model memory, even when
	// inference is CPU-only. The tier and GPU flag describe actual execution.
	if sub.last.Lane != internaljobs.LaneGPU || payload.GPU {
		t.Fatalf("wrong execution lane: %s", sub.last.Lane)
	}
}

func TestSubmit_InvalidSegmentationDoesNotQueue(t *testing.T) {
	for _, raw := range []string{`{}`, `{"segmentation":{}}`, `{"segmentation":{"points":[{"x":0.5,"y":0.5,"negative":true}]}}`, `{"segmentation":{"points":[{"x":"NaN","y":0.5}]}}`, `{"segmentation":{"box":{"x":0.9,"y":0,"width":0.2,"height":1}}}`} {
		t.Run(raw, func(t *testing.T) {
			r, sub := newTestServer(t, true)
			w := httptest.NewRecorder()
			r.ServeHTTP(w, newMultipartSubmit(t, "segment", raw))
			if w.Code != http.StatusBadRequest || sub.last.Operation != "" {
				t.Fatalf("invalid prompt admitted: %d %s", w.Code, w.Body.String())
			}
		})
	}
}

func TestSubmit_EditUsesExistingInputReferenceWithoutUploadingBytes(t *testing.T) {
	r, sub, store := newTestServerWithStore(t, true)
	if err := store.Put(context.Background(), "outputs/parent.png", bytes.NewReader([]byte("png")), "image/png"); err != nil {
		t.Fatal(err)
	}
	var body bytes.Buffer
	mw := multipart.NewWriter(&body)
	_ = mw.WriteField("params", `{"prompt":"brighten"}`)
	_ = mw.WriteField("input_ref", "outputs/parent.png")
	if err := mw.Close(); err != nil {
		t.Fatal(err)
	}
	req := httptest.NewRequest(http.MethodPost, "/api/v1/ai/edit_instruct", &body)
	req.Header.Set("Content-Type", mw.FormDataContentType())
	w := httptest.NewRecorder()
	r.ServeHTTP(w, req)
	if w.Code != http.StatusAccepted {
		t.Fatalf("status=%d body=%s", w.Code, w.Body.String())
	}
	var payload internalai.Payload
	if err := json.Unmarshal(sub.last.Payload, &payload); err != nil {
		t.Fatal(err)
	}
	if payload.InputKey != "outputs/parent.png" {
		t.Fatalf("input key=%q", payload.InputKey)
	}
	reader, _, err := store.Get(context.Background(), "outputs/parent.png")
	if err != nil {
		t.Fatal(err)
	}
	_, _ = io.ReadAll(reader)
	_ = reader.Close()
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

func TestOperationReadRejectsCorruptSavedMetadata(t *testing.T) {
	for _, raw := range []string{"{", "null", "{}", `{"operation":"caption","model_id":"model"}`} {
		t.Run(raw, func(t *testing.T) {
			r, sub := newTestServer(t, true)
			key := uuid.NewString()
			sub.last = internaljobs.Spec{ClientOperationID: key, Operation: "segment", Payload: []byte(raw)}
			w := httptest.NewRecorder()
			r.ServeHTTP(w, httptest.NewRequest(http.MethodGet, "/api/v1/ai/operations/"+key, nil))
			if w.Code != http.StatusInternalServerError {
				t.Fatalf("corrupt metadata returned success: %d %s", w.Code, w.Body.String())
			}
		})
	}
}
