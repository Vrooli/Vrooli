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

// AICreditsTracker tracks AI credit usage per user for entitlement enforcement.
type AICreditsTracker struct {
	db  *sql.DB
	log *logrus.Logger

	// In-memory cache for fast lookups
	cacheMu sync.RWMutex
	cache   map[string]*aiCreditsCache
}

type aiCreditsCache struct {
	creditsUsed   int
	requestsCount int
	month         string // YYYY-MM format
	updatedAt     time.Time
}

// AICreditsUsage represents AI credit usage for a billing period.
type AICreditsUsage struct {
	UserIdentity     string    `json:"user_identity"`
	BillingMonth     string    `json:"billing_month"`
	CreditsUsed      int       `json:"credits_used"`
	RequestsCount    int       `json:"requests_count"`
	CreditsLimit     int       `json:"credits_limit"`     // -1 for unlimited
	CreditsRemaining int       `json:"credits_remaining"` // -1 for unlimited
	PeriodStart      time.Time `json:"period_start"`
	PeriodEnd        time.Time `json:"period_end"`
	ResetDate        time.Time `json:"reset_date"`
}

// AIRequestLog represents a logged AI request.
type AIRequestLog struct {
	UserIdentity     string
	RequestType      string
	Model            string
	PromptTokens     int
	CompletionTokens int
	CreditsCharged   int
	Success          bool
	ErrorMessage     string
	DurationMs       int
}

// NewAICreditsTracker creates a new AI credits tracker.
func NewAICreditsTracker(db *sql.DB, log *logrus.Logger) *AICreditsTracker {
	return &AICreditsTracker{
		db:    db,
		log:   log,
		cache: make(map[string]*aiCreditsCache),
	}
}

// GetAICreditsUsage returns the AI credits usage for a user in the current month.
func (t *AICreditsTracker) GetAICreditsUsage(ctx context.Context, userIdentity string, creditsLimit int) (*AICreditsUsage, error) {
	userIdentity = strings.TrimSpace(strings.ToLower(userIdentity))
	currentMonth := time.Now().Format("2006-01")

	creditsUsed := 0
	requestsCount := 0

	if userIdentity != "" {
		// Check cache first
		if cached := t.getCached(userIdentity, currentMonth); cached != nil {
			creditsUsed = cached.creditsUsed
			requestsCount = cached.requestsCount
		} else {
			// Query database
			var err error
			creditsUsed, requestsCount, err = t.queryUsage(ctx, userIdentity, currentMonth)
			if err != nil {
				return nil, err
			}
			// Update cache
			t.setCached(userIdentity, currentMonth, creditsUsed, requestsCount)
		}
	}

	// Calculate remaining
	creditsRemaining := -1
	if creditsLimit >= 0 {
		creditsRemaining = creditsLimit - creditsUsed
		if creditsRemaining < 0 {
			creditsRemaining = 0
		}
	}

	now := time.Now()
	return &AICreditsUsage{
		UserIdentity:     userIdentity,
		BillingMonth:     currentMonth,
		CreditsUsed:      creditsUsed,
		RequestsCount:    requestsCount,
		CreditsLimit:     creditsLimit,
		CreditsRemaining: creditsRemaining,
		PeriodStart:      firstDayOfMonth(now),
		PeriodEnd:        lastDayOfMonth(now),
		ResetDate:        firstDayOfMonth(now).AddDate(0, 1, 0),
	}, nil
}

// CanUseCredits checks if the user has credits available.
// Returns (canUse, remaining, error).
func (t *AICreditsTracker) CanUseCredits(ctx context.Context, userIdentity string, creditsLimit int) (bool, int, error) {
	if creditsLimit < 0 {
		// Unlimited
		return true, -1, nil
	}
	if creditsLimit == 0 {
		// No access
		return false, 0, nil
	}

	userIdentity = strings.TrimSpace(strings.ToLower(userIdentity))
	currentMonth := time.Now().Format("2006-01")

	creditsUsed := 0
	if userIdentity != "" {
		if cached := t.getCached(userIdentity, currentMonth); cached != nil {
			creditsUsed = cached.creditsUsed
		} else {
			var err error
			creditsUsed, _, err = t.queryUsage(ctx, userIdentity, currentMonth)
			if err != nil {
				return false, 0, err
			}
		}
	}

	remaining := creditsLimit - creditsUsed
	if remaining < 0 {
		remaining = 0
	}

	return remaining > 0, remaining, nil
}

