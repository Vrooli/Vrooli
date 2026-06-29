package backends

import (
	"context"
	"errors"
	"strings"
	"testing"

	"image-tools/internal/models"
)

// fakeProvider is a configurable Provider for table tests.
type fakeProvider struct {
	name       string
	ops        []string
	standalone bool
	cloud      bool
	available  bool
	detail     string
	provision  string
}

func (f fakeProvider) Name() string                   { return f.name }
func (f fakeProvider) Operations() []string           { return f.ops }
func (f fakeProvider) Standalone() bool               { return f.standalone }
func (f fakeProvider) IsCloud() bool                  { return f.cloud }
func (f fakeProvider) Available(context.Context) bool { return f.available }
func (f fakeProvider) Availability(context.Context) Availability {
	return Availability{Available: f.available, Detail: f.detail, Provision: f.provision}
}

func (f fakeProvider) Execute(context.Context, Request) (Result, error) {
	return Result{OutputRef: "out", Tier: TierLocalCPU}, nil
}

// cpuOnlyProvider is a standalone backend that cannot use the GPU (e.g. the
// onnxruntime sidecar bound to CPUExecutionProvider). It implements the optional
// GPUCapable() capability returning false.
type cpuOnlyProvider struct{ fakeProvider }

func (cpuOnlyProvider) GPUCapable() bool { return false }

// adapterProvider is a standalone backend that advertises conditioning-adapter
// support (the diffusers sidecar). It implements the optional SupportsAdapters
// capability returning true for its ops.
type adapterProvider struct{ fakeProvider }

func (adapterProvider) SupportsAdapters(string) bool { return true }

// TestSelectRequireAdaptersRoutesToCapableBackend proves a conditioned request
// (RequireAdapters) skips the model's native backend when it cannot apply
// adapters (stable-diffusion.cpp) and selects the adapter-capable one (diffusers)
// with an explanatory warning — rather than letting the request reach a backend
// that would reject the modifiers.
func TestSelectRequireAdaptersRoutesToCapableBackend(t *testing.T) {
	ctx := context.Background()
	r := New()
	_ = r.Register(fakeProvider{name: "stable-diffusion.cpp", ops: []string{"text_to_image"}, standalone: true, available: true})
	_ = r.Register(adapterProvider{fakeProvider{name: "diffusers", ops: []string{"text_to_image"}, standalone: true, available: true}})

	// Without adapters, the native backend is preferred.
	sel, err := r.SelectProvider(ctx, SelectRequest{Operation: "text_to_image", ModelBackend: "stable-diffusion.cpp", GPUViable: true})
	if err != nil {
		t.Fatalf("select (no adapters): %v", err)
	}
	if sel.Provider.Name() != "stable-diffusion.cpp" {
		t.Fatalf("no-adapter select = %q, want stable-diffusion.cpp", sel.Provider.Name())
	}

	// With adapters required, selection routes to the diffusers backend.
	sel, err = r.SelectProvider(ctx, SelectRequest{Operation: "text_to_image", ModelBackend: "stable-diffusion.cpp", GPUViable: true, RequireAdapters: true})
	if err != nil {
		t.Fatalf("select (adapters): %v", err)
	}
	if sel.Provider.Name() != "diffusers" {
		t.Fatalf("adapter select = %q, want diffusers", sel.Provider.Name())
	}
	if !strings.Contains(strings.Join(sel.Warnings, " "), "conditioning adapters require") {
		t.Fatalf("expected a conditioning-routing warning, got %v", sel.Warnings)
	}
}

// TestSelectRequireAdaptersFailsWhenNoneCapable proves a conditioned request
// fails closed (no silent drop) when no available backend can apply adapters.
func TestSelectRequireAdaptersFailsWhenNoneCapable(t *testing.T) {
	ctx := context.Background()
	r := New()
	_ = r.Register(fakeProvider{name: "stable-diffusion.cpp", ops: []string{"text_to_image"}, standalone: true, available: true})

	if _, err := r.SelectProvider(ctx, SelectRequest{Operation: "text_to_image", ModelBackend: "stable-diffusion.cpp", GPUViable: true, RequireAdapters: true}); err == nil {
		t.Fatal("expected an error when no adapter-capable backend is available, got nil")
	}
}

