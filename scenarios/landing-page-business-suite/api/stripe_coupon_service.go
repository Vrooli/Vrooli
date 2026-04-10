package main

import (
	"context"
	"database/sql"
	"encoding/json"
	"errors"
	"fmt"
	"net/http"
	"net/url"
	"strconv"
	"strings"
)

// --- StripeCouponService Interface Implementation ---
// This file contains all coupon-related operations: CRUD for coupons,
// intro coupon eligibility checking, and coupon import preview.

// StripeCoupon represents a Stripe coupon for admin management.
type StripeCoupon struct {
	ID               string   `json:"id"`
	Name             string   `json:"name,omitempty"`
	AmountOff        *int64   `json:"amount_off,omitempty"`
	PercentOff       *float64 `json:"percent_off,omitempty"`
	Currency         string   `json:"currency,omitempty"`
	Duration         string   `json:"duration"`
	DurationInMonths *int     `json:"duration_in_months,omitempty"`
	MaxRedemptions   *int     `json:"max_redemptions,omitempty"`
	RedeemBy         *int64   `json:"redeem_by,omitempty"`
	TimesRedeemed    int      `json:"times_redeemed"`
	Valid            bool     `json:"valid"`
	Created          int64    `json:"created"`
	IsIntroCoupon    bool     `json:"is_intro_coupon"`
	IntroTier        string   `json:"intro_tier,omitempty"`
}

// CreateCouponRequest contains parameters for creating a new coupon.
type CreateCouponRequest struct {
	ID               string   `json:"id,omitempty"`
	Name             string   `json:"name,omitempty"`
	AmountOff        *int64   `json:"amount_off,omitempty"`
	PercentOff       *float64 `json:"percent_off,omitempty"`
	Currency         string   `json:"currency,omitempty"`
	Duration         string   `json:"duration"`
	DurationInMonths *int     `json:"duration_in_months,omitempty"`
	MaxRedemptions   *int     `json:"max_redemptions,omitempty"`
	RedeemBy         *int64   `json:"redeem_by,omitempty"`
}

// UpdateCouponRequest contains the fields that can be updated on a coupon.
// Note: Stripe only allows updating name and metadata on existing coupons.
type UpdateCouponRequest struct {
	Name string `json:"name,omitempty"`
}

// CouponImportPreviewItem represents a single coupon in the import preview.
type CouponImportPreviewItem struct {
	ID               string   `json:"id"`
	Name             string   `json:"name,omitempty"`
	AmountOff        *int64   `json:"amount_off,omitempty"`
	PercentOff       *float64 `json:"percent_off,omitempty"`
	Currency         string   `json:"currency,omitempty"`
	Duration         string   `json:"duration"`
	DurationInMonths *int     `json:"duration_in_months,omitempty"`
	TimesRedeemed    int      `json:"times_redeemed"`
	Valid            bool     `json:"valid"`
	ExistsLocally    bool     `json:"exists_locally"`
}

// CouponImportPreview contains the preview of coupons available to import from Stripe.
type CouponImportPreview struct {
	Coupons       []CouponImportPreviewItem `json:"coupons"`
	TotalCoupons  int                       `json:"total_coupons"`
	ExistingCount int                       `json:"existing_count"`
	NewCount      int                       `json:"new_count"`
}

// ListCoupons fetches all coupons from Stripe.
func (s *StripeService) ListCoupons(ctx context.Context) ([]StripeCoupon, error) {
	values := url.Values{}
	values.Set("limit", "100")
	path := "/v1/coupons?" + values.Encode()

	body, err := s.doStripeRequest(ctx, http.MethodGet, path, nil, "")
	if err != nil {
		return nil, err
	}

	var resp struct {
		Data []struct {
			ID               string   `json:"id"`
			Name             string   `json:"name"`
			AmountOff        *int64   `json:"amount_off"`
			PercentOff       *float64 `json:"percent_off"`
			Currency         string   `json:"currency"`
			Duration         string   `json:"duration"`
			DurationInMonths *int     `json:"duration_in_months"`
			MaxRedemptions   *int     `json:"max_redemptions"`
			RedeemBy         *int64   `json:"redeem_by"`
			TimesRedeemed    int      `json:"times_redeemed"`
			Valid            bool     `json:"valid"`
			Created          int64    `json:"created"`
		} `json:"data"`
	}
	if err := json.Unmarshal(body, &resp); err != nil {
		return nil, fmt.Errorf("decode stripe coupons: %w", err)
	}

	coupons := make([]StripeCoupon, 0, len(resp.Data))
	for _, c := range resp.Data {
		coupon := StripeCoupon{
			ID:               c.ID,
			Name:             c.Name,
			AmountOff:        c.AmountOff,
			PercentOff:       c.PercentOff,
			Currency:         c.Currency,
			Duration:         c.Duration,
			DurationInMonths: c.DurationInMonths,
			MaxRedemptions:   c.MaxRedemptions,
			RedeemBy:         c.RedeemBy,
			TimesRedeemed:    c.TimesRedeemed,
			Valid:            c.Valid,
			Created:          c.Created,
		}
		// Check if this coupon is configured for intro pricing
		coupon.IsIntroCoupon, coupon.IntroTier = s.checkIntroCouponMapping(c.ID)
		coupons = append(coupons, coupon)
	}

	return coupons, nil
}

