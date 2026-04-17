package credits

import (
	"bytes"
	"context"
	"database/sql"
	"encoding/json"
	"errors"
	"fmt"
	"net/http"
	"strings"
	"sync"
	"time"

	"github.com/google/uuid"
	"github.com/sirupsen/logrus"
	"github.com/vrooli/browser-automation-studio/services/entitlement"
)

// LPBSReporter is an interface for reporting usage to LPBS.
// This allows for mocking in tests.
type LPBSReporter interface {
	ReportUsage(ctx context.Context, report LPBSUsageReport) error
}

// Service implements CreditService with database persistence and caching.
//
// # Architecture
//
// The Service coordinates between several subsystems:
//   - Database layer: Persists credit_usage and operation_log tables
//   - Entitlement layer: Determines tier limits via EntitlementProvider interface
//   - LPBS layer: Reports usage to central landing-page-business-suite (async)
//   - Cache layer: In-memory cache with 1-minute TTL for fast lookups
//
// # Testing Seams
//
// The Service has several intentional seams for testability:
//   - EntitlementProvider: Interface for entitlement lookups (mock in tests)
//   - LPBSReporter: Interface for usage reporting (mock in tests)
//
// See SEAMS.md for full documentation of testing boundaries.
type Service struct {
	db                  *sql.DB
	log                 *logrus.Logger
	entitlementSvc      *entitlement.Service // Legacy: concrete service for backward compat
	entitlementProvider EntitlementProvider  // Preferred: injectable interface for testing
	costs               OperationCosts

	// LPBS integration for centralized usage reporting
	lpbsURL        string       // LPBS service URL for usage reporting
	lpbsSecret     string       // Service-to-service auth secret
	lpbsHTTPClient *http.Client // HTTP client for LPBS requests
	lpbsReporter   LPBSReporter // Optional: injectable reporter for testing
	appBundleKey   string       // App identifier for usage records

	// In-memory cache for fast lookups
	cacheMu sync.RWMutex
	cache   map[string]*usageCache
}

type usageCache struct {
	totalCreditsUsed int
	totalOperations  int
	byOperation      map[OperationType]int
	operationCounts  map[OperationType]int
	month            string // YYYY-MM format
	updatedAt        time.Time
}

// ServiceOptions configures the credit service.
//
// # Testing Seams
//
// For unit testing credit logic without real dependencies:
//   - Use EntitlementProvider with MockEntitlementProvider (preferred for new tests)
//   - Use LPBSReporter with a mock to capture/verify usage reports
//
// # Example Test Setup
//
//	svc := credits.NewService(credits.ServiceOptions{
//	    DB:      sqliteDB,
//	    Logger:  logrus.New(),
//	    EntitlementProvider: &credits.MockEntitlementProvider{
//	        Entitlement: &entitlement.Entitlement{Tier: entitlement.TierPro},
//	        AICreditsLimit: 500,
//	        CanUseAI: true,
//	    },
//	    LPBSReporter: &mockLPBSReporter{},
//	})
type ServiceOptions struct {
	DB             *sql.DB
	Logger         *logrus.Logger
	EntitlementSvc *entitlement.Service // Legacy: concrete entitlement service
	// Note: Operation costs are intentionally NOT configurable here.
	// They are hard-coded in DefaultOperationCosts() to prevent bypassing charges.

	// EntitlementProvider is the preferred way to inject entitlement dependencies.
	// If set, this takes precedence over EntitlementSvc.
	// Use MockEntitlementProvider in tests to control entitlement behavior.
	EntitlementProvider EntitlementProvider

	// LPBS integration for centralized usage reporting
	// When configured, usage is reported to LPBS after local charges
	LPBSURL      string // LPBS service URL (e.g., "http://localhost:15000" or "https://vrooli.com")
	LPBSSecret   string // Service-to-service auth secret
	AppBundleKey string // App identifier (default: "browser-automation-studio")

	// LPBSReporter allows injecting a custom LPBS reporter for testing.
	// If nil, the default HTTP-based reporter will be used.
	LPBSReporter LPBSReporter
}

// NewService creates a new CreditService.
//
// The service can be configured with either EntitlementProvider (preferred) or
// EntitlementSvc (legacy). If EntitlementProvider is set, it takes precedence.
// If neither is set, all operations are treated as unlimited (useful for desktop apps).
func NewService(opts ServiceOptions) *Service {
	appBundleKey := opts.AppBundleKey
	if appBundleKey == "" {
		appBundleKey = "browser-automation-studio"
	}

	var lpbsHTTPClient *http.Client
	if opts.LPBSURL != "" {
		lpbsHTTPClient = &http.Client{
			Timeout: 5 * time.Second, // Short timeout for async reporting
		}
	}

	// Use provided EntitlementProvider, or wrap the legacy EntitlementSvc
	entProvider := opts.EntitlementProvider
	if entProvider == nil && opts.EntitlementSvc != nil {
		entProvider = NewDefaultEntitlementProvider(opts.EntitlementSvc)
	}

	return &Service{
		db:                  opts.DB,
		log:                 opts.Logger,
		entitlementSvc:      opts.EntitlementSvc, // Keep for backward compat
		entitlementProvider: entProvider,
		costs:               DefaultOperationCosts(),
		lpbsURL:             opts.LPBSURL,
		lpbsSecret:          opts.LPBSSecret,
		lpbsHTTPClient:      lpbsHTTPClient,
		lpbsReporter:        opts.LPBSReporter,
		appBundleKey:        appBundleKey,
		cache:               make(map[string]*usageCache),
	}
}