// TestSelectCPUOnlyBackendNeverLabelsGPU proves the honest-tier fix: a CPU-only
// backend reports local-cpu even when the host has a GPU-viable path, with a
// warning, so the tier label never overstates where the op ran.
func TestSelectCPUOnlyBackendNeverLabelsGPU(t *testing.T) {
	ctx := context.Background()
	r := New()
	_ = r.Register(cpuOnlyProvider{fakeProvider{name: "onnxruntime", ops: []string{"background_removal"}, standalone: true, available: true}})

	sel, err := r.SelectProvider(ctx, SelectRequest{Operation: "background_removal", ModelBackend: "onnxruntime", GPUViable: true})
	if err != nil {
		t.Fatalf("select: %v", err)
	}
	if sel.Tier != TierLocalCPU {
		t.Fatalf("CPU-only backend must report local-cpu even on a GPU host, got %s", sel.Tier)
	}
	if len(sel.Warnings) == 0 || !strings.Contains(strings.Join(sel.Warnings, " "), "CPU-only") {
		t.Fatalf("expected a CPU-only warning, got %v", sel.Warnings)
	}
}

func TestRegisterRejectsNoOps(t *testing.T) {
	r := New()
	if err := r.Register(fakeProvider{name: "x"}); !errors.Is(err, ErrNoOperations) {
		t.Fatalf("want ErrNoOperations, got %v", err)
	}
}

func TestValidateEnforcesStandalone(t *testing.T) {
	// upscale has a standalone provider; generate has only ComfyUI → invalid.
	r := New()
	_ = r.Register(fakeProvider{name: "real-esrgan", ops: []string{"upscale"}, standalone: true, available: true})
	_ = r.Register(fakeProvider{name: "comfyui", ops: []string{"generate"}, standalone: false, available: true})

	err := r.Validate()
	if !errors.Is(err, ErrMissingStandalone) {
		t.Fatalf("want ErrMissingStandalone, got %v", err)
	}
	if !strings.Contains(err.Error(), "generate") {
		t.Fatalf("error should name the bad op: %v", err)
	}

	// Add a standalone generate backend → now valid.
	_ = r.Register(fakeProvider{name: "sd.cpp", ops: []string{"generate"}, standalone: true, available: true})
	if err := r.Validate(); err != nil {
		t.Fatalf("expected valid, got %v", err)
	}
}

func TestValidateCloudDoesNotSatisfyStandalone(t *testing.T) {
	r := New()
	// A cloud provider is standalone in the "non-ComfyUI" sense but must NOT
	// satisfy the headless tenet (it needs a key + network).
	_ = r.Register(fakeProvider{name: "openai", ops: []string{"generate"}, standalone: true, cloud: true, available: true})
	if err := r.Validate(); !errors.Is(err, ErrMissingStandalone) {
		t.Fatalf("cloud should not satisfy standalone, got %v", err)
	}
}

func TestSelectPrefersMatchingLocalBackend(t *testing.T) {
	ctx := context.Background()
	r := New()
	_ = r.Register(fakeProvider{name: "sd.cpp", ops: []string{"generate"}, standalone: true, available: true})
	_ = r.Register(fakeProvider{name: "other", ops: []string{"generate"}, standalone: true, available: true})

	sel, err := r.SelectProvider(ctx, SelectRequest{Operation: "generate", ModelBackend: "sd.cpp", GPUViable: true})
	if err != nil {
		t.Fatal(err)
	}
	if sel.Provider.Name() != "sd.cpp" || sel.Tier != TierLocalGPU {
		t.Fatalf("got %s/%s", sel.Provider.Name(), sel.Tier)
	}
	if len(sel.Warnings) != 0 {
		t.Fatalf("clean GPU match should have no warnings: %v", sel.Warnings)
	}
}

func TestSelectSubstitutesWhenNativeUnavailable(t *testing.T) {
	ctx := context.Background()
	r := New()
	_ = r.Register(fakeProvider{name: "sd.cpp", ops: []string{"generate"}, standalone: true, available: false})
	_ = r.Register(fakeProvider{name: "diffusers", ops: []string{"generate"}, standalone: true, available: true})

	sel, err := r.SelectProvider(ctx, SelectRequest{Operation: "generate", ModelBackend: "sd.cpp", GPUViable: false})
	if err != nil {
		t.Fatal(err)
	}
	if sel.Provider.Name() != "diffusers" || sel.Tier != TierLocalCPU {
		t.Fatalf("got %s/%s", sel.Provider.Name(), sel.Tier)
	}
	var sawSub, sawCPU bool
	for _, w := range sel.Warnings {
		if strings.Contains(w, "substituting") {
			sawSub = true
		}
		if strings.Contains(w, "CPU") {
			sawCPU = true
		}
	}
	if !sawSub || !sawCPU {
		t.Fatalf("expected substitution+CPU warnings, got %v", sel.Warnings)
	}
}