// GetCoupon fetches a single coupon from Stripe by ID.
func (s *StripeService) GetCoupon(ctx context.Context, couponID string) (*StripeCoupon, error) {
	if strings.TrimSpace(couponID) == "" {
		return nil, errors.New("coupon ID is required")
	}

	path := "/v1/coupons/" + url.PathEscape(couponID)
	body, err := s.doStripeRequest(ctx, http.MethodGet, path, nil, "")
	if err != nil {
		return nil, err
	}

	var c struct {
		ID               string   `json:"id"`
		Name             string   `json:"name"`
		AmountOff        *int64   `json:"amount_off"`
		PercentOff       *float64 `json:"percent_off"`
		Currency         string   `json:"currency"`
		Duration         string   `json:"duration"`
		DurationInMonths *int     `json:"duration_in_months"`
		MaxRedemptions   *int     `json:"max_redemptions"`
		RedeemBy         *int64   `json:"redeem_by"`
		TimesRedeemed    int      `json:"times_redeemed"`
		Valid            bool     `json:"valid"`
		Created          int64    `json:"created"`
	}
	if err := json.Unmarshal(body, &c); err != nil {
		return nil, fmt.Errorf("decode stripe coupon: %w", err)
	}

	coupon := &StripeCoupon{
		ID:               c.ID,
		Name:             c.Name,
		AmountOff:        c.AmountOff,
		PercentOff:       c.PercentOff,
		Currency:         c.Currency,
		Duration:         c.Duration,
		DurationInMonths: c.DurationInMonths,
		MaxRedemptions:   c.MaxRedemptions,
		RedeemBy:         c.RedeemBy,
		TimesRedeemed:    c.TimesRedeemed,
		Valid:            c.Valid,
		Created:          c.Created,
	}
	coupon.IsIntroCoupon, coupon.IntroTier = s.checkIntroCouponMapping(c.ID)

	return coupon, nil
}

// CreateCoupon creates a new coupon in Stripe.
func (s *StripeService) CreateCoupon(ctx context.Context, req CreateCouponRequest) (*StripeCoupon, error) {
	// Validate discount type
	if req.AmountOff == nil && req.PercentOff == nil {
		return nil, errors.New("either amount_off or percent_off is required")
	}
	if req.AmountOff != nil && req.PercentOff != nil {
		return nil, errors.New("cannot specify both amount_off and percent_off")
	}

	// Validate duration
	duration := strings.TrimSpace(req.Duration)
	if duration == "" {
		return nil, errors.New("duration is required")
	}
	if duration != "once" && duration != "repeating" && duration != "forever" {
		return nil, errors.New("duration must be once, repeating, or forever")
	}
	if duration == "repeating" && (req.DurationInMonths == nil || *req.DurationInMonths <= 0) {
		return nil, errors.New("duration_in_months is required for repeating coupons")
	}

	// Build form values
	values := url.Values{}
	values.Set("duration", duration)

	if id := strings.TrimSpace(req.ID); id != "" {
		values.Set("id", id)
	}
	if name := strings.TrimSpace(req.Name); name != "" {
		values.Set("name", name)
	}
	if req.AmountOff != nil {
		values.Set("amount_off", strconv.FormatInt(*req.AmountOff, 10))
		currency := strings.ToLower(strings.TrimSpace(req.Currency))
		if currency == "" {
			currency = "usd"
		}
		values.Set("currency", currency)
	}
	if req.PercentOff != nil {
		values.Set("percent_off", strconv.FormatFloat(*req.PercentOff, 'f', 2, 64))
	}
	if req.DurationInMonths != nil && duration == "repeating" {
		values.Set("duration_in_months", strconv.Itoa(*req.DurationInMonths))
	}
	if req.MaxRedemptions != nil && *req.MaxRedemptions > 0 {
		values.Set("max_redemptions", strconv.Itoa(*req.MaxRedemptions))
	}
	if req.RedeemBy != nil && *req.RedeemBy > 0 {
		values.Set("redeem_by", strconv.FormatInt(*req.RedeemBy, 10))
	}

	body, err := s.doStripeForm(ctx, http.MethodPost, "/v1/coupons", values)
	if err != nil {
		return nil, err
	}

	var c struct {
		ID               string   `json:"id"`
		Name             string   `json:"name"`
		AmountOff        *int64   `json:"amount_off"`
		PercentOff       *float64 `json:"percent_off"`
		Currency         string   `json:"currency"`
		Duration         string   `json:"duration"`
		DurationInMonths *int     `json:"duration_in_months"`
		MaxRedemptions   *int     `json:"max_redemptions"`
		RedeemBy         *int64   `json:"redeem_by"`
		TimesRedeemed    int      `json:"times_redeemed"`
		Valid            bool     `json:"valid"`
		Created          int64    `json:"created"`
	}
	if err := json.Unmarshal(body, &c); err != nil {
		return nil, fmt.Errorf("decode stripe coupon: %w", err)
	}

	coupon := &StripeCoupon{
		ID:               c.ID,
		Name:             c.Name,
		AmountOff:        c.AmountOff,
		PercentOff:       c.PercentOff,
		Currency:         c.Currency,
		Duration:         c.Duration,
		DurationInMonths: c.DurationInMonths,
		MaxRedemptions:   c.MaxRedemptions,
		RedeemBy:         c.RedeemBy,
		TimesRedeemed:    c.TimesRedeemed,
		Valid:            c.Valid,
		Created:          c.Created,
	}
	coupon.IsIntroCoupon, coupon.IntroTier = s.checkIntroCouponMapping(c.ID)

	return coupon, nil
}

