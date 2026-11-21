package commerce

import (
	"context"
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
	dialect             string
	logf                func(string, map[string]interface{})
	insufficientCredits error
	liveMode            bool
	// serviceAuthConfigured reports whether the service-to-service credential
	// the request gate actually checks (the shared LPBS service secret) resolves.
	// It is a composition-owned seam so this package does not read credentials.
	serviceAuthConfigured func() bool
}

// NewUsageService creates a new usage service.
// UsageServiceOptions provides full configurability for testing.
type UsageServiceOptions struct {
	DB                    UsageStore
	LimitsService         LimitsServicer
	Dialect               string
	Log                   func(string, map[string]interface{})
	InsufficientCredits   error
	LiveMode              *bool
	ServiceAuthConfigured func() bool
}

// NewUsageServiceWithOptions creates a usage service with explicit configuration.
func NewUsageServiceWithOptions(opts UsageServiceOptions) *UsageService {
	liveMode := true
	if opts.LiveMode != nil {
		liveMode = *opts.LiveMode
	}
	return &UsageService{
		db:                    opts.DB,
		limitsSvc:             opts.LimitsService,
		dialect:               opts.Dialect,
		logf:                  opts.Log,
		insufficientCredits:   opts.InsufficientCredits,
		liveMode:              liveMode,
		serviceAuthConfigured: opts.ServiceAuthConfigured,
	}
}

// serviceAuthMode projects the configured boolean onto the stable mode string
// that the service-auth status readers depend on.
func serviceAuthMode(configured bool) string {
	if configured {
		return "enabled"
	}
	return "disabled"
}

func (s *UsageService) log(event string, fields map[string]interface{}) {
	if s.logf != nil {
		s.logf(event, fields)
	}
}

func (s *UsageService) quotaReservations() *QuotaReservationService {
	return NewQuotaReservationService(s.db, s.limitsSvc, s.dialect, s.insufficientCredits)
}

func (s *UsageService) businessAccountUsageService() *BusinessAccountUsageService {
	return NewBusinessAccountUsageService(BusinessAccountUsageOptions{DB: s.db, Limits: s.limitsSvc, Dialect: s.dialect, Insufficient: s.insufficientCredits, Log: s.logf, LiveMode: &s.liveMode})
}

func (s *UsageService) MeterKind(ctx context.Context, tier, key string) (string, error) {
	limit, err := requireMeterKind(ctx, s.limitsSvc, tier, key, "")
	if err != nil {
		return "", err
	}
	return limit.LimitType, nil
}

func (s *UsageService) ReserveQuota(ctx context.Context, actor, tier, key string, amount int64, window time.Duration) (string, error) {
	return s.quotaReservations().Reserve(ctx, actor, tier, key, amount, window)
}

func (s *UsageService) FinalizeQuota(ctx context.Context, actor, tier, id string, amount int64) error {
	return s.quotaReservations().Finalize(ctx, actor, tier, id, amount)
}

func (s *UsageService) ReleaseQuota(ctx context.Context, actor, tier, id string) error {
	return s.quotaReservations().Release(ctx, actor, tier, id)
}

func (s *UsageService) ReserveBusinessAccountCredits(ctx context.Context, account, actor, tier, key string, amount int64) (string, error) {
	return s.businessAccountUsageService().ReserveCredits(ctx, account, actor, tier, key, amount)
}

func (s *UsageService) ReserveBusinessAccountCreditsWithID(ctx context.Context, account, actor, tier, key string, amount int64, operation string) (string, bool, error) {
	return s.businessAccountUsageService().ReserveCreditsWithID(ctx, account, actor, tier, key, amount, operation)
}

func (s *UsageService) FinalizeBusinessAccountReservationWithCost(ctx context.Context, account, actor, id string, amount int64, app, model string, cost int64, prompt, completion int, provider string) error {
	owner := s.businessAccountUsageService()
	if err := owner.VerifyActor(ctx, account, actor, id); err != nil {
		return err
	}
	return owner.FinalizeReservationWithCost(ctx, account, id, amount, app, model, cost, prompt, completion, provider)
}

