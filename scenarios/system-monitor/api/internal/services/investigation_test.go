package services

import (
	"context"
	"strings"
	"testing"
	"time"

	domainpb "github.com/vrooli/vrooli/packages/proto/gen/go/agent-manager/v1/domain"
	"system-monitor-api/internal/agentmanager"
	"system-monitor-api/internal/config"
	"system-monitor-api/internal/models"
	"system-monitor-api/internal/repository/memory"
)

// stubAgentExecutor is a minimal AgentExecutor for testing.
type stubAgentExecutor struct{ available bool }

func (s *stubAgentExecutor) IsEnabled() bool                    { return true }
func (s *stubAgentExecutor) IsAvailable(_ context.Context) bool { return s.available }
func (s *stubAgentExecutor) Initialize(_ context.Context, _ *agentmanager.ProfileConfig) error {
	return nil
}

func (s *stubAgentExecutor) Execute(_ context.Context, _ agentmanager.ExecuteRequest) (*agentmanager.ExecuteResult, error) {
	return &agentmanager.ExecuteResult{Success: true, Output: "ok"}, nil
}

func (s *stubAgentExecutor) GetProfile(_ context.Context) (*domainpb.AgentProfile, error) {
	return nil, nil
}
func (s *stubAgentExecutor) GetProfileID() string { return "" }
func (s *stubAgentExecutor) UpdateProfile(_ context.Context, _ *agentmanager.ProfileConfig) (*domainpb.AgentProfile, error) {
	return nil, nil
}

func (s *stubAgentExecutor) GetAvailableRunners(_ context.Context) ([]agentmanager.RunnerInfo, error) {
	return nil, nil
}

func (s *stubAgentExecutor) GetRunByTag(_ context.Context, _ string) (*domainpb.Run, error) {
	return nil, nil
}

func (s *stubAgentExecutor) ListActiveRuns(_ context.Context) ([]*domainpb.Run, error) {
	return nil, nil
}
func (s *stubAgentExecutor) StopRun(_ context.Context, _ string) error { return nil }
func (s *stubAgentExecutor) ResolveURL(_ context.Context) (string, error) {
	return "", nil
}

func newTestInvestigationService(t *testing.T, clock Clock) *InvestigationService {
	t.Helper()
	cfg := &config.Config{}
	repo := memory.NewRepository()
	return NewInvestigationService(cfg, repo, nil, &stubAgentExecutor{available: true},
		WithInvestigationClock(clock),
	)
}

func TestCooldownEnforced(t *testing.T) {
	clk := NewStubClock(time.Date(2026, 1, 1, 0, 0, 0, 0, time.UTC))
	svc := newTestInvestigationService(t, clk)

	ctx := context.Background()

	// First trigger should succeed
	inv, err := svc.TriggerInvestigation(ctx, false, "")
	if err != nil {
		t.Fatalf("first trigger failed: %v", err)
	}
	if inv == nil {
		t.Fatal("expected investigation, got nil")
	}

	// Advance less than cooldown period (5 min default)
	clk.Advance(3 * time.Minute)

	// Second trigger should fail due to cooldown
	_, err = svc.TriggerInvestigation(ctx, false, "")
	if err == nil {
		t.Fatal("expected cooldown error, got nil")
	}
	if !strings.Contains(err.Error(), "cooldown") {
		t.Fatalf("expected cooldown error, got: %v", err)
	}

	// Advance past cooldown
	clk.Advance(3 * time.Minute) // total 6 min > 5 min cooldown

	// Third trigger should succeed
	inv2, err := svc.TriggerInvestigation(ctx, false, "")
	if err != nil {
		t.Fatalf("third trigger (post-cooldown) failed: %v", err)
	}
	if inv2 == nil {
		t.Fatal("expected investigation, got nil")
	}
}

func TestCooldownReset(t *testing.T) {
	clk := NewStubClock(time.Date(2026, 1, 1, 0, 0, 0, 0, time.UTC))
	svc := newTestInvestigationService(t, clk)

	ctx := context.Background()

	// Trigger to start cooldown
	_, err := svc.TriggerInvestigation(ctx, false, "")
	if err != nil {
		t.Fatalf("first trigger failed: %v", err)
	}

	// Advance a bit but still within cooldown
	clk.Advance(1 * time.Minute)

	// Reset cooldown
	if err := svc.ResetCooldown(ctx); err != nil {
		t.Fatalf("reset cooldown failed: %v", err)
	}

	// Should be able to trigger immediately
	inv, err := svc.TriggerInvestigation(ctx, false, "")
	if err != nil {
		t.Fatalf("trigger after reset failed: %v", err)
	}
	if inv == nil {
		t.Fatal("expected investigation, got nil")
	}
}