func TestSelectFallsBackToComfyUI(t *testing.T) {
	ctx := context.Background()
	r := New()
	_ = r.Register(fakeProvider{name: "sd.cpp", ops: []string{"generate"}, standalone: true, available: false})
	_ = r.Register(fakeProvider{name: "comfyui", ops: []string{"generate"}, standalone: false, available: true})

	sel, err := r.SelectProvider(ctx, SelectRequest{Operation: "generate", ModelBackend: "sd.cpp", GPUViable: true})
	if err != nil {
		t.Fatal(err)
	}
	if sel.Provider.Name() != "comfyui" {
		t.Fatalf("got %s", sel.Provider.Name())
	}
	if !strings.Contains(strings.Join(sel.Warnings, " "), "ComfyUI") {
		t.Fatalf("expected ComfyUI fallback warning, got %v", sel.Warnings)
	}
}

func TestSelectBYOKGatedByFlag(t *testing.T) {
	ctx := context.Background()
	r := New()
	_ = r.Register(fakeProvider{name: "sd.cpp", ops: []string{"generate"}, standalone: true, available: false})
	_ = r.Register(fakeProvider{name: "openai", ops: []string{"generate"}, standalone: true, cloud: true, available: true})

	// BYOK disabled → refuse.
	if _, err := r.SelectProvider(ctx, SelectRequest{Operation: "generate", AllowBYOK: false}); !errors.Is(err, ErrNoneAvailable) {
		t.Fatalf("want ErrNoneAvailable when BYOK off, got %v", err)
	}
	// BYOK enabled → choose cloud with cost warning.
	sel, err := r.SelectProvider(ctx, SelectRequest{Operation: "generate", AllowBYOK: true})
	if err != nil {
		t.Fatal(err)
	}
	if sel.Provider.Name() != "openai" || sel.Tier != TierBYOK {
		t.Fatalf("got %s/%s", sel.Provider.Name(), sel.Tier)
	}
	if !strings.Contains(strings.Join(sel.Warnings, " "), "cost") {
		t.Fatalf("expected cost warning, got %v", sel.Warnings)
	}
}

func TestSelectNoProvider(t *testing.T) {
	r := New()
	if _, err := r.SelectProvider(context.Background(), SelectRequest{Operation: "ghost"}); !errors.Is(err, ErrNoProvider) {
		t.Fatalf("want ErrNoProvider, got %v", err)
	}
}

func TestSelectNoneAvailable(t *testing.T) {
	r := New()
	_ = r.Register(fakeProvider{name: "sd.cpp", ops: []string{"generate"}, standalone: true, available: false, detail: "sd missing", provision: "install sd"})
	_, err := r.SelectProvider(context.Background(), SelectRequest{Operation: "generate"})
	if !errors.Is(err, ErrNoneAvailable) {
		t.Fatalf("want ErrNoneAvailable, got %v", err)
	}
	if !strings.Contains(err.Error(), "sd missing") || !strings.Contains(err.Error(), "install sd") {
		t.Fatalf("error should include availability detail and provision guidance, got %v", err)
	}
}

