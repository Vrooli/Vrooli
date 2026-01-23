package main

import (
	"context"
	cryptoRand "crypto/rand"
	"crypto/subtle"
	"database/sql"
	"encoding/json"
	"fmt"
	"net/http"
	"strings"
	"time"
)

// UsageService tracks credit usage per user and billing period.
type UsageService struct {
	db           *sql.DB
	limitsSvc    LimitsServicer // Interface for testing
	serviceToken string         // Token for service-to-service auth
	dialect      string         // "postgres" or "sqlite"
}

// UsageRecord represents a single usage record.
type UsageRecord struct {
	ID              string     `json:"id"`
	UserIdentity    string     `json:"user_identity"`
	BillingPeriod   string     `json:"billing_period"` // YYYY-MM
	LimitKey        string     `json:"limit_key"`
	UsageAmount     int64      `json:"usage_amount"`
	AppBundleKey    *string    `json:"app_bundle_key"`
	LastOperationAt *time.Time `json:"last_operation_at,omitempty"`
	CreatedAt       time.Time  `json:"created_at"`
	UpdatedAt       time.Time  `json:"updated_at"`
}

// UsageReportRequest is a request to report usage from an app.
type UsageReportRequest struct {
	UserIdentity string            `json:"user_identity"`
	LimitKey     string            `json:"limit_key"` // e.g., "ai_credits"
	Amount       int64             `json:"amount"`    // In internal units (for cost_based) or simple count
	AppBundleKey string            `json:"app_bundle_key"`
	Operation    string            `json:"operation,omitempty"`    // e.g., "ai.analysis"
	IsBYOK       bool              `json:"is_byok,omitempty"`      // True if user used their own API key
	Metadata     map[string]string `json:"metadata,omitempty"`     // Additional context
	OperationID  *string           `json:"operation_id,omitempty"` // Idempotency key - retries with same ID won't double-count
}

// UsageSummary summarizes usage for a user.
type UsageSummary struct {
	UserIdentity   string             `json:"user_identity"`
	BillingPeriod  string             `json:"billing_period"`
	Tier           string             `json:"tier,omitempty"`
	Limits         map[string]int64   `json:"limits"`          // limit_key -> limit_value
	Usage          map[string]int64   `json:"usage"`           // limit_key -> usage_amount
	Remaining      map[string]int64   `json:"remaining"`       // limit_key -> remaining (or -1 for unlimited)
	DisplayCredits map[string]float64 `json:"display_credits"` // limit_key -> display value (divided by 100000)
	ResetDate      time.Time          `json:"reset_date"`
	ByApp          map[string]int64   `json:"by_app,omitempty"` // app_bundle_key -> total usage
}

// NewUsageService creates a new usage service.
func NewUsageService(db *sql.DB, limitsSvc LimitsServicer, dialect string) *UsageService {
	return NewUsageServiceWithOptions(UsageServiceOptions{
		DB:            db,
		LimitsService: limitsSvc,
		Dialect:       dialect,
	})
}

// UsageServiceOptions provides full configurability for testing.
type UsageServiceOptions struct {
	DB            *sql.DB
	LimitsService LimitsServicer
	Dialect       string
	ServiceToken  string // If empty, resolves from environment
}

// NewUsageServiceWithOptions creates a usage service with explicit configuration.
func NewUsageServiceWithOptions(opts UsageServiceOptions) *UsageService {
	token := opts.ServiceToken
	if token == "" {
		token = resolveSecret("LPBS_SERVICE_SECRET")
		if token == "" {
			logStructured("usage_service_no_token", map[string]interface{}{
				"level":   "warn",
				"message": "LPBS_SERVICE_SECRET not set; service-to-service auth disabled",
			})
		}
	}

	return &UsageService{
		db:           opts.DB,
		limitsSvc:    opts.LimitsService,
		serviceToken: token,
		dialect:      opts.Dialect,
	}
}

// getCurrentBillingPeriod returns the current billing period in YYYY-MM format.
func getCurrentBillingPeriod() string {
	return time.Now().Format("2006-01")
}

// getResetDate returns the first day of next month.
func getResetDate() time.Time {
	now := time.Now()
	return time.Date(now.Year(), now.Month()+1, 1, 0, 0, 0, 0, now.Location())
}

