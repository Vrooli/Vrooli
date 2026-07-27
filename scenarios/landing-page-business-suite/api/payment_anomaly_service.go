package main

import (
	"context"
	"database/sql"
	"encoding/json"
	"fmt"
	"strings"
	"sync/atomic"
	"time"
)

// Scenario identifier embedded in every dispatched anomaly payload so that a
// single monitoring endpoint shared across scenarios can route alerts.
const paymentAnomalyScenarioName = "landing-page-business-suite"

// Dispatch status values stored on payment_anomaly_log.dispatch_status.
const (
	anomalyDispatchPending = "pending"
	anomalyDispatchSent    = "sent"
	anomalyDispatchSkipped = "skipped"
	anomalyDispatchFailed  = "failed"
)

// anomalySeverity default when caller omits severity.
const anomalySeverityWarn = "warn"

// defaultAnomalyRateLimitBurst + refill bound alert rate per type when no
// override is configured in payment_settings.anomaly_rate_limits.
const (
	defaultAnomalyRateLimitBurst   = 5
	defaultAnomalyRateLimitRefillS = 60
)

// PaymentAnomaly is the input record callers build and pass to Log. All
// string fields are optional except Type; Details is marshalled to JSONB.
type PaymentAnomaly struct {
	Type        string
	Severity    string
	Email       string
	CustomerID  string
	SubjectID   string
	SubjectKind string
	Details     map[string]interface{}
}

// anomalyRateLimit captures a per-type override loaded from JSONB.
type anomalyRateLimit struct {
	Burst         int `json:"burst"`
	RefillSeconds int `json:"refill_seconds"`
}

// anomalyConfig is an immutable snapshot of the dispatch-relevant portion of
// payment_settings. Readers load it via atomic.Pointer; writers replace it
// wholesale on RefreshConfig.
type anomalyConfig struct {
	webhookURL string
	enabled    bool
	rateLimits map[string]anomalyRateLimit
}

// PaymentAnomalyStore is the context-aware persistence contract for anomaly
// records, configuration, and dispatch state.
//
// seam: PaymentAnomalyStore keeps payment anomaly persistence independent of a
// concrete pool and preserves request-scoped test isolation.
type PaymentAnomalyStore interface {
	QueryRowContext(context.Context, string, ...any) *sql.Row
	ExecContext(context.Context, string, ...any) (sql.Result, error)
}

// PaymentAnomalyService records anomalies and triggers webhook alerts.
type PaymentAnomalyService struct {
	db          PaymentAnomalyStore
	dispatcher  *AnomalyAlertDispatcher
	cfg         atomic.Pointer[anomalyConfig]
	shutdownCtx context.Context
}

// NewPaymentAnomalyService wires the service and loads its initial config.
// shutdownCtx is the server lifetime context used for background dispatch
// goroutines so an in-flight POST aborts cleanly on server shutdown. Two
// context parameters are intentional: ctx scopes the initial config load,
// shutdownCtx scopes long-lived background dispatch goroutines.
//
//revive:disable-next-line:context-as-argument
func NewPaymentAnomalyService(ctx context.Context, db PaymentAnomalyStore, shutdownCtx context.Context) *PaymentAnomalyService {
	if shutdownCtx == nil {
		shutdownCtx = ctx
	}
	if shutdownCtx == nil {
		shutdownCtx = context.TODO()
	}
	s := &PaymentAnomalyService{
		db:          db,
		dispatcher:  NewAnomalyAlertDispatcher(db),
		shutdownCtx: shutdownCtx,
	}
	s.cfg.Store(&anomalyConfig{rateLimits: map[string]anomalyRateLimit{}})
	if err := s.RefreshConfig(ctx); err != nil {
		logStructured("payment_anomaly_config_initial_load_failed", map[string]interface{}{
			"level": "warn",
			"error": err.Error(),
		})
	}
	return s
}

