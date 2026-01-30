package main

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"net/http"
	"net/url"
	"strings"

	landing_page_react_vite_v1 "github.com/vrooli/vrooli/packages/proto/gen/go/landing-page-react-vite/v1"
	"google.golang.org/protobuf/types/known/timestamppb"
)

// --- StripeCheckoutService Interface Implementation ---
// This file contains checkout session creation and price verification.

// StripePriceInfo provides typed price verification results.
type StripePriceInfo struct {
	ID          string `json:"id"`
	LookupKey   string `json:"lookup_key,omitempty"`
	Currency    string `json:"currency"`
	AmountCents int64  `json:"amount_cents"`
	Interval    string `json:"interval,omitempty"`
	Active      bool   `json:"active"`
	ProductName string `json:"product_name,omitempty"`
}

// VerifyStripePriceTyped returns typed price info (testable wrapper).
func (s *StripeService) VerifyStripePriceTyped(key string) (*StripePriceInfo, error) {
	result, err := s.VerifyStripePrice(key)
	if err != nil {
		return nil, err
	}

	info := &StripePriceInfo{}

	if id, ok := result["id"].(string); ok {
		info.ID = id
	}
	if lookupKey, ok := result["lookup_key"].(string); ok {
		info.LookupKey = lookupKey
	}
	if currency, ok := result["currency"].(string); ok {
		info.Currency = currency
	}
	if amountCents, ok := result["amount_cents"].(int64); ok {
		info.AmountCents = amountCents
	}
	if interval, ok := result["interval"].(string); ok {
		info.Interval = interval
	}
	if active, ok := result["active"].(bool); ok {
		info.Active = active
	}
	if productName, ok := result["product"].(string); ok {
		info.ProductName = productName
	}

	return info, nil
}

// resolveStripePriceID accepts a Stripe price ID or lookup key and returns a concrete price ID.
func (s *StripeService) resolveStripePriceID(key string) (string, error) {
	trimmed := strings.TrimSpace(key)
	if trimmed == "" {
		return "", errors.New("stripe price id is required")
	}
	if strings.HasPrefix(trimmed, "price_") {
		return trimmed, nil
	}

	// Treat as lookup key; fetch the first matching price.
	values := url.Values{}
	values.Set("lookup_keys[]", trimmed)
	values.Set("limit", "1")
	values.Set("expand[]", "data.product")
	path := "/v1/prices?" + values.Encode()

	body, err := s.doStripeRequest(context.Background(), http.MethodGet, path, nil, "")
	if err != nil {
		return "", err
	}

	var resp struct {
		Data []struct {
			ID string `json:"id"`
		} `json:"data"`
	}
	if err := json.Unmarshal(body, &resp); err != nil {
		return "", fmt.Errorf("decode stripe price lookup: %w", err)
	}
	if len(resp.Data) == 0 || strings.TrimSpace(resp.Data[0].ID) == "" {
		return "", fmt.Errorf("no stripe price found for lookup key %s", trimmed)
	}
	return resp.Data[0].ID, nil
}

// VerifyStripePrice fetches price details from Stripe using either a price ID or lookup key.
func (s *StripeService) VerifyStripePrice(key string) (map[string]interface{}, error) {
	resolved, err := s.resolveStripePriceID(key)
	if err != nil {
		return nil, err
	}

	path := "/v1/prices/" + url.PathEscape(resolved) + "?expand[]=product"
	body, err := s.doStripeRequest(context.Background(), http.MethodGet, path, nil, "")
	if err != nil {
		return nil, err
	}

	var price struct {
		ID         string `json:"id"`
		LookupKey  string `json:"lookup_key"`
		Currency   string `json:"currency"`
		UnitAmount int64  `json:"unit_amount"`
		Active     bool   `json:"active"`
		Recurring  *struct {
			Interval string `json:"interval"`
		} `json:"recurring"`
		Product struct {
			Name string `json:"name"`
		} `json:"product"`
	}
	if err := json.Unmarshal(body, &price); err != nil {
		return nil, fmt.Errorf("decode stripe price: %w", err)
	}

	interval := "one_time"
	if price.Recurring != nil && strings.TrimSpace(price.Recurring.Interval) != "" {
		interval = price.Recurring.Interval
	}

	return map[string]interface{}{
		"id":           price.ID,
		"lookup_key":   price.LookupKey,
		"currency":     price.Currency,
		"amount_cents": price.UnitAmount,
		"interval":     interval,
		"active":       price.Active,
		"product":      price.Product.Name,
	}, nil
}

