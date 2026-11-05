package checks

import (
	"context"
	"encoding/json"
	"net"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"github.com/vrooli/vrooli/internal/scenarioruntime"
)

func TestRuntimeRecoveryGateSuppressesRestartDuringGatedEpoch(t *testing.T) {
	home := t.TempDir()
	store, err := scenarioruntime.NewSQLiteStore(context.Background(), scenarioruntime.Config{HomeDir: home})
	if err != nil {
		t.Fatalf("NewSQLiteStore: %v", err)
	}
	if _, err := store.CreatePressureEpoch(context.Background(), scenarioruntime.PressureEpoch{EpochID: "epoch-gated", Status: scenarioruntime.PressureEpochGated, DetectedAt: time.Now()}); err != nil {
		t.Fatalf("CreatePressureEpoch: %v", err)
	}
	_ = store.Close()
	gate := RuntimeRecoveryGate{HomeDir: home}
	allowed, reason := gate.AllowsAutoHealRestart(context.Background(), "scenario-system-monitor", "restart")
	if allowed || reason == "" {
		t.Fatalf("gate = allowed=%t reason=%q, want suppression", allowed, reason)
	}
}

func TestRuntimeRecoveryGateSuppressesOwnerRestartDuringStartup(t *testing.T) {
	home := t.TempDir()
	store, err := scenarioruntime.NewSQLiteStore(context.Background(), scenarioruntime.Config{HomeDir: home})
	if err != nil {
		t.Fatalf("NewSQLiteStore: %v", err)
	}
	if _, err := store.CreateLease(context.Background(), scenarioruntime.Instance{
		Scenario: "agent-manager", Variant: scenarioruntime.DefaultVariant, Status: scenarioruntime.StatusStarting,
	}, time.Minute); err != nil {
		t.Fatalf("CreateLease: %v", err)
	}
	instances, err := store.ListInstances(context.Background(), scenarioruntime.InstanceFilter{Scenario: "agent-manager", Variant: scenarioruntime.DefaultVariant, Statuses: scenarioruntime.ActiveInstanceStatuses()})
	if err != nil || len(instances) != 1 || instances[0].Status != scenarioruntime.StatusStarting {
		t.Fatalf("startup fixture instances = %#v, err=%v", instances, err)
	}
	if err := store.Close(); err != nil {
		t.Fatalf("close store: %v", err)
	}

	gate := RuntimeRecoveryGate{HomeDir: home, CoordinatedScenario: "agent-manager"}
	allowed, reason := gate.AllowsAutoHealRestart(context.Background(), "scenario-agent-manager", "restart")
	if allowed || reason == "" {
		t.Fatalf("gate = allowed=%t reason=%q, want startup suppression", allowed, reason)
	}
	allowed, reason = gate.AllowsAutoHealRestart(context.Background(), "scenario-system-monitor", "restart")
	if !allowed || reason != "" {
		t.Fatalf("unrelated gate = allowed=%t reason=%q, want ordinary behavior", allowed, reason)
	}
}

func TestRuntimeRecoveryGateAllowsRepairAfterDeadOwnerLease(t *testing.T) {
	server := httptest.NewServer(http.NotFoundHandler())
	port := server.Listener.Addr().(*net.TCPAddr).Port
	server.Close()

	home := t.TempDir()
	store, err := scenarioruntime.NewSQLiteStore(context.Background(), scenarioruntime.Config{HomeDir: home})
	if err != nil {
		t.Fatalf("NewSQLiteStore: %v", err)
	}
	expired := time.Now().Add(-time.Minute)
	deadPID := 999999999
	instance, err := store.CreateInstance(context.Background(), scenarioruntime.Instance{
		Scenario: "agent-manager", Variant: scenarioruntime.DefaultVariant, Status: scenarioruntime.StatusRunning,
		OwnerPID: &deadPID, HeartbeatDeadlineAt: &expired,
	})
	if err != nil {
		t.Fatalf("CreateInstance: %v", err)
	}
	claim, err := store.AcquirePortClaim(context.Background(), scenarioruntime.PortClaim{
		InstanceID: instance.InstanceID, Scenario: "agent-manager", Variant: scenarioruntime.DefaultVariant,
		PortName: "API_PORT", EnvVar: "API_PORT", Port: port,
	})
	if err != nil {
		t.Fatalf("AcquirePortClaim: %v", err)
	}
	if _, err := store.BindPortClaim(context.Background(), claim.ClaimID); err != nil {
		t.Fatalf("BindPortClaim: %v", err)
	}
	if err := store.Close(); err != nil {
		t.Fatalf("close store: %v", err)
	}

	gate := RuntimeRecoveryGate{HomeDir: home, CoordinatedScenario: "agent-manager"}
	allowed, reason := gate.AllowsAutoHealRestart(context.Background(), "scenario-agent-manager", "restart")
	if !allowed || reason != "" {
		t.Fatalf("gate = allowed=%t reason=%q, want stale dead owner to be recoverable", allowed, reason)
	}
}