// getEntitlement retrieves the entitlement for a user, checking context first
// (for middleware overrides like tier testing), then falling back to the entitlement provider.
//
// Lookup order:
//  1. Context (middleware overrides, e.g., tier testing)
//  2. EntitlementProvider interface (preferred, supports mocking)
//  3. Returns nil if no provider configured (treated as unlimited)
func (s *Service) getEntitlement(ctx context.Context, userIdentity string) (*entitlement.Entitlement, error) {
	// Check context first - respects middleware overrides (e.g., tier override for testing)
	if ent := entitlement.FromContext(ctx); ent != nil {
		return ent, nil
	}

	// Use the provider interface (preferred path for testability)
	if s.entitlementProvider != nil {
		return s.entitlementProvider.GetEntitlement(ctx, userIdentity)
	}

	// No provider configured - return nil (treated as unlimited)
	return nil, nil
}

// CanCharge checks if the user has sufficient credits for the operation.
func (s *Service) CanCharge(ctx context.Context, userIdentity string, op OperationType) (bool, int, error) {
	userIdentity = normalizeIdentity(userIdentity)

	// Get user's credit limit from entitlement
	creditsLimit, err := s.getUserCreditsLimit(ctx, userIdentity)
	if err != nil {
		return false, 0, err
	}

	// Unlimited tier
	if creditsLimit < 0 {
		return true, -1, nil
	}

	// No access (creditsLimit == 0)
	if creditsLimit == 0 {
		return false, 0, ErrNoCreditsAccess
	}

	// Get current usage
	usage, err := s.getUsageFromDB(ctx, userIdentity)
	if err != nil {
		return false, 0, err
	}

	// Calculate cost for this operation
	cost := s.costs.GetCost(op)

	// Calculate remaining
	remaining := creditsLimit - usage.totalCreditsUsed
	if remaining < 0 {
		remaining = 0
	}

	// Check if user can afford this operation
	canCharge := remaining >= cost
	return canCharge, remaining, nil
}

// Charge deducts credits for a completed operation.
func (s *Service) Charge(ctx context.Context, req ChargeRequest) (*ChargeResult, error) {
	userIdentity := normalizeIdentity(req.UserIdentity)

	if userIdentity == "" {
		// Can't charge without user identity, but don't fail
		s.log.Debug("Cannot charge credits: no user identity")
		return &ChargeResult{Charged: 0, RemainingCredits: -1, WasCharged: false}, nil
	}

	cost := s.costs.GetCost(req.Operation)

	// BYOK operations are logged for analytics but not charged (user pays their own way)
	if req.IsBYOK {
		cost = 0
	}

	// Free operations (including BYOK) - just log
	if cost == 0 {
		_ = s.logOperation(ctx, userIdentity, req.Operation, 0, true, req.Metadata, "")
		// Report BYOK operations to LPBS for analytics (with 0 cost)
		s.reportUsageToLPBS(userIdentity, req.Operation, 0, 0, req.IsBYOK, req.Metadata)
		remaining, _ := s.getRemainingCredits(ctx, userIdentity)
		return &ChargeResult{Charged: 0, RemainingCredits: remaining, WasCharged: false}, nil
	}

	currentMonth := s.getBillingMonth(ctx, userIdentity)

	// Upsert credit usage
	if err := s.upsertUsage(ctx, userIdentity, currentMonth, req.Operation, cost); err != nil {
		return nil, fmt.Errorf("charge credits: %w", err)
	}

	// Log the operation
	if err := s.logOperation(ctx, userIdentity, req.Operation, cost, true, req.Metadata, ""); err != nil {
		s.log.WithError(err).Warn("Failed to log operation")
	}

	// Invalidate cache
	s.invalidateCache(userIdentity)

	// Report usage to LPBS (async, non-blocking)
	s.reportUsageToLPBS(userIdentity, req.Operation, cost, req.ActualCostCents, req.IsBYOK, req.Metadata)

	// Get remaining balance
	remaining, _ := s.getRemainingCredits(ctx, userIdentity)

	return &ChargeResult{
		Charged:          cost,
		RemainingCredits: remaining,
		WasCharged:       true,
	}, nil
}