func (s *UsageService) ReleaseBusinessAccountReservation(ctx context.Context, account, actor, id string) error {
	owner := s.businessAccountUsageService()
	if err := owner.VerifyActor(ctx, account, actor, id); err != nil {
		return err
	}
	return owner.ReleaseReservation(ctx, account, id)
}

func (s *UsageService) RetainUncertainBusinessAccountReservation(ctx context.Context, account, actor, id string) error {
	owner := s.businessAccountUsageService()
	if err := owner.VerifyActor(ctx, account, actor, id); err != nil {
		return err
	}
	return owner.RetainUncertainReservation(ctx, account, id)
}

func (s *UsageService) RecordBusinessAccountUsage(ctx context.Context, account, actor, tier, key string, amount int64, operation, app, model string, cost int64, prompt, completion int, provider string) error {
	return s.businessAccountUsageService().RecordUsage(ctx, account, actor, tier, key, amount, operation, app, model, cost, prompt, completion, provider)
}

func (s *UsageService) GetBusinessAccountUsageSummary(ctx context.Context, account, tier string) (BusinessAccountUsageSummary, error) {
	owner := s.businessAccountUsageService()
	summary, err := owner.GetUsageSummary(ctx, account, tier)
	if err != nil {
		return summary, err
	}
	summary.Wallet, err = owner.WalletView(ctx, account)
	return summary, err
}

// CreditReservationBinding exposes immutable ownership for the authenticated
// transport to verify membership before any credit reservation mutation.
func (s *UsageService) CreditReservationBinding(ctx context.Context, id string) (account, actor, key string, found bool, err error) {
	err = s.db.QueryRowContext(ctx, `SELECT business_account_id,actor_identity,limit_key FROM business_account_credit_reservations WHERE id=$1`, id).Scan(&account, &actor, &key)
	if errors.Is(err, sql.ErrNoRows) {
		return "", "", "", false, nil
	}
	return account, actor, key, err == nil, err
}

func (s *UsageService) QuotaReservationKey(ctx context.Context, id string) (string, error) {
	_, key, err := s.quotaReservations().binding(ctx, id)
	return key, err
}

func (s *UsageService) ReserveBusinessAccountUSDSpendCap(ctx context.Context, account, actor, tier string, costMicros int64, operation string) (string, int64, int64, error) {
	if strings.TrimSpace(account) == "" {
		return "", 0, 0, ErrBusinessAccountRequired
	}
	limit, err := requireMeterKind(ctx, s.limitsSvc, tier, "ai_credits", "cost_based")
	if err != nil || costMicros <= 0 {
		return "", 0, 0, fmt.Errorf("current credit limit authority is unavailable: %w", err)
	}
	if limit.CostMultiplier <= 0 {
		return "", 0, 0, errors.New("current AI credit conversion is unavailable")
	}
	if costMicros > (int64(^uint64(0)>>1)-9999)/limit.CostMultiplier {
		return "", 0, 0, errors.New("spend ceiling conversion overflows")
	}
	units := (costMicros*limit.CostMultiplier + 9999) / 10000
	if units <= 0 {
		return "", 0, 0, errors.New("spend ceiling conversion is invalid")
	}
	id, created, err := s.ReserveBusinessAccountCreditsWithID(ctx, account, actor, tier, "ai_credits", units, operation)
	if err == nil && !created {
		return "", 0, 0, errors.New("spend operation is already reserved; reconcile its existing receipt")
	}
	return id, units, limit.CostMultiplier, err
}

func (s *UsageService) CleanupExpiredReservations(ctx context.Context) (int, error) {
	quota, err := s.quotaReservations().Cleanup(ctx)
	if err != nil {
		return 0, err
	}
	credit, err := s.businessAccountUsageService().CleanupExpiredReservations(ctx)
	return int(quota + credit), err
}

