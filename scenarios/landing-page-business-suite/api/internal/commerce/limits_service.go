package commerce

import (
	"context"
	"database/sql"
	"fmt"
	"math"
	"strings"
	"time"
)

// LimitsServicer provides limit lookup for dependent services.
// This interface enables testing UsageService without the real LimitsService.
type LimitsServicer interface {
	GetLimit(ctx context.Context, tierID, limitKey string, appBundleKey *string) (*TierLimit, error)
	GetTierLimits(ctx context.Context, tierID string) ([]TierLimit, error)
}

// LimitsStore is the context-aware persistence contract for subscription tier
// limits.
//
// seam: LimitsStore keeps limit persistence independent of a concrete pool and
// preserves request-scoped test isolation.
type LimitsStore interface {
	QueryContext(context.Context, string, ...any) (*sql.Rows, error)
	QueryRowContext(context.Context, string, ...any) *sql.Row
	ExecContext(context.Context, string, ...any) (sql.Result, error)
}

// LimitsService manages subscription tier limits for the credit system.
type LimitsService struct {
	db      LimitsStore
	dialect string
	logf    func(string, map[string]interface{})
}

// Compile-time check that LimitsService implements LimitsServicer
var _ LimitsServicer = (*LimitsService)(nil)

// TierLimit represents a single limit configuration.
type TierLimit struct {
	ID             string    `json:"id"`
	TierID         string    `json:"tier_id"`
	LimitType      string    `json:"limit_type"`      // cost_based or app_specific
	LimitKey       string    `json:"limit_key"`       // ai_credits, workflow_exports, etc.
	LimitValue     int64     `json:"limit_value"`     // -1 = unlimited
	CostMultiplier int64     `json:"cost_multiplier"` // For cost_based: internal units per cent
	AppBundleKey   *string   `json:"app_bundle_key"`  // NULL for cost_based
	ResetPeriod    string    `json:"reset_period"`    // monthly, yearly, etc.
	CreatedAt      time.Time `json:"created_at"`
	UpdatedAt      time.Time `json:"updated_at"`

	// Computed fields for display
	DisplayDollars *float64 `json:"display_dollars,omitempty"` // For cost_based limits
}

// TierLimitUpdate is the request to update a tier limit.
type TierLimitUpdate struct {
	LimitValue     *int64   `json:"limit_value"`     // Optional: new limit value
	DisplayDollars *float64 `json:"display_dollars"` // Optional: set limit in dollars (converted)
	IsUnlimited    *bool    `json:"is_unlimited"`    // Optional: set to unlimited (-1)
}

// NewLimitsService creates a tier-limit service. logf is optional and keeps
// domain logic independent of the application logging implementation.
func NewLimitsService(db LimitsStore, dialect string, logf func(string, map[string]interface{})) *LimitsService {
	if dialect == "" {
		dialect = "postgres"
	}
	return &LimitsService{db: db, dialect: dialect, logf: logf}
}

func (s *LimitsService) nowExpr() string {
	if s.dialect == "sqlite" {
		return "datetime('now')"
	}
	return "NOW()"
}

func (s *LimitsService) log(event string, fields map[string]interface{}) {
	if s.logf != nil {
		s.logf(event, fields)
	}
}

// GetTierLimits returns all limits for a specific tier.
func (s *LimitsService) GetTierLimits(ctx context.Context, tierID string) ([]TierLimit, error) {
	tierID = strings.TrimSpace(strings.ToLower(tierID))

	rows, err := s.db.QueryContext(ctx, `
		SELECT id, tier_id, limit_type, limit_key, limit_value, cost_multiplier,
			   app_bundle_key, reset_period, created_at, updated_at
		FROM subscription_tier_limits
		WHERE tier_id = $1
		ORDER BY limit_type, limit_key
	`, tierID)
	if err != nil {
		return nil, fmt.Errorf("query tier limits: %w", err)
	}
	defer rows.Close()

	var limits []TierLimit
	for rows.Next() {
		var limit TierLimit
		var appBundleKey sql.NullString
		if err := rows.Scan(
			&limit.ID, &limit.TierID, &limit.LimitType, &limit.LimitKey,
			&limit.LimitValue, &limit.CostMultiplier, &appBundleKey,
			&limit.ResetPeriod, &limit.CreatedAt, &limit.UpdatedAt,
		); err != nil {
			return nil, fmt.Errorf("scan tier limit: %w", err)
		}

		if appBundleKey.Valid {
			limit.AppBundleKey = &appBundleKey.String
		}

		// Calculate display dollars for cost-based limits
		presentLimit(&limit)

		limits = append(limits, limit)
	}

	return limits, rows.Err()
}