// ChargeIfAllowed combines CanCharge and Charge.
//
// NOTE ON TOCTOU: This method has a known time-of-check-time-of-use (TOCTOU) race condition
// between the CanCharge check and the Charge call. Concurrent requests could potentially
// exceed limits briefly. This is acceptable for BAS's use case because:
//  1. When using the LPBS AI gateway (VrooliProvider), credits are checked and charged
//     atomically by LPBS using database transactions with row-level locking.
//  2. This local credit tracking is primarily for BYOK usage analytics and dev mode.
//  3. The potential overage is limited to concurrent requests within milliseconds.
//
// For production use with paid credits, always use the LPBS AI gateway which provides
// true atomic credit operations via ReserveAndCharge.
func (s *Service) ChargeIfAllowed(ctx context.Context, req ChargeRequest) (*ChargeResult, error) {
	canCharge, remaining, err := s.CanCharge(ctx, req.UserIdentity, req.Operation)
	if err != nil {
		return nil, err
	}
	if !canCharge {
		cost := s.costs.GetCost(req.Operation)
		return nil, fmt.Errorf("%w: need %d credits, have %d remaining", ErrInsufficientCredits, cost, remaining)
	}

	return s.Charge(ctx, req)
}

// CanPerformAIOperation checks if user can perform an AI operation.
// Combines BYOK bypass, tier check, and credit check in one call.
// Returns (canProceed, errorCode, errorMessage, remaining, error).
func (s *Service) CanPerformAIOperation(ctx context.Context, userIdentity string, op OperationType, hasBYOK bool) (bool, string, string, int, error) {
	// 1. BYOK users bypass all checks - they pay their own way
	if hasBYOK {
		return true, "", "", -1, nil
	}

	userIdentity = normalizeIdentity(userIdentity)

	// 2. Check tier allows AI (uses features array first, then tier fallback)
	ent, err := s.getEntitlement(ctx, userIdentity)
	if err != nil {
		// NOTE: This fail-open is for edge cases like network errors to the entitlement service.
		// When using the LPBS AI gateway, credits are checked atomically by LPBS itself,
		// so this local check is a pre-filter, not the authoritative source.
		s.log.WithError(err).Warn("credits: failed to get entitlement for AI check (proceeding with credit check)")
	} else if ent != nil && s.entitlementProvider != nil && !s.entitlementProvider.CanUseAIWithEntitlement(ent) {
		return false, "AI_NOT_AVAILABLE", "AI features not available for your subscription", 0, nil
	}

	// 3. Check credits
	canCharge, remaining, err := s.CanCharge(ctx, userIdentity, op)
	if err != nil {
		if errors.Is(err, ErrNoCreditsAccess) {
			return false, "AI_NOT_AVAILABLE", "AI features not available for your tier", 0, nil
		}
		return false, "", "", 0, err
	}
	if !canCharge {
		return false, "INSUFFICIENT_CREDITS", fmt.Sprintf("Insufficient AI credits. Remaining: %d", remaining), remaining, nil
	}

	return true, "", "", remaining, nil
}

// GetUsage returns the usage summary for a user in the current billing period.
func (s *Service) GetUsage(ctx context.Context, userIdentity string) (*UsageSummary, error) {
	userIdentity = normalizeIdentity(userIdentity)
	currentMonth := s.getBillingMonth(ctx, userIdentity)

	// Get usage from cache or DB
	var usage *usageCache
	if cached := s.getCached(userIdentity, currentMonth); cached != nil {
		usage = cached
	} else {
		dbUsage, err := s.getUsageFromDB(ctx, userIdentity)
		if err != nil {
			return nil, err
		}
		usage = dbUsage
		s.setCached(userIdentity, currentMonth, usage)
	}

	// Get credit limit from entitlement
	creditsLimit, err := s.getUserCreditsLimit(ctx, userIdentity)
	if err != nil {
		s.log.WithError(err).Warn("Failed to get credit limit, assuming unlimited")
		creditsLimit = -1
	}

	creditsRemaining := -1
	if creditsLimit >= 0 {
		creditsRemaining = creditsLimit - usage.totalCreditsUsed
		if creditsRemaining < 0 {
			creditsRemaining = 0
		}
	}

	// Get billing period boundaries
	now := time.Now()
	var periodStart, periodEnd, resetDate time.Time
	ent, _ := s.getEntitlement(ctx, userIdentity)
	if ent != nil && ent.BillingCycleStart >= 1 && ent.BillingCycleStart <= 28 {
		periodStart, periodEnd = ent.GetBillingPeriod(now)
		resetDate = periodEnd.Add(time.Nanosecond)
	} else {
		periodStart = firstDayOfMonth(now)
		periodEnd = lastDayOfMonth(now)
		resetDate = firstDayOfMonth(now).AddDate(0, 1, 0)
	}

	return &UsageSummary{
		UserIdentity:     userIdentity,
		BillingMonth:     currentMonth,
		TotalCreditsUsed: usage.totalCreditsUsed,
		TotalOperations:  usage.totalOperations,
		ByOperation:      usage.byOperation,
		OperationCounts:  usage.operationCounts,
		CreditsLimit:     creditsLimit,
		CreditsRemaining: creditsRemaining,
		PeriodStart:      periodStart,
		PeriodEnd:        periodEnd,
		ResetDate:        resetDate,
	}, nil
}

// GetOperationCost returns the credit cost for an operation type.
func (s *Service) GetOperationCost(op OperationType) int {
	return s.costs.GetCost(op)
}

// LogFailedOperation logs an operation that failed without charging credits.
func (s *Service) LogFailedOperation(ctx context.Context, req ChargeRequest, opErr error) error {
	userIdentity := normalizeIdentity(req.UserIdentity)
	errMsg := ""
	if opErr != nil {
		errMsg = opErr.Error()
	}
	return s.logOperation(ctx, userIdentity, req.Operation, 0, false, req.Metadata, errMsg)
}