// RecordUsage records usage for a user. This is typically called by external apps.
// If OperationID is provided, this operation is idempotent - duplicate requests with
// the same operation_id will return success without incrementing usage.
func (s *UsageService) RecordUsage(ctx context.Context, req UsageReportRequest) error {
	userIdentity := strings.TrimSpace(strings.ToLower(req.UserIdentity))
	limitKey := strings.TrimSpace(strings.ToLower(req.LimitKey))
	appBundleKey := strings.TrimSpace(strings.ToLower(req.AppBundleKey))

	if userIdentity == "" {
		return fmt.Errorf("user_identity is required")
	}
	if limitKey == "" {
		return fmt.Errorf("limit_key is required")
	}
	if req.Amount <= 0 && !req.IsBYOK {
		return fmt.Errorf("amount must be positive")
	}

	// BYOK operations are logged with 0 amount
	amount := req.Amount
	if req.IsBYOK {
		amount = 0
	}

	// Idempotency check: if operation_id is provided, check if it was already processed
	if req.OperationID != nil && *req.OperationID != "" {
		var checkQuery string
		if s.dialect == "sqlite" {
			checkQuery = "SELECT EXISTS(SELECT 1 FROM usage_records WHERE operation_id = ?)"
		} else {
			checkQuery = "SELECT EXISTS(SELECT 1 FROM usage_records WHERE operation_id = $1)"
		}

		var exists bool
		err := s.db.QueryRowContext(ctx, checkQuery, *req.OperationID).Scan(&exists)
		if err == nil && exists {
			// Already recorded - return success (idempotent)
			logStructured("usage_already_recorded", map[string]interface{}{
				"level":         "debug",
				"operation_id":  *req.OperationID,
				"user_identity": userIdentity,
			})
			return nil
		}
	}

	billingPeriod := getCurrentBillingPeriod()

	// Use UPSERT to atomically increment usage
	var appKey interface{}
	if appBundleKey != "" {
		appKey = appBundleKey
	}

	// Determine operation_id for insert
	var opID interface{}
	if req.OperationID != nil && *req.OperationID != "" {
		opID = *req.OperationID
	}

	var query string
	if s.dialect == "sqlite" {
		query = `
			INSERT INTO usage_records (user_identity, billing_period, limit_key, usage_amount, app_bundle_key, operation_id, last_operation_at)
			VALUES (?, ?, ?, ?, ?, ?, datetime('now'))
			ON CONFLICT (user_identity, billing_period, limit_key, app_bundle_key)
			DO UPDATE SET
				usage_amount = usage_records.usage_amount + excluded.usage_amount,
				operation_id = COALESCE(excluded.operation_id, usage_records.operation_id),
				last_operation_at = datetime('now'),
				updated_at = datetime('now')
		`
	} else {
		query = `
			INSERT INTO usage_records (user_identity, billing_period, limit_key, usage_amount, app_bundle_key, operation_id, last_operation_at)
			VALUES ($1, $2, $3, $4, $5, $6, NOW())
			ON CONFLICT (user_identity, billing_period, limit_key, app_bundle_key)
			DO UPDATE SET
				usage_amount = usage_records.usage_amount + $4,
				operation_id = COALESCE($6, usage_records.operation_id),
				last_operation_at = NOW(),
				updated_at = NOW()
		`
	}

	_, err := s.db.ExecContext(ctx, query, userIdentity, billingPeriod, limitKey, amount, appKey, opID)
	if err != nil {
		return fmt.Errorf("record usage: %w", err)
	}

	logStructured("usage_recorded", map[string]interface{}{
		"level":          "debug",
		"user_identity":  userIdentity,
		"limit_key":      limitKey,
		"amount":         amount,
		"app_bundle_key": appBundleKey,
		"is_byok":        req.IsBYOK,
		"operation":      req.Operation,
		"operation_id":   req.OperationID,
	})

	return nil
}

// GetUsage returns the current usage for a user and limit key.
func (s *UsageService) GetUsage(ctx context.Context, userIdentity, limitKey string, appBundleKey *string) (int64, error) {
	userIdentity = strings.TrimSpace(strings.ToLower(userIdentity))
	limitKey = strings.TrimSpace(strings.ToLower(limitKey))
	billingPeriod := getCurrentBillingPeriod()

	var query string
	var args []interface{}

	if appBundleKey == nil {
		// Sum all usage for this limit key (across all apps)
		query = `
			SELECT COALESCE(SUM(usage_amount), 0)
			FROM usage_records
			WHERE user_identity = $1 AND billing_period = $2 AND limit_key = $3
		`
		args = []interface{}{userIdentity, billingPeriod, limitKey}
	} else {
		query = `
			SELECT COALESCE(usage_amount, 0)
			FROM usage_records
			WHERE user_identity = $1 AND billing_period = $2 AND limit_key = $3 AND app_bundle_key = $4
		`
		args = []interface{}{userIdentity, billingPeriod, limitKey, *appBundleKey}
	}

	var usage int64
	err := s.db.QueryRowContext(ctx, query, args...).Scan(&usage)
	if err != nil && err != sql.ErrNoRows {
		return 0, fmt.Errorf("get usage: %w", err)
	}

	return usage, nil
}

