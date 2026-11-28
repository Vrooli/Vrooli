package ai

import (
	"context"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"testing"
	"time"

	"image-tools/internal/backends"
	"image-tools/internal/models"
)

const (
	gpuInventoryTimeout = 30 * time.Second
	gpuDoctorTimeout    = 5 * time.Minute
)

// TestGPUCapabilityDoctorRejectsCPUOnlyBackendsOnGPUHosts is part of the
// image-tools Test Genie unit phase. On a machine with an NVIDIA GPU it checks
// the same installed binaries and Python runtime that the serving API probes;
// CPU-only or missing GPU paths are test failures. Hosts without NVIDIA
// hardware keep the ordinary unit suite portable.
//
// The probing is real: each Python-backed provider imports its runtime (torch,
// onnxruntime) in a child process, about 15 s in total on an idle host and over
// 50 s when the host is saturated by other work. Time therefore bounds only a hang:
// the inventory call and the doctor probing each get a generous ceiling, and
// running into it fails with the elapsed time instead of reporting the
// half-probed backends as "unavailable". A genuine CPU-only or missing GPU path
// returns well inside the bound and still fails on its own message.
func TestGPUCapabilityDoctorRejectsCPUOnlyBackendsOnGPUHosts(t *testing.T) {
	if _, err := exec.LookPath("nvidia-smi"); err != nil {
		t.Skip("NVIDIA GPU is not available on this host")
	}
	inventoryCtx, cancelInventory := context.WithTimeout(context.Background(), gpuInventoryTimeout)
	defer cancelInventory()
	if output, err := exec.CommandContext(inventoryCtx, "nvidia-smi", "-L").CombinedOutput(); err != nil || !strings.Contains(string(output), "GPU") {
		t.Skipf("NVIDIA GPU inventory is unavailable: %v: %s", err, strings.TrimSpace(string(output)))
	}

	home, err := os.UserHomeDir()
	if err != nil {
		t.Fatalf("resolve user home: %v", err)
	}
	python := filepath.Join(home, ".vrooli", "cache", "vrooli", "image-tools", "pyenv", "bin", "python")
	if _, err := os.Stat(python); err != nil {
		t.Fatalf("GPU host has no provisioned Image Tools Python runtime at %s: %v", python, err)
	}

	modelRegistry, err := models.Load()
	if err != nil {
		t.Fatalf("load model catalog: %v", err)
	}
	providerRegistry := backends.New()
	if err := RegisterProviders(providerRegistry, exec.LookPath, nil, python); err != nil {
		t.Fatalf("register installed backends: %v", err)
	}
	defer providerRegistry.Close()
	ctx, cancel := context.WithTimeout(context.Background(), gpuDoctorTimeout)
	defer cancel()
	started := time.Now()
	report := providerRegistry.DoctorForModels(ctx, modelRegistry.Models())
	if ctx.Err() != nil {
		t.Fatalf("backend doctor did not finish within %s (%s elapsed): a probe hung or the host is starved; the per-backend results below it are not trustworthy", gpuDoctorTimeout, time.Since(started).Round(time.Second))
	}
	assertGPUOperations := func(backend string, want ...string) {
		t.Helper()
		for _, operation := range want {
			found, available, gpuReady := false, false, false
			detail := ""
			for _, row := range report.Backends {
				if row.Name != backend {
					continue
				}
				found = true
				available = available || row.Available
				detail = row.Detail
				for _, gpuOperation := range row.GPUOperations {
					if gpuOperation == operation {
						gpuReady = true
						break
					}
				}
			}
			if !found {
				t.Errorf("doctor did not report backend %s for operation %s", backend, operation)
				continue
			}
			if !available {
				t.Errorf("%s is unavailable for GPU-capable operation %s: %s", backend, operation, detail)
			}
			if !gpuReady {
				t.Errorf("%s operation %s is not GPU-capable: %s", backend, operation, detail)
			}
		}
	}
	assertGPUOperations("stable-diffusion.cpp", "text_to_image")
	assertGPUOperations("onnxruntime", "background_removal", "segment")
	assertGPUOperations("diffusers", "text_to_image", "image_to_image", "inpaint", "outpaint", "background_replace", "edit_instruct")
	assertGPUOperations("sam2-transformers", "segment")
	assertGPUOperations("triposr", "image_to_object")
	assertGPUOperations("florence-transformers", "caption", "object_detection")
}
