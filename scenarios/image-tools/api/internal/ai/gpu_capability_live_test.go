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

// TestGPUCapabilityDoctorRejectsCPUOnlyBackendsOnGPUHosts is part of the
// image-tools Test Genie unit phase. On a machine with an NVIDIA GPU it checks
// the same installed binaries and Python runtime that the serving API probes;
// CPU-only or missing GPU paths are test failures. Hosts without NVIDIA
// hardware keep the ordinary unit suite portable.
func TestGPUCapabilityDoctorRejectsCPUOnlyBackendsOnGPUHosts(t *testing.T) {
	if _, err := exec.LookPath("nvidia-smi"); err != nil {
		t.Skip("NVIDIA GPU is not available on this host")
	}
	ctx, cancel := context.WithTimeout(context.Background(), 90*time.Second)
	defer cancel()
	if output, err := exec.CommandContext(ctx, "nvidia-smi", "-L").CombinedOutput(); err != nil || !strings.Contains(string(output), "GPU") {
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
	report := providerRegistry.DoctorForModels(ctx, modelRegistry.Models())
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
}