// GetUsageHistory returns usage summaries for multiple billing periods.
func (s *Service) GetUsageHistory(ctx context.Context, userIdentity string, months, offset int) ([]UsageSummary, bool, error) {
	userIdentity = normalizeIdentity(userIdentity)

	if months <= 0 {
		months = 6 // Default to 6 months
	}
	if offset < 0 {
		offset = 0
	}

	// Calculate the billing months to query
	// Start from current month minus offset, go back 'months' periods
	now := time.Now()
	startMonth := firstDayOfMonth(now).AddDate(0, -offset, 0)

	// Query for months + 1 to check if there's more
	queryMonths := make([]string, 0, months+1)
	for i := 0; i <= months; i++ {
		m := startMonth.AddDate(0, -i, 0)
		queryMonths = append(queryMonths, m.Format("2006-01"))
	}

	// Query database for these months - aggregate across all user_identities for single-user desktop app
	placeholders := make([]string, len(queryMonths))
	args := make([]interface{}, len(queryMonths))
	for i, m := range queryMonths {
		placeholders[i] = "?"
		args[i] = m
	}
	query := fmt.Sprintf(`
		SELECT billing_month, total_credits_used, total_operations, credits_by_operation, operations_by_type
		FROM credit_usage
		WHERE billing_month IN (%s)
		ORDER BY billing_month DESC
	`, strings.Join(placeholders, ","))
	rows, err := s.db.QueryContext(ctx, query, args...)
	if err != nil {
		return nil, false, fmt.Errorf("query usage history: %w", err)
	}
	defer rows.Close()

	// Collect results into a map, aggregating across all user_identities per month
	usageByMonth := make(map[string]*usageCache)
	for rows.Next() {
		var month string
		var totalCreditsUsed, totalOperations int
		var creditsByOpJSON, opCountsJSON []byte

		if err := rows.Scan(&month, &totalCreditsUsed, &totalOperations, &creditsByOpJSON, &opCountsJSON); err != nil {
			return nil, false, fmt.Errorf("scan usage row: %w", err)
		}

		// Get or create the cache entry for this month
		existing := usageByMonth[month]
		if existing == nil {
			existing = &usageCache{
				totalCreditsUsed: 0,
				totalOperations:  0,
				byOperation:      make(map[OperationType]int),
				operationCounts:  make(map[OperationType]int),
				month:            month,
			}
			usageByMonth[month] = existing
		}

		// Aggregate totals
		existing.totalCreditsUsed += totalCreditsUsed
		existing.totalOperations += totalOperations

		// Aggregate operation breakdowns
		if len(creditsByOpJSON) > 0 {
			var rawByOp map[string]int
			if err := json.Unmarshal(creditsByOpJSON, &rawByOp); err == nil {
				for k, v := range rawByOp {
					existing.byOperation[OperationType(k)] += v
				}
			}
		}

		if len(opCountsJSON) > 0 {
			var rawCounts map[string]int
			if err := json.Unmarshal(opCountsJSON, &rawCounts); err == nil {
				for k, v := range rawCounts {
					existing.operationCounts[OperationType(k)] += v
				}
			}
		}
	}

	if err := rows.Err(); err != nil {
		return nil, false, fmt.Errorf("iterate usage rows: %w", err)
	}

	// Get credit limit for the user
	creditsLimit, err := s.getUserCreditsLimit(ctx, userIdentity)
	if err != nil {
		s.log.WithError(err).Warn("Failed to get credit limit for history")
		creditsLimit = -1
	}

	// Build summaries for the requested months
	summaries := make([]UsageSummary, 0, months)
	for i := 0; i < months && i < len(queryMonths); i++ {
		month := queryMonths[i]
		monthTime, _ := time.Parse("2006-01", month)

		usage := usageByMonth[month]
		if usage == nil {
			// No usage for this month - create empty summary
			usage = &usageCache{
				totalCreditsUsed: 0,
				totalOperations:  0,
				byOperation:      make(map[OperationType]int),
				operationCounts:  make(map[OperationType]int),
			}
		}

		creditsRemaining := -1
		if creditsLimit >= 0 {
			creditsRemaining = creditsLimit - usage.totalCreditsUsed
			if creditsRemaining < 0 {
				creditsRemaining = 0
			}
		}

		summaries = append(summaries, UsageSummary{
			UserIdentity:     userIdentity,
			BillingMonth:     month,
			TotalCreditsUsed: usage.totalCreditsUsed,
			TotalOperations:  usage.totalOperations,
			ByOperation:      usage.byOperation,
			OperationCounts:  usage.operationCounts,
			CreditsLimit:     creditsLimit,
			CreditsRemaining: creditsRemaining,
			PeriodStart:      firstDayOfMonth(monthTime),
			PeriodEnd:        lastDayOfMonth(monthTime),
			ResetDate:        firstDayOfMonth(monthTime).AddDate(0, 1, 0),
		})
	}

	// Check if there's more data
	hasMore := len(queryMonths) > months && usageByMonth[queryMonths[months]] != nil

	return summaries, hasMore, nil
}

