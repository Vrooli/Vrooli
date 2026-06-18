package backends

import (
	"context"
	"errors"
	"strings"
	"testing"
)

// fakeProvider is a configurable Provider for table tests.
type fakeProvider struct {
	name       string
	ops        []string
	standalone bool
	cloud      bool
	available  bool
}

func (f fakeProvider) Name() string                   { return f.name }
func (f fakeProvider) Operations() []string           { return f.ops }
func (f fakeProvider) Standalone() bool               { return f.standalone }
func (f fakeProvider) IsCloud() bool                  { return f.cloud }
func (f fakeProvider) Available(context.Context) bool { return f.available }
func (f fakeProvider) Execute(context.Context, Request) (Result, error) {
	return Result{OutputRef: "out", Tier: TierLocalCPU}, nil
}

// cpuOnlyProvider is a standalone backend that cannot use the GPU (e.g. the
// onnxruntime sidecar bound to CPUExecutionProvider). It implements the optional
// GPUCapable() capability returning false.
type cpuOnlyProvider struct{ fakeProvider }

func (cpuOnlyProvider) GPUCapable() bool { return false }

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
	_ = r.Register(fakeProvider{name: "sd.cpp", ops: []string{"generate"}, standalone: true, available: false})
	if _, err := r.SelectProvider(context.Background(), SelectRequest{Operation: "generate"}); !errors.Is(err, ErrNoneAvailable) {
		t.Fatalf("want ErrNoneAvailable, got %v", err)
	}
}
