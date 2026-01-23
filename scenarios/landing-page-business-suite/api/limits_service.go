package main

import (
	"context"
	"database/sql"
	"encoding/json"
	"fmt"
	"net/http"
	"strings"
	"time"

	"github.com/gorilla/mux"
)

// LimitsService manages subscription tier limits for the credit system.
type LimitsService struct {
	db       *sql.DB
	dialects *DialectHelper
}

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

// NewLimitsService creates a new limits service.
func NewLimitsService(db *sql.DB, dialect string) *LimitsService {
	return &LimitsService{
		db:       db,
		dialects: NewDialectHelper(dialect),
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
		if limit.LimitType == "cost_based" && limit.LimitValue > 0 && limit.CostMultiplier > 0 {
			// Convert internal units to dollars
			// internal_units / cost_multiplier = cents, cents / 100 = dollars
			dollars := float64(limit.LimitValue) / float64(limit.CostMultiplier) / 100.0
			limit.DisplayDollars = &dollars
		}

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
		if limit.LimitType == "cost_based" && limit.LimitValue > 0 && limit.CostMultiplier > 0 {
			dollars := float64(limit.LimitValue) / float64(limit.CostMultiplier) / 100.0
			limit.DisplayDollars = &dollars
		}

		result[limit.TierID] = append(result[limit.TierID], limit)
	}

	return result, rows.Err()
}