// GetAllTierLimits returns all limits grouped by tier.
func (s *LimitsService) GetAllTierLimits(ctx context.Context) (map[string][]TierLimit, error) {
	rows, err := s.db.QueryContext(ctx, `
		SELECT id, tier_id, limit_type, limit_key, limit_value, cost_multiplier,
			   app_bundle_key, reset_period, created_at, updated_at
		FROM subscription_tier_limits
		ORDER BY tier_id, limit_type, limit_key
	`)
	if err != nil {
		return nil, fmt.Errorf("query all tier limits: %w", err)
	}
	defer rows.Close()

	result := make(map[string][]TierLimit)
	for rows.Next() {
		var limit TierLimit
		var appBundleKey sql.NullString
		if err := rows.Scan(
			&limit.ID, &limit.TierID, &limit.LimitType, &limit.LimitKey,
			&limit.LimitValue, &limit.CostMultiplier, &appBundleKey,
			&limit.ResetPeriod, &limit.CreatedAt, &limit.UpdatedAt,
		); err != nil {
			return nil, fmt.Errorf("scan tier limit: %w", err)
		}

		if appBundleKey.Valid {
			limit.AppBundleKey = &appBundleKey.String
		}

		// Calculate display dollars
		presentLimit(&limit)

		result[limit.TierID] = append(result[limit.TierID], limit)
	}

	return result, rows.Err()
}

// GetLimit returns a specific limit for a tier and key.
func (s *LimitsService) GetLimit(ctx context.Context, tierID, limitKey string, appBundleKey *string) (*TierLimit, error) {
	tierID, limitKey = strings.ToLower(strings.TrimSpace(tierID)), strings.ToLower(strings.TrimSpace(limitKey))
	query := `SELECT id,tier_id,limit_type,limit_key,limit_value,cost_multiplier,app_bundle_key,reset_period,created_at,updated_at FROM subscription_tier_limits WHERE tier_id=$1 AND limit_key=$2 AND app_bundle_key IS NULL`
	args := []any{tierID, limitKey}
	if appBundleKey != nil {
		if strings.TrimSpace(*appBundleKey) == "" {
			return nil, fmt.Errorf("app scope must be an exact nonempty key")
		}
		query = `SELECT id,tier_id,limit_type,limit_key,limit_value,cost_multiplier,app_bundle_key,reset_period,created_at,updated_at FROM subscription_tier_limits WHERE tier_id=$1 AND limit_key=$2 AND app_bundle_key=$3`
		args = append(args, *appBundleKey)
	}
	rows, err := s.db.QueryContext(ctx, query, args...)
	if err != nil {
		return nil, fmt.Errorf("get limit: %w", err)
	}
	defer rows.Close()
	var result *TierLimit
	for rows.Next() {
		if result != nil {
			return nil, fmt.Errorf("multiple meter definitions apply to the same tier, key and app scope")
		}
		var value TierLimit
		var app sql.NullString
		if err := rows.Scan(&value.ID, &value.TierID, &value.LimitType, &value.LimitKey, &value.LimitValue, &value.CostMultiplier, &app, &value.ResetPeriod, &value.CreatedAt, &value.UpdatedAt); err != nil {
			return nil, err
		}
		if app.Valid {
			value.AppBundleKey = &app.String
		}
		kind, err := meterKind(value)
		if err != nil {
			return nil, err
		}
		value.LimitType = kind
		decorateLimitDollars(&value)
		result = &value
	}
	return result, rows.Err()
}