// GetUsageSummary returns a comprehensive usage summary for a user.
func (s *UsageService) GetUsageSummary(ctx context.Context, userIdentity, tier string) (*UsageSummary, error) {
	userIdentity = strings.TrimSpace(strings.ToLower(userIdentity))
	tier = strings.TrimSpace(strings.ToLower(tier))
	billingPeriod := getCurrentBillingPeriod()

	// Get all usage records for this user and billing period
	rows, err := s.db.QueryContext(ctx, `
		SELECT limit_key, usage_amount, app_bundle_key
		FROM usage_records
		WHERE user_identity = $1 AND billing_period = $2
	`, userIdentity, billingPeriod)
	if err != nil {
		return nil, fmt.Errorf("query usage records: %w", err)
	}
	defer rows.Close()

	usage := make(map[string]int64)
	byApp := make(map[string]int64)

	for rows.Next() {
		var limitKey string
		var amount int64
		var appKey sql.NullString

		if err := rows.Scan(&limitKey, &amount, &appKey); err != nil {
			return nil, fmt.Errorf("scan usage record: %w", err)
		}

		usage[limitKey] += amount
		if appKey.Valid {
			byApp[appKey.String] += amount
		}
	}

	if err := rows.Err(); err != nil {
		return nil, err
	}

	// Get limits for the tier
	limits := make(map[string]int64)
	remaining := make(map[string]int64)
	displayCredits := make(map[string]float64)

	if tier != "" && s.limitsSvc != nil {
		tierLimits, err := s.limitsSvc.GetTierLimits(ctx, tier)
		if err != nil {
			logStructuredError("get_tier_limits_for_summary_failed", map[string]interface{}{
				"error": err.Error(),
				"tier":  tier,
			})
		} else {
			for _, limit := range tierLimits {
				limits[limit.LimitKey] = limit.LimitValue

				// Calculate remaining
				used := usage[limit.LimitKey]
				if limit.LimitValue < 0 {
					// Unlimited
					remaining[limit.LimitKey] = -1
				} else {
					rem := limit.LimitValue - used
					if rem < 0 {
						rem = 0
					}
					remaining[limit.LimitKey] = rem
				}

				// Calculate display credits (divide by 100000 for user-friendly display)
				// This shows "5000 credits" instead of "500000000 internal units" for $5
				if limit.LimitType == "cost_based" {
					usedDisplay := float64(used) / 100000.0
					displayCredits[limit.LimitKey+"_used"] = usedDisplay

					if limit.LimitValue > 0 {
						limitDisplay := float64(limit.LimitValue) / 100000.0
						displayCredits[limit.LimitKey+"_limit"] = limitDisplay

						remDisplay := float64(remaining[limit.LimitKey]) / 100000.0
						displayCredits[limit.LimitKey+"_remaining"] = remDisplay
					} else if limit.LimitValue < 0 {
						displayCredits[limit.LimitKey+"_limit"] = -1
						displayCredits[limit.LimitKey+"_remaining"] = -1
					}
				}
			}
		}
	}

	return &UsageSummary{
		UserIdentity:   userIdentity,
		BillingPeriod:  billingPeriod,
		Tier:           tier,
		Limits:         limits,
		Usage:          usage,
		Remaining:      remaining,
		DisplayCredits: displayCredits,
		ResetDate:      getResetDate(),
		ByApp:          byApp,
	}, nil
}

// CheckLimit checks if a user can perform an operation based on their limits.
// Returns (canProceed, remaining, error).
func (s *UsageService) CheckLimit(ctx context.Context, userIdentity, tier, limitKey string, amount int64) (bool, int64, error) {
	userIdentity = strings.TrimSpace(strings.ToLower(userIdentity))
	tier = strings.TrimSpace(strings.ToLower(tier))
	limitKey = strings.TrimSpace(strings.ToLower(limitKey))

	if tier == "" || s.limitsSvc == nil {
		// No tier or limits service - allow everything
		return true, -1, nil
	}

	// Get the limit for this tier and key
	limit, err := s.limitsSvc.GetLimit(ctx, tier, limitKey, nil)
	if err != nil {
		return false, 0, err
	}

	if limit == nil {
		// No limit configured - allow everything
		return true, -1, nil
	}

	// Check if unlimited
	if limit.LimitValue < 0 {
		return true, -1, nil
	}

	// Check if tier has no access (limit == 0)
	if limit.LimitValue == 0 {
		return false, 0, nil
	}

	// Get current usage
	usage, err := s.GetUsage(ctx, userIdentity, limitKey, nil)
	if err != nil {
		return false, 0, err
	}

	remaining := limit.LimitValue - usage
	if remaining < 0 {
		remaining = 0
	}

	canProceed := remaining >= amount
	return canProceed, remaining, nil
}

// GetAllUsageForPeriod returns usage records for all users in a billing period (admin view).
func (s *UsageService) GetAllUsageForPeriod(ctx context.Context, billingPeriod string) ([]UsageRecord, error) {
	if billingPeriod == "" {
		billingPeriod = getCurrentBillingPeriod()
	}

	rows, err := s.db.QueryContext(ctx, `
		SELECT id, user_identity, billing_period, limit_key, usage_amount,
			   app_bundle_key, last_operation_at, created_at, updated_at
		FROM usage_records
		WHERE billing_period = $1
		ORDER BY user_identity, limit_key
	`, billingPeriod)
	if err != nil {
		return nil, fmt.Errorf("query usage records: %w", err)
	}
	defer rows.Close()

	var records []UsageRecord
	for rows.Next() {
		var record UsageRecord
		var appKey sql.NullString
		var lastOp sql.NullTime

		if err := rows.Scan(
			&record.ID, &record.UserIdentity, &record.BillingPeriod,
			&record.LimitKey, &record.UsageAmount, &appKey,
			&lastOp, &record.CreatedAt, &record.UpdatedAt,
		); err != nil {
			return nil, fmt.Errorf("scan usage record: %w", err)
		}

		if appKey.Valid {
			record.AppBundleKey = &appKey.String
		}
		if lastOp.Valid {
			record.LastOperationAt = &lastOp.Time
		}

		records = append(records, record)
	}

	return records, rows.Err()
}

// ValidateServiceToken validates the service-to-service auth token.
// Uses constant-time comparison to prevent timing attacks where an attacker
// could deduce token characters by measuring response times.
func (s *UsageService) ValidateServiceToken(token string) bool {
	if s.serviceToken == "" {
		// No token configured - reject all
		return false
	}
	// Use constant-time comparison to prevent timing attacks
	return subtle.ConstantTimeCompare([]byte(token), []byte(s.serviceToken)) == 1
}

// UsageHealthStatus contains the health status of the usage service.
type UsageHealthStatus struct {
	Healthy           bool       `json:"healthy"`
	DatabaseConnected bool       `json:"database_connected"`
	LastRecordAt      *time.Time `json:"last_record_at,omitempty"`
	RecordsThisPeriod int64      `json:"records_this_period"`
}