func TestInvestigationIDDeterministic(t *testing.T) {
	fixedTime := time.Date(2026, 6, 15, 12, 0, 0, 0, time.UTC)
	clk := NewStubClock(fixedTime)
	svc := newTestInvestigationService(t, clk)

	ctx := context.Background()
	inv, err := svc.TriggerInvestigation(ctx, false, "")
	if err != nil {
		t.Fatalf("trigger failed: %v", err)
	}

	expectedPrefix := "inv_"
	if !strings.HasPrefix(inv.ID, expectedPrefix) {
		t.Errorf("expected ID to start with %s, got %s", expectedPrefix, inv.ID)
	}

	// Verify the unix timestamp portion matches
	expectedID := "inv_1781524800"
	if inv.ID != expectedID {
		t.Errorf("expected ID %s, got %s", expectedID, inv.ID)
	}
}

func TestCooldownStatus(t *testing.T) {
	clk := NewStubClock(time.Date(2026, 1, 1, 0, 0, 0, 0, time.UTC))
	svc := newTestInvestigationService(t, clk)

	ctx := context.Background()

	// Before any trigger, should be ready
	status, err := svc.GetCooldownStatus(ctx)
	if err != nil {
		t.Fatalf("get cooldown status failed: %v", err)
	}
	if !status.IsReady {
		t.Error("expected ready before any trigger")
	}

	// Trigger
	_, _ = svc.TriggerInvestigation(ctx, false, "")

	// Advance 2 minutes
	clk.Advance(2 * time.Minute)

	status, err = svc.GetCooldownStatus(ctx)
	if err != nil {
		t.Fatalf("get cooldown status failed: %v", err)
	}
	if status.IsReady {
		t.Error("expected not ready during cooldown")
	}
	if status.RemainingSeconds <= 0 {
		t.Error("expected positive remaining seconds")
	}
}

func TestGetTriggers_Defaults(t *testing.T) {
	clk := NewStubClock(time.Date(2026, 1, 1, 0, 0, 0, 0, time.UTC))
	svc := newTestInvestigationService(t, clk)

	triggers, err := svc.GetTriggers(context.Background())
	if err != nil {
		t.Fatalf("get triggers failed: %v", err)
	}
	if len(triggers) == 0 {
		t.Fatal("expected default triggers")
	}
	if _, ok := triggers["high_cpu"]; !ok {
		t.Error("expected high_cpu trigger in defaults")
	}
}

func TestUpdateTrigger(t *testing.T) {
	clk := NewStubClock(time.Date(2026, 1, 1, 0, 0, 0, 0, time.UTC))
	svc := newTestInvestigationService(t, clk)

	ctx := context.Background()
	enabled := false
	threshold := 95.0
	err := svc.UpdateTrigger(ctx, "high_cpu", &enabled, nil, &threshold)
	if err != nil {
		t.Fatalf("update trigger failed: %v", err)
	}

	triggers, _ := svc.GetTriggers(ctx)
	trigger := triggers["high_cpu"]
	if trigger.Enabled {
		t.Error("expected trigger to be disabled")
	}
	if trigger.Threshold != 95.0 {
		t.Errorf("expected threshold 95.0, got %f", trigger.Threshold)
	}
}

func TestUpdateTrigger_NotFound(t *testing.T) {
	clk := NewStubClock(time.Date(2026, 1, 1, 0, 0, 0, 0, time.UTC))
	svc := newTestInvestigationService(t, clk)

	err := svc.UpdateTrigger(context.Background(), "nonexistent", nil, nil, nil)
	if err == nil {
		t.Fatal("expected error for nonexistent trigger")
	}
}

func TestListInvestigations_Empty(t *testing.T) {
	clk := NewStubClock(time.Date(2026, 1, 1, 0, 0, 0, 0, time.UTC))
	svc := newTestInvestigationService(t, clk)

	investigations, err := svc.ListInvestigations(context.Background(), 10)
	if err != nil {
		t.Fatalf("list investigations failed: %v", err)
	}
	if len(investigations) != 0 {
		t.Errorf("expected 0 investigations, got %d", len(investigations))
	}
}

func TestGetLatestInvestigation_Default(t *testing.T) {
	clk := NewStubClock(time.Date(2026, 1, 1, 0, 0, 0, 0, time.UTC))
	svc := newTestInvestigationService(t, clk)

	inv, err := svc.GetLatestInvestigation(context.Background())
	if err != nil {
		t.Fatalf("get latest investigation failed: %v", err)
	}
	if inv.ID != "inv_default" {
		t.Errorf("expected default investigation, got %s", inv.ID)
	}
	if inv.Status != models.StatusQueued {
		t.Errorf("expected queued status, got %s", inv.Status)
	}
}
