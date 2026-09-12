package services

import (
	"context"
	"testing"
	"time"

	"github.com/vrooli/vrooli/scenarios/system-monitor/api/internal/config"
	"github.com/vrooli/vrooli/scenarios/system-monitor/api/internal/models"
	"github.com/vrooli/vrooli/scenarios/system-monitor/api/internal/repository/memory"
)

func newTestAlertService(t *testing.T, clk Clock, cooldownMinutes int) *AlertService {
	t.Helper()
	cfg := &config.Config{
		Alerts: config.AlertConfig{
			CooldownMinutes: cooldownMinutes,
			MinSeverity:     1,
		},
	}
	repo := memory.NewRepository()
	return NewAlertService(cfg, repo, WithAlertClock(clk))
}

// twoAlertScenario describes a "create alert, advance the clock, create a second
// alert, then assert the active-alert count" test flow. It single-sources the
// boilerplate shared by the cooldown tests.
type twoAlertScenario struct {
	cooldownMinutes int
	firstAlert      *models.Alert
	advance         time.Duration
	secondAlert     *models.Alert
	wantActive      int
}

// runTwoAlertScenario executes the scenario and asserts the resulting active
// alert count, failing the test on any unexpected error.
func runTwoAlertScenario(t *testing.T, sc twoAlertScenario) {
	t.Helper()

	clk := NewStubClock(time.Date(2026, 1, 1, 0, 0, 0, 0, time.UTC))
	svc := newTestAlertService(t, clk, sc.cooldownMinutes)
	ctx := context.Background()

	if err := svc.CreateAlert(ctx, sc.firstAlert); err != nil {
		t.Fatalf("first alert failed: %v", err)
	}

	clk.Advance(sc.advance)

	if err := svc.CreateAlert(ctx, sc.secondAlert); err != nil {
		t.Fatalf("second alert failed: %v", err)
	}

	alerts, err := svc.GetActiveAlerts(ctx)
	if err != nil {
		t.Fatalf("get alerts failed: %v", err)
	}
	if len(alerts) != sc.wantActive {
		t.Errorf("expected %d active alerts, got %d", sc.wantActive, len(alerts))
	}
}

func TestAlertCooldownPreventsSpam(t *testing.T) {
	// Second alert lands inside the 5-minute cooldown and is silently dropped.
	runTwoAlertScenario(t, twoAlertScenario{
		cooldownMinutes: 5,
		firstAlert:      &models.Alert{MetricName: "cpu", Severity: "high", Message: "high cpu"},
		advance:         2 * time.Minute,
		secondAlert:     &models.Alert{MetricName: "cpu", Severity: "high", Message: "high cpu again"},
		wantActive:      1,
	})
}

func TestAlertCooldownExpires(t *testing.T) {
	// Second alert lands after the cooldown expires and is stored.
	runTwoAlertScenario(t, twoAlertScenario{
		cooldownMinutes: 5,
		firstAlert:      &models.Alert{MetricName: "memory", Severity: "critical", Message: "high memory"},
		advance:         6 * time.Minute,
		secondAlert:     &models.Alert{MetricName: "memory", Severity: "critical", Message: "high memory again"},
		wantActive:      2,
	})
}

func TestCleanupCooldowns(t *testing.T) {
	clk := NewStubClock(time.Date(2026, 1, 1, 0, 0, 0, 0, time.UTC))
	svc := newTestAlertService(t, clk, 5)

	ctx := context.Background()

	// Create alerts for multiple metrics
	for _, metric := range []string{"cpu", "memory", "disk"} {
		_ = svc.CreateAlert(ctx, &models.Alert{
			MetricName: metric,
			Severity:   "high",
			Message:    "test " + metric,
		})
	}

	// Verify cooldowns exist
	svc.mu.RLock()
	count := len(svc.cooldowns)
	svc.mu.RUnlock()
	if count != 3 {
		t.Fatalf("expected 3 cooldown entries, got %d", count)
	}

	// Advance past cooldown
	clk.Advance(6 * time.Minute)

	// Cleanup should remove all expired entries
	svc.CleanupCooldowns()

	svc.mu.RLock()
	count = len(svc.cooldowns)
	svc.mu.RUnlock()
	if count != 0 {
		t.Errorf("expected 0 cooldown entries after cleanup, got %d", count)
	}
}

func TestAlertDifferentMetricsNoCooldown(t *testing.T) {
	// Alerts for different metrics never share a cooldown, so both are stored.
	// The 1s advance gives the second alert a distinct auto-generated ID.
	runTwoAlertScenario(t, twoAlertScenario{
		cooldownMinutes: 5,
		firstAlert:      &models.Alert{MetricName: "cpu", Severity: "high", Message: "high cpu"},
		advance:         1 * time.Second,
		secondAlert:     &models.Alert{MetricName: "memory", Severity: "high", Message: "high memory"},
		wantActive:      2,
	})
}
