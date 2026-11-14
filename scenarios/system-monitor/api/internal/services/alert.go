package services

// DOC: docs/concepts/ARCHITECTURE.md#alerting

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"sync"
	"time"

	"github.com/vrooli/api-core/eventbus"
	"github.com/vrooli/vrooli/scenarios/system-monitor/api/internal/config"
	"github.com/vrooli/vrooli/scenarios/system-monitor/api/internal/logx"
	"github.com/vrooli/vrooli/scenarios/system-monitor/api/internal/models"
	"github.com/vrooli/vrooli/scenarios/system-monitor/api/internal/repository"
)

// AlertService manages alerts and notifications
type AlertService struct {
	logger         logx.Logger
	config         *config.Config
	repo           repository.AlertRepository
	events         domainEventPublisher
	clock          Clock
	httpClient     HTTPDoer
	cooldown       time.Duration
	mu             sync.RWMutex
	cooldowns      map[string]time.Time
	notificationWG sync.WaitGroup
}

// AlertOption configures an AlertService.
type AlertOption func(*AlertService)

// AlertTransition is one persisted alert lifecycle edge or point-in-time
// event. Threshold records use occurred because they are already resolved
// when written; condition alerts move through opened, severity_changed,
// escalated, and resolved.
type AlertTransition string

const (
	AlertTransitionOpened         AlertTransition = "opened"
	AlertTransitionOccurred       AlertTransition = "occurred"
	AlertTransitionSeverityChange AlertTransition = "severity_changed"
	AlertTransitionEscalated      AlertTransition = "escalated"
	AlertTransitionResolved       AlertTransition = "resolved"
)

type domainEventPublisher interface {
	PublishDomainEvent(context.Context, eventbus.DomainEvent) error
}

// WithAlertClock sets the clock used by the alert service.
func WithAlertClock(c Clock) AlertOption {
	return func(s *AlertService) { s.clock = c }
}

// WithAlertLogger sets the structured logger used for alert operations.
func WithAlertLogger(logger logx.Logger) AlertOption {
	return func(s *AlertService) {
		if logger != nil {
			s.logger = logger
		}
	}
}

// WithAlertEventPublisher sets the Vrooli Events publisher used for
// Notification Hub requests. A nil publisher keeps alerts durable while
// notifications are unavailable.
func WithAlertEventPublisher(p domainEventPublisher) AlertOption {
	return func(s *AlertService) { s.events = p }
}

// WithHTTPClient injects the optional legacy webhook transport.
func WithHTTPClient(c HTTPDoer) AlertOption {
	return func(s *AlertService) {
		if c != nil {
			s.httpClient = c
		}
	}
}

// NewAlertService creates a new alert service.
func NewAlertService(cfg *config.Config, repo repository.AlertRepository, opts ...AlertOption) *AlertService {
	cooldown := 0
	if cfg != nil {
		cooldown = cfg.Alerts.CooldownMinutes
	}
	s := &AlertService{
		logger:     logx.Default(),
		config:     cfg,
		repo:       repo,
		clock:      RealClock{},
		httpClient: &http.Client{Timeout: 10 * time.Second},
		cooldown:   time.Duration(cooldown) * time.Minute,
		cooldowns:  make(map[string]time.Time),
	}
	for _, opt := range opts {
		opt(s)
	}
	return s
}

// CreateAlert creates a new alert
func (s *AlertService) CreateAlert(ctx context.Context, alert *models.Alert) error {
	// Generate ID if not set
	if alert.ID == "" {
		alert.ID = fmt.Sprintf("alert_%d", s.clock.Now().Unix())
	}

	// Check cooldown
	if !s.checkCooldown(alert.MetricName) {
		s.logger.Info("alert suppressed by cooldown", "metric", alert.MetricName)
		return nil
	}

	// Save to repository
	if err := s.repo.CreateAlert(ctx, alert); err != nil {
		return fmt.Errorf("failed to create alert: %w", err)
	}

	transition := AlertTransitionOpened
	if alert.ResolvedAt != nil {
		transition = AlertTransitionOccurred
	}
	s.NotifyTransition(ctx, alert, transition)
	// Update cooldown
	s.updateCooldown(alert.MetricName)

	return nil
}

// OpenConditionAlert persists a lifecycle-managed condition alert and sends
// notifications. Unlike CreateAlert it applies no per-metric cooldown: the
// evaluator already guarantees one open alert per rule/key, and a cooldown
// here would silently drop the record the evaluator then believes is open.
func (s *AlertService) OpenConditionAlert(ctx context.Context, alert *models.Alert) error {
	if err := s.repo.CreateAlert(ctx, alert); err != nil {
		return fmt.Errorf("failed to create alert: %w", err)
	}
	s.NotifyTransition(ctx, alert, AlertTransitionOpened)
	return nil
}

