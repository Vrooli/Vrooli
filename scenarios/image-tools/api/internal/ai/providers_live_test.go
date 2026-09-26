package ai

import (
	"context"
	"os"
	"os/exec"
	"path/filepath"
	"testing"
	"time"

	"image-tools/internal/backends"
	"image-tools/internal/models"
	"image-tools/internal/storage"
	"image-tools/internal/technique"
)

// TestLive_StableDiffusion exercises the real installed sd-cli end to end:
// the GPU-capability probe and the streaming progress runner. It is skipped
// unless IMGTOOLS_SD_LIVE=1 and an `sd` binary resolves on PATH, so CI/unit
// runs stay hermetic. Run with:
//
//	IMGTOOLS_SD_LIVE=1 go test ./internal/ai/ -run TestLive_StableDiffusion -v
func TestLive_StableDiffusion(t *testing.T) {
	if os.Getenv("IMGTOOLS_SD_LIVE") != "1" {
		t.Skip("set IMGTOOLS_SD_LIVE=1 to run the live sd-cli check")
	}
	sd, err := exec.LookPath("sd")
	if err != nil {
		t.Skipf("sd not on PATH: %v", err)
	}

	// 1) GPU probe reports the truth about the installed binary (not a static lie).
	gpu := probeStableDiffusionGPU(context.Background(), sd)
	t.Logf("probeStableDiffusionGPU(%s) = %v", sd, gpu)

	// 2) Streaming run surfaces live sampling progress.
	model := os.Getenv("IMGTOOLS_SD_MODEL")
	if model == "" {
		t.Skip("set IMGTOOLS_SD_MODEL to a .safetensors path for the streaming check")
	}
	out := filepath.Join(t.TempDir(), "live-out.png")
	p := &execProvider{
		name:         "stable-diffusion.cpp",
		program:      sd,
		techniques:   technique.Single("text_to_image", technique.StableDiffusionCpp),
		progressScan: scanStableDiffusionProgress,
		lookPath:     exec.LookPath,
	}
	var fracs []float64
	req := backends.Request{
		Operation: "text_to_image",
		Model:     models.Model{ID: "sd-1.5"},
		ModelDir:  filepath.Dir(model),
		Output:    storage.OutputTarget{LocalPath: out},
		Params:    map[string]string{"prompt": "a red apple on a table", "steps": "4", "width": "256", "height": "256"},
		Progress: func(frac float64, msg string) {
			fracs = append(fracs, frac)
			t.Logf("progress %.2f  %s", frac, msg)
		},
	}
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Minute)
	defer cancel()
	if _, err := p.Execute(ctx, req); err != nil {
		t.Fatalf("execute: %v", err)
	}
	if len(fracs) < 2 {
		t.Fatalf("expected multiple live progress updates, got %v", fracs)
	}
	if last := fracs[len(fracs)-1]; last < 1.0 {
		t.Fatalf("expected progress to reach 1.0, last=%v (all=%v)", last, fracs)
	}
	if fi, err := os.Stat(out); err != nil || fi.Size() == 0 {
		t.Fatalf("expected a non-empty output image at %s: %v", out, err)
	}
}