// HealthCheck returns the health status of the usage service.
func (s *UsageService) HealthCheck(ctx context.Context) (*UsageHealthStatus, error) {
	status := &UsageHealthStatus{
		Healthy:           true,
		DatabaseConnected: true,
	}

	// Check database connectivity
	if err := s.db.PingContext(ctx); err != nil {
		status.Healthy = false
		status.DatabaseConnected = false
		return status, nil
	}

	currentPeriod := getCurrentBillingPeriod()

	// Get last record timestamp and count for current period
	var lastRecordAt sql.NullTime
	var recordCount int64

	var query string
	if s.dialect == "sqlite" {
		query = `
			SELECT MAX(last_operation_at), COUNT(*)
			FROM usage_records
			WHERE billing_period = ?
		`
	} else {
		query = `
			SELECT MAX(last_operation_at), COUNT(*)
			FROM usage_records
			WHERE billing_period = $1
		`
	}

	err := s.db.QueryRowContext(ctx, query, currentPeriod).Scan(&lastRecordAt, &recordCount)
	if err != nil && err != sql.ErrNoRows {
		status.Healthy = false
		return status, nil
	}

	if lastRecordAt.Valid {
		status.LastRecordAt = &lastRecordAt.Time
	}
	status.RecordsThisPeriod = recordCount

	return status, nil
}

// API Handlers

// requireServiceAuth is middleware for service-to-service authentication.
func (s *UsageService) requireServiceAuth(next http.HandlerFunc) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		// Extract Bearer token
		auth := r.Header.Get("Authorization")
		if !strings.HasPrefix(auth, "Bearer ") {
			writeJSONError(w, http.StatusUnauthorized, "Missing or invalid authorization header", ApiErrorTypeUnauthorized)
			return
		}

		token := strings.TrimPrefix(auth, "Bearer ")
		if !s.ValidateServiceToken(token) {
			writeJSONError(w, http.StatusUnauthorized, "Invalid service token", ApiErrorTypeUnauthorized)
			return
		}

		next(w, r)
	}
}

func handleReportUsage(svc *UsageService) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		var req UsageReportRequest
		if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
			writeJSONError(w, http.StatusBadRequest, "Invalid request body", ApiErrorTypeValidation)
			return
		}

		if err := svc.RecordUsage(r.Context(), req); err != nil {
			logStructuredError("report_usage_failed", map[string]interface{}{
				"error":         err.Error(),
				"user_identity": req.UserIdentity,
				"limit_key":     req.LimitKey,
			})
			writeJSONError(w, http.StatusBadRequest, err.Error(), ApiErrorTypeValidation)
			return
		}

		w.Header().Set("Content-Type", "application/json")
		if err := json.NewEncoder(w).Encode(map[string]interface{}{
			"success": true,
		}); err != nil {
			logStructuredError("encode_response_failed", map[string]interface{}{"error": err.Error()})
		}
	}
}

func handleGetUsageSummary(svc *UsageService, accountSvc *AccountService) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		userIdentity := getUserEmail(r.Context())
		if userIdentity == "" {
			writeJSONError(w, http.StatusUnauthorized, "Authentication required", ApiErrorTypeUnauthorized)
			return
		}
		tier := r.URL.Query().Get("tier")

		// If no tier provided, try to get it from the account service
		if tier == "" && accountSvc != nil {
			// Try to get subscription info to determine tier
			sub, err := accountSvc.GetSubscription(userIdentity)
			if err == nil && sub != nil && sub.PlanTier != nil {
				tier = *sub.PlanTier
			}
		}

		summary, err := svc.GetUsageSummary(r.Context(), userIdentity, tier)
		if err != nil {
			logStructuredError("get_usage_summary_failed", map[string]interface{}{
				"error":         err.Error(),
				"user_identity": userIdentity,
			})
			writeJSONError(w, http.StatusInternalServerError, "Failed to get usage summary", ApiErrorTypeServerError)
			return
		}

		w.Header().Set("Content-Type", "application/json")
		if err := json.NewEncoder(w).Encode(summary); err != nil {
			logStructuredError("encode_response_failed", map[string]interface{}{"error": err.Error()})
		}
	}
}

func handleAdminUsageSummary(svc *UsageService) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		billingPeriod := r.URL.Query().Get("period")

		records, err := svc.GetAllUsageForPeriod(r.Context(), billingPeriod)
		if err != nil {
			logStructuredError("admin_usage_summary_failed", map[string]interface{}{
				"error":          err.Error(),
				"billing_period": billingPeriod,
			})
			writeJSONError(w, http.StatusInternalServerError, "Failed to get usage summary", ApiErrorTypeServerError)
			return
		}

		if records == nil {
			records = []UsageRecord{}
		}

		// Aggregate by user
		userTotals := make(map[string]int64)
		appTotals := make(map[string]int64)
		for _, record := range records {
			userTotals[record.UserIdentity] += record.UsageAmount
			if record.AppBundleKey != nil {
				appTotals[*record.AppBundleKey] += record.UsageAmount
			}
		}

		w.Header().Set("Content-Type", "application/json")
		if err := json.NewEncoder(w).Encode(map[string]interface{}{
			"billing_period": billingPeriod,
			"records":        records,
			"user_totals":    userTotals,
			"app_totals":     appTotals,
			"total_users":    len(userTotals),
			"total_records":  len(records),
		}); err != nil {
			logStructuredError("encode_response_failed", map[string]interface{}{"error": err.Error()})
		}
	}
}

