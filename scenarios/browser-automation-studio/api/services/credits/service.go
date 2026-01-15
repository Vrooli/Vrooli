package credits

import (
	"context"
	"database/sql"
	"encoding/json"
	"fmt"
	"strings"
	"sync"
	"time"

	"github.com/lib/pq"
	"github.com/sirupsen/logrus"
	"github.com/vrooli/browser-automation-studio/services/entitlement"
)

// Service implements CreditService with database persistence and caching.
type Service struct {
	db             *sql.DB
	log            *logrus.Logger
	entitlementSvc *entitlement.Service
	costs          OperationCosts
	enabled        bool

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
type ServiceOptions struct {
	DB             *sql.DB
	Logger         *logrus.Logger
	EntitlementSvc *entitlement.Service
	Enabled        bool // Whether credit tracking is enabled
	// Note: Operation costs are intentionally NOT configurable here.
	// They are hard-coded in DefaultOperationCosts() to prevent bypassing charges.
}

// NewService creates a new CreditService.
func NewService(opts ServiceOptions) *Service {
	return &Service{
		db:             opts.DB,
		log:            opts.Logger,
		entitlementSvc: opts.EntitlementSvc,
		costs:          DefaultOperationCosts(),
		enabled:        opts.Enabled,
		cache:          make(map[string]*usageCache),
	}
}

// IsEnabled reports whether credit tracking is enabled.
func (s *Service) IsEnabled() bool {
	return s.enabled
}

// CanCharge checks if the user has sufficient credits for the operation.
func (s *Service) CanCharge(ctx context.Context, userIdentity string, op OperationType) (bool, int, error) {
	// If credits disabled, always allow
	if !s.enabled {
		return true, -1, nil
	}

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

	// If credits disabled or no user identity, just log but don't charge
	if !s.enabled {
		// Still log the operation for analytics
		_ = s.logOperation(ctx, userIdentity, req.Operation, 0, true, req.Metadata, "")
		return &ChargeResult{Charged: 0, RemainingCredits: -1, WasCharged: false}, nil
	}

	if userIdentity == "" {
		// Can't charge without user identity, but don't fail
		s.log.Debug("Cannot charge credits: no user identity")
		return &ChargeResult{Charged: 0, RemainingCredits: -1, WasCharged: false}, nil
	}

	cost := s.costs.GetCost(req.Operation)

	// Free operations - just log
	if cost == 0 {
		_ = s.logOperation(ctx, userIdentity, req.Operation, 0, true, req.Metadata, "")
		remaining, _ := s.getRemainingCredits(ctx, userIdentity)
		return &ChargeResult{Charged: 0, RemainingCredits: remaining, WasCharged: false}, nil
	}

	currentMonth := time.Now().Format("2006-01")

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

	// Get remaining balance
	remaining, _ := s.getRemainingCredits(ctx, userIdentity)

	return &ChargeResult{
		Charged:          cost,
		RemainingCredits: remaining,
		WasCharged:       true,
	}, nil
}

// ChargeIfAllowed combines CanCharge and Charge atomically.
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

// GetUsage returns the usage summary for a user in the current billing period.
func (s *Service) GetUsage(ctx context.Context, userIdentity string) (*UsageSummary, error) {
	userIdentity = normalizeIdentity(userIdentity)
	currentMonth := time.Now().Format("2006-01")

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
	creditsLimit := -1 // Default to unlimited
	creditsRemaining := -1
	if s.enabled {
		var err error
		creditsLimit, err = s.getUserCreditsLimit(ctx, userIdentity)
		if err != nil {
			s.log.WithError(err).Warn("Failed to get credit limit, assuming unlimited")
		}
		if creditsLimit >= 0 {
			creditsRemaining = creditsLimit - usage.totalCreditsUsed
			if creditsRemaining < 0 {
				creditsRemaining = 0
			}
		}
	}

	now := time.Now()
	return &UsageSummary{
		UserIdentity:     userIdentity,
		BillingMonth:     currentMonth,
		TotalCreditsUsed: usage.totalCreditsUsed,
		TotalOperations:  usage.totalOperations,
		ByOperation:      usage.byOperation,
		OperationCounts:  usage.operationCounts,
		CreditsLimit:     creditsLimit,
		CreditsRemaining: creditsRemaining,
		PeriodStart:      firstDayOfMonth(now),
		PeriodEnd:        lastDayOfMonth(now),
		ResetDate:        firstDayOfMonth(now).AddDate(0, 1, 0),
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

	// Query database for these months
	query := `
		SELECT billing_month, total_credits_used, total_operations, credits_by_operation, operations_by_type
		FROM credit_usage
		WHERE user_identity = $1 AND billing_month = ANY($2)
		ORDER BY billing_month DESC
	`

	rows, err := s.db.QueryContext(ctx, query, userIdentity, pq.Array(queryMonths))
	if err != nil {
		return nil, false, fmt.Errorf("query usage history: %w", err)
	}
	defer rows.Close()

	// Collect results into a map
	usageByMonth := make(map[string]*usageCache)
	for rows.Next() {
		var month string
		var totalCreditsUsed, totalOperations int
		var creditsByOpJSON, opCountsJSON []byte

		if err := rows.Scan(&month, &totalCreditsUsed, &totalOperations, &creditsByOpJSON, &opCountsJSON); err != nil {
			return nil, false, fmt.Errorf("scan usage row: %w", err)
		}

		byOperation := make(map[OperationType]int)
		operationCounts := make(map[OperationType]int)

		if len(creditsByOpJSON) > 0 {
			var rawByOp map[string]int
			if err := json.Unmarshal(creditsByOpJSON, &rawByOp); err == nil {
				for k, v := range rawByOp {
					byOperation[OperationType(k)] = v
				}
			}
		}

		if len(opCountsJSON) > 0 {
			var rawCounts map[string]int
			if err := json.Unmarshal(opCountsJSON, &rawCounts); err == nil {
				for k, v := range rawCounts {
					operationCounts[OperationType(k)] = v
				}
			}
		}

		usageByMonth[month] = &usageCache{
			totalCreditsUsed: totalCreditsUsed,
			totalOperations:  totalOperations,
			byOperation:      byOperation,
			operationCounts:  operationCounts,
			month:            month,
		}
	}

	if err := rows.Err(); err != nil {
		return nil, false, fmt.Errorf("iterate usage rows: %w", err)
	}

	// Get credit limit for the user
	creditsLimit := -1
	if s.enabled {
		var err error
		creditsLimit, err = s.getUserCreditsLimit(ctx, userIdentity)
		if err != nil {
			s.log.WithError(err).Warn("Failed to get credit limit for history")
		}
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

	// Build category filter
	var categoryFilter string
	var args []interface{}

	args = append(args, userIdentity, monthStart, monthEnd)

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

	// Count total
	countQuery := fmt.Sprintf(`
		SELECT COUNT(*)
		FROM operation_log
		WHERE user_identity = $1 AND created_at >= $2 AND created_at <= $3%s
	`, categoryFilter)

	var total int
	if err := s.db.QueryRowContext(ctx, countQuery, args...).Scan(&total); err != nil {
		return nil, fmt.Errorf("count operations: %w", err)
	}

	// Query operations
	query := fmt.Sprintf(`
		SELECT id, operation_type, credits_charged, success, created_at, metadata, error_message
		FROM operation_log
		WHERE user_identity = $1 AND created_at >= $2 AND created_at <= $3%s
		ORDER BY created_at DESC
		LIMIT $4 OFFSET $5
	`, categoryFilter)

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
func (s *Service) getUserCreditsLimit(ctx context.Context, userIdentity string) (int, error) {
	if s.entitlementSvc == nil || !s.entitlementSvc.IsEnabled() {
		return -1, nil // Unlimited when entitlements disabled
	}

	ent, err := s.entitlementSvc.GetEntitlement(ctx, userIdentity)
	if err != nil {
		s.log.WithError(err).Warn("Failed to get entitlement, assuming unlimited")
		return -1, nil // Fail open
	}

	return s.entitlementSvc.GetAICreditsLimit(ent.Tier), nil
}

// getRemainingCredits returns the remaining credits for a user.
func (s *Service) getRemainingCredits(ctx context.Context, userIdentity string) (int, error) {
	if !s.enabled {
		return -1, nil
	}

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
func (s *Service) getUsageFromDB(ctx context.Context, userIdentity string) (*usageCache, error) {
	currentMonth := time.Now().Format("2006-01")

	query := `
		SELECT total_credits_used, total_operations, credits_by_operation, operations_by_type
		FROM credit_usage
		WHERE user_identity = $1 AND billing_month = $2
	`

	var totalCreditsUsed, totalOperations int
	var creditsByOpJSON, opCountsJSON []byte

	err := s.db.QueryRowContext(ctx, query, userIdentity, currentMonth).Scan(
		&totalCreditsUsed, &totalOperations, &creditsByOpJSON, &opCountsJSON,
	)
	if err == sql.ErrNoRows {
		// No usage yet
		return &usageCache{
			totalCreditsUsed: 0,
			totalOperations:  0,
			byOperation:      make(map[OperationType]int),
			operationCounts:  make(map[OperationType]int),
			month:            currentMonth,
			updatedAt:        time.Now(),
		}, nil
	}
	if err != nil {
		return nil, fmt.Errorf("query credit usage: %w", err)
	}

	// Parse JSONB fields
	byOperation := make(map[OperationType]int)
	operationCounts := make(map[OperationType]int)

	if len(creditsByOpJSON) > 0 {
		var rawByOp map[string]int
		if err := json.Unmarshal(creditsByOpJSON, &rawByOp); err == nil {
			for k, v := range rawByOp {
				byOperation[OperationType(k)] = v
			}
		}
	}

	if len(opCountsJSON) > 0 {
		var rawCounts map[string]int
		if err := json.Unmarshal(opCountsJSON, &rawCounts); err == nil {
			for k, v := range rawCounts {
				operationCounts[OperationType(k)] = v
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

// upsertUsage increments credit usage in the database.
func (s *Service) upsertUsage(ctx context.Context, userIdentity, month string, op OperationType, credits int) error {
	// We need to use a PostgreSQL function to properly increment JSONB values
	query := `
		INSERT INTO credit_usage (
			user_identity, billing_month, total_credits_used, total_operations,
			credits_by_operation, operations_by_type, last_operation_at, updated_at
		)
		VALUES (
			$1, $2, $3, 1,
			jsonb_build_object($4::text, $3),
			jsonb_build_object($4::text, 1),
			NOW(), NOW()
		)
		ON CONFLICT (user_identity, billing_month)
		DO UPDATE SET
			total_credits_used = credit_usage.total_credits_used + $3,
			total_operations = credit_usage.total_operations + 1,
			credits_by_operation = credit_usage.credits_by_operation ||
				jsonb_build_object($4::text, COALESCE((credit_usage.credits_by_operation->>$4::text)::int, 0) + $3),
			operations_by_type = credit_usage.operations_by_type ||
				jsonb_build_object($4::text, COALESCE((credit_usage.operations_by_type->>$4::text)::int, 0) + 1),
			last_operation_at = NOW(),
			updated_at = NOW()
	`

	_, err := s.db.ExecContext(ctx, query, userIdentity, month, credits, string(op))
	if err != nil {
		return fmt.Errorf("upsert credit usage: %w", err)
	}

	return nil
}

// logOperation inserts a log entry for an operation.
func (s *Service) logOperation(ctx context.Context, userIdentity string, op OperationType, credits int, success bool, metadata ChargeMetadata, errMsg string) error {
	if s.db == nil {
		return nil
	}

	metadataJSON, _ := json.Marshal(metadata)

	query := `
		INSERT INTO operation_log (
			user_identity, operation_type, credits_charged, success, metadata, error_message, duration_ms
		) VALUES ($1, $2, $3, $4, $5, $6, $7)
	`

	_, err := s.db.ExecContext(ctx, query,
		userIdentity,
		string(op),
		credits,
		success,
		metadataJSON,
		errMsg,
		metadata.DurationMs,
	)
	if err != nil {
		return fmt.Errorf("insert operation log: %w", err)
	}

	return nil
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

// Ensure Service implements CreditService
var _ CreditService = (*Service)(nil)
