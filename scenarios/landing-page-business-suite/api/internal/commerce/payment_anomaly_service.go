package commerce

import (
	"context"
	"database/sql"
	"encoding/json"
	"fmt"
	"strings"
	"sync/atomic"
	"time"
)

const (
	defaultAnomalySeverity        = "warn"
	defaultAnomalyRateLimitBurst  = 5
	defaultAnomalyRateLimitRefill = 60
)

// PaymentAnomaly is an anomaly record produced by commerce workflows.
type PaymentAnomaly struct {
	Type        string
	Severity    string
	Email       string
	CustomerID  string
	SubjectID   string
	SubjectKind string
	Details     map[string]interface{}
}

// PaymentAnomalyRuntime supplies application-specific behavior without making
// payment policy import the API composition package.
type PaymentAnomalyRuntime struct {
	ScenarioName   string
	NormalizeEmail func(string) string
	Log            func(string, map[string]interface{})
	LogError       func(string, map[string]interface{})
}

// PaymentAnomalyDispatchConfig is an immutable observation of the active
// dispatch policy. It lets API composition and diagnostics inspect the
// refreshed configuration without reaching into service internals.
type PaymentAnomalyDispatchConfig struct {
	WebhookURL string
	Enabled    bool
	RateLimits map[string]AnomalyRateLimit
}

type anomalyConfig struct {
	webhookURL string
	enabled    bool
	rateLimits map[string]AnomalyRateLimit
}

// PaymentAnomalyService persists anomalies and dispatches configured alerts.
type PaymentAnomalyService struct {
	db          PaymentAnomalyStore
	dispatcher  *AnomalyAlertDispatcher
	runtime     PaymentAnomalyRuntime
	cfg         atomic.Pointer[anomalyConfig]
	shutdownCtx context.Context
}

// NewPaymentAnomalyService initializes the service with the production
// dispatcher. shutdownCtx intentionally differs from ctx: ctx bounds initial
// configuration I/O while shutdownCtx owns the service's asynchronous life.
//
//nolint:revive // Two independently scoped contexts are required by this constructor.
func NewPaymentAnomalyService(ctx context.Context, db PaymentAnomalyStore, shutdownCtx context.Context, runtime PaymentAnomalyRuntime) *PaymentAnomalyService {
	return NewPaymentAnomalyServiceWithDispatcher(ctx, db, shutdownCtx, runtime, nil)
}

// NewPaymentAnomalyServiceWithDispatcher supports composition with a supplied
// dispatcher, including deterministic integration tests. The two contexts
// deliberately have separate lifetimes; see NewPaymentAnomalyService.
//
//nolint:revive // Two independently scoped contexts are required by this constructor.
func NewPaymentAnomalyServiceWithDispatcher(ctx context.Context, db PaymentAnomalyStore, shutdownCtx context.Context, runtime PaymentAnomalyRuntime, dispatcher *AnomalyAlertDispatcher) *PaymentAnomalyService {
	if shutdownCtx == nil {
		shutdownCtx = ctx
	}
	if shutdownCtx == nil {
		shutdownCtx = context.TODO()
	}
	if dispatcher == nil {
		dispatcher = NewAnomalyAlertDispatcher(db, DispatcherRuntime{
			ScenarioName: runtime.ScenarioName,
			LogError:     runtime.LogError,
		})
	}
	service := &PaymentAnomalyService{
		db:          db,
		dispatcher:  dispatcher,
		runtime:     runtime,
		shutdownCtx: shutdownCtx,
	}
	service.cfg.Store(emptyAnomalyConfig())
	if err := service.RefreshConfig(ctx); err != nil {
		service.log("payment_anomaly_config_initial_load_failed", map[string]interface{}{
			"level": "warn",
			"error": err.Error(),
		})
	}
	return service
}

func (s *PaymentAnomalyService) RefreshConfig(ctx context.Context) error {
	row := s.db.QueryRowContext(ctx, `
		SELECT anomaly_webhook_url, anomaly_webhook_enabled, anomaly_rate_limits
		FROM payment_settings WHERE id = 1
	`)
	var url sql.NullString
	var enabled sql.NullBool
	var limits sql.NullString
	if err := row.Scan(&url, &enabled, &limits); err != nil {
		if err == sql.ErrNoRows {
			s.cfg.Store(emptyAnomalyConfig())
			return nil
		}
		return fmt.Errorf("load anomaly config: %w", err)
	}

	config := emptyAnomalyConfig()
	if url.Valid {
		config.webhookURL = strings.TrimSpace(url.String)
	}
	if enabled.Valid {
		config.enabled = enabled.Bool
	}
	if limits.Valid && strings.TrimSpace(limits.String) != "" {
		var raw map[string]AnomalyRateLimit
		if err := json.Unmarshal([]byte(limits.String), &raw); err == nil {
			for kind, limit := range raw {
				if limit.Burst > 0 && limit.RefillSeconds > 0 {
					config.rateLimits[kind] = limit
				}
			}
		}
	}
	s.cfg.Store(config)
	return nil
}

