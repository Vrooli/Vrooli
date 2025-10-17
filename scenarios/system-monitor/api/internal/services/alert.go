package services

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"log"
	"net/http"
	"sync"
	"time"

	"system-monitor-api/internal/config"
	"system-monitor-api/internal/models"
	"system-monitor-api/internal/repository"
)

// AlertService manages alerts and notifications
type AlertService struct {
	config     *config.Config
	repo       repository.AlertRepository
	httpClient *http.Client
	mu         sync.RWMutex
	cooldowns  map[string]time.Time
}

// NewAlertService creates a new alert service
func NewAlertService(cfg *config.Config, repo repository.AlertRepository) *AlertService {
	return &AlertService{
		config:     cfg,
		repo:       repo,
		httpClient: &http.Client{Timeout: 10 * time.Second},
		cooldowns:  make(map[string]time.Time),
	}
}

// CreateAlert creates a new alert
func (s *AlertService) CreateAlert(ctx context.Context, alert *models.Alert) error {
	// Generate ID if not set
	if alert.ID == "" {
		alert.ID = fmt.Sprintf("alert_%d", time.Now().Unix())
	}
	
	// Check cooldown
	if !s.checkCooldown(alert.MetricName) {
		log.Printf("Alert for %s is in cooldown period", alert.MetricName)
		return nil
	}
	
	// Save to repository
	if err := s.repo.CreateAlert(ctx, alert); err != nil {
		return fmt.Errorf("failed to create alert: %w", err)
	}
	
	// Send notifications based on severity
	if s.shouldNotify(alert) {
		go s.sendNotifications(alert)
	}
	
	// Update cooldown
	s.updateCooldown(alert.MetricName)
	
	return nil
}

// SendAnomaly sends an alert for an anomaly
func (s *AlertService) SendAnomaly(ctx context.Context, anomaly *models.Anomaly) error {
	alert := &models.Alert{
		ID:          fmt.Sprintf("alert_%s_%d", anomaly.Type, time.Now().Unix()),
		Type:        "anomaly",
		Severity:    anomaly.Severity,
		Message:     anomaly.Description,
		MetricName:  anomaly.Type,
		Details:     anomaly.MetricData,
		Timestamp:   anomaly.DetectedAt,
	}
	
	return s.CreateAlert(ctx, alert)
}

// SendThresholdViolation sends an alert for a threshold violation
func (s *AlertService) SendThresholdViolation(ctx context.Context, violation *models.ThresholdViolation) error {
	alert := &models.Alert{
		ID:          fmt.Sprintf("alert_threshold_%s_%d", violation.MetricName, time.Now().Unix()),
		Type:        "threshold_violation",
		Severity:    violation.Severity,
		Message:     fmt.Sprintf("%s threshold violated: %.2f (threshold: %.2f)", violation.MetricName, violation.CurrentValue, violation.ThresholdValue),
		MetricName:  violation.MetricName,
		MetricValue: violation.CurrentValue,
		Details: map[string]interface{}{
			"violation_type": violation.ViolationType,
			"trend":         violation.Trend,
			"duration":      violation.Duration,
		},
		Timestamp: violation.Timestamp,
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

// shouldNotify determines if an alert should trigger notifications
func (s *AlertService) shouldNotify(alert *models.Alert) bool {
	// Check minimum severity
	severityMap := map[string]int{
		"low":      1,
		"medium":   2,
		"high":     3,
		"critical": 4,
	}
	
	alertSeverity, ok := severityMap[alert.Severity]
	if !ok {
		alertSeverity = 1
	}
	
	return alertSeverity >= s.config.Alerts.MinSeverity
}

// sendNotifications sends alert notifications
func (s *AlertService) sendNotifications(alert *models.Alert) {
	// Send webhook notification
	if s.config.Alerts.EnableWebhooks && s.config.Alerts.WebhookURL != "" {
		s.sendWebhook(alert)
	}
	
	// Send email notification (if configured)
	if s.config.Alerts.EnableEmail {
		s.sendEmail(alert)
	}
}

// sendWebhook sends a webhook notification
func (s *AlertService) sendWebhook(alert *models.Alert) {
	payload := map[string]interface{}{
		"type":        "system_alert",
		"alert":       alert,
		"timestamp":   time.Now(),
		"system_name": s.config.Server.ServiceName,
	}
	
	data, err := json.Marshal(payload)
	if err != nil {
		log.Printf("Failed to marshal webhook payload: %v", err)
		return
	}
	
	resp, err := s.httpClient.Post(s.config.Alerts.WebhookURL, "application/json", bytes.NewBuffer(data))
	if err != nil {
		log.Printf("Failed to send webhook: %v", err)
		return
	}
	defer resp.Body.Close()
	
	if resp.StatusCode >= 400 {
		log.Printf("Webhook returned error status: %d", resp.StatusCode)
	}
}

// sendEmail sends an email notification
func (s *AlertService) sendEmail(alert *models.Alert) {
	// Email implementation would go here
	log.Printf("Email notification for alert %s (not implemented)", alert.ID)
}

// checkCooldown checks if an alert is in cooldown period
func (s *AlertService) checkCooldown(metricName string) bool {
	s.mu.RLock()
	lastAlert, exists := s.cooldowns[metricName]
	s.mu.RUnlock()
	
	if !exists {
		return true
	}
	
	cooldownDuration := time.Duration(s.config.Alerts.CooldownMinutes) * time.Minute
	return time.Since(lastAlert) > cooldownDuration
}

// updateCooldown updates the cooldown timestamp for a metric
func (s *AlertService) updateCooldown(metricName string) {
	s.mu.Lock()
	s.cooldowns[metricName] = time.Now()
	s.mu.Unlock()
}

// CleanupCooldowns removes expired cooldown entries
func (s *AlertService) CleanupCooldowns() {
	s.mu.Lock()
	defer s.mu.Unlock()
	
	cooldownDuration := time.Duration(s.config.Alerts.CooldownMinutes) * time.Minute
	now := time.Now()
	
	for metric, lastAlert := range s.cooldowns {
		if now.Sub(lastAlert) > cooldownDuration {
			delete(s.cooldowns, metric)
		}
	}
}