// UpdateCoupon updates a coupon in Stripe (only name can be updated).
func (s *StripeService) UpdateCoupon(ctx context.Context, couponID string, req UpdateCouponRequest) (*StripeCoupon, error) {
	if strings.TrimSpace(couponID) == "" {
		return nil, errors.New("coupon ID is required")
	}

	params := url.Values{}
	if req.Name != "" {
		params.Set("name", req.Name)
	}

	path := "/v1/coupons/" + url.PathEscape(couponID)
	body, err := s.doStripeRequest(ctx, http.MethodPost, path, strings.NewReader(params.Encode()), "application/x-www-form-urlencoded")
	if err != nil {
		return nil, fmt.Errorf("update coupon: %w", err)
	}

	var resp struct {
		ID               string   `json:"id"`
		Name             string   `json:"name"`
		AmountOff        *int64   `json:"amount_off"`
		PercentOff       *float64 `json:"percent_off"`
		Currency         string   `json:"currency"`
		Duration         string   `json:"duration"`
		DurationInMonths *int     `json:"duration_in_months"`
		MaxRedemptions   *int     `json:"max_redemptions"`
		RedeemBy         *int64   `json:"redeem_by"`
		TimesRedeemed    int      `json:"times_redeemed"`
		Valid            bool     `json:"valid"`
		Created          int64    `json:"created"`
	}
	if err := json.Unmarshal(body, &resp); err != nil {
		return nil, fmt.Errorf("parse coupon response: %w", err)
	}

	coupon := &StripeCoupon{
		ID:               resp.ID,
		Name:             resp.Name,
		AmountOff:        resp.AmountOff,
		PercentOff:       resp.PercentOff,
		Currency:         resp.Currency,
		Duration:         resp.Duration,
		DurationInMonths: resp.DurationInMonths,
		MaxRedemptions:   resp.MaxRedemptions,
		RedeemBy:         resp.RedeemBy,
		TimesRedeemed:    resp.TimesRedeemed,
		Valid:            resp.Valid,
		Created:          resp.Created,
	}
	coupon.IsIntroCoupon, coupon.IntroTier = s.checkIntroCouponMapping(coupon.ID)

	return coupon, nil
}

// DeleteCoupon deletes a coupon from Stripe.
func (s *StripeService) DeleteCoupon(ctx context.Context, couponID string) error {
	if strings.TrimSpace(couponID) == "" {
		return errors.New("coupon ID is required")
	}

	path := "/v1/coupons/" + url.PathEscape(couponID)
	_, err := s.doStripeRequest(ctx, http.MethodDelete, path, nil, "")
	return err
}