// meterKind is the only compatibility mapping for the old scope-as-kind field.
// It does not change stored configuration or guess units for an arbitrary key.
func meterKind(limit TierLimit) (string, error) {
	switch limit.LimitType {
	case "cost_based":
		if limit.CostMultiplier <= 0 {
			return "", fmt.Errorf("cost meter conversion must be positive")
		}
		return "cost_based", nil
	case "count_based":
		return "count_based", nil
	case "app_specific":
		if limit.LimitKey == "ai_credits" && limit.AppBundleKey != nil && strings.TrimSpace(*limit.AppBundleKey) != "" && limit.CostMultiplier > 0 {
			return "cost_based", nil
		}
	}
	return "", fmt.Errorf("meter kind is ambiguous or unsupported for %s/%s", limit.LimitType, limit.LimitKey)
}

func decorateLimitDollars(limit *TierLimit) {
	if kind, err := meterKind(*limit); err == nil && kind == "cost_based" && limit.LimitValue >= 0 {
		dollars := float64(limit.LimitValue) / float64(limit.CostMultiplier) / 100
		limit.DisplayDollars = &dollars
	}
}

// Admin reads expose the reviewed kind where it is known, retaining an
// ambiguous legacy row visibly for explicit operator correction.
func presentLimit(limit *TierLimit) {
	if kind, err := meterKind(*limit); err == nil {
		limit.LimitType = kind
		decorateLimitDollars(limit)
	}
}

// MeterLimits separates a meter's units from its two possible applicable caps.
// The caller supplies its verified app identity; this resolver never authenticates
// client metadata and never substitutes a different app's definition.
type MeterLimits struct {
	Kind           string
	Global, App    *TierLimit
	CostMultiplier int64
}

func ResolveMeterLimits(ctx context.Context, authority LimitsServicer, tier, key, app string) (MeterLimits, error) {
	var result MeterLimits
	if authority == nil || strings.TrimSpace(tier) == "" || strings.TrimSpace(key) == "" {
		return result, fmt.Errorf("meter limit authority is unavailable")
	}
	rows, err := authority.GetTierLimits(ctx, tier)
	if err != nil {
		return result, err
	}
	for _, row := range rows {
		if row.LimitKey != key {
			continue
		}
		if row.AppBundleKey != nil && strings.TrimSpace(*row.AppBundleKey) == "" {
			return result, fmt.Errorf("meter has an empty app scope")
		}
		global := row.AppBundleKey == nil
		if !global && (app == "" || *row.AppBundleKey != app) {
			continue
		}
		kind, err := meterKind(row)
		if err != nil {
			return result, err
		}
		row.LimitType = kind
		if result.Kind != "" && result.Kind != kind {
			return result, fmt.Errorf("global and app meter kinds contradict each other")
		}
		if kind == "cost_based" && result.CostMultiplier != 0 && result.CostMultiplier != row.CostMultiplier {
			return result, fmt.Errorf("global and app credit conversions contradict each other")
		}
		result.Kind = kind
		if kind == "cost_based" {
			result.CostMultiplier = row.CostMultiplier
		}
		if global {
			if result.Global != nil {
				return result, fmt.Errorf("duplicate global meter definition")
			}
			value := row
			result.Global = &value
		} else {
			if result.App != nil {
				return result, fmt.Errorf("duplicate exact-app meter definition")
			}
			value := row
			result.App = &value
		}
	}
	if result.Kind == "" {
		return result, fmt.Errorf("no meter definition applies to the verified app scope")
	}
	return result, nil
}

