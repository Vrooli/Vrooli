package checks

import (
	"context"
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