// GetCouponImportPreview returns a preview of coupons available to import from Stripe.
// It marks which coupons are already assigned to plans locally.
func (s *StripeService) GetCouponImportPreview(ctx context.Context) (*CouponImportPreview, error) {
	// Fetch all coupons from Stripe
	coupons, err := s.ListCoupons(ctx)
	if err != nil {
		return nil, fmt.Errorf("list stripe coupons: %w", err)
	}

	// Get existing coupon mappings to determine which coupons are "in use"
	localMappings := s.planService.GetCouponMappings()
	usedCouponIDs := make(map[string]struct{})
	for _, couponID := range localMappings {
		usedCouponIDs[couponID] = struct{}{}
	}

	// Also include intro coupon config as "used"
	for _, couponID := range s.introCouponConfig.CouponMap {
		usedCouponIDs[couponID] = struct{}{}
	}

	preview := &CouponImportPreview{
		Coupons: make([]CouponImportPreviewItem, 0, len(coupons)),
	}

	for _, c := range coupons {
		_, existsLocally := usedCouponIDs[c.ID]
		item := CouponImportPreviewItem{
			ID:               c.ID,
			Name:             c.Name,
			AmountOff:        c.AmountOff,
			PercentOff:       c.PercentOff,
			Currency:         c.Currency,
			Duration:         c.Duration,
			DurationInMonths: c.DurationInMonths,
			TimesRedeemed:    c.TimesRedeemed,
			Valid:            c.Valid,
			ExistsLocally:    existsLocally,
		}
		preview.Coupons = append(preview.Coupons, item)

		if existsLocally {
			preview.ExistingCount++
		} else {
			preview.NewCount++
		}
	}

	preview.TotalCoupons = len(coupons)
	return preview, nil
}

// GetIntroCouponMap returns the configured intro coupon mapping.
func (s *StripeService) GetIntroCouponMap() map[string]string {
	if !s.introCouponConfig.Enabled || s.introCouponConfig.CouponMap == nil {
		return nil
	}
	// Return a copy to prevent external modification
	result := make(map[string]string, len(s.introCouponConfig.CouponMap))
	for k, v := range s.introCouponConfig.CouponMap {
		result[k] = v
	}
	return result
}

// --- Intro Coupon Eligibility Methods ---

// checkIntroEligibility checks if a user is eligible for the intro pricing coupon.
// Returns true if the user has never used the intro offer before.
func (s *StripeService) checkIntroEligibility(ctx context.Context, email string) (bool, error) {
	email = NormalizeEmail(email)
	if email == "" {
		// No email means we can't track eligibility, so don't apply intro
		return false, nil
	}

	// Check if user exists and has already used intro
	var hasUsedIntro sql.NullBool
	err := s.db.QueryRowContext(ctx, `
		SELECT has_used_intro FROM users WHERE email = $1
	`, email).Scan(&hasUsedIntro)

	if err == sql.ErrNoRows {
		// New user - eligible for intro
		return true, nil
	}
	if err != nil {
		return false, fmt.Errorf("check intro eligibility: %w", err)
	}

	// User exists - check if they've used intro
	if hasUsedIntro.Valid && hasUsedIntro.Bool {
		return false, nil // Already used intro
	}

	return true, nil // Eligible
}

// markIntroUsed marks that a user has used their intro pricing offer.
// This should be called after successful payment with an intro coupon.
func (s *StripeService) markIntroUsed(ctx context.Context, email, customerID, couponID, planTier, subscriptionID string) error {
	email = NormalizeEmail(email)
	if email == "" {
		return errors.New("email required to mark intro used")
	}

	// Start a transaction to ensure consistency
	tx, err := s.db.BeginTx(ctx, nil)
	if err != nil {
		return fmt.Errorf("begin transaction: %w", err)
	}
	defer func() { _ = tx.Rollback() }()

	// Update the user's has_used_intro flag (upsert)
	_, err = tx.ExecContext(ctx, `
		INSERT INTO users (email, has_used_intro, stripe_customer_id)
		VALUES ($1, TRUE, $2)
		ON CONFLICT (email) DO UPDATE SET
			has_used_intro = TRUE,
			stripe_customer_id = COALESCE(NULLIF($2, ''), users.stripe_customer_id),
			updated_at = NOW()
	`, email, customerID)
	if err != nil {
		return fmt.Errorf("update user intro flag: %w", err)
	}

	// Record in audit table
	_, err = tx.ExecContext(ctx, `
		INSERT INTO intro_coupon_usage (email, stripe_customer_id, coupon_id, plan_tier, subscription_id)
		VALUES ($1, $2, $3, $4, $5)
	`, email, customerID, couponID, planTier, subscriptionID)
	if err != nil {
		return fmt.Errorf("insert intro coupon usage: %w", err)
	}

	if err := tx.Commit(); err != nil {
		return fmt.Errorf("commit transaction: %w", err)
	}

	// Also update Stripe customer metadata as backup
	if customerID != "" {
		go func() {
			values := url.Values{}
			values.Set("metadata[has_used_intro]", "true")
			values.Set("metadata[intro_coupon_id]", couponID)
			_, updateErr := s.doStripeForm(context.Background(), http.MethodPost, "/v1/customers/"+url.PathEscape(customerID), values)
			if updateErr != nil {
				logStructuredError("stripe_customer_metadata_update_failed", map[string]interface{}{
					"customer_id": customerID,
					"error":       updateErr.Error(),
				})
			}
		}()
	}

	logStructured("intro_coupon_marked_used", map[string]interface{}{
		"level":           "info",
		"email":           email,
		"customer_id":     customerID,
		"coupon_id":       couponID,
		"plan_tier":       planTier,
		"subscription_id": subscriptionID,
	})

	return nil
}

