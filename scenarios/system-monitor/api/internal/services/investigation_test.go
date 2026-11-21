package services

import (
	"context"
	"errors"
	"fmt"
	"strings"
	"testing"
	"time"

	"github.com/vrooli/vrooli/scenarios/system-monitor/api/internal/agentmanager"
	"github.com/vrooli/vrooli/scenarios/system-monitor/api/internal/apierrors"
	"github.com/vrooli/vrooli/scenarios/system-monitor/api/internal/config"
	"github.com/vrooli/vrooli/scenarios/system-monitor/api/internal/models"
	"github.com/vrooli/vrooli/scenarios/system-monitor/api/internal/repository"
	"github.com/vrooli/vrooli/scenarios/system-monitor/api/internal/repository/memory"
	"github.com/vrooli/vrooli/scenarios/system-monitor/api/internal/services/mocks"
)

func newTestInvestigationService(t *testing.T, clock Clock) *InvestigationService {
	t.Helper()
	cfg := &config.Config{}
	repo := memory.NewRepository()
	promptStore := NewMemoryConfigStore()
	if err := promptStore.WriteConfig("anomaly-check.md", []byte("Investigation {{INVESTIGATION_ID}}")); err != nil {
		t.Fatalf("seed prompt store: %v", err)
	}
	svc, err := NewInvestigationService(cfg, repo, nil, mocks.NewAgentExecutor().WithAvailable(true), NewSettingsManager(WithSettingsConfigStore(NewMemoryConfigStore())),
		WithInvestigationClock(clock),
		WithPromptStore(promptStore),
	)
	if err != nil {
		t.Fatalf("NewInvestigationService: %v", err)
	}
	return svc
}

func TestDetachedCleanupContextPreservesValuesAndBoundsWork(t *testing.T) {
	type contextKey struct{}
	parent, cancelParent := context.WithCancel(context.WithValue(context.Background(), contextKey{}, "trace-id"))
	cancelParent()

	cleanup, cancelCleanup := detachedCleanupContext(parent, time.Second)
	defer cancelCleanup()
	if err := cleanup.Err(); err != nil {
		t.Fatalf("cleanup inherited cancellation: %v", err)
	}
	if got := cleanup.Value(contextKey{}); got != "trace-id" {
		t.Fatalf("cleanup lost parent value: %v", got)
	}
	deadline, ok := cleanup.Deadline()
	if !ok || time.Until(deadline) > time.Second {
		t.Fatalf("cleanup deadline = %v, present=%v; want a one-second bound", deadline, ok)
	}
}

