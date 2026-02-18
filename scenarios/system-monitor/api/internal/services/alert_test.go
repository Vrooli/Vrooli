package services

import (
	"context"
	"testing"
	"time"

	"system-monitor-api/internal/config"
	"system-monitor-api/internal/models"
	"system-monitor-api/internal/repository/memory"
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

func TestAlertCooldownPreventsSpam(t *testing.T) {
	clk := NewStubClock(time.Date(2026, 1, 1, 0, 0, 0, 0, time.UTC))
	svc := newTestAlertService(t, clk, 5) // 5-minute cooldown

	ctx := context.Background()

	// First alert should succeed
	err := svc.CreateAlert(ctx, &models.Alert{
		MetricName: "cpu",
		Severity:   "high",
		Message:    "high cpu",
	})
	if err != nil {
		t.Fatalf("first alert failed: %v", err)
	}

	// Advance 2 minutes - still in cooldown
	clk.Advance(2 * time.Minute)

	// Second alert for same metric - should be silently dropped (cooldown)
	err = svc.CreateAlert(ctx, &models.Alert{
		MetricName: "cpu",
		Severity:   "high",
		Message:    "high cpu again",
	})
	if err != nil {
		t.Fatalf("second alert should succeed (silently dropped): %v", err)
	}

	// Only 1 alert should be stored (second was dropped by cooldown)
	alerts, err := svc.GetActiveAlerts(ctx)
	if err != nil {
		t.Fatalf("get alerts failed: %v", err)
	}
	if len(alerts) != 1 {
		t.Errorf("expected 1 alert (cooldown dropped second), got %d", len(alerts))
	}
}

func TestAlertCooldownExpires(t *testing.T) {
	clk := NewStubClock(time.Date(2026, 1, 1, 0, 0, 0, 0, time.UTC))
	svc := newTestAlertService(t, clk, 5)

	ctx := context.Background()

	// First alert
	err := svc.CreateAlert(ctx, &models.Alert{
		MetricName: "memory",
		Severity:   "critical",
		Message:    "high memory",
	})
	if err != nil {
		t.Fatalf("first alert failed: %v", err)
	}

	// Advance past cooldown
	clk.Advance(6 * time.Minute)

	// Second alert should go through
	err = svc.CreateAlert(ctx, &models.Alert{
		MetricName: "memory",
		Severity:   "critical",
		Message:    "high memory again",
	})
	if err != nil {
		t.Fatalf("second alert (post-cooldown) failed: %v", err)
	}

	alerts, err := svc.GetActiveAlerts(ctx)
	if err != nil {
		t.Fatalf("get alerts failed: %v", err)
	}
	if len(alerts) != 2 {
		t.Errorf("expected 2 alerts, got %d", len(alerts))
	}
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
	clk := NewStubClock(time.Date(2026, 1, 1, 0, 0, 0, 0, time.UTC))
	svc := newTestAlertService(t, clk, 5)

	ctx := context.Background()

	// Alerts for different metrics should not block each other
	err := svc.CreateAlert(ctx, &models.Alert{
		MetricName: "cpu",
		Severity:   "high",
		Message:    "high cpu",
	})
	if err != nil {
		t.Fatalf("cpu alert failed: %v", err)
	}

	// Advance clock so second alert gets a different auto-generated ID
	clk.Advance(1 * time.Second)

	err = svc.CreateAlert(ctx, &models.Alert{
		MetricName: "memory",
		Severity:   "high",
		Message:    "high memory",
	})
	if err != nil {
		t.Fatalf("memory alert failed: %v", err)
	}

	alerts, _ := svc.GetActiveAlerts(ctx)
	if len(alerts) != 2 {
		t.Errorf("expected 2 alerts (different metrics), got %d", len(alerts))
	}
}
