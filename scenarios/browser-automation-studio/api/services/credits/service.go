package credits

import (
	"context"
	"database/sql"
	"encoding/json"
	"errors"
	"fmt"
	"net/http"
	"strings"
	"sync"
	"time"

	"github.com/sirupsen/logrus"
	coredb "github.com/vrooli/api-core/database"
	"github.com/vrooli/browser-automation-studio/services/entitlement"
	monetization "github.com/vrooli/vrooli/packages/monetization-go"
)

// Service implements the local account usage read model and durable usage outbox.
//
// # Architecture
//
// The Service coordinates between several subsystems:
//   - Database layer: Reads legacy credit_usage and operation_log records
//   - Entitlement layer: Determines tier limits via EntitlementProvider interface
//   - LPBS layer: Durable outbox used by the governed desktop usage journey
//   - Cache layer: In-memory cache with 1-minute TTL for fast lookups
//
// # Testing Seams
//
// The Service has several intentional seams for testability:
//   - EntitlementProvider: Interface for entitlement lookups (mock in tests)
//
// See SEAMS.md for full documentation of testing boundaries.
type Service struct {
	db                  sqlExecutor
	log                 *logrus.Logger
	entitlementSvc      *entitlement.Service // Production source of verified shared snapshots.
	entitlementProvider EntitlementProvider  // Preferred: injectable interface for testing

	// LPBS integration for the shared durable usage outbox.
	lpbsURL            string
	lpbsHTTPClient     *http.Client // HTTP client for LPBS requests
	lpbsAccessToken    func(context.Context) (string, error)
	appBundleKey       string // App identifier for usage records
	monetizationOutbox *monetization.Outbox

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
// Tests inject EntitlementProvider to keep entitlement reads isolated.
//
// # Example Test Setup
//
//	svc := credits.NewService(credits.ServiceOptions{
//	    DB:      sqliteDB,
//	    Logger:  logrus.New(),
//	    EntitlementProvider: testEntitlementProvider,
//
//	})
type ServiceOptions struct {
	DB             sqlExecutor
	Logger         *logrus.Logger
	EntitlementSvc *entitlement.Service // Production source; test seams use EntitlementProvider.

	// EntitlementProvider is the preferred way to inject entitlement dependencies.
	// If set, this takes precedence over EntitlementSvc.
	// Tests can inject an implementation to control entitlement behavior.
	EntitlementProvider EntitlementProvider

	// LPBS integration for the durable usage outbox.
	LPBSURL      string
	AppBundleKey string // App identifier (default: "browser-automation-studio")

	// LPBSAccessToken is a test seam for an already-resolved consumer token.
	// Production composition leaves it unset and uses the entitlement session.
	LPBSAccessToken func(context.Context) (string, error)
}

// sqlExecutor keeps the credit domain independent of a concrete SQL pool and
// lets request contexts select the leased Test Genie database in production.
type sqlExecutor interface {
	ExecContext(context.Context, string, ...any) (sql.Result, error)
	QueryContext(context.Context, string, ...any) (*sql.Rows, error)
	QueryRowContext(context.Context, string, ...any) *sql.Row
	BeginTx(context.Context, *sql.TxOptions) (*sql.Tx, error)
}

var (
	_ sqlExecutor = (*sql.DB)(nil)
	_ sqlExecutor = (*coredb.RoutedDB)(nil)
)

// NewService creates a new CreditService.
//
// Production composition supplies EntitlementSvc, whose snapshots come from
// the shared signed-lease client. Tests may inject EntitlementProvider to keep
// local entitlement reads isolated.
func NewService(opts ServiceOptions) *Service {
	appBundleKey := opts.AppBundleKey
	if appBundleKey == "" {
		appBundleKey = "browser-automation-studio"
	}

	var lpbsHTTPClient *http.Client
	if opts.LPBSURL != "" {
		lpbsHTTPClient = &http.Client{
			Timeout: 5 * time.Second, // Bound LPBS outbox delivery requests.
		}
	}

	service := &Service{
		db:                  opts.DB,
		log:                 opts.Logger,
		entitlementSvc:      opts.EntitlementSvc, // Keep for backward compat
		entitlementProvider: opts.EntitlementProvider,
		lpbsURL:             opts.LPBSURL,
		lpbsHTTPClient:      lpbsHTTPClient,
		lpbsAccessToken:     opts.LPBSAccessToken,
		appBundleKey:        appBundleKey,
		cache:               make(map[string]*usageCache),
	}
	if service.db != nil && service.lpbsURL != "" {
		resolveToken := func(ctx context.Context, _ string) (string, error) {
			return service.resolveLPBSAccess(ctx)
		}
		transport := monetization.NewLPBSUsageTransport(service.lpbsURL, resolveToken, service.lpbsHTTPClient)
		service.monetizationOutbox = monetization.NewOutbox(monetization.NewSQLStore(service.db, monetization.SQLDialectSQLite), transport)
	}
	return service
}

// getEntitlement retrieves the entitlement for a user, checking context first
// (for middleware overrides like tier testing), then falling back to the entitlement provider.
//
// Lookup order:
//  1. Shared verified snapshot already attached to this request.
//  2. Injected EntitlementProvider used by isolated tests.
//  3. Production entitlement.Service backed by the shared lease client.
//  4. Nil when neither source is configured (standalone local use).
func (s *Service) getEntitlement(ctx context.Context, userIdentity string) (*entitlement.Entitlement, error) {
	// Check context first - respects middleware overrides (e.g., tier override for testing)
	if ent := monetization.SnapshotFromContext(ctx); ent != nil {
		return ent, nil
	}

	// The injectable provider is a test seam; production uses entitlement.Service.
	if s.entitlementProvider != nil {
		return s.entitlementProvider.GetEntitlement(ctx, userIdentity)
	}
	if s.entitlementSvc != nil {
		return s.entitlementSvc.GetEntitlement(ctx, userIdentity)
	}

	// No provider configured - return nil (treated as unlimited)
	return nil, nil
}

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
	if ent == nil {
		return -1, nil
	}
	if limits, ok := s.entitlementProvider.(EntitlementLimitProvider); ok {
		if limit, found := limits.LimitForEntitlement(ent); found {
			return limit, nil
		}
	}
	if limit, ok := ent.LimitValueForApp("browser-automation-studio", "ai_credits", "business_suite"); ok {
		return int(limit), nil
	}
	return 0, nil
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

// DrainOutbox delivers pending reports using the current consumer session.
// A delivered operation remains durably marked; LPBS also deduplicates by the
// operation_id carried in the payload, making retries exactly-once upstream.
func (s *Service) DrainOutbox(ctx context.Context, limit int) (int, error) {
	if s.lpbsURL == "" || limit <= 0 {
		return 0, nil
	}
	if s.monetizationOutbox == nil {
		return 0, errors.New("shared usage outbox is unavailable")
	}
	return s.monetizationOutbox.Drain(ctx, limit)
}

func (s *Service) resolveLPBSAccess(ctx context.Context) (string, error) {
	if s.lpbsAccessToken != nil {
		return s.lpbsAccessToken(ctx)
	}
	if s.entitlementSvc == nil {
		return "", errors.New("entitlement session resolver is unavailable")
	}
	return s.entitlementSvc.ResolveAccessToken(ctx, s.lpbsURL)
}

// StartOutboxDrainer starts the process-owned durable retry loop.
func (s *Service) StartOutboxDrainer(ctx context.Context, interval time.Duration) {
	if interval <= 0 {
		interval = 10 * time.Second
	}
	go func() {
		ticker := time.NewTicker(interval)
		defer ticker.Stop()
		for {
			select {
			case <-ctx.Done():
				return
			case <-ticker.C:
				if _, err := s.DrainOutbox(ctx, 25); err != nil {
					s.log.WithError(err).Debug("lpbs: outbox drain failed")
				}
			}
		}
	}()
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
