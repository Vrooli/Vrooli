package ai

import (
	"context"
	"errors"
	"strings"
	"testing"

	"image-tools/internal/backends"
	"image-tools/internal/models"
	"image-tools/internal/storage"
	"image-tools/internal/technique"
)

// TestScanStableDiffusionProgress checks that only the sd-cli SAMPLING bar
// (iteration-rate suffix) drives progress, while the model-load / vae-decode
// byte bars and noise lines are ignored.
func TestScanStableDiffusionProgress(t *testing.T) {
	cases := []struct {
		name     string
		line     string
		wantOK   bool
		wantFrac float64
	}{
		{"sampler mid", "  |============>      | 3/20 - 3.58s/it", true, 0.15},
		{"sampler done", "  |####################| 20/20 - 3.58s/it", true, 1.0},
		{"sampler it/s", "  |=====>              | 1/4 - 2.0it/s", true, 0.25},
		{"tensor byte bar", "  |####| 196/196 - 2.28GB/s", false, 0},
		{"vae byte bar", "  |####| 140/140 - 939.25MB/s", false, 0},
		{"info line", "[INFO ] stable-diffusion.cpp:4432 - generate_image 512x512", false, 0},
		{"empty", "", false, 0},
		{"clamp over-total", "  |####| 21/20 - 3.0s/it", true, 1.0},
	}
	for _, c := range cases {
		t.Run(c.name, func(t *testing.T) {
			frac, msg, ok := scanStableDiffusionProgress(c.line)
			if ok != c.wantOK {
				t.Fatalf("ok=%v want %v (line=%q)", ok, c.wantOK, c.line)
			}
			if ok && frac != c.wantFrac {
				t.Fatalf("frac=%v want %v (line=%q)", frac, c.wantFrac, c.line)
			}
			if ok && msg == "" {
				t.Fatalf("expected non-empty message for %q", c.line)
			}
		})
	}
}

// TestExecProvider_StreamsProgress verifies that when a Request carries a
// Progress sink and the provider has a progressScan, Execute streams parsed
// progress fractions through it (instead of running silently to completion).
func TestExecProvider_StreamsProgress(t *testing.T) {
	lines := []string{
		"[INFO ] loading model",
		"  |#####| 196/196 - 2.28GB/s", // ignored (byte bar)
		"  |==>  | 1/4 - 3.0s/it",
		"  |#### | 2/4 - 3.0s/it",
		"  |#####| 4/4 - 3.0s/it",
	}
	p := &execProvider{
		name:         "stable-diffusion.cpp",
		program:      "sd",
		techniques:   technique.Single("text_to_image", technique.StableDiffusionCpp),
		progressScan: scanStableDiffusionProgress,
		stream: func(_ context.Context, _ string, _ []string, onLine func(string)) error {
			for _, l := range lines {
				onLine(l)
			}
			return nil
		},
	}
	var got []float64
	req := backends.Request{
		Operation: "text_to_image",
		Model:     models.Model{ID: "sd-1.5"},
		ModelDir:  "/models/sd-1.5",
		Output:    storage.OutputTarget{LocalPath: "/out.png"},
		Params:    map[string]string{"prompt": "a cat"},
		Progress:  func(frac float64, _ string) { got = append(got, frac) },
	}
	res, err := p.Execute(context.Background(), req)
	if err != nil {
		t.Fatalf("execute: %v", err)
	}
	want := []float64{0.25, 0.5, 1.0}
	if len(got) != len(want) {
		t.Fatalf("progress updates=%v want %v", got, want)
	}
	for i := range want {
		if got[i] != want[i] {
			t.Fatalf("progress[%d]=%v want %v (all=%v)", i, got[i], want[i], got)
		}
	}
	if res.Meta["runner"] != "stream" {
		t.Fatalf("expected stream runner metadata, got %+v", res.Meta)
	}
}

