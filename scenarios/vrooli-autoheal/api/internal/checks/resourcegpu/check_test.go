package resourcegpu

import (
	"context"
	"errors"
	"testing"
	"time"

	"github.com/vrooli/vrooli/internal/hostinventory"
	sharedresources "github.com/vrooli/vrooli/internal/resources"
	"github.com/vrooli/vrooli/scenarios/vrooli-autoheal/api/internal/checks"
)

func TestCheckMapsContainerGPUStates(t *testing.T) {
	cases := []struct {
		name   string
		state  sharedresources.GPUAccessState
		status checks.Status
	}{
		{"healthy", sharedresources.GPUAccessOK, checks.StatusOK},
		{"revoked", sharedresources.GPUAccessRevoked, checks.StatusCritical},
		{"unknown", sharedresources.GPUAccessUnknown, checks.StatusWarning},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			check := New(
				WithHostCollector(func(context.Context) (hostinventory.Snapshot, error) {
					return hostinventory.Snapshot{GPUs: []hostinventory.GPU{{Name: "NVIDIA", Source: "nvidia-smi"}}, DockerGPU: hostinventory.DockerGPU{NvidiaRuntime: true}}, nil
				}),
				WithTargets(func(context.Context) ([]Target, error) {
					return []Target{{Name: "ollama", Container: "ollama", Probe: "nvidia", Running: true}}, nil
				}),
				WithVerifier(func(context.Context, string, string) (sharedresources.GPUAccessState, string) {
					return tc.state, "fixture"
				}),
			)
			got := check.Run(context.Background())
			if got.Status != tc.status {
				t.Fatalf("status=%q, want %q; message=%q", got.Status, tc.status, got.Message)
			}
			if tc.state == sharedresources.GPUAccessUnknown && got.Status == checks.StatusOK {
				t.Fatal("unknown access must never report healthy")
			}
		})
	}
}

func TestCheckReportsNotApplicableWithoutDockerGPU(t *testing.T) {
	check := New(WithHostCollector(func(context.Context) (hostinventory.Snapshot, error) {
		return hostinventory.Snapshot{}, nil
	}))
	got := check.Run(context.Background())
	if got.Status != checks.StatusNotApplicable {
		t.Fatalf("status=%q, want not-applicable", got.Status)
	}
}

func TestCheckUnknownHostInventoryIsWarning(t *testing.T) {
	check := New(WithHostCollector(func(context.Context) (hostinventory.Snapshot, error) {
		return hostinventory.Snapshot{}, errors.New("probe unavailable")
	}))
	got := check.Run(context.Background())
	if got.Status != checks.StatusWarning {
		t.Fatalf("status=%q, want warning", got.Status)
	}
}

func TestCheckRecoveryActionsRestartNamedAndAllRevoked(t *testing.T) {
	var restarted []string
	check := New(
		WithRestarter(func(_ context.Context, name string) error { restarted = append(restarted, name); return nil }),
		WithTargets(func(context.Context) ([]Target, error) {
			return []Target{{Name: "ollama", Running: true}, {Name: "whisper", Running: true}}, nil
		}),
		WithClock(func() time.Time { return time.Unix(100, 0) }),
	)
	last := &checks.Result{Status: checks.StatusCritical, Details: map[string]interface{}{"revoked": []string{"ollama"}}}
	actions := check.RecoveryActions(last)
	if len(actions) != 2 {
		t.Fatalf("actions=%v, want named + all", actions)
	}
	if result := check.ExecuteAction(context.Background(), "restart-ollama"); !result.Success || len(restarted) != 1 || restarted[0] != "ollama" {
		t.Fatalf("named action result=%+v restarted=%v", result, restarted)
	}
}

func TestCheckAllRevokedActionRechecksBeforeRestarting(t *testing.T) {
	var restarted []string
	check := New(
		WithRestarter(func(_ context.Context, name string) error { restarted = append(restarted, name); return nil }),
		WithTargets(func(context.Context) ([]Target, error) {
			return []Target{
				{Name: "ollama", Container: "ollama", Probe: "nvidia", Running: true},
				{Name: "whisper", Container: "whisper", Probe: "nvidia", Running: true},
			}, nil
		}),
		WithVerifier(func(_ context.Context, container, _ string) (sharedresources.GPUAccessState, string) {
			if container == "ollama" {
				return sharedresources.GPUAccessRevoked, "revoked"
			}
			return sharedresources.GPUAccessOK, "ok"
		}),
	)
	result := check.ExecuteAction(context.Background(), "restart-all-revoked")
	if !result.Success || len(restarted) != 1 || restarted[0] != "ollama" {
		t.Fatalf("result=%+v restarted=%v", result, restarted)
	}
}
