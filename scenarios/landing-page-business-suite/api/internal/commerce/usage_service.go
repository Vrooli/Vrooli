package commerce

import (
	"context"
	"crypto/subtle"
	"database/sql"
	"errors"
	"fmt"
	"strings"
	"time"
)

// UsageService owns credit metering, limit enforcement, and reservation-aware usage accounting.
type UsageService struct {
	db                  UsageStore
	limitsSvc           LimitsServicer
	serviceToken        string
	dialect             string
	logf                func(string, map[string]interface{})
	insufficientCredits error
}

// NewUsageService creates a new usage service.
// UsageServiceOptions provides full configurability for testing.
type UsageServiceOptions struct {
	DB                  UsageStore
	LimitsService       LimitsServicer
	Dialect             string
	ServiceToken        string
	Log                 func(string, map[string]interface{})
	InsufficientCredits error
}

// NewUsageServiceWithOptions creates a usage service with explicit configuration.
func NewUsageServiceWithOptions(opts UsageServiceOptions) *UsageService {
	return &UsageService{
		db:                  opts.DB,
		limitsSvc:           opts.LimitsService,
		serviceToken:        opts.ServiceToken,
		dialect:             opts.Dialect,
		logf:                opts.Log,
		insufficientCredits: opts.InsufficientCredits,
	}
}

func (s *UsageService) log(event string, fields map[string]interface{}) {
	if s.logf != nil {
		s.logf(event, fields)
	}
}

func (s *UsageService) reservationService() *ReservationService {
	insufficientCredits := s.insufficientCredits
	if insufficientCredits == nil {
		insufficientCredits = errors.New("insufficient credits")
	}
	return NewReservationService(s.db, s.limitsSvc, s.dialect, ReservationRuntime{
		InsufficientCredits: insufficientCredits,
		Log:                 s.logf,
	})
}

func (s *UsageService) ReserveAndCharge(ctx context.Context, userIdentity, tier, limitKey string, amount int64, metadata UsageReportRequest) error {
	return s.reservationService().ReserveAndCharge(ctx, userIdentity, tier, limitKey, amount, metadata)
}

func (s *UsageService) ReserveCredits(ctx context.Context, userIdentity, tier, limitKey string, amount int64) (string, error) {
	return s.reservationService().ReserveCredits(ctx, userIdentity, tier, limitKey, amount)
}

func (s *UsageService) FinalizeReservation(ctx context.Context, reservationID string, actualAmount int64) error {
	return s.reservationService().FinalizeReservation(ctx, reservationID, actualAmount)
}

func (s *UsageService) ReleaseReservation(ctx context.Context, reservationID string) error {
	return s.reservationService().ReleaseReservation(ctx, reservationID)
}

func (s *UsageService) CleanupExpiredReservations(ctx context.Context) (int, error) {
	return s.reservationService().CleanupExpiredReservations(ctx)
}

func (s *UsageService) StartReservationCleanup(interval time.Duration) func() {
	return s.reservationService().StartReservationCleanup(interval)
}

func (s *UsageService) AdjustUsage(ctx context.Context, userIdentity, limitKey string, adjustment int64, reason string) error {
	return s.reservationService().AdjustUsage(ctx, userIdentity, limitKey, adjustment, reason)
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
	normalized, err := NormalizeUsageReport(req)
	if err != nil {
		return err
	}
	req, amount := normalized.UsageReportRequest, normalized.Amount
	userIdentity, limitKey, appBundleKey := req.UserIdentity, req.LimitKey, req.AppBundleKey

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
			s.log("usage_already_recorded", map[string]interface{}{
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

	_, err = s.db.ExecContext(ctx, query, userIdentity, billingPeriod, limitKey, amount, appKey, opID)
	if err != nil {
		return fmt.Errorf("record usage: %w", err)
	}

	s.log("usage_recorded", map[string]interface{}{
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
			s.log("get_tier_limits_for_summary_failed", map[string]interface{}{
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

// HealthCheck returns the health status of the usage service.
func (s *UsageService) HealthCheck(ctx context.Context) (*UsageHealthStatus, error) {
	status := &UsageHealthStatus{
		Healthy:               true,
		DatabaseConnected:     true,
		ServiceAuthConfigured: strings.TrimSpace(s.serviceToken) != "",
		ServiceAuthMode:       "disabled",
	}
	if status.ServiceAuthConfigured {
		status.ServiceAuthMode = "token"
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