func TestCooldownEnforced(t *testing.T) {
	clk := NewStubClock(time.Date(2026, 1, 1, 0, 0, 0, 0, time.UTC))
	svc := newTestInvestigationService(t, clk)

	ctx := context.Background()

	// First trigger should succeed
	inv, err := svc.TriggerInvestigation(auth01FixtureCaller(t, ctx), false, "")
	if err != nil {
		t.Fatalf("first trigger failed: %v", err)
	}
	if inv == nil {
		t.Fatal("expected investigation, got nil")
	}

	// Advance less than cooldown period (5 min default)
	clk.Advance(3 * time.Minute)

	// Second trigger should fail due to cooldown
	_, err = svc.TriggerInvestigation(auth01FixtureCaller(t, ctx), false, "")
	if err == nil {
		t.Fatal("expected cooldown error, got nil")
	}
	if !strings.Contains(err.Error(), "cooldown") {
		t.Fatalf("expected cooldown error, got: %v", err)
	}

	// Advance past cooldown
	clk.Advance(3 * time.Minute) // total 6 min > 5 min cooldown

	// Third trigger should succeed
	inv2, err := svc.TriggerInvestigation(auth01FixtureCaller(t, ctx), false, "")
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
	_, err := svc.TriggerInvestigation(auth01FixtureCaller(t, ctx), false, "")
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
	inv, err := svc.TriggerInvestigation(auth01FixtureCaller(t, ctx), false, "")
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
	inv, err := svc.TriggerInvestigation(auth01FixtureCaller(t, ctx), false, "")
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
	_, _ = svc.TriggerInvestigation(auth01FixtureCaller(t, ctx), false, "")

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

func TestInvestigationCooldownUsesCanonicalSettings(t *testing.T) {
	clk := NewStubClock(time.Date(2026, 1, 1, 0, 0, 0, 0, time.UTC))
	svc := newTestInvestigationService(t, clk)
	settings := svc.settings
	updated := settings.GetSettings()
	updated.CooldownPeriodSeconds = 60
	if err := settings.UpdateSettings(updated); err != nil {
		t.Fatalf("set canonical cooldown: %v", err)
	}
	ctx := auth01FixtureCaller(t, context.Background())
	if _, err := svc.TriggerInvestigation(ctx, false, ""); err != nil {
		t.Fatalf("trigger investigation: %v", err)
	}
	status, err := svc.GetCooldownStatus(context.Background())
	if err != nil {
		t.Fatalf("GetCooldownStatus: %v", err)
	}
	if status.CooldownPeriodSeconds != 60 || status.RemainingSeconds != 60 {
		t.Fatalf("cooldown status = %+v, want 60 second canonical cooldown", status)
	}
	if err := svc.UpdateCooldownPeriod(context.Background(), 120); err != nil {
		t.Fatalf("UpdateCooldownPeriod: %v", err)
	}
	if got := settings.GetSettings().CooldownPeriodSeconds; got != 120 {
		t.Fatalf("canonical cooldown = %d after compatibility update, want 120", got)
	}
}

func TestLegacyTriggerEndpointsReturnMigrationError(t *testing.T) {
	clk := NewStubClock(time.Date(2026, 1, 1, 0, 0, 0, 0, time.UTC))
	svc := newTestInvestigationService(t, clk)

	enabled := false
	threshold := 95.0
	if _, err := svc.GetTriggers(context.Background()); err == nil || !strings.Contains(err.Error(), "Notification Hub") {
		t.Fatalf("GetTriggers error = %v, want migration guidance", err)
	}
	if err := svc.UpdateTrigger(context.Background(), "high_cpu", &enabled, nil, &threshold); err == nil || !strings.Contains(err.Error(), "System Settings") {
		t.Fatalf("UpdateTrigger error = %v, want migration guidance", err)
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

// panicAgentExecutor is a test double whose Execute method panics.
type panicAgentExecutor struct{ mocks.AgentExecutor }

func (p *panicAgentExecutor) Execute(_ context.Context, _ agentmanager.ExecuteRequest) (*agentmanager.ExecuteResult, error) {
	panic("boom")
}

func TestPanicRecovery_MarksInvestigationFailed(t *testing.T) {
	clk := NewStubClock(time.Date(2026, 1, 1, 0, 0, 0, 0, time.UTC))
	repo := memory.NewRepository()
	cfg := &config.Config{}
	agent := &panicAgentExecutor{*mocks.NewAgentExecutor().WithAvailable(true)}

	svc, err := NewInvestigationService(cfg, repo, nil, agent, NewSettingsManager(WithSettingsConfigStore(NewMemoryConfigStore())), WithInvestigationClock(clk))
	if err != nil {
		t.Fatal(err)
	}

	inv, err := svc.TriggerInvestigation(auth01FixtureCaller(t, context.Background()), false, "")
	if err != nil {
		t.Fatalf("trigger failed: %v", err)
	}

	// Wait for the goroutine to complete (panic + recovery)
	deadline := time.Now().Add(5 * time.Second)
	for time.Now().Before(deadline) {
		got, getErr := repo.GetInvestigation(context.Background(), inv.ID)
		if getErr == nil && got.Status == models.StatusFailed {
			// Verify findings mention the panic
			if !strings.Contains(got.Findings, "internal error") {
				t.Errorf("expected findings to mention internal error, got: %s", got.Findings)
			}
			return
		}
		time.Sleep(50 * time.Millisecond)
	}
	t.Fatal("investigation did not reach failed status after panic within timeout")
}

// Ensure the test double satisfies AgentExecutor at compile time.
var _ AgentExecutor = (*panicAgentExecutor)(nil)

func TestExecuteError_MarksInvestigationFailed(t *testing.T) {
	clk := NewStubClock(time.Date(2026, 1, 1, 0, 0, 0, 0, time.UTC))
	repo := memory.NewRepository()
	cfg := &config.Config{}
	agent := mocks.NewAgentExecutor().
		WithAvailable(true).
		WithExecuteError(context.DeadlineExceeded)

	svc, err := NewInvestigationService(cfg, repo, nil, agent, NewSettingsManager(WithSettingsConfigStore(NewMemoryConfigStore())), WithInvestigationClock(clk))
	if err != nil {
		t.Fatal(err)
	}

	inv, err := svc.TriggerInvestigation(auth01FixtureCaller(t, context.Background()), false, "")
	if err != nil {
		t.Fatalf("trigger failed: %v", err)
	}

	// Wait for the background goroutine to finish and mark status.
	deadline := time.Now().Add(5 * time.Second)
	for time.Now().Before(deadline) {
		got, getErr := repo.GetInvestigation(context.Background(), inv.ID)
		if getErr == nil && models.IsTerminalStatus(got.Status) {
			if got.Status != models.StatusFailed {
				t.Errorf("expected failed status, got %s", got.Status)
			}
			return
		}
		time.Sleep(50 * time.Millisecond)
	}
	t.Fatal("investigation did not reach failed status after agent error within timeout")
}

// failCreateRepo wraps a real repo but forces CreateInvestigation to fail.
type failCreateRepo struct {
	repository.InvestigationRepository
}

func (f *failCreateRepo) CreateInvestigation(_ context.Context, _ *models.Investigation) error {
	return fmt.Errorf("disk full")
}

func TestAddInvestigationStep_NotFound(t *testing.T) {
	clk := NewStubClock(time.Date(2026, 1, 1, 0, 0, 0, 0, time.UTC))
	svc := newTestInvestigationService(t, clk)

	ctx := context.Background()
	step := models.InvestigationStep{Name: "test-step", Status: "running"}
	err := svc.AddInvestigationStep(ctx, "nonexistent-id", step)
	if err == nil {
		t.Fatal("expected error for nonexistent investigation")
	}
	// Verify it's a structured NotFound error
	var apiErr *apierrors.APIError
	if !errors.As(err, &apiErr) {
		t.Fatalf("expected *apierrors.APIError, got %T: %v", err, err)
	}
	if apiErr.Category != apierrors.CategoryNotFound {
		t.Errorf("category = %q, want %q", apiErr.Category, apierrors.CategoryNotFound)
	}
}

func TestAddInvestigationStep_Success(t *testing.T) {
	clk := NewStubClock(time.Date(2026, 1, 1, 0, 0, 0, 0, time.UTC))
	svc := newTestInvestigationService(t, clk)

	ctx := context.Background()

	// First create an investigation
	inv, err := svc.TriggerInvestigation(auth01FixtureCaller(t, ctx), false, "")
	if err != nil {
		t.Fatalf("trigger failed: %v", err)
	}

	// Add a step
	step := models.InvestigationStep{Name: "check-cpu", Status: "running"}
	if err := svc.AddInvestigationStep(ctx, inv.ID, step); err != nil {
		t.Fatalf("add step failed: %v", err)
	}

	// Verify step was added
	got, err := svc.GetInvestigation(ctx, inv.ID)
	if err != nil {
		t.Fatalf("get investigation failed: %v", err)
	}
	if len(got.Steps) == 0 {
		t.Fatal("expected at least one step")
	}
	found := false
	for _, s := range got.Steps {
		if s.Name == "check-cpu" {
			found = true
			break
		}
	}
	if !found {
		t.Error("expected step 'check-cpu' in investigation steps")
	}
}

func TestTriggerInvestigation_RepoFailureDoesNotConsumeCooldown(t *testing.T) {
	clk := NewStubClock(time.Date(2026, 1, 1, 0, 0, 0, 0, time.UTC))
	cfg := &config.Config{}
	realRepo := memory.NewRepository()
	badRepo := &failCreateRepo{realRepo}
	agent := mocks.NewAgentExecutor().WithAvailable(true)

	svc, err := NewInvestigationService(cfg, badRepo, nil, agent, NewSettingsManager(WithSettingsConfigStore(NewMemoryConfigStore())), WithInvestigationClock(clk))
	if err != nil {
		t.Fatal(err)
	}

	ctx := context.Background()

	// First trigger fails because the repo rejects CreateInvestigation.
	_, err = svc.TriggerInvestigation(auth01FixtureCaller(t, ctx), false, "")
	if err == nil {
		t.Fatal("expected error from failing repo, got nil")
	}

	// Swap in a working repo so the next trigger can succeed.
	svc.repo = realRepo

	// The cooldown must NOT have been consumed by the failed attempt.
	inv, err := svc.TriggerInvestigation(auth01FixtureCaller(t, ctx), false, "")
	if err != nil {
		t.Fatalf("second trigger should succeed (no cooldown consumed), got: %v", err)
	}
	if inv == nil {
		t.Fatal("expected investigation, got nil")
	}
}