// handleCheckLimit checks if a user can perform an operation (for entitlements endpoint).
func handleCheckLimit(svc *UsageService) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		userIdentity := getUserEmail(r.Context())
		if userIdentity == "" {
			writeJSONError(w, http.StatusUnauthorized, "Authentication required", ApiErrorTypeUnauthorized)
			return
		}
		tier := r.URL.Query().Get("tier")
		limitKey := r.URL.Query().Get("limit_key")

		if limitKey == "" {
			writeJSONError(w, http.StatusBadRequest, "limit_key is required", ApiErrorTypeValidation)
			return
		}

		// Default check for 1 unit
		amount := int64(1)

		canProceed, remaining, err := svc.CheckLimit(r.Context(), userIdentity, tier, limitKey, amount)
		if err != nil {
			logStructuredError("check_limit_failed", map[string]interface{}{
				"error":         err.Error(),
				"user_identity": userIdentity,
				"limit_key":     limitKey,
			})
			writeJSONError(w, http.StatusInternalServerError, "Failed to check limit", ApiErrorTypeServerError)
			return
		}

		w.Header().Set("Content-Type", "application/json")
		if err := json.NewEncoder(w).Encode(map[string]interface{}{
			"can_proceed": canProceed,
			"remaining":   remaining,
			"limit_key":   limitKey,
		}); err != nil {
			logStructuredError("encode_response_failed", map[string]interface{}{"error": err.Error()})
		}
	}
}

// handleUsageHealth returns the health status of the usage service.
// This endpoint is unauthenticated for monitoring/observability purposes.
func handleUsageHealth(svc *UsageService) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		ctx, cancel := context.WithTimeout(r.Context(), 5*time.Second)
		defer cancel()

		status, err := svc.HealthCheck(ctx)
		if err != nil {
			logStructuredError("usage_health_check_failed", map[string]interface{}{
				"error": err.Error(),
			})
			w.WriteHeader(http.StatusServiceUnavailable)
			if encErr := json.NewEncoder(w).Encode(map[string]interface{}{
				"healthy": false,
				"error":   err.Error(),
			}); encErr != nil {
				logStructuredError("encode_response_failed", map[string]interface{}{"error": encErr.Error()})
			}
			return
		}

		w.Header().Set("Content-Type", "application/json")
		if !status.Healthy {
			w.WriteHeader(http.StatusServiceUnavailable)
		}
		if err := json.NewEncoder(w).Encode(status); err != nil {
			logStructuredError("encode_response_failed", map[string]interface{}{"error": err.Error()})
		}
	}
}

// ReserveAndCharge atomically checks the credit limit and records usage in a single transaction.
// This prevents TOCTOU (time-of-check to time-of-use) race conditions where a user could
// exceed their limit by making concurrent requests.
//
// The method uses SELECT FOR UPDATE to lock the user's usage records during the transaction,
// ensuring that concurrent requests are serialized.
func (s *UsageService) ReserveAndCharge(ctx context.Context, userIdentity, tier, limitKey string, amount int64, metadata UsageReportRequest) error {
	userIdentity = strings.TrimSpace(strings.ToLower(userIdentity))
	tier = strings.TrimSpace(strings.ToLower(tier))
	limitKey = strings.TrimSpace(strings.ToLower(limitKey))

	if userIdentity == "" {
		return fmt.Errorf("user_identity is required")
	}
	if limitKey == "" {
		return fmt.Errorf("limit_key is required")
	}
	if amount <= 0 {
		return fmt.Errorf("amount must be positive")
	}

	billingPeriod := getCurrentBillingPeriod()

	// Start a serializable transaction to prevent concurrent limit bypass
	tx, err := s.db.BeginTx(ctx, &sql.TxOptions{Isolation: sql.LevelSerializable})
	if err != nil {
		return fmt.Errorf("begin transaction: %w", err)
	}
	defer func() { _ = tx.Rollback() }()

	// Get current usage with row locking (FOR UPDATE)
	var currentUsage int64
	var query string
	if s.dialect == "sqlite" {
		// SQLite doesn't support FOR UPDATE, but serializable isolation provides equivalent protection
		query = `
			SELECT COALESCE(SUM(usage_amount), 0)
			FROM usage_records
			WHERE user_identity = ? AND billing_period = ? AND limit_key = ?
		`
	} else {
		query = `
			SELECT COALESCE(SUM(usage_amount), 0)
			FROM usage_records
			WHERE user_identity = $1 AND billing_period = $2 AND limit_key = $3
			FOR UPDATE
		`
	}

	err = tx.QueryRowContext(ctx, query, userIdentity, billingPeriod, limitKey).Scan(&currentUsage)
	if err != nil && err != sql.ErrNoRows {
		return fmt.Errorf("get current usage: %w", err)
	}

	// Check the limit if tier is specified
	if tier != "" && s.limitsSvc != nil {
		limit, err := s.limitsSvc.GetLimit(ctx, tier, limitKey, nil)
		if err != nil {
			return fmt.Errorf("get limit: %w", err)
		}

		if limit != nil && limit.LimitValue >= 0 {
			// Check if the new usage would exceed the limit
			if currentUsage+amount > limit.LimitValue {
				return fmt.Errorf("%w: would use %d, limit is %d, remaining is %d",
					ErrInsufficientCredits, currentUsage+amount, limit.LimitValue, limit.LimitValue-currentUsage)
			}
		}
		// If limit is nil or negative (unlimited), allow the operation
	}

	// Record the usage within the same transaction
	var appKey interface{}
	if metadata.AppBundleKey != "" {
		appKey = strings.TrimSpace(strings.ToLower(metadata.AppBundleKey))
	}

	var insertQuery string
	if s.dialect == "sqlite" {
		insertQuery = `
			INSERT INTO usage_records (user_identity, billing_period, limit_key, usage_amount, app_bundle_key, last_operation_at)
			VALUES (?, ?, ?, ?, ?, datetime('now'))
			ON CONFLICT (user_identity, billing_period, limit_key, app_bundle_key)
			DO UPDATE SET
				usage_amount = usage_records.usage_amount + excluded.usage_amount,
				last_operation_at = datetime('now'),
				updated_at = datetime('now')
		`
	} else {
		insertQuery = `
			INSERT INTO usage_records (user_identity, billing_period, limit_key, usage_amount, app_bundle_key, last_operation_at)
			VALUES ($1, $2, $3, $4, $5, NOW())
			ON CONFLICT (user_identity, billing_period, limit_key, app_bundle_key)
			DO UPDATE SET
				usage_amount = usage_records.usage_amount + $4,
				last_operation_at = NOW(),
				updated_at = NOW()
		`
	}

	_, err = tx.ExecContext(ctx, insertQuery, userIdentity, billingPeriod, limitKey, amount, appKey)
	if err != nil {
		return fmt.Errorf("record usage: %w", err)
	}

	// Commit the transaction
	if err := tx.Commit(); err != nil {
		return fmt.Errorf("commit transaction: %w", err)
	}

	logStructured("usage_reserved_and_charged", map[string]interface{}{
		"level":          "debug",
		"user_identity":  userIdentity,
		"limit_key":      limitKey,
		"amount":         amount,
		"previous_usage": currentUsage,
		"new_usage":      currentUsage + amount,
		"operation":      metadata.Operation,
	})

	return nil
}