// GetOperationLog returns paginated operation log entries for a billing period.
func (s *Service) GetOperationLog(ctx context.Context, userIdentity, month, category string, limit, offset int) (*OperationLogPage, error) {
	userIdentity = normalizeIdentity(userIdentity)

	if limit <= 0 {
		limit = 20
	}
	if limit > 100 {
		limit = 100 // Cap at 100
	}
	if offset < 0 {
		offset = 0
	}

	// Parse month to get date range
	monthTime, err := time.Parse("2006-01", month)
	if err != nil {
		monthTime = firstDayOfMonth(time.Now())
		month = monthTime.Format("2006-01")
	}
	monthStart := firstDayOfMonth(monthTime)
	monthEnd := lastDayOfMonth(monthTime)

	// Build category filter - no user_identity filter for single-user desktop app
	var categoryFilter string
	var args []interface{}

	args = append(args, monthStart, monthEnd)

	if category != "" {
		// Map category to operation type prefix
		switch category {
		case "ai":
			categoryFilter = " AND operation_type LIKE 'ai.%'"
		case "execution":
			categoryFilter = " AND operation_type LIKE 'execution.%'"
		case "export":
			categoryFilter = " AND operation_type LIKE 'export.%'"
		}
	}

	countQuery := fmt.Sprintf(`
		SELECT COUNT(*)
		FROM operation_log
		WHERE created_at >= ? AND created_at <= ?%s
	`, categoryFilter)

	query := fmt.Sprintf(`
		SELECT id, operation_type, credits_charged, success, created_at, metadata, error_message
		FROM operation_log
		WHERE created_at >= ? AND created_at <= ?%s
		ORDER BY created_at DESC
		LIMIT ? OFFSET ?
	`, categoryFilter)

	var total int
	if err := s.db.QueryRowContext(ctx, countQuery, args...).Scan(&total); err != nil {
		return nil, fmt.Errorf("count operations: %w", err)
	}

	args = append(args, limit, offset)

	rows, err := s.db.QueryContext(ctx, query, args...)
	if err != nil {
		return nil, fmt.Errorf("query operations: %w", err)
	}
	defer rows.Close()

	operations := make([]OperationLogEntry, 0, limit)
	for rows.Next() {
		var entry OperationLogEntry
		var opType string
		var metadataJSON []byte
		var errorMsg sql.NullString

		if err := rows.Scan(&entry.ID, &opType, &entry.CreditsCharged, &entry.Success, &entry.CreatedAt, &metadataJSON, &errorMsg); err != nil {
			return nil, fmt.Errorf("scan operation row: %w", err)
		}

		entry.OperationType = OperationType(opType)
		if errorMsg.Valid {
			entry.ErrorMessage = errorMsg.String
		}

		if len(metadataJSON) > 0 {
			var metadata map[string]interface{}
			if err := json.Unmarshal(metadataJSON, &metadata); err == nil {
				entry.Metadata = metadata
			}
		}

		operations = append(operations, entry)
	}

	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("iterate operation rows: %w", err)
	}

	return &OperationLogPage{
		UserIdentity: userIdentity,
		BillingMonth: month,
		Operations:   operations,
		Total:        total,
		Limit:        limit,
		Offset:       offset,
		HasMore:      offset+len(operations) < total,
	}, nil
}

// getUserCreditsLimit gets the credit limit for a user based on their entitlement tier.
//
// Returns:
//   - -1: Unlimited (no provider, business tier, or fail-open on errors)
//   - 0: No access (free tier with AI disabled)
//   - >0: Credit limit for the billing period
func (s *Service) getUserCreditsLimit(ctx context.Context, userIdentity string) (int, error) {
	// Use helper that checks context first (respects tier overrides)
	ent, err := s.getEntitlement(ctx, userIdentity)
	if err != nil {
		s.log.WithError(err).Warn("Failed to get entitlement, assuming unlimited")
		return -1, nil // Fail open
	}

	// No entitlement means no entitlement provider configured - unlimited
	if ent == nil || s.entitlementProvider == nil {
		return -1, nil
	}

	return s.entitlementProvider.GetAICreditsLimit(ent.Tier), nil
}

// getRemainingCredits returns the remaining credits for a user.
func (s *Service) getRemainingCredits(ctx context.Context, userIdentity string) (int, error) {
	creditsLimit, err := s.getUserCreditsLimit(ctx, userIdentity)
	if err != nil {
		return -1, err
	}

	if creditsLimit < 0 {
		return -1, nil // Unlimited
	}

	usage, err := s.getUsageFromDB(ctx, userIdentity)
	if err != nil {
		return 0, err
	}

	remaining := creditsLimit - usage.totalCreditsUsed
	if remaining < 0 {
		remaining = 0
	}
	return remaining, nil
}

