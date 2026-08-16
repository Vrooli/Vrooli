package resources

import (
	"context"
	"errors"
	"os/exec"
	"slices"
	"strings"
	"testing"

	manifestpkg "github.com/vrooli/vrooli/internal/resources/manifest"
)

func TestBuildDockerRunArgsAppendsMemoryLimit(t *testing.T) {
	controller := NewController(t.TempDir(), t.TempDir())
	manifest := ResourceManifest{
		Name:   "ollama",
		Driver: "docker-service",
		Runtime: ResourceRuntime{
			Image:       "example/inference-service:0.11.7",
			MemoryLimit: "12g",
		},
	}

	args, err := buildDockerRunArgs(context.Background(), controller, manifest, "inference-service")
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

	args, err := buildDockerRunArgs(context.Background(), controller, manifest, "redis")
	if err != nil {
		t.Fatalf("buildDockerRunArgs: %v", err)
	}
	if slices.Contains(args, "--memory") {
		t.Fatalf("expected no --memory flag, got %v", args)
	}
}

func TestBuildDockerRunArgsPinsRuncForServiceWithoutGPUDeclaration(t *testing.T) {
	controller := NewController(t.TempDir(), t.TempDir())
	manifest := ResourceManifest{
		Name:    "ollama",
		Driver:  "docker-service",
		Runtime: ResourceRuntime{Image: "example/inference-service:0.30.10"},
	}

	args, err := buildDockerRunArgs(context.Background(), controller, manifest, "inference-service")
	if err != nil {
		t.Fatalf("buildDockerRunArgs: %v", err)
	}
	if !containsSubsequence(args, "--runtime", "runc") {
		t.Fatalf("non-GPU docker service must pin runc instead of inheriting the daemon default runtime: %v", args)
	}
}

func TestBuildDockerRunArgsSupportsHostIPAndProtocol(t *testing.T) {
	controller := NewController(t.TempDir(), t.TempDir())
	manifest := ResourceManifest{
		Name:   "adguard-home",
		Driver: "docker-service",
		Ports: []ResourcePort{
			{Name: "dns-tcp", HostIP: "192.168.1.173", Container: 53, Host: 53, Protocol: "tcp"},
			{Name: "dns-udp", HostIP: "192.168.1.173", Container: 53, Host: 53, Protocol: "udp"},
		},
		Runtime: ResourceRuntime{
			Image: "adguard/adguardhome:v0.107.77",
		},
	}

	args, err := buildDockerRunArgs(context.Background(), controller, manifest, "adguard-home")
	if err != nil {
		t.Fatalf("buildDockerRunArgs: %v", err)
	}

	if !containsSubsequence(args, "-p", "192.168.1.173:53:53") {
		t.Fatalf("expected tcp bind mapping, got %v", args)
	}
	if !containsSubsequence(args, "-p", "192.168.1.173:53:53/udp") {
		t.Fatalf("expected udp bind mapping, got %v", args)
	}
}

func TestBuildDockerRunArgsUsesGPUOnlyWhenDeclaredAndAvailable(t *testing.T) {
	controller := NewController(t.TempDir(), t.TempDir())
	manifest := ResourceManifest{
		Name:   "ollama",
		Driver: "docker-service",
		GPU: &manifestpkg.ResourceGPU{
			Probe:        "nvidia",
			EnvOverrides: map[string]string{"NVIDIA_VISIBLE_DEVICES": "all"},
		},
		Runtime: ResourceRuntime{Image: "example/inference-service:0.30.10"},
	}
	withStubGPUProbe(t, true)

	args, err := buildDockerRunArgs(context.Background(), controller, manifest, "inference-service")
	if err != nil {
		t.Fatalf("buildDockerRunArgs: %v", err)
	}
	if !containsSubsequence(args, "--gpus", "all") {
		t.Fatalf("GPU-declared service with a passing probe must receive GPU devices: %v", args)
	}
	if slices.Contains(args, "--runtime") {
		t.Fatalf("GPU-declared service must not pin the CPU runtime: %v", args)
	}
	if !containsSubsequence(args, "-e", "NVIDIA_VISIBLE_DEVICES=all") {
		t.Fatalf("GPU environment overrides are missing: %v", args)
	}
}

func TestDockerRuntimeForManifestUsesRuncWhenGPUIsUnavailable(t *testing.T) {
	manifest := ResourceManifest{
		Name: "ollama",
		GPU:  &manifestpkg.ResourceGPU{Probe: "nvidia", EnvOverrides: map[string]string{"NVIDIA_VISIBLE_DEVICES": "all"}},
	}
	withStubGPUProbe(t, false)
	if got := dockerRuntimeForManifest(context.Background(), manifest); got != "runc" {
		t.Fatalf("runtime = %q, want runc when the GPU probe fails", got)
	}
}

func TestValidateExistingDockerMountsRejectsStaleRepoLocalBind(t *testing.T) {
	controller := NewController(t.TempDir(), t.TempDir())
	manifest := ResourceManifest{
		Name: "postgres",
		Runtime: ResourceRuntime{
			ContainerName: "vrooli-postgres-main",
			Volumes: []ResourceVolume{
				{Source: "${RESOURCE_DATA_DIR}/instances/main/data", Target: "/var/lib/postgresql/data"},
			},
		},
	}

	originalRun := runCommandResource
	t.Cleanup(func() {
		runCommandResource = originalRun
	})
	runCommandResource = func(ctx context.Context, cmd *exec.Cmd) commandResult {
		if !strings.Contains(strings.Join(cmd.Args, " "), "{{json .Mounts}}") {
			return commandResult{err: errors.New("unexpected docker command")}
		}
		return commandResult{output: []byte(`[{"Source":"/repo/resources/postgres/instances/main/data","Destination":"/var/lib/postgresql/data"}]`)}
	}

	err := validateExistingDockerMounts(context.Background(), controller, manifest)
	if err == nil {
		t.Fatal("expected stale bind mount to be rejected")
	}
	if got := err.Error(); !strings.Contains(got, "stale docker mount") || !strings.Contains(got, "docker rm -f vrooli-postgres-main") {
		t.Fatalf("error = %q, want stale mount remediation", got)
	}
}

func containsSubsequence(values []string, want ...string) bool {
	for i := 0; i+len(want) <= len(values); i++ {
		if slices.Equal(values[i:i+len(want)], want) {
			return true
		}
	}
	return false
}
