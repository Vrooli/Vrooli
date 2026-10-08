package resources

import (
	"context"
	"errors"
	"testing"

	"github.com/vrooli/vrooli/internal/hostinventory"
)

func withStubGPUProbe(t *testing.T, result bool) {
	t.Helper()
	orig := gpuProbe
	gpuProbe = func(context.Context, string) bool { return result }
	t.Cleanup(func() { gpuProbe = orig })
}

func TestShouldUseGPU(t *testing.T) {
	cases := []struct {
		name     string
		probe    string
		override string
		stub     bool
		want     bool
	}{
		{"empty probe never uses GPU", "", "auto", true, false},
		{"override=on forces true", "nvidia", "on", false, true},
		{"override=off forces false", "nvidia", "off", true, false},
		{"override=auto delegates to probe (pass)", "nvidia", "auto", true, true},
		{"override=auto delegates to probe (fail)", "nvidia", "auto", false, false},
		{"unset override defaults to auto (pass)", "nvidia", "", true, true},
		{"unset override defaults to auto (fail)", "nvidia", "", false, false},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			withStubGPUProbe(t, tc.stub)
			t.Setenv(gpuOverrideEnvVar, tc.override)
			if got := shouldUseGPU(context.Background(), tc.probe); got != tc.want {
				t.Fatalf("shouldUseGPU(%q) with override=%q stub=%v: got %v, want %v", tc.probe, tc.override, tc.stub, got, tc.want)
			}
		})
	}
}

func TestRunGPUProbeUnknownProbeReturnsFalse(t *testing.T) {
	if runGPUProbe(context.Background(), "radeon") {
		t.Fatal("unknown probe should return false")
	}
}

func TestNvidiaProbeUsesSharedHostInventory(t *testing.T) {
	orig := collectGPUInventory
	collectGPUInventory = func(context.Context) (hostinventory.Snapshot, error) {
		return hostinventory.Snapshot{GPUs: []hostinventory.GPU{{Name: "NVIDIA RTX 4090", Source: "nvidia-smi"}}, DockerGPU: hostinventory.DockerGPU{NvidiaRuntime: true}}, nil
	}
	t.Cleanup(func() { collectGPUInventory = orig })
	if !nvidiaProbe(context.Background()) {
		t.Fatal("nvidiaProbe should return true for Docker-addressable NVIDIA GPUs")
	}
}

func TestNvidiaProbeReturnsFalseWhenSharedHostInventoryFails(t *testing.T) {
	orig := collectGPUInventory
	collectGPUInventory = func(context.Context) (hostinventory.Snapshot, error) {
		return hostinventory.Snapshot{}, errors.New("probe failed")
	}
	t.Cleanup(func() { collectGPUInventory = orig })
	if nvidiaProbe(context.Background()) {
		t.Fatal("nvidiaProbe should return false when shared host inventory fails")
	}
}

func TestVerifyContainerGPUMapsDeviceOpenOutcomes(t *testing.T) {
	cases := []struct {
		name   string
		output string
		err    error
		want   GPUAccessState
	}{
		{name: "open succeeds", output: "ok", want: GPUAccessOK},
		{name: "device access revoked", output: "failed to open /dev/nvidiactl: operation not permitted", err: errors.New("exit status 1"), want: GPUAccessRevoked},
		{name: "device permission denied", output: "cannot create /dev/nvidiactl: Permission denied", err: errors.New("exit status 1"), want: GPUAccessRevoked},
		{name: "container cannot answer", output: "exec: \"sh\": executable file not found", err: errors.New("exit status 1"), want: GPUAccessUnknown},
		{name: "container absent", output: "Error response from daemon: No such container", err: errors.New("exit status 1"), want: GPUAccessUnknown},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			old := verifyContainerGPUExec
			t.Cleanup(func() { verifyContainerGPUExec = old })
			verifyContainerGPUExec = func(context.Context, string, string) ([]byte, error) {
				return []byte(tc.output), tc.err
			}
			got, reason := VerifyContainerGPU(context.Background(), "gpu-test", "nvidia")
			if got != tc.want {
				t.Fatalf("state=%q, want %q (reason=%q)", got, tc.want, reason)
			}
			if got != GPUAccessOK && reason == "" {
				t.Fatal("non-OK state must preserve a reason")
			}
		})
	}
}

func TestVerifyContainerGPURejectsUnknownProbeWithoutExecuting(t *testing.T) {
	called := false
	old := verifyContainerGPUExec
	t.Cleanup(func() { verifyContainerGPUExec = old })
	verifyContainerGPUExec = func(context.Context, string, string) ([]byte, error) {
		called = true
		return []byte("ok"), nil
	}
	got, reason := VerifyContainerGPU(context.Background(), "gpu-test", "cuda")
	if got != GPUAccessUnknown || reason == "" {
		t.Fatalf("state=%q reason=%q, want unknown with reason", got, reason)
	}
	if called {
		t.Fatal("unsupported probe must not execute a container probe")
	}
}