func TestRuntimeRecoveryGateSuppressesOwnerRestartDuringMaintenance(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path != "/api/v1/maintenance/admission" {
			http.NotFound(w, r)
			return
		}
		_ = json.NewEncoder(w).Encode(map[string]any{"closed": true, "revision": 4})
	}))
	defer server.Close()
	port := server.Listener.Addr().(*net.TCPAddr).Port

	home := t.TempDir()
	store, err := scenarioruntime.NewSQLiteStore(context.Background(), scenarioruntime.Config{HomeDir: home})
	if err != nil {
		t.Fatalf("NewSQLiteStore: %v", err)
	}
	instance, err := store.CreateLease(context.Background(), scenarioruntime.Instance{
		Scenario: "agent-manager", Variant: scenarioruntime.DefaultVariant, Status: scenarioruntime.StatusRunning,
	}, time.Minute)
	if err != nil {
		t.Fatalf("CreateLease: %v", err)
	}
	claim, err := store.AcquirePortClaim(context.Background(), scenarioruntime.PortClaim{
		InstanceID: instance.InstanceID, Scenario: "agent-manager", Variant: scenarioruntime.DefaultVariant,
		PortName: "API_PORT", EnvVar: "API_PORT", Port: port,
	})
	if err != nil {
		t.Fatalf("AcquirePortClaim: %v", err)
	}
	if _, err := store.BindPortClaim(context.Background(), claim.ClaimID); err != nil {
		t.Fatalf("BindPortClaim: %v", err)
	}
	if err := store.Close(); err != nil {
		t.Fatalf("close store: %v", err)
	}

	gate := RuntimeRecoveryGate{HomeDir: home, CoordinatedScenario: "agent-manager"}
	allowed, reason := gate.AllowsAutoHealRestart(context.Background(), "scenario-agent-manager", "restart")
	if allowed || reason == "" {
		t.Fatalf("gate = allowed=%t reason=%q, want maintenance suppression", allowed, reason)
	}
}

func TestRuntimeRecoveryGateRequiresExplicitOwnerAdmissionClosed(t *testing.T) {
	tests := map[string]string{
		"empty response": `{}`,
		"wrong service":  `{"service":"other"}`,
		"malformed json": `{"closed":`,
	}
	for name, body := range tests {
		t.Run(name, func(t *testing.T) {
			server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
				if r.URL.Path != "/api/v1/maintenance/admission" {
					http.NotFound(w, r)
					return
				}
				w.Header().Set("Content-Type", "application/json")
				_, _ = w.Write([]byte(body))
			}))
			defer server.Close()
			port := server.Listener.Addr().(*net.TCPAddr).Port

			home := t.TempDir()
			store, err := scenarioruntime.NewSQLiteStore(context.Background(), scenarioruntime.Config{HomeDir: home})
			if err != nil {
				t.Fatalf("NewSQLiteStore: %v", err)
			}
			instance, err := store.CreateLease(context.Background(), scenarioruntime.Instance{
				Scenario: "agent-manager", Variant: scenarioruntime.DefaultVariant, Status: scenarioruntime.StatusRunning,
			}, time.Minute)
			if err != nil {
				t.Fatalf("CreateLease: %v", err)
			}
			claim, err := store.AcquirePortClaim(context.Background(), scenarioruntime.PortClaim{
				InstanceID: instance.InstanceID, Scenario: "agent-manager", Variant: scenarioruntime.DefaultVariant,
				PortName: "API_PORT", EnvVar: "API_PORT", Port: port,
			})
			if err != nil {
				t.Fatalf("AcquirePortClaim: %v", err)
			}
			if _, err := store.BindPortClaim(context.Background(), claim.ClaimID); err != nil {
				t.Fatalf("BindPortClaim: %v", err)
			}
			if err := store.Close(); err != nil {
				t.Fatalf("close store: %v", err)
			}

			gate := RuntimeRecoveryGate{HomeDir: home, CoordinatedScenario: "agent-manager"}
			allowed, reason := gate.AllowsAutoHealRestart(context.Background(), "scenario-agent-manager", "restart")
			if allowed || reason == "" {
				t.Fatalf("gate = allowed=%t reason=%q, want incomplete admission to fail closed", allowed, reason)
			}
		})
	}
}