// NotifyTransition publishes a persisted alert lifecycle transition to
// Notification Hub. Alert state remains authoritative here; delivery
// preferences and channels belong to Notification Hub.
func (s *AlertService) NotifyTransition(ctx context.Context, alert *models.Alert, transition AlertTransition) {
	if alert == nil {
		return
	}
	category := alertNotificationCategory(alert)
	if transition == AlertTransitionEscalated {
		category = "alert.investigation"
	}
	if (transition == AlertTransitionOpened || transition == AlertTransitionOccurred) && s.shouldNotify(alert) {
		s.notificationWG.Add(1)
		go func() {
			defer s.notificationWG.Done()
			s.sendNotifications(alert)
		}()
	}
	if category == "" {
		return
	}
	urgency := "normal"
	if alert.Severity == SeverityCritical || alert.Severity == "high" {
		urgency = "high"
	}
	title := "System Monitor alert"
	switch transition {
	case AlertTransitionOccurred:
		title = "System Monitor threshold event"
	case AlertTransitionResolved:
		title = "System Monitor alert resolved"
	case AlertTransitionEscalated:
		title = "System Monitor alert escalated"
	}
	body := alert.Message
	if body == "" {
		body = alert.MetricName + " alert " + string(transition)
	}
	correlation := map[string]string{"alert_id": alert.ID, "transition": string(transition)}
	if ruleID, ok := alert.Details["rule_id"].(string); ok && ruleID != "" {
		correlation["rule_id"] = ruleID
	}
	s.publishNotification(ctx, title, body, category, urgency, alert.ID+":"+string(transition), correlation)
}

// WaitNotifications joins webhook delivery workers that were admitted before
// the lifecycle fence closed. Callers must first stop alert producers and drain
// admitted requests so no new worker can be added while waiting.
func (s *AlertService) WaitNotifications(ctx context.Context) error {
	done := make(chan struct{})
	go func() {
		s.notificationWG.Wait()
		close(done)
	}()
	select {
	case <-done:
		return nil
	case <-ctx.Done():
		return ctx.Err()
	}
}

// NotifySignal publishes an operator-worthy signal that has no alert row,
// such as a storage-writer episode. The caller owns transition detection and
// must pass a stable subject key so repeat observations deduplicate.
func (s *AlertService) NotifySignal(ctx context.Context, title, body, category, urgency, dedupeKey string, correlation map[string]string) {
	s.publishNotification(ctx, title, body, category, urgency, dedupeKey, correlation)
}

func (s *AlertService) publishNotification(ctx context.Context, title, body, category, urgency, dedupeKey string, correlation map[string]string) {
	if s == nil || s.events == nil {
		return
	}
	payload := map[string]any{
		"title": title, "body": body, "category": category,
		"urgency": urgency, "sensitivity": "private", "dedupe_key": dedupeKey,
		"correlation": correlation,
	}
	if err := s.events.PublishDomainEvent(ctx, eventbus.DomainEvent{
		Source: "system-monitor", EventType: "system_monitor.alert.notification.v1",
		Occurred: s.clock.Now().UTC(), Payload: payload,
	}); err != nil {
		s.logger.Warn("failed to publish notification event", "dedupe_key", dedupeKey, "error", err)
	}
}

// shouldNotify preserves optional direct webhook delivery. Per-category
// preferences for first-party operator notifications remain in Notification
// Hub and do not change this external integration.
func (s *AlertService) shouldNotify(alert *models.Alert) bool {
	if s == nil || s.config == nil || alert == nil {
		return false
	}
	severity, ok := map[string]int{"low": 1, "warning": 2, "medium": 2, "high": 3, "critical": 4}[alert.Severity]
	if !ok {
		severity = 1
	}
	return severity >= s.config.Alerts.MinSeverity
}

func (s *AlertService) sendNotifications(alert *models.Alert) {
	if s.config.Alerts.EnableWebhooks && s.config.Alerts.WebhookURL != "" {
		s.sendWebhook(alert)
	}
	if s.config.Alerts.EnableEmail {
		s.logger.Warn("email alert delivery is not implemented", "alert_id", alert.ID)
	}
}

func (s *AlertService) sendWebhook(alert *models.Alert) {
	data, err := json.Marshal(map[string]any{
		"type": "system_alert", "alert": alert, "timestamp": s.clock.Now(),
		"system_name": s.config.Server.ServiceName,
	})
	if err != nil {
		s.logger.Error("failed to marshal webhook payload", "error", err)
		return
	}
	req, err := http.NewRequest(http.MethodPost, s.config.Alerts.WebhookURL, bytes.NewReader(data))
	if err != nil {
		s.logger.Error("failed to create webhook request", "error", err)
		return
	}
	req.Header.Set("Content-Type", "application/json")
	resp, err := s.httpClient.Do(req)
	if err != nil {
		s.logger.Error("failed to send webhook", "error", err, "alert_id", alert.ID)
		return
	}
	defer resp.Body.Close()
	if resp.StatusCode >= http.StatusBadRequest {
		s.logger.Warn("webhook returned error status", "status_code", resp.StatusCode, "alert_id", alert.ID)
	}
}

