package resources

import (
	"context"
	"path/filepath"
	"strings"
	"testing"

	manifestpkg "github.com/vrooli/vrooli/internal/resources/manifest"
)

func gpuTestManifest(name string, gpu *manifestpkg.ResourceGPU) ResourceManifest {
	return ResourceManifest{
		Name:        name,
		Driver:      "compose-service",
		ComposeFile: "docker/docker-compose.yml",
		GPU:         gpu,
	}
}

func TestComposeInvocationArgsNoGPUBlock(t *testing.T) {
	controller := NewController(t.TempDir(), t.TempDir())
	manifest := gpuTestManifest("whisper", nil)

	args, env := composeInvocationArgs(context.Background(), controller, manifest)

	if countFlag(args, "-f") != 1 {
		t.Fatalf("expected exactly one -f flag, got args=%v", args)
	}
	for _, kv := range env {
		if strings.HasPrefix(kv, "WHISPER_IMAGE=") {
			t.Fatalf("no env overrides should be present when gpu block is absent; got %q", kv)
		}
	}
}

func TestComposeInvocationArgsGPUOverrideOn(t *testing.T) {
	controller := NewController(t.TempDir(), t.TempDir())
	manifest := gpuTestManifest("whisper", &manifestpkg.ResourceGPU{
		Probe:          "nvidia",
		ComposeOverlay: "docker/docker-compose.gpu.yml",
		EnvOverrides: map[string]string{
			"WHISPER_IMAGE": "onerahmet/openai-whisper-asr-webservice:latest-gpu",
		},
	})
	t.Setenv(gpuOverrideEnvVar, "on")

	args, env := composeInvocationArgs(context.Background(), controller, manifest)

	if countFlag(args, "-f") != 2 {
		t.Fatalf("expected two -f flags (base + overlay), got args=%v", args)
	}
	overlayPath := filepath.Join(controller.Root, "resources", "whisper", "docker", "docker-compose.gpu.yml")
	if !containsString(args, overlayPath) {
		t.Fatalf("expected overlay path %q in args %v", overlayPath, args)
	}
	if !containsEnv(env, "WHISPER_IMAGE=onerahmet/openai-whisper-asr-webservice:latest-gpu") {
		t.Fatalf("expected env override in %v", env)
	}
}

func TestComposeInvocationArgsGPUOverrideOff(t *testing.T) {
	controller := NewController(t.TempDir(), t.TempDir())
	manifest := gpuTestManifest("whisper", &manifestpkg.ResourceGPU{
		Probe:          "nvidia",
		ComposeOverlay: "docker/docker-compose.gpu.yml",
		EnvOverrides:   map[string]string{"WHISPER_IMAGE": "something-gpu"},
	})
	t.Setenv(gpuOverrideEnvVar, "off")

	args, env := composeInvocationArgs(context.Background(), controller, manifest)

	if countFlag(args, "-f") != 1 {
		t.Fatalf("expected exactly one -f flag when override=off, got args=%v", args)
	}
	if containsEnv(env, "WHISPER_IMAGE=something-gpu") {
		t.Fatal("env override must not be present when override=off")
	}
}

func TestComposeInvocationArgsGPUAutoProbeFails(t *testing.T) {
	controller := NewController(t.TempDir(), t.TempDir())
	manifest := gpuTestManifest("whisper", &manifestpkg.ResourceGPU{
		Probe:          "nvidia",
		ComposeOverlay: "docker/docker-compose.gpu.yml",
		EnvOverrides:   map[string]string{"WHISPER_IMAGE": "something-gpu"},
	})
	t.Setenv(gpuOverrideEnvVar, "auto")
	withStubGPUProbe(t, false)

	args, env := composeInvocationArgs(context.Background(), controller, manifest)

	if countFlag(args, "-f") != 1 {
		t.Fatalf("probe fail must not add overlay; got args=%v", args)
	}
	if containsEnv(env, "WHISPER_IMAGE=something-gpu") {
		t.Fatal("env override must not be present when probe fails")
	}
}

func TestComposeInvocationArgsGPUAutoProbePasses(t *testing.T) {
	controller := NewController(t.TempDir(), t.TempDir())
	manifest := gpuTestManifest("whisper", &manifestpkg.ResourceGPU{
		Probe:          "nvidia",
		ComposeOverlay: "docker/docker-compose.gpu.yml",
		EnvOverrides:   map[string]string{"WHISPER_IMAGE": "something-gpu"},
	})
	t.Setenv(gpuOverrideEnvVar, "auto")
	withStubGPUProbe(t, true)

	args, env := composeInvocationArgs(context.Background(), controller, manifest)

	if countFlag(args, "-f") != 2 {
		t.Fatalf("probe pass must add overlay; got args=%v", args)
	}
	if !containsEnv(env, "WHISPER_IMAGE=something-gpu") {
		t.Fatalf("env override must be present when probe passes; got env=%v", env)
	}
}

func countFlag(args []string, flag string) int {
	n := 0
	for _, a := range args {
		if a == flag {
			n++
		}
	}
	return n
}

func containsEnv(env []string, kv string) bool {
	for _, e := range env {
		if e == kv {
			return true
		}
	}
	return false
}