// RefreshConfig reloads the dispatch-relevant fields from payment_settings
// and atomically swaps the in-memory snapshot. Called at startup and after
// every successful admin save.
func (s *PaymentAnomalyService) RefreshConfig(ctx context.Context) error {
	row := s.db.QueryRowContext(ctx, `
		SELECT anomaly_webhook_url, anomaly_webhook_enabled, anomaly_rate_limits
		FROM payment_settings WHERE id = 1
	`)
	var (
		url     sql.NullString
		enabled sql.NullBool
		limits  sql.NullString
	)
	if err := row.Scan(&url, &enabled, &limits); err != nil {
		if err == sql.ErrNoRows {
			s.cfg.Store(&anomalyConfig{rateLimits: map[string]anomalyRateLimit{}})
			return nil
		}
		return fmt.Errorf("load anomaly config: %w", err)
	}

	cfg := &anomalyConfig{rateLimits: map[string]anomalyRateLimit{}}
	if url.Valid {
		cfg.webhookURL = strings.TrimSpace(url.String)
	}
	if enabled.Valid {
		cfg.enabled = enabled.Bool
	}
	if limits.Valid && strings.TrimSpace(limits.String) != "" {
		raw := map[string]anomalyRateLimit{}
		if err := json.Unmarshal([]byte(limits.String), &raw); err == nil {
			for k, v := range raw {
				if v.Burst > 0 && v.RefillSeconds > 0 {
					cfg.rateLimits[k] = v
				}
			}
		}
	}
	s.cfg.Store(cfg)
	return nil
}

// currentConfig returns the active snapshot. Never nil — the service always
// publishes a (possibly empty) config.
func (s *PaymentAnomalyService) currentConfig() *anomalyConfig {
	c := s.cfg.Load()
	if c == nil {
		return &anomalyConfig{rateLimits: map[string]anomalyRateLimit{}}
	}
	return c
}

// Log inserts a row into payment_anomaly_log and (if dispatch is enabled and
// within rate limits) fires a background webhook POST. Returns the new row
// id. Callers should treat the id as opaque; admin tooling can correlate via
// payment_anomaly_log.
func (s *PaymentAnomalyService) Log(ctx context.Context, a PaymentAnomaly) (int64, error) {
	if strings.TrimSpace(a.Type) == "" {
		return 0, fmt.Errorf("anomaly type required")
	}
	severity := strings.TrimSpace(a.Severity)
	if severity == "" {
		severity = anomalySeverityWarn
	}
	email := ""
	if a.Email != "" {
		email = NormalizeEmail(a.Email)
	}

	details := a.Details
	if details == nil {
		details = map[string]interface{}{}
	}
	detailsJSON, err := json.Marshal(details)
	if err != nil {
		return 0, fmt.Errorf("marshal details: %w", err)
	}

	cfg := s.currentConfig()
	shouldDispatch := cfg.enabled && cfg.webhookURL != ""
	rateLimited := false
	if shouldDispatch {
		rateLimited = !s.dispatcher.Allow(a.Type, cfg)
	}

	initialStatus := anomalyDispatchPending
	var initialErr sql.NullString
	switch {
	case !shouldDispatch:
		initialStatus = anomalyDispatchSkipped
		initialErr = sql.NullString{String: "dispatch_disabled", Valid: true}
	case rateLimited:
		initialStatus = anomalyDispatchSkipped
		initialErr = sql.NullString{String: "rate_limited", Valid: true}
	}

	var rowID int64
	insertErr := s.db.QueryRowContext(ctx, `
		INSERT INTO payment_anomaly_log
			(anomaly_type, severity, email, customer_id, subject_id, subject_kind, details, dispatch_status, dispatch_error, created_at)
		VALUES ($1, $2, NULLIF($3, ''), NULLIF($4, ''), NULLIF($5, ''), NULLIF($6, ''), $7::jsonb, $8, $9, NOW())
		RETURNING id
	`, a.Type, severity, email, a.CustomerID, a.SubjectID, a.SubjectKind, string(detailsJSON), initialStatus, initialErr).Scan(&rowID)
	if insertErr != nil {
		logStructuredError("payment_anomaly_log_insert_failed", map[string]interface{}{
			"anomaly_type": a.Type,
			"email":        a.Email,
			"error":        insertErr.Error(),
		})
		return 0, insertErr
	}

	logStructured("payment_anomaly_logged", map[string]interface{}{
		"level":           "warn",
		"id":              rowID,
		"anomaly_type":    a.Type,
		"severity":        severity,
		"email":           email,
		"subject_kind":    a.SubjectKind,
		"subject_id":      a.SubjectID,
		"dispatch_status": initialStatus,
	})

	if initialStatus == anomalyDispatchPending {
		payload := anomalyDispatchPayload{
			ID:          rowID,
			Type:        a.Type,
			Severity:    severity,
			Email:       email,
			CustomerID:  a.CustomerID,
			SubjectID:   a.SubjectID,
			SubjectKind: a.SubjectKind,
			Details:     details,
			CreatedAt:   time.Now().UTC(),
			WebhookURL:  cfg.webhookURL,
		}
		go s.dispatcher.Dispatch(s.shutdownCtx, payload)
	}

	return rowID, nil
}