// getUsageFromDB queries the database for credit usage.
// For single-user desktop apps, this aggregates ALL usage regardless of user_identity.
func (s *Service) getUsageFromDB(ctx context.Context, userIdentity string) (*usageCache, error) {
	currentMonth := s.getBillingMonth(ctx, userIdentity)

	query := `
		SELECT
			COALESCE(SUM(total_credits_used), 0) as total_credits,
			COALESCE(SUM(total_operations), 0) as total_ops
		FROM credit_usage
		WHERE billing_month = ?
	`

	var totalCreditsUsed, totalOperations int

	err := s.db.QueryRowContext(ctx, query, currentMonth).Scan(
		&totalCreditsUsed, &totalOperations,
	)

	if err != nil && err != sql.ErrNoRows {
		return nil, fmt.Errorf("query credit usage: %w", err)
	}

	// Get detailed breakdown with a separate query
	breakdownQuery := `
		SELECT credits_by_operation, operations_by_type
		FROM credit_usage
		WHERE billing_month = ?
	`

	rows, err := s.db.QueryContext(ctx, breakdownQuery, currentMonth)
	if err != nil {
		return nil, fmt.Errorf("query credit breakdown: %w", err)
	}
	defer rows.Close()

	byOperation := make(map[OperationType]int)
	operationCounts := make(map[OperationType]int)

	for rows.Next() {
		var creditsByOpJSON, opCountsJSON []byte
		if err := rows.Scan(&creditsByOpJSON, &opCountsJSON); err != nil {
			continue
		}

		if len(creditsByOpJSON) > 0 {
			var rawByOp map[string]int
			if err := json.Unmarshal(creditsByOpJSON, &rawByOp); err == nil {
				for k, v := range rawByOp {
					byOperation[OperationType(k)] += v
				}
			}
		}

		if len(opCountsJSON) > 0 {
			var rawCounts map[string]int
			if err := json.Unmarshal(opCountsJSON, &rawCounts); err == nil {
				for k, v := range rawCounts {
					operationCounts[OperationType(k)] += v
				}
			}
		}
	}

	return &usageCache{
		totalCreditsUsed: totalCreditsUsed,
		totalOperations:  totalOperations,
		byOperation:      byOperation,
		operationCounts:  operationCounts,
		month:            currentMonth,
		updatedAt:        time.Now(),
	}, nil
}

// upsertUsage increments credit usage in the database using a read-modify-write
// transaction. SQLite has no JSONB merge operators, so the per-operation breakdown
// JSON is rehydrated, mutated, and re-serialized in Go.
func (s *Service) upsertUsage(ctx context.Context, userIdentity, month string, op OperationType, credits int) (err error) {
	tx, err := s.db.BeginTx(ctx, nil)
	if err != nil {
		return fmt.Errorf("begin transaction: %w", err)
	}
	defer func() {
		if err == nil {
			return
		}
		if rollbackErr := tx.Rollback(); rollbackErr != nil && rollbackErr != sql.ErrTxDone {
			s.log.WithError(rollbackErr).Warn("Failed to rollback credit usage transaction")
		}
	}()

	// Try to get existing row
	var existingID string
	var totalCreditsUsed, totalOperations int
	var creditsByOpJSON, opsByTypeJSON string

	err = tx.QueryRowContext(ctx, `
		SELECT id, total_credits_used, total_operations, credits_by_operation, operations_by_type
		FROM credit_usage
		WHERE user_identity = ? AND billing_month = ?
	`, userIdentity, month).Scan(&existingID, &totalCreditsUsed, &totalOperations, &creditsByOpJSON, &opsByTypeJSON)

	opKey := string(op)

	if err == sql.ErrNoRows {
		// Insert new row
		newID := uuid.New().String()
		creditsByOp := map[string]int{opKey: credits}
		opsByType := map[string]int{opKey: 1}
		creditsByOpBytes, _ := json.Marshal(creditsByOp)
		opsByTypeBytes, _ := json.Marshal(opsByType)

		_, err = tx.ExecContext(ctx, `
			INSERT INTO credit_usage (id, user_identity, billing_month, total_credits_used, total_operations, credits_by_operation, operations_by_type, last_operation_at, created_at, updated_at)
			VALUES (?, ?, ?, ?, 1, ?, ?, CURRENT_TIMESTAMP, CURRENT_TIMESTAMP, CURRENT_TIMESTAMP)
		`, newID, userIdentity, month, credits, string(creditsByOpBytes), string(opsByTypeBytes))
		if err != nil {
			return fmt.Errorf("insert credit usage: %w", err)
		}
	} else if err != nil {
		return fmt.Errorf("query existing credit usage: %w", err)
	} else {
		// Update existing row - parse JSON, increment, re-serialize
		var creditsByOp map[string]int
		var opsByType map[string]int
		if err := json.Unmarshal([]byte(creditsByOpJSON), &creditsByOp); err != nil || creditsByOp == nil {
			creditsByOp = make(map[string]int)
		}
		if err := json.Unmarshal([]byte(opsByTypeJSON), &opsByType); err != nil || opsByType == nil {
			opsByType = make(map[string]int)
		}

		creditsByOp[opKey] += credits
		opsByType[opKey]++

		creditsByOpBytes, _ := json.Marshal(creditsByOp)
		opsByTypeBytes, _ := json.Marshal(opsByType)

		_, err = tx.ExecContext(ctx, `
			UPDATE credit_usage SET
				total_credits_used = total_credits_used + ?,
				total_operations = total_operations + 1,
				credits_by_operation = ?,
				operations_by_type = ?,
				last_operation_at = CURRENT_TIMESTAMP,
				updated_at = CURRENT_TIMESTAMP
			WHERE id = ?
		`, credits, string(creditsByOpBytes), string(opsByTypeBytes), existingID)
		if err != nil {
			return fmt.Errorf("update credit usage: %w", err)
		}
	}

	if commitErr := tx.Commit(); commitErr != nil {
		err = fmt.Errorf("commit credit usage: %w", commitErr)
		return err
	}

	return nil
}