func TestDoctorReportsBackendAvailability(t *testing.T) {
	r := New()
	_ = r.Register(fakeProvider{name: "onnxruntime", ops: []string{"background_removal", "denoise"}, standalone: true, available: true, detail: "python ready", provision: "install python deps"})
	_ = r.Register(fakeProvider{name: "sd.cpp", ops: []string{"text_to_image"}, standalone: true, available: false, detail: "sd missing", provision: "install sd"})
	_ = r.Register(fakeProvider{name: "openai", ops: []string{"text_to_image"}, standalone: true, cloud: true, available: false, detail: "api key missing", provision: "set API key"})

	report := r.Doctor(context.Background())
	if report.OK {
		t.Fatalf("missing local backend should make doctor red")
	}
	if len(report.Backends) != 3 {
		t.Fatalf("got %d backend rows, want 3", len(report.Backends))
	}
	var sawONNX, sawSD, sawCloud bool
	for _, b := range report.Backends {
		switch b.Name {
		case "onnxruntime":
			sawONNX = true
			if !b.Available || !b.Standalone || b.Cloud || len(b.Operations) != 2 {
				t.Fatalf("bad onnxruntime row: %+v", b)
			}
		case "sd.cpp":
			sawSD = true
			if b.Available || b.Detail != "sd missing" || b.Provision != "install sd" {
				t.Fatalf("bad sd row: %+v", b)
			}
		case "openai":
			sawCloud = true
			if !b.Cloud {
				t.Fatalf("cloud row should be marked cloud: %+v", b)
			}
		}
	}
	if !sawONNX || !sawSD || !sawCloud {
		t.Fatalf("missing expected rows: onnx=%t sd=%t cloud=%t", sawONNX, sawSD, sawCloud)
	}
}

func TestDoctorForModelsReportsDeclaredButUnregisteredBackends(t *testing.T) {
	r := New()
	_ = r.Register(fakeProvider{name: "builtin", ops: []string{"naturalize"}, standalone: true, available: true, detail: "built in", provision: "none"})

	report := r.DoctorForModels(context.Background(), []models.Model{
		{ID: "naturalize-detail-v1", Backend: models.BackendBuiltin, Enabled: true, Operations: []string{"naturalize"}},
		{ID: "caption-default", Backend: "llama.cpp", Enabled: true, Operations: []string{"caption"}},
		{ID: "disabled-gap", Backend: "python-sidecar", Enabled: false, Operations: []string{"face_restore"}},
	})
	if report.OK {
		t.Fatalf("declared but unregistered local backend should make doctor red")
	}
	var sawBuiltin, sawLlama, sawDisabled bool
	for _, b := range report.Backends {
		switch b.Name {
		case "builtin":
			sawBuiltin = true
			if !b.Available {
				t.Fatalf("registered builtin row should stay available: %+v", b)
			}
		case "llama.cpp":
			sawLlama = true
			if b.Available || !strings.Contains(b.Detail, "no runtime provider") || !strings.Contains(strings.Join(b.Operations, ","), "caption") {
				t.Fatalf("bad llama.cpp gap row: %+v", b)
			}
		case "python-sidecar":
			sawDisabled = true
		}
	}
	if !sawBuiltin || !sawLlama {
		t.Fatalf("missing expected rows: %+v", report.Backends)
	}
	if sawDisabled {
		t.Fatalf("disabled catalog models must not create backend doctor gaps: %+v", report.Backends)
	}
}

func TestDoctorKeepsSameBackendDistinctProviderRows(t *testing.T) {
	r := New()
	_ = r.Register(fakeProvider{name: "library-cgo", ops: []string{"ocr"}, standalone: true, available: false, detail: "tesseract missing", provision: "install tesseract"})
	_ = r.Register(fakeProvider{name: "library-cgo", ops: []string{"face_detection"}, standalone: true, available: true, detail: "opencv ready", provision: "opencv present"})

	report := r.DoctorForModels(context.Background(), []models.Model{
		{ID: "tesseract", Backend: "library-cgo", Enabled: true, Operations: []string{"ocr"}},
		{ID: "yunet", Backend: "library-cgo", Enabled: true, Operations: []string{"face_detection"}},
	})
	if report.OK {
		t.Fatalf("missing tesseract should keep report red: %+v", report.Backends)
	}
	var sawOCR, sawFace bool
	for _, b := range report.Backends {
		if b.Name != "library-cgo" {
			continue
		}
		ops := strings.Join(b.Operations, ",")
		switch ops {
		case "ocr":
			sawOCR = true
			if b.Available || b.Detail != "tesseract missing" {
				t.Fatalf("bad OCR row: %+v", b)
			}
		case "face_detection":
			sawFace = true
			if !b.Available || b.Detail != "opencv ready" {
				t.Fatalf("bad face_detection row: %+v", b)
			}
		default:
			t.Fatalf("unexpected merged library-cgo row: %+v", b)
		}
	}
	if !sawOCR || !sawFace {
		t.Fatalf("missing distinct provider rows: %+v", report.Backends)
	}
}
