package resources

import (
	"slices"
	"testing"
)

func TestBuildDockerRunArgsAppendsMemoryLimit(t *testing.T) {
	controller := NewController(t.TempDir(), t.TempDir())
	manifest := ResourceManifest{
		Name:   "ollama",
		Driver: "docker-service",
		Runtime: ResourceRuntime{
			Image:       "ollama/ollama:0.11.7",
			MemoryLimit: "12g",
		},
	}

	args, err := buildDockerRunArgs(controller, manifest, "ollama")
	if err != nil {
		t.Fatalf("buildDockerRunArgs: %v", err)
	}

	idx := slices.Index(args, "--memory")
	if idx < 0 {
		t.Fatalf("expected --memory flag in args, got %v", args)
	}
	if got := args[idx+1]; got != "12g" {
		t.Fatalf("--memory value = %q, want %q", got, "12g")
	}
	imageIdx := slices.Index(args, manifest.Runtime.Image)
	if imageIdx < 0 || idx >= imageIdx {
		t.Fatalf("--memory must precede the image positional; idx=%d imageIdx=%d args=%v", idx, imageIdx, args)
	}
}

func TestBuildDockerRunArgsOmitsMemoryLimitWhenUnset(t *testing.T) {
	controller := NewController(t.TempDir(), t.TempDir())
	manifest := ResourceManifest{
		Name:   "redis",
		Driver: "docker-service",
		Runtime: ResourceRuntime{
			Image: "redis:7",
		},
	}

	args, err := buildDockerRunArgs(controller, manifest, "redis")
	if err != nil {
		t.Fatalf("buildDockerRunArgs: %v", err)
	}
	if slices.Contains(args, "--memory") {
		t.Fatalf("expected no --memory flag, got %v", args)
	}
}
