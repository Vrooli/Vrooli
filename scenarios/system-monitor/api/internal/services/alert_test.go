package services

import (
	"context"
	"errors"
	"net/http"
	"testing"
	"time"

	"github.com/vrooli/api-core/eventbus"
	"github.com/vrooli/vrooli/scenarios/system-monitor/api/internal/config"
	"github.com/vrooli/vrooli/scenarios/system-monitor/api/internal/models"
	"github.com/vrooli/vrooli/scenarios/system-monitor/api/internal/repository/memory"
)

type recordingDomainPublisher struct {
	events []eventbus.DomainEvent
	err    error
}

type blockingAlertHTTPClient struct {
	started chan struct{}
	release chan struct{}
}

func (c blockingAlertHTTPClient) Do(*http.Request) (*http.Response, error) {
	close(c.started)
	<-c.release
	return &http.Response{StatusCode: http.StatusNoContent, Body: http.NoBody}, nil
}

func (p *recordingDomainPublisher) PublishDomainEvent(_ context.Context, event eventbus.DomainEvent) error {
	p.events = append(p.events, event)
	return p.err
}

func newTestAlertService(t *testing.T, clk Clock, cooldownMinutes int) *AlertService {
	t.Helper()
	cfg := &config.Config{
		Alerts: config.AlertConfig{
			CooldownMinutes: cooldownMinutes,
			MinSeverity:     2,
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

func TestNotifyTransitionPublishesNotificationHubContract(t *testing.T) {
	clock := NewStubClock(time.Date(2026, 11, 2, 12, 0, 0, 0, time.UTC))
	publisher := &recordingDomainPublisher{}
	svc := NewAlertService(&config.Config{}, memory.NewRepository(), WithAlertClock(clock), WithAlertEventPublisher(publisher))
	alert := &models.Alert{
		ID: "cond_cpu_usage_host_1", Type: ConditionAlertType, Severity: SeverityCritical,
		Message: "CPU usage is 98%.", MetricName: "cpu_usage_percent",
		Details: map[string]interface{}{"rule_id": RuleCPUUsage},
	}

	svc.NotifyTransition(context.Background(), alert, AlertTransitionEscalated)

	if len(publisher.events) != 1 {
		t.Fatalf("published events = %d, want 1", len(publisher.events))
	}
	event := publisher.events[0]
	if event.Source != "system-monitor" || event.EventType != "system_monitor.alert.notification.v1" {
		t.Fatalf("event identity = %s/%s", event.Source, event.EventType)
	}
	payload := event.Payload
	if payload["category"] != "alert.investigation" || payload["urgency"] != "high" || payload["sensitivity"] != "private" {
		t.Fatalf("notification preferences fields = %#v", payload)
	}
	if payload["dedupe_key"] != "cond_cpu_usage_host_1:escalated" {
		t.Fatalf("dedupe key = %v", payload["dedupe_key"])
	}
	correlation, ok := payload["correlation"].(map[string]string)
	if !ok || correlation["alert_id"] != alert.ID || correlation["rule_id"] != RuleCPUUsage {
		t.Fatalf("correlation = %#v", payload["correlation"])
	}
}

func TestNotifyTransitionDoesNothingWithoutPublisher(t *testing.T) {
	service := NewAlertService(&config.Config{}, memory.NewRepository())
	service.NotifyTransition(context.Background(), &models.Alert{ID: "a", MetricName: RuleCPUUsage}, AlertTransitionOpened)
}

func TestWaitNotificationsReportsUnjoinedWebhookUntilCompletion(t *testing.T) {
	client := blockingAlertHTTPClient{started: make(chan struct{}), release: make(chan struct{})}
	service := NewAlertService(&config.Config{Alerts: config.AlertConfig{
		MinSeverity: 2, EnableWebhooks: true, WebhookURL: "https://example.invalid/webhook",
	}}, memory.NewRepository(), WithHTTPClient(client))
	service.NotifyTransition(context.Background(), &models.Alert{ID: "a", Severity: "high", MetricName: "cpu"}, AlertTransitionOpened)
	select {
	case <-client.started:
	case <-time.After(time.Second):
		t.Fatal("webhook worker did not start")
	}

	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Millisecond)
	defer cancel()
	if err := service.WaitNotifications(ctx); !errors.Is(err, context.DeadlineExceeded) {
		t.Fatalf("WaitNotifications error = %v, want timeout while webhook is active", err)
	}
	close(client.release)
	if err := service.WaitNotifications(context.Background()); err != nil {
		t.Fatalf("WaitNotifications after delivery completion: %v", err)
	}
}

func TestThresholdEventPublishesOccurredTransition(t *testing.T) {
	clock := NewStubClock(time.Date(2026, 11, 2, 12, 0, 0, 0, time.UTC))
	publisher := &recordingDomainPublisher{}
	repo := memory.NewRepository()
	cfg := &config.Config{Alerts: config.AlertConfig{MinSeverity: 2}}
	service := NewAlertService(cfg, repo, WithAlertClock(clock), WithAlertEventPublisher(publisher))
	if err := service.SendThresholdViolation(context.Background(), &models.ThresholdViolation{
		MetricName: "disk_used_percent", CurrentValue: 96, Severity: "critical", Timestamp: clock.Now(),
	}); err != nil {
		t.Fatalf("send threshold event: %v", err)
	}
	if len(publisher.events) != 1 {
		t.Fatalf("published events = %d, want 1", len(publisher.events))
	}
	payload := publisher.events[0].Payload
	if payload["category"] != "alert.disk_pressure" || payload["title"] != "System Monitor threshold event" {
		t.Fatalf("threshold notification = %#v", payload)
	}
	correlation, ok := payload["correlation"].(map[string]string)
	if !ok || correlation["transition"] != string(AlertTransitionOccurred) {
		t.Fatalf("threshold transition = %#v", payload["correlation"])
	}
}
