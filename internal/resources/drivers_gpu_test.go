package resources

import (
	"context"
	"encoding/json"
	"strings"
	"testing"

	"github.com/vrooli/vrooli/internal/accel"
	"github.com/vrooli/vrooli/internal/hostinventory"
	manifestpkg "github.com/vrooli/vrooli/internal/resources/manifest"
)

// Feature: the compose invocation follows the backend the host can give
//
//	As a compose-service resource
//	I want my accelerator overlay applied only when the host can serve it
//	So that a CPU-only host gets the base compose file instead of one demanding
//	a device it does not have.

func TestStatusRawWithModeReportsSelectedExecutionMode(t *testing.T) {
	raw := statusRawWithMode(nil, "cpu")
	var payload map[string]any
	if err := json.Unmarshal(raw, &payload); err != nil {
		t.Fatal(err)
	}
	if payload["mode"] != "cpu" {
		t.Fatalf("mode=%v, want cpu", payload["mode"])
	}
}

// acceleratedComposeManifest builds a compose resource declaring cuda with a
// cpu floor, the shape every migrated compose resource now has.
func acceleratedComposeManifest(name string, cuda manifestpkg.BackendConfig) ResourceManifest {
	return ResourceManifest{
		Name:        name,
		Driver:      "compose-service",
		ComposeFile: "docker/docker-compose.yml",
		Acceleration: &manifestpkg.AccelerationSpec{
			Backends: []string{manifestpkg.BackendCUDA, manifestpkg.BackendCPU},
			Require:  manifestpkg.RequirePreferred,
			Backend: map[string]manifestpkg.BackendConfig{
				manifestpkg.BackendCUDA: cuda,
				manifestpkg.BackendCPU:  {},
			},
		},
	}
}

func acceleratedCUDAConfig() manifestpkg.BackendConfig {
	return manifestpkg.BackendConfig{
		ComposeOverlay: "docker/docker-compose.gpu.yml",
		Env:            map[string]string{"WHISPER_IMAGE": "onerahmet/openai-whisper-asr-webservice:latest-gpu"},
	}
}

// Scenario: a resource declaring no accelerator gets the base compose file.
func TestComposeInvocationArgsWithoutAnAccelerationBlock(t *testing.T) {
	// Given a compose resource that declares no accelerator
	controller := NewController(t.TempDir(), t.TempDir())
	manifest := ResourceManifest{Name: "whisper", Driver: "compose-service", ComposeFile: "docker/docker-compose.yml"}

	// When its compose invocation is built
	args, env := composeInvocationArgs(context.Background(), controller, manifest)

	// Then exactly one compose file is passed, and no backend env is applied
	if countFlag(args, "-f") != 1 {
		t.Fatalf("expected exactly one -f flag, got args=%v", args)
	}
	for _, kv := range env {
		if strings.HasPrefix(kv, "WHISPER_IMAGE=") {
			t.Fatalf("a resource declaring no accelerator must get no backend env; got %q", kv)
		}
	}
}

// Scenario: VROOLI_GPU=on forces the accelerator overlay.
func TestComposeInvocationArgsWithTheOverrideOn(t *testing.T) {
	// Given an accelerated compose resource and an operator forcing the device
	controller := NewController(t.TempDir(), t.TempDir())
	manifest := acceleratedComposeManifest("whisper", acceleratedCUDAConfig())
	t.Setenv(gpuOverrideEnvVar, "on")

	// When its compose invocation is built
	args, env := composeInvocationArgs(context.Background(), controller, manifest)

	// Then the overlay is layered on and the backend env is applied
	if countFlag(args, "-f") != 2 {
		t.Fatalf("expected the overlay to be layered on, got args=%v", args)
	}
	if !envContainsPrefix(env, "WHISPER_IMAGE=") {
		t.Fatalf("expected the cuda env override, got env=%v", env)
	}
}

// Scenario: VROOLI_GPU=off falls back to the base compose file.
func TestComposeInvocationArgsWithTheOverrideOff(t *testing.T) {
	// Given an accelerated compose resource and an operator forcing the CPU
	controller := NewController(t.TempDir(), t.TempDir())
	manifest := acceleratedComposeManifest("whisper", acceleratedCUDAConfig())
	t.Setenv(gpuOverrideEnvVar, "off")

	// When its compose invocation is built
	args, env := composeInvocationArgs(context.Background(), controller, manifest)

	// Then only the base compose file is used
	if countFlag(args, "-f") != 1 {
		t.Fatalf("the cpu backend must not layer an overlay, got args=%v", args)
	}
	// And the accelerator env is not applied, because the resource is not on it
	if envContainsPrefix(env, "WHISPER_IMAGE=") {
		t.Fatalf("the cuda env must not apply while running on the cpu, got env=%v", env)
	}
}

// Scenario: the cpu floor may carry its own env.
func TestComposeInvocationArgsAppliesTheCPUBackendEnv(t *testing.T) {
	// Given a resource whose cpu floor declares its own environment
	controller := NewController(t.TempDir(), t.TempDir())
	manifest := acceleratedComposeManifest("kyutai-stt", acceleratedCUDAConfig())
	manifest.Acceleration.Backend[manifestpkg.BackendCPU] = manifestpkg.BackendConfig{
		Env: map[string]string{"KYUTAI_STT_DEVICE": "cpu"},
	}
	t.Setenv(gpuOverrideEnvVar, "off")

	// When its compose invocation is built
	args, env := composeInvocationArgs(context.Background(), controller, manifest)

	// Then the cpu env is applied and no overlay is layered on
	if !envContainsPrefix(env, "KYUTAI_STT_DEVICE=cpu") {
		t.Fatalf("expected the cpu backend env, got env=%v", env)
	}
	if countFlag(args, "-f") != 1 {
		t.Fatalf("the cpu backend must not layer an overlay, got args=%v", args)
	}
}

func envContainsPrefix(env []string, prefix string) bool {
	for _, kv := range env {
		if strings.HasPrefix(kv, prefix) {
			return true
		}
	}
	return false
}

// countFlag counts how many times a flag appears in an argument vector.
func countFlag(args []string, flag string) int {
	count := 0
	for _, arg := range args {
		if arg == flag {
			count++
		}
	}
	return count
}

// withStubAcceleratorFacts replaces the package's host-fact seam for one test,
// so a unit run never depends on whether the machine running it has a device.
func withStubAcceleratorFacts(t *testing.T, snapshot hostinventory.Snapshot) {
	t.Helper()
	original := accelFactSource
	accelFactSource = accel.StaticFactSource{Inventory: snapshot}
	t.Cleanup(func() { accelFactSource = original })
}

// acceleratedHostSnapshot is a host with a working CUDA device.
func acceleratedHostSnapshot() hostinventory.Snapshot {
	return hostinventory.Snapshot{
		OS: "linux", Arch: "amd64",
		RuntimeTools:  map[string]hostinventory.Tool{hostinventory.ToolNvidiaSMI: {Present: true}},
		ProbeStatuses: map[string]string{"nvidia_gpu": "ok"},
		GPUs: []hostinventory.GPU{
			{Index: 0, Name: "NVIDIA RTX", CUDAComputeCapability: "8.9", VRAMBytes: 16 << 30, Source: hostinventory.SourceNvidiaSMI},
		},
	}
}