// UpdateLimit updates a tier limit.
func (s *LimitsService) UpdateLimit(ctx context.Context, tierID, limitKey string, appBundleKey *string, update TierLimitUpdate) (*TierLimit, error) {
	tierID = strings.TrimSpace(strings.ToLower(tierID))
	limitKey = strings.TrimSpace(strings.ToLower(limitKey))

	// Get current limit to get cost_multiplier
	current, err := s.GetLimit(ctx, tierID, limitKey, appBundleKey)
	if err != nil {
		return nil, err
	}
	if current == nil {
		return nil, fmt.Errorf("limit not found: %s/%s", tierID, limitKey)
	}

	// Calculate new limit value
	var newValue int64
	if update.IsUnlimited != nil && *update.IsUnlimited {
		newValue = -1
	} else if update.DisplayDollars != nil {
		if current.LimitType != "cost_based" {
			return nil, fmt.Errorf("dollar amounts require a cost-based meter")
		}
		units := *update.DisplayDollars * 100 * float64(current.CostMultiplier)
		if math.IsNaN(units) || math.IsInf(units, 0) || units < 0 || units >= float64(math.MaxInt64) {
			return nil, fmt.Errorf("dollar limit is invalid or overflows")
		}
		newValue = int64(math.Round(units))
	} else if update.LimitValue != nil {
		newValue = *update.LimitValue
	} else {
		return nil, fmt.Errorf("must provide limit_value, display_dollars, or is_unlimited")
	}

	if newValue < -1 {
		return nil, fmt.Errorf("limit_value must be non-negative or -1 for unlimited")
	}
	// Update the limit
	var query string
	var args []interface{}
	nowExpr := s.nowExpr()

	if appBundleKey == nil {
		query = `
			UPDATE subscription_tier_limits
			SET limit_value = $1, updated_at = ` + nowExpr + `
			WHERE tier_id = $2 AND limit_key = $3 AND app_bundle_key IS NULL
			RETURNING id, tier_id, limit_type, limit_key, limit_value, cost_multiplier,
					  app_bundle_key, reset_period, created_at, updated_at
		`
		args = []interface{}{newValue, tierID, limitKey}
	} else {
		query = `
			UPDATE subscription_tier_limits
			SET limit_value = $1, updated_at = ` + nowExpr + `
			WHERE tier_id = $2 AND limit_key = $3 AND app_bundle_key = $4
			RETURNING id, tier_id, limit_type, limit_key, limit_value, cost_multiplier,
					  app_bundle_key, reset_period, created_at, updated_at
		`
		args = []interface{}{newValue, tierID, limitKey, *appBundleKey}
	}

	var limit TierLimit
	var appKey sql.NullString
	err = s.db.QueryRowContext(ctx, query, args...).Scan(
		&limit.ID, &limit.TierID, &limit.LimitType, &limit.LimitKey,
		&limit.LimitValue, &limit.CostMultiplier, &appKey,
		&limit.ResetPeriod, &limit.CreatedAt, &limit.UpdatedAt,
	)
	if err != nil {
		return nil, fmt.Errorf("update limit: %w", err)
	}

	if appKey.Valid {
		limit.AppBundleKey = &appKey.String
	}

	// Calculate display dollars
	presentLimit(&limit)

	s.log("tier_limit_updated", map[string]interface{}{
		"level":     "info",
		"tier_id":   tierID,
		"limit_key": limitKey,
		"new_value": newValue,
	})

	return &limit, nil
}

// CreateLimit creates a new tier limit.
func (s *LimitsService) CreateLimit(ctx context.Context, limit TierLimit) (*TierLimit, error) {
	limit.TierID = strings.TrimSpace(strings.ToLower(limit.TierID))
	limit.LimitKey = strings.TrimSpace(strings.ToLower(limit.LimitKey))
	limit.LimitType = strings.TrimSpace(strings.ToLower(limit.LimitType))

	if limit.TierID == "" || limit.LimitKey == "" || limit.LimitType == "" {
		return nil, fmt.Errorf("tier_id, limit_key, and limit_type are required")
	}

	if limit.AppBundleKey != nil {
		value := strings.TrimSpace(*limit.AppBundleKey)
		if value == "" {
			return nil, fmt.Errorf("app scope must be an exact nonempty key")
		}
		limit.AppBundleKey = &value
	}
	kind, err := meterKind(limit)
	if err != nil {
		return nil, err
	}
	limit.LimitType = kind
	if limit.LimitValue < -1 {
		return nil, fmt.Errorf("limit_value must be non-negative or -1 for unlimited")
	}

	if limit.ResetPeriod == "" {
		limit.ResetPeriod = "monthly"
	}

	query := `
		INSERT INTO subscription_tier_limits
			(tier_id, limit_type, limit_key, limit_value, cost_multiplier, app_bundle_key, reset_period)
		VALUES ($1, $2, $3, $4, $5, $6, $7)
		RETURNING id, tier_id, limit_type, limit_key, limit_value, cost_multiplier,
				  app_bundle_key, reset_period, created_at, updated_at
	`

	var result TierLimit
	var appKey sql.NullString
	err = s.db.QueryRowContext(ctx, query,
		limit.TierID, limit.LimitType, limit.LimitKey, limit.LimitValue,
		limit.CostMultiplier, limit.AppBundleKey, limit.ResetPeriod,
	).Scan(
		&result.ID, &result.TierID, &result.LimitType, &result.LimitKey,
		&result.LimitValue, &result.CostMultiplier, &appKey,
		&result.ResetPeriod, &result.CreatedAt, &result.UpdatedAt,
	)
	if err != nil {
		return nil, fmt.Errorf("create limit: %w", err)
	}

	if appKey.Valid {
		result.AppBundleKey = &appKey.String
	}

	// Calculate display dollars
	presentLimit(&result)

	s.log("tier_limit_created", map[string]interface{}{
		"level":     "info",
		"tier_id":   result.TierID,
		"limit_key": result.LimitKey,
	})

	return &result, nil
}