// ReserveCredits atomically checks if the user has enough credits and creates a reservation.
// This prevents TOCTOU race conditions in streaming requests where multiple concurrent
// requests could exceed the credit limit.
// Returns the reservation ID on success.
func (s *UsageService) ReserveCredits(ctx context.Context, userIdentity, tier, limitKey string, amount int64) (string, error) {
	userIdentity = strings.TrimSpace(strings.ToLower(userIdentity))
	tier = strings.TrimSpace(strings.ToLower(tier))
	limitKey = strings.TrimSpace(strings.ToLower(limitKey))

	if userIdentity == "" {
		return "", fmt.Errorf("user_identity is required")
	}
	if limitKey == "" {
		return "", fmt.Errorf("limit_key is required")
	}
	if amount <= 0 {
		return "", fmt.Errorf("amount must be positive")
	}

	billingPeriod := getCurrentBillingPeriod()

	// Start a serializable transaction for atomic check-and-reserve
	tx, err := s.db.BeginTx(ctx, &sql.TxOptions{Isolation: sql.LevelSerializable})
	if err != nil {
		return "", fmt.Errorf("begin transaction: %w", err)
	}
	defer func() { _ = tx.Rollback() }()

	// Get current usage + pending reservations with row locking
	var currentUsage int64
	var pendingReservations int64

	if s.dialect == "sqlite" {
		// Get current usage
		err = tx.QueryRowContext(ctx, `
			SELECT COALESCE(SUM(usage_amount), 0)
			FROM usage_records
			WHERE user_identity = ? AND billing_period = ? AND limit_key = ?
		`, userIdentity, billingPeriod, limitKey).Scan(&currentUsage)
		if err != nil && err != sql.ErrNoRows {
			return "", fmt.Errorf("get current usage: %w", err)
		}

		// Get pending reservations
		err = tx.QueryRowContext(ctx, `
			SELECT COALESCE(SUM(reserved_amount), 0)
			FROM credit_reservations
			WHERE user_identity = ? AND billing_period = ? AND limit_key = ? AND status = 'pending'
		`, userIdentity, billingPeriod, limitKey).Scan(&pendingReservations)
		if err != nil && err != sql.ErrNoRows {
			return "", fmt.Errorf("get pending reservations: %w", err)
		}
	} else {
		// Get current usage with FOR UPDATE lock
		err = tx.QueryRowContext(ctx, `
			SELECT COALESCE(SUM(usage_amount), 0)
			FROM usage_records
			WHERE user_identity = $1 AND billing_period = $2 AND limit_key = $3
			FOR UPDATE
		`, userIdentity, billingPeriod, limitKey).Scan(&currentUsage)
		if err != nil && err != sql.ErrNoRows {
			return "", fmt.Errorf("get current usage: %w", err)
		}

		// Get pending reservations with FOR UPDATE lock
		err = tx.QueryRowContext(ctx, `
			SELECT COALESCE(SUM(reserved_amount), 0)
			FROM credit_reservations
			WHERE user_identity = $1 AND billing_period = $2 AND limit_key = $3 AND status = 'pending'
			FOR UPDATE
		`, userIdentity, billingPeriod, limitKey).Scan(&pendingReservations)
		if err != nil && err != sql.ErrNoRows {
			return "", fmt.Errorf("get pending reservations: %w", err)
		}
	}

	effectiveUsage := currentUsage + pendingReservations

	// Check the limit if tier is specified
	if tier != "" && s.limitsSvc != nil {
		limit, err := s.limitsSvc.GetLimit(ctx, tier, limitKey, nil)
		if err != nil {
			return "", fmt.Errorf("get limit: %w", err)
		}

		if limit != nil && limit.LimitValue >= 0 {
			// Check if the new usage would exceed the limit
			if effectiveUsage+amount > limit.LimitValue {
				return "", fmt.Errorf("%w: would use %d, limit is %d, remaining is %d",
					ErrInsufficientCredits, effectiveUsage+amount, limit.LimitValue, limit.LimitValue-effectiveUsage)
			}
		}
	}

	// Create the reservation (expires in 10 minutes)
	reservationID := generateUUID()
	expiresAt := time.Now().Add(10 * time.Minute)

	var insertQuery string
	if s.dialect == "sqlite" {
		insertQuery = `
			INSERT INTO credit_reservations (id, user_identity, billing_period, limit_key, reserved_amount, status, expires_at, created_at)
			VALUES (?, ?, ?, ?, ?, 'pending', ?, datetime('now'))
		`
		_, err = tx.ExecContext(ctx, insertQuery, reservationID, userIdentity, billingPeriod, limitKey, amount, expiresAt)
	} else {
		insertQuery = `
			INSERT INTO credit_reservations (id, user_identity, billing_period, limit_key, reserved_amount, status, expires_at, created_at)
			VALUES ($1, $2, $3, $4, $5, 'pending', $6, NOW())
		`
		_, err = tx.ExecContext(ctx, insertQuery, reservationID, userIdentity, billingPeriod, limitKey, amount, expiresAt)
	}
	if err != nil {
		return "", fmt.Errorf("create reservation: %w", err)
	}

	// Commit the transaction
	if err := tx.Commit(); err != nil {
		return "", fmt.Errorf("commit transaction: %w", err)
	}

	logStructured("credits_reserved", map[string]interface{}{
		"level":          "debug",
		"user_identity":  userIdentity,
		"limit_key":      limitKey,
		"reserved":       amount,
		"reservation_id": reservationID,
		"current_usage":  currentUsage,
		"pending":        pendingReservations,
	})

	return reservationID, nil
}