// logOperation inserts a log entry for an operation.
func (s *Service) logOperation(ctx context.Context, userIdentity string, op OperationType, credits int, success bool, metadata ChargeMetadata, errMsg string) error {
	if s.db == nil {
		return nil
	}

	metadataJSON, _ := json.Marshal(metadata)

	// SQLite uses INTEGER for boolean (0/1).
	successVal := 0
	if success {
		successVal = 1
	}

	_, err := s.db.ExecContext(ctx, `
		INSERT INTO operation_log (
			id, user_identity, operation_type, credits_charged, success, metadata, error_message, duration_ms
		) VALUES (?, ?, ?, ?, ?, ?, ?, ?)
	`,
		uuid.New().String(),
		userIdentity,
		string(op),
		credits,
		successVal,
		string(metadataJSON),
		errMsg,
		metadata.DurationMs,
	)
	if err != nil {
		return fmt.Errorf("insert operation log: %w", err)
	}

	return nil
}

// LPBS usage reporting

// LPBSUsageReportMetadata contains additional context for usage reports.
type LPBSUsageReportMetadata struct {
	Operation    string `json:"operation,omitempty"`
	Model        string `json:"model,omitempty"`
	PromptTokens int    `json:"prompt_tokens,omitempty"`
	IsBYOK       bool   `json:"is_byok,omitempty"`
}

// LPBSUsageReport represents the request payload for LPBS usage reporting.
// Exported for testing purposes.
type LPBSUsageReport struct {
	UserIdentity string                  `json:"user_identity"`
	LimitKey     string                  `json:"limit_key"`
	UsageAmount  int64                   `json:"usage_amount"` // In internal units (cents × 1,000,000)
	Amount       int64                   `json:"amount"`       // Alias for LPBS compatibility
	AppBundleKey string                  `json:"app_bundle_key"`
	OperationID  string                  `json:"operation_id,omitempty"` // Idempotency key - same ID across retries prevents double-counting
	Metadata     LPBSUsageReportMetadata `json:"metadata,omitempty"`
}

// lpbsUsageReport is an alias for internal use.
type lpbsUsageReport = LPBSUsageReport