func (s *PaymentAnomalyService) Log(ctx context.Context, anomaly PaymentAnomaly) (int64, error) {
	if strings.TrimSpace(anomaly.Type) == "" {
		return 0, fmt.Errorf("anomaly type required")
	}
	severity := strings.TrimSpace(anomaly.Severity)
	if severity == "" {
		severity = defaultAnomalySeverity
	}
	email := strings.TrimSpace(anomaly.Email)
	if email != "" && s.runtime.NormalizeEmail != nil {
		email = s.runtime.NormalizeEmail(email)
	}
	details := anomaly.Details
	if details == nil {
		details = map[string]interface{}{}
	}
	detailsJSON, err := json.Marshal(details)
	if err != nil {
		return 0, fmt.Errorf("marshal details: %w", err)
	}

	config := s.currentConfig()
	shouldDispatch := config.enabled && config.webhookURL != ""
	rateLimited := shouldDispatch && !s.dispatcher.Allow(anomaly.Type, config.rateLimits, defaultAnomalyRateLimitBurst, defaultAnomalyRateLimitRefill)
	status := DispatchPending
	var dispatchErr sql.NullString
	switch {
	case !shouldDispatch:
		status = DispatchSkipped
		dispatchErr = sql.NullString{String: "dispatch_disabled", Valid: true}
	case rateLimited:
		status = DispatchSkipped
		dispatchErr = sql.NullString{String: "rate_limited", Valid: true}
	}

	var rowID int64
	err = s.db.QueryRowContext(ctx, `
		INSERT INTO payment_anomaly_log
			(anomaly_type, severity, email, customer_id, subject_id, subject_kind, details, dispatch_status, dispatch_error, created_at)
		VALUES ($1, $2, NULLIF($3, ''), NULLIF($4, ''), NULLIF($5, ''), NULLIF($6, ''), $7::jsonb, $8, $9, NOW())
		RETURNING id
	`, anomaly.Type, severity, email, anomaly.CustomerID, anomaly.SubjectID, anomaly.SubjectKind, string(detailsJSON), status, dispatchErr).Scan(&rowID)
	if err != nil {
		s.logError("payment_anomaly_log_insert_failed", map[string]interface{}{
			"anomaly_type": anomaly.Type,
			"email":        anomaly.Email,
			"error":        err.Error(),
		})
		return 0, err
	}

	s.log("payment_anomaly_logged", map[string]interface{}{
		"level":           "warn",
		"id":              rowID,
		"anomaly_type":    anomaly.Type,
		"severity":        severity,
		"email":           email,
		"subject_kind":    anomaly.SubjectKind,
		"subject_id":      anomaly.SubjectID,
		"dispatch_status": status,
	})
	if status == DispatchPending {
		go s.dispatcher.Dispatch(s.shutdownCtx, AnomalyDispatchPayload{
			ID: rowID, Type: anomaly.Type, Severity: severity, Email: email,
			CustomerID: anomaly.CustomerID, SubjectID: anomaly.SubjectID,
			SubjectKind: anomaly.SubjectKind, Details: details, CreatedAt: time.Now().UTC(),
			WebhookURL: config.webhookURL,
		})
	}
	return rowID, nil
}

func (s *PaymentAnomalyService) WaitForDispatch(ctx context.Context, rowID int64) (string, error) {
	ticker := time.NewTicker(25 * time.Millisecond)
	defer ticker.Stop()
	for {
		var status string
		err := s.db.QueryRowContext(ctx, `SELECT dispatch_status FROM payment_anomaly_log WHERE id = $1`, rowID).Scan(&status)
		if err == nil && status != DispatchPending {
			return status, nil
		}
		if err != nil && err != sql.ErrNoRows {
			return "", err
		}
		select {
		case <-ctx.Done():
			return "", ctx.Err()
		case <-ticker.C:
		}
	}
}

// DispatchConfig returns a copy of the current dispatch policy.
func (s *PaymentAnomalyService) DispatchConfig() PaymentAnomalyDispatchConfig {
	config := s.currentConfig()
	limits := make(map[string]AnomalyRateLimit, len(config.rateLimits))
	for kind, limit := range config.rateLimits {
		limits[kind] = limit
	}
	return PaymentAnomalyDispatchConfig{
		WebhookURL: config.webhookURL,
		Enabled:    config.enabled,
		RateLimits: limits,
	}
}

func (s *PaymentAnomalyService) currentConfig() *anomalyConfig {
	if config := s.cfg.Load(); config != nil {
		return config
	}
	return emptyAnomalyConfig()
}

func emptyAnomalyConfig() *anomalyConfig {
	return &anomalyConfig{rateLimits: map[string]AnomalyRateLimit{}}
}

func (s *PaymentAnomalyService) log(event string, fields map[string]interface{}) {
	if s.runtime.Log != nil {
		s.runtime.Log(event, fields)
	}
}

func (s *PaymentAnomalyService) logError(event string, fields map[string]interface{}) {
	if s.runtime.LogError != nil {
		s.runtime.LogError(event, fields)
	}
}