// FinalizeReservation marks a reservation as finalized and records the actual usage.
// Call this after a streaming request completes successfully.
func (s *UsageService) FinalizeReservation(ctx context.Context, reservationID string, actualAmount int64) error {
	if reservationID == "" {
		return fmt.Errorf("reservation_id is required")
	}
	if actualAmount < 0 {
		return fmt.Errorf("actual_amount must be non-negative")
	}

	// Start a transaction to finalize the reservation and record usage atomically
	tx, err := s.db.BeginTx(ctx, nil)
	if err != nil {
		return fmt.Errorf("begin transaction: %w", err)
	}
	defer func() { _ = tx.Rollback() }()

	// Get reservation details
	var userIdentity, billingPeriod, limitKey, status string
	var reservedAmount int64
	var getQuery string

	if s.dialect == "sqlite" {
		getQuery = `
			SELECT user_identity, billing_period, limit_key, reserved_amount, status
			FROM credit_reservations
			WHERE id = ?
		`
	} else {
		getQuery = `
			SELECT user_identity, billing_period, limit_key, reserved_amount, status
			FROM credit_reservations
			WHERE id = $1
			FOR UPDATE
		`
	}

	err = tx.QueryRowContext(ctx, getQuery, reservationID).Scan(&userIdentity, &billingPeriod, &limitKey, &reservedAmount, &status)
	if err != nil {
		if err == sql.ErrNoRows {
			return fmt.Errorf("reservation not found: %s", reservationID)
		}
		return fmt.Errorf("get reservation: %w", err)
	}

	if status != "pending" {
		return fmt.Errorf("reservation already %s", status)
	}

	// Mark reservation as finalized
	var updateQuery string
	if s.dialect == "sqlite" {
		updateQuery = `
			UPDATE credit_reservations
			SET status = 'finalized', finalized_at = datetime('now')
			WHERE id = ?
		`
	} else {
		updateQuery = `
			UPDATE credit_reservations
			SET status = 'finalized', finalized_at = NOW()
			WHERE id = $1
		`
	}
	_, err = tx.ExecContext(ctx, updateQuery, reservationID)
	if err != nil {
		return fmt.Errorf("finalize reservation: %w", err)
	}

	// Record actual usage
	var insertQuery string
	if s.dialect == "sqlite" {
		insertQuery = `
			INSERT INTO usage_records (user_identity, billing_period, limit_key, usage_amount, last_operation_at)
			VALUES (?, ?, ?, ?, datetime('now'))
			ON CONFLICT (user_identity, billing_period, limit_key, app_bundle_key)
			DO UPDATE SET
				usage_amount = usage_records.usage_amount + excluded.usage_amount,
				last_operation_at = datetime('now'),
				updated_at = datetime('now')
		`
	} else {
		insertQuery = `
			INSERT INTO usage_records (user_identity, billing_period, limit_key, usage_amount, last_operation_at)
			VALUES ($1, $2, $3, $4, NOW())
			ON CONFLICT (user_identity, billing_period, limit_key, app_bundle_key)
			DO UPDATE SET
				usage_amount = usage_records.usage_amount + $4,
				last_operation_at = NOW(),
				updated_at = NOW()
		`
	}
	_, err = tx.ExecContext(ctx, insertQuery, userIdentity, billingPeriod, limitKey, actualAmount)
	if err != nil {
		return fmt.Errorf("record usage: %w", err)
	}

	// Commit the transaction
	if err := tx.Commit(); err != nil {
		return fmt.Errorf("commit transaction: %w", err)
	}

	logStructured("reservation_finalized", map[string]interface{}{
		"level":          "debug",
		"reservation_id": reservationID,
		"user_identity":  userIdentity,
		"reserved":       reservedAmount,
		"actual":         actualAmount,
	})

	return nil
}