// WaitForDispatch blocks until the row's dispatch_status transitions out of
// "pending" or ctx cancels. Returns the terminal status on success or
// ctx.Err() on timeout. Polls at 25ms intervals. Available to admin tooling
// and tests; there is no separate test-only code path.
func (s *PaymentAnomalyService) WaitForDispatch(ctx context.Context, rowID int64) (string, error) {
	ticker := time.NewTicker(25 * time.Millisecond)
	defer ticker.Stop()
	for {
		var status string
		err := s.db.QueryRowContext(ctx, `SELECT dispatch_status FROM payment_anomaly_log WHERE id = $1`, rowID).Scan(&status)
		if err == nil && status != anomalyDispatchPending {
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

// LogPaymentAnomaly is a package-level convenience for callers that already
// hold a Server. It is the sole recommended entrypoint for new producers.
func LogPaymentAnomaly(ctx context.Context, srv *Server, a PaymentAnomaly) (int64, error) {
	if srv == nil || srv.paymentAnomaly == nil {
		return 0, fmt.Errorf("payment anomaly service not initialized")
	}
	return srv.paymentAnomaly.Log(ctx, a)
}

// --- Rate limiter (in-process token bucket, per anomaly type) ---

type tokenBucket struct {
	tokens     int
	lastRefill time.Time
}

// Allow consumes a token for the given anomaly type. Returns false when the
// bucket is empty. Uses the config's per-type override when set, falling
// back to the built-in default. Internally caps to avoid unbounded growth —
// buckets are GC'd only when the service is GC'd, which is fine for
// server-lifetime scope.
func (d *AnomalyAlertDispatcher) Allow(anomalyType string, cfg *anomalyConfig) bool {
	if d == nil {
		return true
	}
	burst, refillS := defaultAnomalyRateLimitBurst, defaultAnomalyRateLimitRefillS
	if cfg != nil {
		if override, ok := cfg.rateLimits[anomalyType]; ok {
			burst, refillS = override.Burst, override.RefillSeconds
		}
	}
	if burst <= 0 || refillS <= 0 {
		return true
	}

	d.mu.Lock()
	defer d.mu.Unlock()
	now := time.Now()
	b, ok := d.buckets[anomalyType]
	if !ok {
		b = &tokenBucket{tokens: burst, lastRefill: now}
		d.buckets[anomalyType] = b
	}
	// Refill: one token per refillS seconds since lastRefill, capped at burst.
	elapsed := now.Sub(b.lastRefill)
	if elapsed > 0 {
		add := int(elapsed / (time.Duration(refillS) * time.Second))
		if add > 0 {
			b.tokens += add
			if b.tokens > burst {
				b.tokens = burst
			}
			b.lastRefill = b.lastRefill.Add(time.Duration(add) * time.Duration(refillS) * time.Second)
		}
	}
	if b.tokens <= 0 {
		return false
	}
	b.tokens--
	return true
}
