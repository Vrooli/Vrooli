package entitlement

import (
	"context"
	"database/sql"
	"fmt"
	"strings"
	"sync"
	"time"

	"github.com/sirupsen/logrus"
)

// UsageTracker tracks execution usage per user for entitlement enforcement.
type UsageTracker struct {
	db  *sql.DB
	log *logrus.Logger

	// In-memory cache for fast lookups
	cacheMu sync.RWMutex
	cache   map[string]*usageCache
}

type usageCache struct {
	count     int
	month     string // YYYY-MM format
	updatedAt time.Time
}

// NewUsageTracker creates a new usage tracker.
func NewUsageTracker(db *sql.DB, log *logrus.Logger) *UsageTracker {
	return &UsageTracker{
		db:    db,
		log:   log,
		cache: make(map[string]*usageCache),
	}
}

// GetMonthlyExecutionCount returns the number of executions for a user in the current month.
func (t *UsageTracker) GetMonthlyExecutionCount(ctx context.Context, userIdentity string) (int, error) {
	userIdentity = strings.TrimSpace(strings.ToLower(userIdentity))
	if userIdentity == "" {
		return 0, nil
	}

	currentMonth := time.Now().Format("2006-01")

	// Check cache first
	if cached := t.getCached(userIdentity, currentMonth); cached != nil {
		return cached.count, nil
	}

	// Query database
	count, err := t.queryCount(ctx, userIdentity, currentMonth)
	if err != nil {
		return 0, err
	}

	// Update cache
	t.setCached(userIdentity, currentMonth, count)

	return count, nil
}

// IncrementExecutionCount increments the execution count for a user.
// Call this after successfully starting a workflow execution.
func (t *UsageTracker) IncrementExecutionCount(ctx context.Context, userIdentity string) error {
	userIdentity = strings.TrimSpace(strings.ToLower(userIdentity))
	if userIdentity == "" {
		return nil
	}

	currentMonth := time.Now().Format("2006-01")

	// Update database
	if err := t.upsertCount(ctx, userIdentity, currentMonth); err != nil {
		return err
	}

	// Invalidate cache to force refresh
	t.invalidateCache(userIdentity)

	return nil
}

// GetUsageSummary returns a summary of usage for a user.
func (t *UsageTracker) GetUsageSummary(ctx context.Context, userIdentity string) (*UsageSummary, error) {
	userIdentity = strings.TrimSpace(strings.ToLower(userIdentity))
	currentMonth := time.Now().Format("2006-01")

	count, err := t.GetMonthlyExecutionCount(ctx, userIdentity)
	if err != nil {
		return nil, err
	}

	return &UsageSummary{
		UserIdentity:   userIdentity,
		CurrentMonth:   currentMonth,
		ExecutionCount: count,
		PeriodStart:    firstDayOfMonth(time.Now()),
		PeriodEnd:      lastDayOfMonth(time.Now()),
	}, nil
}

// UsageSummary contains usage statistics for a user.
type UsageSummary struct {
	UserIdentity   string    `json:"user_identity"`
	CurrentMonth   string    `json:"current_month"`
	ExecutionCount int       `json:"execution_count"`
	PeriodStart    time.Time `json:"period_start"`
	PeriodEnd      time.Time `json:"period_end"`
}

// queryCount queries the database for the execution count.
func (t *UsageTracker) queryCount(ctx context.Context, userIdentity, month string) (int, error) {
	query := `
		SELECT execution_count
		FROM entitlement_usage
		WHERE user_identity = $1 AND billing_month = $2
	`

	var count int
	err := t.db.QueryRowContext(ctx, query, userIdentity, month).Scan(&count)
	if err == sql.ErrNoRows {
		return 0, nil
	}
	if err != nil {
		return 0, fmt.Errorf("query usage count: %w", err)
	}

	return count, nil
}

// upsertCount increments the execution count in the database.
func (t *UsageTracker) upsertCount(ctx context.Context, userIdentity, month string) error {
	// Use PostgreSQL upsert
	query := `
		INSERT INTO entitlement_usage (user_identity, billing_month, execution_count, updated_at)
		VALUES ($1, $2, 1, NOW())
		ON CONFLICT (user_identity, billing_month)
		DO UPDATE SET
			execution_count = entitlement_usage.execution_count + 1,
			updated_at = NOW()
	`

	_, err := t.db.ExecContext(ctx, query, userIdentity, month)
	if err != nil {
		return fmt.Errorf("upsert usage count: %w", err)
	}

	return nil
}

// getCached retrieves a cached count.
func (t *UsageTracker) getCached(userIdentity, month string) *usageCache {
	t.cacheMu.RLock()
	defer t.cacheMu.RUnlock()

	cached, ok := t.cache[userIdentity]
	if !ok {
		return nil
	}

	// Check if cache is for current month and not too stale (1 minute)
	if cached.month != month || time.Since(cached.updatedAt) > time.Minute {
		return nil
	}

	return cached
}

// setCached stores a count in the cache.
func (t *UsageTracker) setCached(userIdentity, month string, count int) {
	t.cacheMu.Lock()
	t.cache[userIdentity] = &usageCache{
		count:     count,
		month:     month,
		updatedAt: time.Now(),
	}
	t.cacheMu.Unlock()
}

// invalidateCache removes a user's cached count.
func (t *UsageTracker) invalidateCache(userIdentity string) {
	t.cacheMu.Lock()
	delete(t.cache, userIdentity)
	t.cacheMu.Unlock()
}

// Helper functions for date calculations.

func firstDayOfMonth(t time.Time) time.Time {
	return time.Date(t.Year(), t.Month(), 1, 0, 0, 0, 0, t.Location())
}

func lastDayOfMonth(t time.Time) time.Time {
	return firstDayOfMonth(t).AddDate(0, 1, 0).Add(-time.Nanosecond)
}