// ReleaseReservation marks a reservation as released without recording usage.
// Call this when a streaming request is cancelled or fails before completing.
func (s *UsageService) ReleaseReservation(ctx context.Context, reservationID string) error {
	if reservationID == "" {
		return fmt.Errorf("reservation_id is required")
	}

	var query string
	if s.dialect == "sqlite" {
		query = `
			UPDATE credit_reservations
			SET status = 'released'
			WHERE id = ? AND status = 'pending'
		`
	} else {
		query = `
			UPDATE credit_reservations
			SET status = 'released'
			WHERE id = $1 AND status = 'pending'
		`
	}

	result, err := s.db.ExecContext(ctx, query, reservationID)
	if err != nil {
		return fmt.Errorf("release reservation: %w", err)
	}

	rowsAffected, _ := result.RowsAffected()
	if rowsAffected == 0 {
		logStructured("reservation_release_noop", map[string]interface{}{
			"level":          "debug",
			"reservation_id": reservationID,
			"reason":         "already finalized/released/expired or not found",
		})
	} else {
		logStructured("reservation_released", map[string]interface{}{
			"level":          "debug",
			"reservation_id": reservationID,
		})
	}

	return nil
}

// CleanupExpiredReservations marks expired pending reservations as expired.
// Returns the number of reservations that were expired.
func (s *UsageService) CleanupExpiredReservations(ctx context.Context) (int, error) {
	var query string
	if s.dialect == "sqlite" {
		query = `
			UPDATE credit_reservations
			SET status = 'expired'
			WHERE status = 'pending' AND expires_at < datetime('now')
		`
	} else {
		query = `
			UPDATE credit_reservations
			SET status = 'expired'
			WHERE status = 'pending' AND expires_at < NOW()
		`
	}

	result, err := s.db.ExecContext(ctx, query)
	if err != nil {
		return 0, fmt.Errorf("cleanup expired reservations: %w", err)
	}

	rowsAffected, _ := result.RowsAffected()
	if rowsAffected > 0 {
		logStructured("reservations_expired", map[string]interface{}{
			"level": "info",
			"count": rowsAffected,
		})
	}

	return int(rowsAffected), nil
}

// StartReservationCleanup starts a background goroutine that periodically cleans up
// expired reservations. Returns a cancel function to stop the cleanup goroutine.
func (s *UsageService) StartReservationCleanup(interval time.Duration) func() {
	done := make(chan struct{})
	go func() {
		ticker := time.NewTicker(interval)
		defer ticker.Stop()
		for {
			select {
			case <-ticker.C:
				ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
				_, _ = s.CleanupExpiredReservations(ctx)
				cancel()
			case <-done:
				return
			}
		}
	}()
	return func() {
		close(done)
	}
}

// generateUUID generates a UUID v4 string.
func generateUUID() string {
	// Simple UUID generation using crypto/rand
	b := make([]byte, 16)
	_, _ = cryptoRand.Read(b)
	b[6] = (b[6] & 0x0f) | 0x40 // Version 4
	b[8] = (b[8] & 0x3f) | 0x80 // Variant is 10
	return fmt.Sprintf("%x-%x-%x-%x-%x", b[0:4], b[4:6], b[6:8], b[8:10], b[10:])
}

// AdjustUsage adjusts usage for a user, allowing both positive and negative amounts.
// Negative amounts are used for refunds when actual usage was less than estimated.
// Usage will not go below 0 (floor at 0).
func (s *UsageService) AdjustUsage(ctx context.Context, userIdentity, limitKey string, adjustment int64, reason string) error {
	userIdentity = strings.TrimSpace(strings.ToLower(userIdentity))
	limitKey = strings.TrimSpace(strings.ToLower(limitKey))

	if userIdentity == "" {
		return fmt.Errorf("user_identity is required")
	}
	if limitKey == "" {
		return fmt.Errorf("limit_key is required")
	}
	if adjustment == 0 {
		return nil // No-op
	}

	billingPeriod := getCurrentBillingPeriod()

	var query string
	if s.dialect == "sqlite" {
		// SQLite: use MAX to floor at 0
		query = `
			UPDATE usage_records
			SET usage_amount = MAX(0, usage_amount + ?),
			    last_operation_at = datetime('now'),
			    updated_at = datetime('now')
			WHERE user_identity = ? AND billing_period = ? AND limit_key = ? AND app_bundle_key IS NULL
		`
		_, err := s.db.ExecContext(ctx, query, adjustment, userIdentity, billingPeriod, limitKey)
		if err != nil {
			return fmt.Errorf("adjust usage: %w", err)
		}
	} else {
		// PostgreSQL: use GREATEST to floor at 0
		query = `
			UPDATE usage_records
			SET usage_amount = GREATEST(0, usage_amount + $1),
			    last_operation_at = NOW(),
			    updated_at = NOW()
			WHERE user_identity = $2 AND billing_period = $3 AND limit_key = $4 AND app_bundle_key IS NULL
		`
		_, err := s.db.ExecContext(ctx, query, adjustment, userIdentity, billingPeriod, limitKey)
		if err != nil {
			return fmt.Errorf("adjust usage: %w", err)
		}
	}

	logStructured("usage_adjusted", map[string]interface{}{
		"level":         "debug",
		"user_identity": userIdentity,
		"limit_key":     limitKey,
		"adjustment":    adjustment,
		"reason":        reason,
	})

	return nil
}

// Note: ErrInsufficientCredits is defined in ai_gateway_errors.go (centralized AI gateway errors)

// Compile-time interface check for UsageServicer
var _ UsageServicer = (*UsageService)(nil)
