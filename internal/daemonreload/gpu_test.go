package daemonreload

import (
	"context"
	"errors"
	"testing"

	"github.com/vrooli/vrooli/internal/gpuaccess"
	"github.com/vrooli/vrooli/internal/hostreqkit"
)

func TestReloadRepairsRevokedGPUContainer(t *testing.T) {
	oldList, oldVerify, oldReload, oldRestart := listGPUContainersFn, verifyGPUFn, reloadFn, restartGPUFn
	t.Cleanup(func() {
		listGPUContainersFn, verifyGPUFn, reloadFn, restartGPUFn = oldList, oldVerify, oldReload, oldRestart
	})
	listGPUContainersFn = func(context.Context, string) ([]target, error) {
		return []target{{resource: "ollama", container: "ollama", probe: "nvidia"}}, nil
	}
	checks := 0
	verifyGPUFn = func(context.Context, string, string) (gpuaccess.State, string) {
		checks++
		if checks == 1 {
			return gpuaccess.OK, "before"
		}
		if checks == 2 {
			return gpuaccess.Revoked, "operation not permitted"
		}
		return gpuaccess.OK, "after restart"
	}
	reloaded := false
	reloadFn = func(string, string, []string, hostreqkit.EnsureOptions) error { reloaded = true; return nil }
	restarted := false
	restartGPUFn = func(context.Context, string, ...string) error { restarted = true; return nil }
	got, err := Reload(context.Background(), "/repo", hostreqkit.EnsureOptions{})
	if err != nil {
		t.Fatal(err)
	}
	if !reloaded || !restarted || len(got.Repairs) != 1 || got.Repairs[0].State != string(gpuaccess.OK) {
		t.Fatalf("result=%+v reloaded=%v restarted=%v", got, reloaded, restarted)
	}
}

func TestReloadSkipsGPUWorkWhenNoGPUResources(t *testing.T) {
	oldList, oldReload := listGPUContainersFn, reloadFn
	t.Cleanup(func() { listGPUContainersFn, reloadFn = oldList, oldReload })
	listGPUContainersFn = func(context.Context, string) ([]target, error) { return nil, nil }
	called := false
	reloadFn = func(string, string, []string, hostreqkit.EnsureOptions) error { called = true; return nil }
	if _, err := Reload(context.Background(), "/repo", hostreqkit.EnsureOptions{}); err != nil {
		t.Fatal(err)
	}
	if !called {
		t.Fatal("daemon-reload must still run on a host with no GPU resources")
	}
}

func TestReloadStillRunsWhenDockerInventoryIsUnavailable(t *testing.T) {
	oldList, oldReload := listGPUContainersFn, reloadFn
	t.Cleanup(func() { listGPUContainersFn, reloadFn = oldList, oldReload })
	listGPUContainersFn = func(context.Context, string) ([]target, error) { return nil, errors.New("docker daemon unavailable") }
	called := false
	reloadFn = func(string, string, []string, hostreqkit.EnsureOptions) error { called = true; return nil }
	result, err := Reload(context.Background(), "/repo", hostreqkit.EnsureOptions{})
	if err != nil {
		t.Fatal(err)
	}
	if !called || len(result.Notes) != 1 {
		t.Fatalf("result=%+v called=%v", result, called)
	}
}

func TestReloadReturnsFailedRestart(t *testing.T) {
	oldList, oldVerify, oldReload, oldRestart := listGPUContainersFn, verifyGPUFn, reloadFn, restartGPUFn
	t.Cleanup(func() {
		listGPUContainersFn, verifyGPUFn, reloadFn, restartGPUFn = oldList, oldVerify, oldReload, oldRestart
	})
	listGPUContainersFn = func(context.Context, string) ([]target, error) {
		return []target{{resource: "whisper", container: "whisper", probe: "nvidia"}}, nil
	}
	checks := 0
	verifyGPUFn = func(context.Context, string, string) (gpuaccess.State, string) {
		checks++
		if checks == 1 {
			return gpuaccess.OK, "before"
		}
		return gpuaccess.Revoked, "revoked"
	}
	reloadFn = func(string, string, []string, hostreqkit.EnsureOptions) error { return nil }
	restartGPUFn = func(context.Context, string, ...string) error { return errors.New("restart failed") }
	if _, err := Reload(context.Background(), "/repo", hostreqkit.EnsureOptions{}); err == nil {
		t.Fatal("expected restart error")
	}
}