// ChargeCredits charges credits for an AI operation.
func (t *AICreditsTracker) ChargeCredits(ctx context.Context, userIdentity string, credits int, requestType, model string) error {
	userIdentity = strings.TrimSpace(strings.ToLower(userIdentity))
	if userIdentity == "" || credits <= 0 {
		return nil
	}

	currentMonth := time.Now().Format("2006-01")

	// Update database
	if err := t.upsertCredits(ctx, userIdentity, currentMonth, credits); err != nil {
		return err
	}

	// Log the request
	if err := t.logRequest(ctx, &AIRequestLog{
		UserIdentity:   userIdentity,
		RequestType:    requestType,
		Model:          model,
		CreditsCharged: credits,
		Success:        true,
	}); err != nil {
		// Log but don't fail the operation
		t.log.WithError(err).Warn("Failed to log AI request")
	}

	// Invalidate cache to force refresh
	t.invalidateCache(userIdentity)

	return nil
}

// LogRequest logs an AI request (can be called separately for detailed tracking).
func (t *AICreditsTracker) LogRequest(ctx context.Context, req *AIRequestLog) error {
	return t.logRequest(ctx, req)
}

// queryUsage queries the database for AI credits usage.
func (t *AICreditsTracker) queryUsage(ctx context.Context, userIdentity, month string) (creditsUsed, requestsCount int, err error) {
	query := `
		SELECT credits_used, requests_count
		FROM ai_credits_usage
		WHERE user_identity = $1 AND billing_month = $2
	`

	err = t.db.QueryRowContext(ctx, query, userIdentity, month).Scan(&creditsUsed, &requestsCount)
	if err == sql.ErrNoRows {
		return 0, 0, nil
	}
	if err != nil {
		return 0, 0, fmt.Errorf("query AI credits usage: %w", err)
	}

	return creditsUsed, requestsCount, nil
}

// upsertCredits increments the credits used in the database.
func (t *AICreditsTracker) upsertCredits(ctx context.Context, userIdentity, month string, credits int) error {
	query := `
		INSERT INTO ai_credits_usage (user_identity, billing_month, credits_used, requests_count, last_request_at, updated_at)
		VALUES ($1, $2, $3, 1, NOW(), NOW())
		ON CONFLICT (user_identity, billing_month)
		DO UPDATE SET
			credits_used = ai_credits_usage.credits_used + $3,
			requests_count = ai_credits_usage.requests_count + 1,
			last_request_at = NOW(),
			updated_at = NOW()
	`

	_, err := t.db.ExecContext(ctx, query, userIdentity, month, credits)
	if err != nil {
		return fmt.Errorf("upsert AI credits: %w", err)
	}

	return nil
}

// logRequest inserts a log entry for an AI request.
func (t *AICreditsTracker) logRequest(ctx context.Context, req *AIRequestLog) error {
	query := `
		INSERT INTO ai_request_log (
			user_identity, request_type, model, prompt_tokens, completion_tokens,
			credits_charged, success, error_message, duration_ms
		) VALUES ($1, $2, $3, $4, $5, $6, $7, $8, $9)
	`

	_, err := t.db.ExecContext(ctx, query,
		req.UserIdentity,
		req.RequestType,
		req.Model,
		req.PromptTokens,
		req.CompletionTokens,
		req.CreditsCharged,
		req.Success,
		req.ErrorMessage,
		req.DurationMs,
	)
	if err != nil {
		return fmt.Errorf("insert AI request log: %w", err)
	}

	return nil
}

// getCached retrieves cached AI credits usage.
func (t *AICreditsTracker) getCached(userIdentity, month string) *aiCreditsCache {
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

// setCached stores AI credits usage in the cache.
func (t *AICreditsTracker) setCached(userIdentity, month string, creditsUsed, requestsCount int) {
	t.cacheMu.Lock()
	t.cache[userIdentity] = &aiCreditsCache{
		creditsUsed:   creditsUsed,
		requestsCount: requestsCount,
		month:         month,
		updatedAt:     time.Now(),
	}
	t.cacheMu.Unlock()
}

// invalidateCache removes a user's cached AI credits.
func (t *AICreditsTracker) invalidateCache(userIdentity string) {
	t.cacheMu.Lock()
	delete(t.cache, userIdentity)
	t.cacheMu.Unlock()
}