func alertNotificationCategory(alert *models.Alert) string {
	ruleID, _ := alert.Details["rule_id"].(string)
	if ruleID == "" {
		ruleID = alert.MetricName
	}
	switch ruleID {
	case RuleMetricsSnapshot, RuleCPUUsage, RuleMemoryUsage, RuleTCPConnections,
		RuleGPUUsage, RuleMemoryPressure, RuleOrphanedAgents:
		return "alert." + ruleID
	case "storage_writer_hot", "storage_writer_cool":
		return "alert.storage_writer"
	case "disk_used_percent", "disk_space", "disk_pressure":
		return "alert.disk_pressure"
	case "cpu":
		return "alert.cpu_usage"
	case "anomaly":
		return "alert.investigation"
	default:
		if alert.Type == "anomaly" {
			return "alert.investigation"
		}
		return ""
	}
}

// SendAnomaly records an anomaly as a point-in-time event alert. Event
// records are created resolved: the open/resolved lifecycle belongs to the
// condition alert that raised the anomaly, not to the event.
func (s *AlertService) SendAnomaly(ctx context.Context, anomaly *models.Anomaly) error {
	at := anomaly.DetectedAt
	alert := &models.Alert{
		ID:         fmt.Sprintf("alert_%s_%d", anomaly.Type, s.clock.Now().UnixNano()),
		Type:       "anomaly",
		Severity:   anomaly.Severity,
		Message:    anomaly.Description,
		MetricName: anomaly.Type,
		Details:    anomaly.MetricData,
		Timestamp:  anomaly.DetectedAt,
		ResolvedAt: &at,
	}
	// No per-metric cooldown: an anomaly is raised once per escalated
	// condition alert, and the cooldown map is keyed by metric names that
	// threshold events share.
	// The condition transition already publishes the operator notification.
	// Keep the anomaly event durable without sending a duplicate notice.
	return s.repo.CreateAlert(ctx, alert)
}

// SendThresholdViolation sends an alert for a threshold violation
func (s *AlertService) SendThresholdViolation(ctx context.Context, violation *models.ThresholdViolation) error {
	resolvedAt := violation.Timestamp
	alert := &models.Alert{
		ID:          fmt.Sprintf("alert_threshold_%s_%d", violation.MetricName, s.clock.Now().Unix()),
		Type:        "threshold_violation",
		Severity:    violation.Severity,
		Message:     fmt.Sprintf("%s threshold violated: %.2f (threshold: %.2f)", violation.MetricName, violation.CurrentValue, violation.ThresholdValue),
		MetricName:  violation.MetricName,
		MetricValue: violation.CurrentValue,
		Details: map[string]interface{}{
			"violation_type": violation.ViolationType,
			"trend":          violation.Trend,
			"duration":       violation.Duration,
		},
		Timestamp: violation.Timestamp,
		// A violation is a band-transition event, not an open condition;
		// condition alerts (AlertEvaluator) own the open/resolved lifecycle.
		// Recording it resolved keeps event history without leaving rows
		// that never close.
		ResolvedAt: &resolvedAt,
	}

	return s.CreateAlert(ctx, alert)
}

// GetActiveAlerts retrieves all active (unresolved) alerts
func (s *AlertService) GetActiveAlerts(ctx context.Context) ([]*models.Alert, error) {
	return s.repo.GetActiveAlerts(ctx)
}

// AcknowledgeAlert marks an alert as acknowledged
func (s *AlertService) AcknowledgeAlert(ctx context.Context, id string, ackedBy string) error {
	return s.repo.AcknowledgeAlert(ctx, id, ackedBy)
}

// ResolveAlert marks an alert as resolved
func (s *AlertService) ResolveAlert(ctx context.Context, id string) error {
	return s.repo.ResolveAlert(ctx, id)
}

// checkCooldown checks if an alert is in cooldown period
func (s *AlertService) checkCooldown(metricName string) bool {
	s.mu.RLock()
	lastAlert, exists := s.cooldowns[metricName]
	s.mu.RUnlock()

	if !exists {
		return true
	}

	return s.clock.Since(lastAlert) > s.cooldown
}

// updateCooldown updates the cooldown timestamp for a metric
func (s *AlertService) updateCooldown(metricName string) {
	s.mu.Lock()
	s.cooldowns[metricName] = s.clock.Now()
	s.mu.Unlock()
}

// CleanupCooldowns removes expired cooldown entries
func (s *AlertService) CleanupCooldowns() {
	s.mu.Lock()
	defer s.mu.Unlock()

	now := s.clock.Now()

	for metric, lastAlert := range s.cooldowns {
		if now.Sub(lastAlert) > s.cooldown {
			delete(s.cooldowns, metric)
		}
	}
}