// GetLimit returns a specific limit for a tier and key.
func (s *LimitsService) GetLimit(ctx context.Context, tierID, limitKey string, appBundleKey *string) (*TierLimit, error) {
	tierID = strings.TrimSpace(strings.ToLower(tierID))
	limitKey = strings.TrimSpace(strings.ToLower(limitKey))

	var query string
	var args []interface{}

	if appBundleKey == nil {
		query = `
			SELECT id, tier_id, limit_type, limit_key, limit_value, cost_multiplier,
				   app_bundle_key, reset_period, created_at, updated_at
			FROM subscription_tier_limits
			WHERE tier_id = $1 AND limit_key = $2 AND app_bundle_key IS NULL
		`
		args = []interface{}{tierID, limitKey}
	} else {
		query = `
			SELECT id, tier_id, limit_type, limit_key, limit_value, cost_multiplier,
				   app_bundle_key, reset_period, created_at, updated_at
			FROM subscription_tier_limits
			WHERE tier_id = $1 AND limit_key = $2 AND app_bundle_key = $3
		`
		args = []interface{}{tierID, limitKey, *appBundleKey}
	}

	var limit TierLimit
	var appKey sql.NullString
	err := s.db.QueryRowContext(ctx, query, args...).Scan(
		&limit.ID, &limit.TierID, &limit.LimitType, &limit.LimitKey,
		&limit.LimitValue, &limit.CostMultiplier, &appKey,
		&limit.ResetPeriod, &limit.CreatedAt, &limit.UpdatedAt,
	)

	if err == sql.ErrNoRows {
		return nil, nil
	}
	if err != nil {
		return nil, fmt.Errorf("get limit: %w", err)
	}

	if appKey.Valid {
		limit.AppBundleKey = &appKey.String
	}

	// Calculate display dollars
	if limit.LimitType == "cost_based" && limit.LimitValue > 0 && limit.CostMultiplier > 0 {
		dollars := float64(limit.LimitValue) / float64(limit.CostMultiplier) / 100.0
		limit.DisplayDollars = &dollars
	}

	return &limit, nil
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
		// Convert dollars to internal units
		// dollars * 100 = cents, cents * cost_multiplier = internal_units
		newValue = int64(*update.DisplayDollars * 100.0 * float64(current.CostMultiplier))
	} else if update.LimitValue != nil {
		newValue = *update.LimitValue
	} else {
		return nil, fmt.Errorf("must provide limit_value, display_dollars, or is_unlimited")
	}

	// Update the limit
	var query string
	var args []interface{}
	nowExpr := s.dialects.NowExpr()

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
	if limit.LimitType == "cost_based" && limit.LimitValue > 0 && limit.CostMultiplier > 0 {
		dollars := float64(limit.LimitValue) / float64(limit.CostMultiplier) / 100.0
		limit.DisplayDollars = &dollars
	}

	logStructured("tier_limit_updated", map[string]interface{}{
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

	if limit.LimitType != "cost_based" && limit.LimitType != "app_specific" {
		return nil, fmt.Errorf("limit_type must be 'cost_based' or 'app_specific'")
	}

	if limit.CostMultiplier <= 0 {
		limit.CostMultiplier = 1000000 // Default multiplier
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
	err := s.db.QueryRowContext(ctx, query,
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
	if result.LimitType == "cost_based" && result.LimitValue > 0 && result.CostMultiplier > 0 {
		dollars := float64(result.LimitValue) / float64(result.CostMultiplier) / 100.0
		result.DisplayDollars = &dollars
	}

	logStructured("tier_limit_created", map[string]interface{}{
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

	logStructured("tier_limit_deleted", map[string]interface{}{
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

// API Handlers

func handleGetTierLimits(svc *LimitsService) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		vars := mux.Vars(r)
		tierID := vars["tier"]

		if tierID == "" {
			// Return all tier limits
			limits, err := svc.GetAllTierLimits(r.Context())
			if err != nil {
				logStructuredError("get_all_tier_limits_failed", map[string]interface{}{"error": err.Error()})
				writeJSONError(w, http.StatusInternalServerError, "Failed to get tier limits", ApiErrorTypeServerError)
				return
			}

			w.Header().Set("Content-Type", "application/json")
			if err := json.NewEncoder(w).Encode(map[string]interface{}{
				"limits": limits,
			}); err != nil {
				logStructuredError("encode_response_failed", map[string]interface{}{"error": err.Error()})
			}
			return
		}

		limits, err := svc.GetTierLimits(r.Context(), tierID)
		if err != nil {
			logStructuredError("get_tier_limits_failed", map[string]interface{}{
				"error":   err.Error(),
				"tier_id": tierID,
			})
			writeJSONError(w, http.StatusInternalServerError, "Failed to get tier limits", ApiErrorTypeServerError)
			return
		}

		if limits == nil {
			limits = []TierLimit{}
		}

		w.Header().Set("Content-Type", "application/json")
		if err := json.NewEncoder(w).Encode(map[string]interface{}{
			"tier_id": tierID,
			"limits":  limits,
		}); err != nil {
			logStructuredError("encode_response_failed", map[string]interface{}{"error": err.Error()})
		}
	}
}

func handleUpdateTierLimits(svc *LimitsService) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		vars := mux.Vars(r)
		tierID := vars["tier"]

		if tierID == "" {
			writeJSONError(w, http.StatusBadRequest, "Tier ID is required", ApiErrorTypeValidation)
			return
		}

		var req struct {
			LimitKey     string          `json:"limit_key"`
			AppBundleKey *string         `json:"app_bundle_key"`
			Update       TierLimitUpdate `json:"update"`
		}

		if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
			writeJSONError(w, http.StatusBadRequest, "Invalid request body", ApiErrorTypeValidation)
			return
		}

		if req.LimitKey == "" {
			writeJSONError(w, http.StatusBadRequest, "limit_key is required", ApiErrorTypeValidation)
			return
		}

		limit, err := svc.UpdateLimit(r.Context(), tierID, req.LimitKey, req.AppBundleKey, req.Update)
		if err != nil {
			logStructuredError("update_tier_limit_failed", map[string]interface{}{
				"error":     err.Error(),
				"tier_id":   tierID,
				"limit_key": req.LimitKey,
			})
			writeJSONError(w, http.StatusBadRequest, err.Error(), ApiErrorTypeValidation)
			return
		}

		w.Header().Set("Content-Type", "application/json")
		if err := json.NewEncoder(w).Encode(limit); err != nil {
			logStructuredError("encode_response_failed", map[string]interface{}{"error": err.Error()})
		}
	}
}

func handleGetAppLimits(svc *LimitsService) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		vars := mux.Vars(r)
		appKey := vars["app"]

		if appKey == "" {
			writeJSONError(w, http.StatusBadRequest, "App key is required", ApiErrorTypeValidation)
			return
		}

		limits, err := svc.GetAppLimits(r.Context(), appKey)
		if err != nil {
			logStructuredError("get_app_limits_failed", map[string]interface{}{
				"error":   err.Error(),
				"app_key": appKey,
			})
			writeJSONError(w, http.StatusInternalServerError, "Failed to get app limits", ApiErrorTypeServerError)
			return
		}

		w.Header().Set("Content-Type", "application/json")
		if err := json.NewEncoder(w).Encode(map[string]interface{}{
			"app_bundle_key": appKey,
			"limits":         limits,
		}); err != nil {
			logStructuredError("encode_response_failed", map[string]interface{}{"error": err.Error()})
		}
	}
}

func handleCreateTierLimit(svc *LimitsService) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		var limit TierLimit
		if err := json.NewDecoder(r.Body).Decode(&limit); err != nil {
			writeJSONError(w, http.StatusBadRequest, "Invalid request body", ApiErrorTypeValidation)
			return
		}

		result, err := svc.CreateLimit(r.Context(), limit)
		if err != nil {
			logStructuredError("create_tier_limit_failed", map[string]interface{}{
				"error": err.Error(),
			})
			writeJSONError(w, http.StatusBadRequest, err.Error(), ApiErrorTypeValidation)
			return
		}

		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusCreated)
		if err := json.NewEncoder(w).Encode(result); err != nil {
			logStructuredError("encode_response_failed", map[string]interface{}{"error": err.Error()})
		}
	}
}

func handleDeleteTierLimit(svc *LimitsService) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		var req struct {
			TierID       string  `json:"tier_id"`
			LimitKey     string  `json:"limit_key"`
			AppBundleKey *string `json:"app_bundle_key"`
		}

		if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
			writeJSONError(w, http.StatusBadRequest, "Invalid request body", ApiErrorTypeValidation)
			return
		}

		if req.TierID == "" || req.LimitKey == "" {
			writeJSONError(w, http.StatusBadRequest, "tier_id and limit_key are required", ApiErrorTypeValidation)
			return
		}

		if err := svc.DeleteLimit(r.Context(), req.TierID, req.LimitKey, req.AppBundleKey); err != nil {
			logStructuredError("delete_tier_limit_failed", map[string]interface{}{
				"error":     err.Error(),
				"tier_id":   req.TierID,
				"limit_key": req.LimitKey,
			})
			writeJSONError(w, http.StatusNotFound, err.Error(), ApiErrorTypeNotFound)
			return
		}

		w.WriteHeader(http.StatusNoContent)
	}
}