// sendLPBSReport sends a single usage report to LPBS. Returns error on failure.
func (s *Service) sendLPBSReport(ctx context.Context, report lpbsUsageReport) error {
	body, err := json.Marshal(report)
	if err != nil {
		return fmt.Errorf("marshal: %w", err)
	}

	req, err := http.NewRequestWithContext(ctx, http.MethodPost, s.lpbsURL+"/api/v1/usage/report", bytes.NewReader(body))
	if err != nil {
		return fmt.Errorf("create request: %w", err)
	}

	req.Header.Set("Content-Type", "application/json")
	if s.lpbsSecret != "" {
		req.Header.Set("Authorization", "Bearer "+s.lpbsSecret)
	}

	resp, err := s.lpbsHTTPClient.Do(req)
	if err != nil {
		return fmt.Errorf("send: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK && resp.StatusCode != http.StatusCreated {
		return fmt.Errorf("status %d", resp.StatusCode)
	}
	return nil
}

// reportUsageToLPBS asynchronously reports usage to the LPBS centralized tracking system.
// This is called after a successful local charge. Failures are logged but do not affect local operations.
// Includes retry logic with exponential backoff for transient failures.
// The operation_id ensures idempotency - the same ID is used across all retries to prevent double-counting.
func (s *Service) reportUsageToLPBS(userIdentity string, op OperationType, localCredits int, actualCostCents float64, isBYOK bool, metadata ChargeMetadata) {
	// Skip if LPBS is not configured (neither URL nor custom reporter)
	if s.lpbsURL == "" && s.lpbsReporter == nil {
		return
	}

	// Generate operation_id ONCE - this ensures retries don't double-count
	operationID := uuid.New().String()

	// Convert cost to LPBS internal units (cents × 1,000,000)
	// If actual cost is provided (from OpenRouter), use it; otherwise estimate from local credits
	var usageAmount int64
	if actualCostCents > 0 {
		// Actual cost from API provider (in cents)
		usageAmount = int64(actualCostCents * 1_000_000)
	} else if !isBYOK && localCredits > 0 {
		// Estimate: 1 local credit ≈ $0.001 = 0.1 cents
		// So usageAmount = localCredits * 0.1 cents * 1_000_000 = localCredits * 100_000
		usageAmount = int64(localCredits) * 100_000
	}
	// BYOK operations get 0 cost (user pays their own way)

	report := lpbsUsageReport{
		UserIdentity: userIdentity,
		LimitKey:     "ai_credits",
		UsageAmount:  usageAmount,
		Amount:       usageAmount, // Alias for LPBS compatibility
		AppBundleKey: s.appBundleKey,
		OperationID:  operationID,
	}
	report.Metadata.Operation = string(op)
	report.Metadata.Model = metadata.Model
	report.Metadata.PromptTokens = metadata.PromptTokens
	report.Metadata.IsBYOK = isBYOK

	// If a custom reporter is provided (e.g., for testing), use it synchronously
	if s.lpbsReporter != nil {
		ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
		defer cancel()
		if err := s.lpbsReporter.ReportUsage(ctx, report); err != nil {
			s.log.WithError(err).Debug("lpbs: custom reporter failed to send usage report")
		}
		return
	}

	// Run asynchronously with retry logic to not block the local charge
	go func() {
		const maxRetries = 3
		baseDelay := 500 * time.Millisecond

		for attempt := 0; attempt < maxRetries; attempt++ {
			ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
			err := s.sendLPBSReport(ctx, report)
			cancel()

			if err == nil {
				s.log.WithFields(logrus.Fields{
					"user":         userIdentity,
					"operation_id": operationID,
				}).Debug("lpbs: usage report sent")
				return
			}

			// Final attempt - log at Warn level for visibility
			if attempt == maxRetries-1 {
				s.log.WithError(err).WithFields(logrus.Fields{
					"user":         userIdentity,
					"operation_id": operationID,
					"attempts":     maxRetries,
				}).Warn("lpbs: usage report failed after retries")
				return
			}

			// Exponential backoff: 500ms, 1s, 2s
			time.Sleep(baseDelay * time.Duration(1<<attempt))
		}
	}()
}

// LPBS Health Check

// LPBSHealthStatus contains the health status of the LPBS connection.
type LPBSHealthStatus struct {
	Configured bool       `json:"configured"`
	Reachable  bool       `json:"reachable"`
	LastSync   *time.Time `json:"last_sync,omitempty"`
	LastError  string     `json:"last_error,omitempty"`
}

// CheckLPBSHealth checks the connectivity to the LPBS service.
// Returns a status indicating whether LPBS is configured and reachable.
func (s *Service) CheckLPBSHealth(ctx context.Context) *LPBSHealthStatus {
	status := &LPBSHealthStatus{
		Configured: s.lpbsURL != "" || s.lpbsReporter != nil,
		Reachable:  false,
	}

	// If not configured, return early
	if !status.Configured {
		return status
	}

	// If using a custom reporter (for testing), assume reachable
	if s.lpbsReporter != nil {
		status.Reachable = true
		return status
	}

	// Try to reach the LPBS health endpoint
	healthURL := s.lpbsURL + "/api/v1/usage/health"
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, healthURL, nil)
	if err != nil {
		status.LastError = fmt.Sprintf("create request: %v", err)
		return status
	}

	resp, err := s.lpbsHTTPClient.Do(req)
	if err != nil {
		status.LastError = fmt.Sprintf("connect: %v", err)
		return status
	}
	defer resp.Body.Close()

	if resp.StatusCode == http.StatusOK {
		status.Reachable = true
	} else {
		status.LastError = fmt.Sprintf("status %d", resp.StatusCode)
	}

	return status
}

// Cache management

func (s *Service) getCached(userIdentity, month string) *usageCache {
	s.cacheMu.RLock()
	defer s.cacheMu.RUnlock()

	cached, ok := s.cache[userIdentity]
	if !ok {
		return nil
	}

	// Check if cache is for current month and not too stale (1 minute)
	if cached.month != month || time.Since(cached.updatedAt) > time.Minute {
		return nil
	}

	return cached
}

func (s *Service) setCached(userIdentity, month string, usage *usageCache) {
	s.cacheMu.Lock()
	defer s.cacheMu.Unlock()

	usage.month = month
	usage.updatedAt = time.Now()
	s.cache[userIdentity] = usage
}

func (s *Service) invalidateCache(userIdentity string) {
	s.cacheMu.Lock()
	delete(s.cache, userIdentity)
	s.cacheMu.Unlock()
}

// Helper functions

func normalizeIdentity(identity string) string {
	return strings.TrimSpace(strings.ToLower(identity))
}

func firstDayOfMonth(t time.Time) time.Time {
	return time.Date(t.Year(), t.Month(), 1, 0, 0, 0, 0, t.Location())
}

func lastDayOfMonth(t time.Time) time.Time {
	return firstDayOfMonth(t).AddDate(0, 1, 0).Add(-time.Nanosecond)
}

// getBillingMonth returns the billing period key for the user.
// Uses custom billing cycle if set, otherwise calendar month.
func (s *Service) getBillingMonth(ctx context.Context, userIdentity string) string {
	ent, err := s.getEntitlement(ctx, userIdentity)
	if err != nil || ent == nil || ent.BillingCycleStart < 1 || ent.BillingCycleStart > 28 {
		return time.Now().Format("2006-01")
	}
	return ent.GetBillingMonth(time.Now())
}

// Ensure Service implements CreditService
var _ CreditService = (*Service)(nil)