// CreateCheckoutSession creates a Stripe checkout session
// [REQ:STRIPE-ROUTES] POST /api/checkout/create endpoint
func (s *StripeService) CreateCheckoutSession(priceID string, successURL string, cancelURL string, customerEmail string) (*landing_page_react_vite_v1.CheckoutSession, error) {
	ctx := context.Background()

	// Normalize email at entry point for consistent lookups and eligibility checks
	customerEmail = NormalizeEmail(customerEmail)

	plan, err := s.planService.GetPlanByPriceID(priceID)
	if err != nil {
		return nil, fmt.Errorf("price %s not found: %w", priceID, err)
	}

	resolvedPriceID, err := s.resolveStripePriceID(priceID)
	if err != nil {
		return nil, fmt.Errorf("resolve price %s: %w", priceID, err)
	}

	if successURL == "" {
		successURL = "/success"
	}
	if cancelURL == "" {
		cancelURL = "/cancel"
	}

	mode := "subscription"
	sessionKind := landing_page_react_vite_v1.SessionKind_SESSION_KIND_SUBSCRIPTION
	sessionType := sessionTypeSubscription
	if plan.Kind == landing_page_react_vite_v1.PlanKind_PLAN_KIND_CREDITS_TOPUP {
		mode = "payment"
		sessionKind = landing_page_react_vite_v1.SessionKind_SESSION_KIND_CREDITS_TOPUP
		sessionType = sessionTypeCreditsTopup
	} else if plan.Kind == landing_page_react_vite_v1.PlanKind_PLAN_KIND_SUPPORTER_CONTRIBUTION {
		mode = "payment"
		sessionKind = landing_page_react_vite_v1.SessionKind_SESSION_KIND_SUPPORTER_CONTRIBUTION
		sessionType = sessionTypeSupporterContribution
	}

	values := url.Values{}
	values.Set("mode", mode)
	values.Set("success_url", successURL)
	values.Set("cancel_url", cancelURL)
	values.Set("line_items[0][price]", resolvedPriceID)
	values.Set("line_items[0][quantity]", "1")
	values.Set("metadata[bundle_key]", plan.BundleKey)
	values.Set("metadata[plan_tier]", plan.PlanTier)
	if customerEmail != "" {
		values.Set("customer_email", customerEmail)
		values.Set("metadata[user_identity]", customerEmail)
	}

	if existingCustomerID := s.lookupCustomerID(customerEmail); existingCustomerID != "" {
		values.Del("customer_email")
		values.Set("customer", existingCustomerID)
	}

	// Apply coupon for subscriptions
	// Priority: 1) plan-specific coupon mapping, 2) tier-based intro coupon (legacy fallback)
	var appliedCouponID string
	if mode == "subscription" && customerEmail != "" {
		// First, check for plan-specific coupon mapping (from admin UI)
		planCouponID := s.planService.GetCouponForPlan(resolvedPriceID)
		if planCouponID != "" {
			// Plan-specific coupon found - check eligibility
			eligible, eligErr := s.checkIntroEligibility(ctx, customerEmail)
			if eligErr != nil {
				logStructuredError("coupon_eligibility_check_failed", map[string]interface{}{
					"email": customerEmail,
					"error": eligErr.Error(),
				})
				// Continue without coupon on error
			} else if eligible {
				values.Set("discounts[0][coupon]", planCouponID)
				values.Set("metadata[coupon_applied]", planCouponID)
				values.Set("metadata[coupon_source]", "plan_mapping")
				appliedCouponID = planCouponID
				logStructured("plan_coupon_applying", map[string]interface{}{
					"level":     "info",
					"email":     customerEmail,
					"coupon_id": planCouponID,
					"price_id":  resolvedPriceID,
					"source":    "plan_mapping",
				})
			}
		} else if plan.BillingInterval == landing_page_react_vite_v1.BillingInterval_BILLING_INTERVAL_MONTH {
			// Fallback to tier-based intro coupon (legacy env var config)
			eligible, eligErr := s.checkIntroEligibility(ctx, customerEmail)
			if eligErr != nil {
				logStructuredError("intro_eligibility_check_failed", map[string]interface{}{
					"email": customerEmail,
					"error": eligErr.Error(),
				})
				// Continue without intro pricing on error
			} else if eligible {
				couponID := s.introCouponConfig.GetCouponForTier(plan.PlanTier)
				if couponID != "" {
					values.Set("discounts[0][coupon]", couponID)
					values.Set("metadata[intro_coupon_applied]", couponID)
					values.Set("metadata[coupon_source]", "tier_config")
					appliedCouponID = couponID
					logStructured("intro_coupon_applying", map[string]interface{}{
						"level":     "info",
						"email":     customerEmail,
						"coupon_id": couponID,
						"plan_tier": plan.PlanTier,
						"source":    "tier_config",
					})
				}
			}
		}
	}

	body, err := s.doStripeForm(ctx, http.MethodPost, "/v1/checkout/sessions", values)
	if err != nil {
		return nil, fmt.Errorf("create checkout for price %s: %w", priceID, err)
	}

	var resp struct {
		ID            string `json:"id"`
		URL           string `json:"url"`
		Status        string `json:"status"`
		Subscription  string `json:"subscription"`
		Customer      string `json:"customer"`
		CustomerEmail string `json:"customer_email"`
		AmountTotal   int64  `json:"amount_total"`
		PaymentStatus string `json:"payment_status"`
		Mode          string `json:"mode"`
		Currency      string `json:"currency"`
	}

	if err := json.Unmarshal(body, &resp); err != nil {
		return nil, fmt.Errorf("decode stripe response: %w", err)
	}

	amount := resp.AmountTotal
	if amount == 0 {
		amount = plan.AmountCents
	}

	meta := map[string]interface{}{
		"bundle_key": plan.BundleKey,
		"plan_tier":  plan.PlanTier,
		"kind":       plan.Kind.String(),
	}
	if appliedCouponID != "" {
		meta["intro_coupon_applied"] = appliedCouponID
	}
	metaBytes, _ := json.Marshal(meta)

	_, err = s.db.Exec(`
		INSERT INTO checkout_sessions (session_id, customer_email, customer_id, price_id, subscription_id, status, session_type, amount_cents, metadata, created_at, updated_at)
		VALUES ($1,$2,$3,$4,$5,$6,$7,$8,$9,NOW(),NOW())
		ON CONFLICT (session_id) DO UPDATE SET
			customer_email = EXCLUDED.customer_email,
			customer_id = EXCLUDED.customer_id,
			price_id = EXCLUDED.price_id,
			subscription_id = EXCLUDED.subscription_id,
			status = EXCLUDED.status,
			session_type = EXCLUDED.session_type,
			amount_cents = EXCLUDED.amount_cents,
			metadata = EXCLUDED.metadata,
			updated_at = NOW()
	`, resp.ID, resp.CustomerEmail, resp.Customer, priceID, resp.Subscription, resp.Status, sessionType, amount, string(metaBytes))
	if err != nil {
		return nil, err
	}

	cfg := s.getConfig()
	session := &landing_page_react_vite_v1.CheckoutSession{
		SessionId:      resp.ID,
		SessionKind:    sessionKind,
		Status:         landing_page_react_vite_v1.CheckoutSessionStatus_CHECKOUT_SESSION_STATUS_OPEN,
		Url:            resp.URL,
		PublishableKey: cfg.publishableKey,
		CustomerEmail:  resp.CustomerEmail,
		StripePriceId:  priceID,
		AmountCents:    amount,
		Currency:       plan.Currency,
		SuccessUrl:     successURL,
		CancelUrl:      cancelURL,
		CreatedAt:      timestamppb.Now(),
	}

	return session, nil
}