func (s *UsageService) StartReservationCleanup(interval time.Duration) func() {
	if interval <= 0 {
		interval = time.Minute
	}
	ctx, cancel := context.WithCancel(context.Background())
	done := make(chan struct{})
	go func() {
		defer close(done)
		ticker := time.NewTicker(interval)
		defer ticker.Stop()
		for {
			select {
			case <-ticker.C:
				if s.dialect != "sqlite" {
					if _, err := DispatchFundingLedger(ctx, s.db); err != nil {
						s.log("funding_ledger_dispatch_failed", map[string]interface{}{"error": err.Error()})
					}
				}
				_, err := s.CleanupExpiredReservations(ctx)
				if err != nil {
					s.log("reservation_cleanup_failed", map[string]interface{}{"error": err.Error()})
				}
			case <-ctx.Done():
				return
			}
		}
	}()
	return func() { cancel(); <-done }
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

// ErrDirectUsageScopeUnavailable is stable while direct reports have no verified
// account/app binding. Request fields cannot establish that authority.
var ErrDirectUsageScopeUnavailable = errors.New("Verified app billing scope is unavailable; contact the app operator to complete LPBS billing setup")

// ErrDirectUsageAuthorityUnavailable identifies retryable server-owned tier/limit
// lookup failures, separately from invalid meter definitions and missing scope.
var ErrDirectUsageAuthorityUnavailable = errors.New("Meter authority is temporarily unavailable; retry the usage report")

type directUsageLimitAuthority struct{ LimitsServicer }

func (a directUsageLimitAuthority) GetTierLimits(ctx context.Context, tier string) ([]TierLimit, error) {
	rows, err := a.LimitsServicer.GetTierLimits(ctx, tier)
	if err != nil {
		return nil, fmt.Errorf("%w: %w", ErrDirectUsageAuthorityUnavailable, err)
	}
	return rows, nil
}

// ValidateDirectUsage performs all report admission checks without mutating usage.
// Only an explicitly configured global count meter can use the existing personal
// quota subject. Any app definition needs a verified binding, even if omitted by
// the caller; absence in the request is not evidence of global authority.
func (s *UsageService) ValidateDirectUsage(ctx context.Context, req UsageReportRequest) error {
	normalized, err := NormalizeUsageReport(req)
	if err != nil {
		return err
	}
	req = normalized.UsageReportRequest
	if req.BusinessAccountID != "" || req.AppBundleKey != "" {
		return ErrDirectUsageScopeUnavailable
	}
	if s.limitsSvc == nil || strings.TrimSpace(req.Tier) == "" {
		return ErrDirectUsageAuthorityUnavailable
	}
	authority := directUsageLimitAuthority{s.limitsSvc}
	rows, err := authority.GetTierLimits(ctx, req.Tier)
	if err != nil {
		return err
	}
	for _, row := range rows {
		if row.LimitKey == req.LimitKey && row.AppBundleKey != nil {
			return ErrDirectUsageScopeUnavailable
		}
	}
	limits, err := ResolveMeterLimits(ctx, authority, req.Tier, req.LimitKey, "")
	if err != nil {
		return err
	}
	if limits.Kind != "count_based" || limits.Global == nil {
		return ErrDirectUsageScopeUnavailable
	}
	return nil
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
	if err := s.ValidateDirectUsage(ctx, req); err != nil {
		return err
	}
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
		WHERE user_identity = $1 AND billing_period = $2 AND limit_key IN (SELECT limit_key FROM subscription_tier_limits WHERE tier_id=$3 AND limit_type='count_based')
	`, userIdentity, billingPeriod, tier)
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
				if limit.LimitType != "count_based" {
					continue
				}
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

	limit, err := requireMeterKind(ctx, s.limitsSvc, tier, limitKey, "count_based")
	if err != nil {
		return false, 0, err
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

// HealthCheck returns the health status of the usage service.
func (s *UsageService) HealthCheck(ctx context.Context) (*UsageHealthStatus, error) {
	configured := s.serviceAuthConfigured != nil && s.serviceAuthConfigured()
	status := &UsageHealthStatus{
		Healthy:               true,
		DatabaseConnected:     true,
		ServiceAuthConfigured: configured,
		ServiceAuthMode:       serviceAuthMode(configured),
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