// DeleteLimit removes a tier limit.
func (s *LimitsService) DeleteLimit(ctx context.Context, tierID, limitKey string, appBundleKey *string) error {
	tierID = strings.TrimSpace(strings.ToLower(tierID))
	limitKey = strings.TrimSpace(strings.ToLower(limitKey))

	var query string
	var args []interface{}

	if appBundleKey == nil {
		query = `DELETE FROM subscription_tier_limits WHERE tier_id = $1 AND limit_key = $2 AND app_bundle_key IS NULL`
		args = []interface{}{tierID, limitKey}
	} else {
		query = `DELETE FROM subscription_tier_limits WHERE tier_id = $1 AND limit_key = $2 AND app_bundle_key = $3`
		args = []interface{}{tierID, limitKey, *appBundleKey}
	}

	result, err := s.db.ExecContext(ctx, query, args...)
	if err != nil {
		return fmt.Errorf("delete limit: %w", err)
	}

	affected, _ := result.RowsAffected()
	if affected == 0 {
		return fmt.Errorf("limit not found: %s/%s", tierID, limitKey)
	}

	s.log("tier_limit_deleted", map[string]interface{}{
		"level":     "info",
		"tier_id":   tierID,
		"limit_key": limitKey,
	})

	return nil
}

// GetAppLimits returns all limits for a specific app across all tiers.
func (s *LimitsService) GetAppLimits(ctx context.Context, appBundleKey string) (map[string][]TierLimit, error) {
	appBundleKey = strings.TrimSpace(strings.ToLower(appBundleKey))

	rows, err := s.db.QueryContext(ctx, `
		SELECT id, tier_id, limit_type, limit_key, limit_value, cost_multiplier,
			   app_bundle_key, reset_period, created_at, updated_at
		FROM subscription_tier_limits
		WHERE app_bundle_key = $1
		ORDER BY tier_id, limit_key
	`, appBundleKey)
	if err != nil {
		return nil, fmt.Errorf("query app limits: %w", err)
	}
	defer rows.Close()

	result := make(map[string][]TierLimit)
	for rows.Next() {
		var limit TierLimit
		var appKey sql.NullString
		if err := rows.Scan(
			&limit.ID, &limit.TierID, &limit.LimitType, &limit.LimitKey,
			&limit.LimitValue, &limit.CostMultiplier, &appKey,
			&limit.ResetPeriod, &limit.CreatedAt, &limit.UpdatedAt,
		); err != nil {
			return nil, fmt.Errorf("scan app limit: %w", err)
		}

		if appKey.Valid {
			limit.AppBundleKey = &appKey.String
		}

		result[limit.TierID] = append(result[limit.TierID], limit)
	}

	return result, rows.Err()
}

// DollarsToInternalUnits converts dollars to internal units.
// Uses the default cost multiplier of 1,000,000.
func DollarsToInternalUnits(dollars float64) int64 {
	// dollars * 100 = cents, cents * 1,000,000 = internal units
	return int64(dollars * 100.0 * 1000000.0)
}

// InternalUnitsToDollars converts internal units to dollars.
// Uses the default cost multiplier of 1,000,000.
func InternalUnitsToDollars(units int64) float64 {
	// units / 1,000,000 = cents, cents / 100 = dollars
	return float64(units) / 1000000.0 / 100.0
}
