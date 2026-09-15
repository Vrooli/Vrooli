package handlers

import (
	"context"
	"testing"
	"time"

	"connectrpc.com/connect"
	healing "github.com/vrooli/vrooli/packages/proto/gen/go/vrooli-autoheal/v1/healing"
	"github.com/vrooli/vrooli/scenarios/vrooli-autoheal/api/internal/checks"
	"github.com/vrooli/vrooli/scenarios/vrooli-autoheal/api/internal/checks/system"
)

// [REQ:BOOT-RECOVERY-001] The typed readiness projection carries the last
// boot-recovery verdict with every precondition, and says "unknown" (never
// ok) before the check has run.
func TestGetReadinessProjectsBootRecovery(t *testing.T) {
	h := setupTestHandlers(&mockStore{})
	svc := &typedHealing{h: h}

	resp, err := svc.GetReadiness(context.Background(), connect.NewRequest(&healing.GetReadinessRequest{}))
	if err != nil {
		t.Fatal(err)
	}
	if got := resp.Msg.GetBootRecovery(); got == nil || got.GetStatus() != "unknown" || got.GetRemediation() != "vrooli setup" {
		t.Fatalf("before the check runs: %+v", got)
	}

	at := time.Date(2026, 9, 2, 18, 0, 0, 0, time.UTC)
	h.registry.SetResult(checks.Result{
		CheckID:   system.BootRecoveryReadinessCheckID,
		Status:    checks.StatusCritical,
		Message:   "Boot recovery would not work: unit-active failed",
		Timestamp: at,
		Details: map[string]interface{}{
			"remediation": "vrooli setup",
			"preconditions": []interface{}{
				map[string]any{"name": "safeguards", "state": "ok", "reason": "applied"},
				map[string]any{"name": "unit-active", "state": "failed", "reason": "vrooli-runtime-supervisor.service is inactive"},
			},
		},
	})
	resp, err = svc.GetReadiness(context.Background(), connect.NewRequest(&healing.GetReadinessRequest{}))
	if err != nil {
		t.Fatal(err)
	}
	got := resp.Msg.GetBootRecovery()
	if got.GetStatus() != "critical" || len(got.GetPreconditions()) != 2 || got.GetPreconditions()[1].GetState() != "failed" {
		t.Fatalf("projection = %+v", got)
	}
	if !got.GetEvaluatedAt().AsTime().Equal(at) {
		t.Fatalf("evaluated_at = %v, want %v", got.GetEvaluatedAt().AsTime(), at)
	}
}