// isIntroCoupon checks if a coupon ID matches one of our configured intro coupons.
func (s *StripeService) isIntroCoupon(couponID string) bool {
	if !s.introCouponConfig.Enabled {
		return false
	}
	for _, configuredID := range s.introCouponConfig.CouponMap {
		if configuredID == couponID {
			return true
		}
	}
	return false
}

// extractIntroCouponFromInvoice extracts the intro coupon ID from an invoice object.
// Returns empty string if no intro coupon was applied.
func (s *StripeService) extractIntroCouponFromInvoice(obj map[string]interface{}) string {
	// Check discount object (single discount)
	if discount, ok := obj["discount"].(map[string]interface{}); ok {
		if coupon, ok := discount["coupon"].(map[string]interface{}); ok {
			if couponID, ok := coupon["id"].(string); ok {
				if s.isIntroCoupon(couponID) {
					return couponID
				}
			}
		}
	}

	// Check discounts array (multiple discounts)
	if discounts, ok := obj["discounts"].([]interface{}); ok {
		for _, d := range discounts {
			if discountMap, ok := d.(map[string]interface{}); ok {
				if coupon, ok := discountMap["coupon"].(map[string]interface{}); ok {
					if couponID, ok := coupon["id"].(string); ok {
						if s.isIntroCoupon(couponID) {
							return couponID
						}
					}
				}
			}
		}
	}

	// Check total_discount_amounts for coupon references
	if tdAmounts, ok := obj["total_discount_amounts"].([]interface{}); ok {
		for _, tda := range tdAmounts {
			if tdaMap, ok := tda.(map[string]interface{}); ok {
				if discount, ok := tdaMap["discount"].(map[string]interface{}); ok {
					if coupon, ok := discount["coupon"].(map[string]interface{}); ok {
						if couponID, ok := coupon["id"].(string); ok {
							if s.isIntroCoupon(couponID) {
								return couponID
							}
						}
					}
				}
			}
		}
	}

	return ""
}

// checkIntroCouponMapping checks if a coupon ID matches a configured intro coupon.
// Returns (isIntro, tier) where tier is the plan tier name if matched.
func (s *StripeService) checkIntroCouponMapping(couponID string) (bool, string) {
	if !s.introCouponConfig.Enabled || s.introCouponConfig.CouponMap == nil {
		return false, ""
	}
	for tier, configuredID := range s.introCouponConfig.CouponMap {
		if configuredID == couponID {
			return true, tier
		}
	}
	return false, ""
}

// logIntroAnomaly records an anomalous intro coupon usage for admin review.
// This is used for fraud detection when a user appears ineligible at payment time
// but had a coupon applied at checkout time.
func (s *StripeService) logIntroAnomaly(email, customerID, couponID, anomalyType string, details map[string]interface{}) {
	if email == "" {
		return
	}

	detailsJSON, _ := json.Marshal(details)
	if detailsJSON == nil {
		detailsJSON = []byte("{}")
	}

	_, err := s.db.Exec(`
		INSERT INTO intro_anomaly_log (email, customer_id, coupon_id, anomaly_type, details)
		VALUES ($1, $2, $3, $4, $5)
	`, NormalizeEmail(email), customerID, couponID, anomalyType, string(detailsJSON))
	if err != nil {
		logStructuredError("intro_anomaly_log_insert_failed", map[string]interface{}{
			"email":        email,
			"customer_id":  customerID,
			"coupon_id":    couponID,
			"anomaly_type": anomalyType,
			"error":        err.Error(),
		})
		return
	}

	logStructured("intro_anomaly_logged", map[string]interface{}{
		"level":        "warn",
		"email":        email,
		"customer_id":  customerID,
		"coupon_id":    couponID,
		"anomaly_type": anomalyType,
	})
}