// TestExecProvider_StreamErrorPropagates ensures a non-zero exit from the
// streaming backend surfaces as an execution failure.
func TestExecProvider_StreamErrorPropagates(t *testing.T) {
	p := &execProvider{
		name:         "stable-diffusion.cpp",
		program:      "sd",
		techniques:   technique.Single("text_to_image", technique.StableDiffusionCpp),
		progressScan: scanStableDiffusionProgress,
		stream: func(context.Context, string, []string, func(string)) error {
			return errors.New("sd boom")
		},
	}
	req := backends.Request{
		Operation: "text_to_image",
		Model:     models.Model{ID: "sd-1.5"},
		ModelDir:  "/models/sd-1.5",
		Output:    storage.OutputTarget{LocalPath: "/out.png"},
		Params:    map[string]string{"prompt": "a cat"},
		Progress:  func(float64, string) {},
	}
	if _, err := p.Execute(context.Background(), req); err == nil || !strings.Contains(err.Error(), "execution failed") {
		t.Fatalf("expected execution failed error, got %v", err)
	}
}

// TestExecProvider_NoProgressUsesPlainRunner confirms backward compatibility:
// with no Progress sink, Execute uses the plain run path (not the stream path),
// so existing callers/tests are unaffected.
func TestExecProvider_NoProgressUsesPlainRunner(t *testing.T) {
	var plainCalls, streamCalls int
	p := &execProvider{
		name:         "stable-diffusion.cpp",
		program:      "sd",
		techniques:   technique.Single("text_to_image", technique.StableDiffusionCpp),
		progressScan: scanStableDiffusionProgress,
		run: func(context.Context, string, []string) error {
			plainCalls++
			return nil
		},
		stream: func(context.Context, string, []string, func(string)) error {
			streamCalls++
			return nil
		},
	}
	req := backends.Request{
		Operation: "text_to_image",
		Model:     models.Model{ID: "sd-1.5"},
		ModelDir:  "/models/sd-1.5",
		Output:    storage.OutputTarget{LocalPath: "/out.png"},
		Params:    map[string]string{"prompt": "a cat"},
		// Progress nil
	}
	if _, err := p.Execute(context.Background(), req); err != nil {
		t.Fatalf("execute: %v", err)
	}
	if plainCalls != 1 || streamCalls != 0 {
		t.Fatalf("plain=%d stream=%d want 1/0", plainCalls, streamCalls)
	}
}

// TestExecProvider_GPUCapableProbe verifies the install probe overrides the
// static type capability: a GPU-capable backend TYPE whose installed binary has
// no GPU backend (probe=false) is reported CPU-only, so the selector labels its
// runs local-cpu instead of lying with "Running on your GPU".
func TestExecProvider_GPUCapableProbe(t *testing.T) {
	mk := func(typeCap bool, probe func(context.Context, string) bool) *execProvider {
		return &execProvider{
			name:       "stable-diffusion.cpp",
			program:    "sd",
			gpuCapable: typeCap,
			gpuProbe:   probe,
			lookPath:   func(string) (string, error) { return "/usr/bin/sd", nil },
		}
	}
	var probeCalls int
	cpuOnly := mk(true, func(context.Context, string) bool { probeCalls++; return false })
	if cpuOnly.GPUCapable() {
		t.Fatal("expected CPU-only install to report GPUCapable=false")
	}
	// Cached: a second call must not re-probe.
	_ = cpuOnly.GPUCapable()
	if probeCalls != 1 {
		t.Fatalf("probe called %d times, want 1 (cached)", probeCalls)
	}

	gpuBuild := mk(true, func(context.Context, string) bool { return true })
	if !gpuBuild.GPUCapable() {
		t.Fatal("expected GPU build to report GPUCapable=true")
	}

	// A type that is not GPU-capable never probes.
	cpuType := mk(false, func(context.Context, string) bool { t.Fatal("should not probe a non-GPU type"); return true })
	if cpuType.GPUCapable() {
		t.Fatal("non-GPU type must report false")
	}

	// Not installed (lookPath fails) → cannot use a GPU.
	notInstalled := mk(true, func(context.Context, string) bool { return true })
	notInstalled.lookPath = func(string) (string, error) { return "", errors.New("not found") }
	if notInstalled.GPUCapable() {
		t.Fatal("uninstalled backend must report GPUCapable=false")
	}
}
