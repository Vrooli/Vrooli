package main

import (
	"context"
	"database/sql"
	"encoding/json"
	"errors"
	"fmt"
	"net/http"
	"net/url"
	"strings"
	"time"

	landing_page_business_suite_v1 "github.com/vrooli/vrooli/packages/proto/gen/go/landing-page-business-suite/v1"
	shared "github.com/vrooli/vrooli/packages/proto/gen/go/landing-page-business-suite/v1/shared"
	"google.golang.org/protobuf/proto"
	"google.golang.org/protobuf/types/known/timestamppb"
	"landing-page-business-suite-api/internal/commerce"
)

// --- StripeSubscriptionService Interface Implementation ---
// This file contains subscription verification, cancellation, and billing portal operations.

// VerifySubscription checks subscription status for a user
// [REQ:SUB-VERIFY] GET /api/subscription/verify endpoint
func (s *StripeService) VerifySubscription(userIdentity string) (*shared.SubscriptionStatus, error) {
	user := strings.TrimSpace(userIdentity)
	if user == "" {
		return &shared.SubscriptionStatus{
			State:        shared.SubscriptionState_SUBSCRIPTION_STATE_INACTIVE,
			UserIdentity: "",
			Message:      proto.String("user not provided"),
		}, nil
	}

	// Normalize email for case-insensitive lookup (but preserve customer IDs as-is)
	if strings.Contains(user, "@") {
		user = NormalizeEmail(user)
	}

	var status string
	var canceledAt *time.Time
	var updatedAt time.Time
	var priceID sql.NullString
	var planTier sql.NullString
	var bundleKey sql.NullString
	var subscriptionID sql.NullString

	err := s.db.QueryRow(`
		SELECT status, canceled_at, updated_at, price_id, plan_tier, bundle_key, subscription_id
		FROM subscriptions
		WHERE customer_email = $1 OR customer_id = $1
		ORDER BY updated_at DESC
		LIMIT 1
	`, user).Scan(&status, &canceledAt, &updatedAt, &priceID, &planTier, &bundleKey, &subscriptionID)

	needRefresh := false
	if err == sql.ErrNoRows {
		needRefresh = true
	} else if err != nil {
		return nil, err
	} else if time.Since(updatedAt) > s.checkoutCacheTTL {
		needRefresh = true
		logStructured("Subscription cache stale", map[string]interface{}{
			"level":         "warn",
			"user_identity": user,
			"cache_age_ms":  time.Since(updatedAt).Milliseconds(),
		})
	}

	if needRefresh {
		if refreshed, err := s.refreshSubscriptionFromStripe(user, subscriptionID.String); err == nil && refreshed != nil {
			return refreshed, nil
		} else if err != nil {
			logStructured("Stripe verification fallback to cache", map[string]interface{}{
				"level": "warn",
				"user":  user,
				"error": err.Error(),
			})
		}
	}

	if err == sql.ErrNoRows {
		return &shared.SubscriptionStatus{
			State:        shared.SubscriptionState_SUBSCRIPTION_STATE_INACTIVE,
			UserIdentity: user,
			Message:      proto.String("No subscription found"),
		}, nil
	}

	state := mapSubscriptionState(status)
	result := &shared.SubscriptionStatus{
		State:        state,
		UserIdentity: user,
		CachedAt:     timestamppb.New(updatedAt),
		CacheAgeMs:   time.Since(updatedAt).Milliseconds(),
	}
	if canceledAt != nil {
		result.CanceledAt = timestamppb.New(*canceledAt)
	}
	if priceID.Valid {
		result.StripePriceId = proto.String(priceID.String)
		if planTier.String == "" {
			if plan, err := s.planService.GetPlanByPriceID(priceID.String); err == nil {
				planTier.String = plan.PlanTier
				bundleKey.String = plan.BundleKey
			}
		}
		if planTier.String != "" {
			if _, err := commerce.NormalizePlanTier(planTier.String); err != nil {
				logStructured("stripe_subscription_plan_tier_invalid", map[string]interface{}{
					"level":        "warn",
					"plan_tier":    planTier.String,
					"price_id":     priceID.String,
					"subscription": subscriptionID.String,
				})
				planTier.String = ""
			} else if subscriptionID.Valid {
				_, _ = s.db.Exec(`
					UPDATE subscriptions
					SET plan_tier = COALESCE(NULLIF($1,''), plan_tier),
						bundle_key = COALESCE(NULLIF($2,''), bundle_key),
						updated_at = NOW()
					WHERE subscription_id = $3
				`, planTier.String, bundleKey.String, subscriptionID.String)
			}
		}
	}
	if planTier.Valid {
		result.PlanTier = proto.String(planTier.String)
	}
	if bundleKey.Valid {
		result.BundleKey = proto.String(bundleKey.String)
	}
	if subscriptionID.Valid {
		result.SubscriptionId = proto.String(subscriptionID.String)
	}

	return result, nil
}

// CancelSubscription cancels an active subscription
// [REQ:SUB-CANCEL] POST /api/subscription/cancel endpoint
func (s *StripeService) CancelSubscription(userIdentity string) (*landing_page_business_suite_v1.CancelSubscriptionResponse, error) {
	var subscriptionID string
	var status string
	var customerID sql.NullString

	// Find active subscription
	err := s.db.QueryRow(`
		SELECT subscription_id, status, customer_id
		FROM subscriptions
		WHERE (customer_email = $1 OR customer_id = $1)
		AND status IN ('active', 'trialing')
		ORDER BY created_at DESC
		LIMIT 1
	`, userIdentity).Scan(&subscriptionID, &status, &customerID)

	if err == sql.ErrNoRows {
		return nil, errors.New("no active subscription found")
	}

	if err != nil {
		return nil, err
	}

	// Call Stripe FIRST - fail if this fails to prevent local DB update
	// while Stripe continues to charge the customer
	_, stripeErr := s.doStripeForm(context.Background(), http.MethodPost, "/v1/subscriptions/"+url.PathEscape(subscriptionID), url.Values{
		"cancel_at_period_end": {"true"},
	})
	if stripeErr != nil {
		logStructuredError("failed_to_cancel_subscription_on_stripe", map[string]interface{}{
			"id":    subscriptionID,
			"user":  userIdentity,
			"error": stripeErr.Error(),
		})
		return nil, fmt.Errorf("failed to cancel subscription with Stripe: %w", stripeErr)
	}

	// Only update local DB after Stripe confirms success
	now := time.Now()
	_, err = s.db.Exec(`
		UPDATE subscriptions
		SET status = $1, canceled_at = $2, updated_at = $3
		WHERE subscription_id = $4
	`, "canceled", now, now, subscriptionID)
	if err != nil {
		return nil, err
	}

	return &landing_page_business_suite_v1.CancelSubscriptionResponse{
		SubscriptionId: proto.String(subscriptionID),
		State:          mapSubscriptionState("canceled"),
		CanceledAt:     timestamppb.New(now),
		Message:        proto.String("Subscription canceled successfully"),
	}, nil
}

// CreateBillingPortalSession creates a Stripe billing portal session for subscription management.
func (s *StripeService) CreateBillingPortalSession(ctx context.Context, userIdentity string, returnURL string) (*landing_page_business_suite_v1.BillingPortalResponse, error) {
	user := strings.TrimSpace(userIdentity)
	if user == "" {
		return nil, errors.New("user identity is required")
	}

	customerID := s.lookupCustomerID(user)
	if customerID == "" {
		if strings.Contains(user, "@") {
			customer, err := s.findCustomerByEmail(ctx, user)
			if err != nil {
				return nil, err
			}
			if customer != nil {
				customerID = customer.ID
			}
		} else {
			customerID = user
		}
	}

	if customerID == "" {
		return nil, errors.New("no Stripe customer found for user")
	}

	values := url.Values{}
	values.Set("customer", customerID)
	if strings.TrimSpace(returnURL) != "" {
		values.Set("return_url", returnURL)
	}

	body, err := s.doStripeForm(ctx, http.MethodPost, "/v1/billing_portal/sessions", values)
	if err != nil {
		return nil, err
	}
	var resp struct {
		URL string `json:"url"`
	}
	if err := json.Unmarshal(body, &resp); err != nil {
		return nil, err
	}
	if resp.URL == "" {
		return nil, errors.New("Stripe portal URL not returned")
	}

	return &landing_page_business_suite_v1.BillingPortalResponse{Url: resp.URL}, nil
}

// refreshSubscriptionFromStripe fetches subscription data from Stripe and updates local cache.
func (s *StripeService) refreshSubscriptionFromStripe(userIdentity string, currentSubscriptionID string) (*shared.SubscriptionStatus, error) {
	ctx := context.Background()

	if currentSubscriptionID != "" {
		if sub, err := s.fetchSubscription(ctx, currentSubscriptionID); err == nil {
			return s.persistSubscriptionFromStripe(userIdentity, sub)
		} else {
			logStructured("stripe fetch subscription failed", map[string]interface{}{
				"level":   "warn",
				"id":      currentSubscriptionID,
				"error":   err.Error(),
				"user_id": userIdentity,
			})
		}
	}

	if strings.HasPrefix(strings.TrimSpace(userIdentity), "sub_") {
		if sub, err := s.fetchSubscription(ctx, userIdentity); err == nil {
			return s.persistSubscriptionFromStripe(userIdentity, sub)
		}
	}

	if strings.Contains(userIdentity, "@") {
		customer, err := s.findCustomerByEmail(ctx, userIdentity)
		if err != nil {
			return nil, err
		}
		if customer == nil {
			return nil, nil
		}
		sub, err := s.latestSubscriptionForCustomer(ctx, customer.ID)
		if err != nil {
			return nil, err
		}
		return s.persistSubscriptionFromStripe(userIdentity, sub)
	}

	if strings.TrimSpace(userIdentity) != "" {
		sub, err := s.latestSubscriptionForCustomer(ctx, userIdentity)
		if err != nil {
			return nil, err
		}
		return s.persistSubscriptionFromStripe(userIdentity, sub)
	}

	return nil, nil
}

// persistSubscriptionFromStripe saves subscription data from Stripe to local database.
func (s *StripeService) persistSubscriptionFromStripe(userHint string, sub *stripeSubscription) (*shared.SubscriptionStatus, error) {
	if sub == nil {
		return nil, nil
	}

	priceID := ""
	if len(sub.Items.Data) > 0 {
		priceID = sub.Items.Data[0].Price.ID
	}

	planTier := ""
	bundleKey := s.planService.BundleKey()

	if planTierVal, ok := sub.Metadata["plan_tier"].(string); ok && planTierVal != "" {
		planTier = planTierVal
	}
	if bundleVal, ok := sub.Metadata["bundle_key"].(string); ok && bundleVal != "" {
		bundleKey = bundleVal
	}

	if priceID != "" {
		if plan, err := s.planService.GetPlanByPriceID(priceID); err == nil {
			if plan.PlanTier != "" {
				planTier = plan.PlanTier
			}
			if plan.BundleKey != "" {
				bundleKey = plan.BundleKey
			}
		} else {
			logStructured("stripe_plan_lookup_failed", map[string]interface{}{
				"level":    "warn",
				"price_id": priceID,
				"error":    err.Error(),
			})
		}
	}
	if strings.TrimSpace(planTier) == "" && strings.TrimSpace(priceID) != "" {
		if inferred, ok := commerce.DetectTierToken(priceID); ok {
			planTier = inferred
		}
	}
	if strings.TrimSpace(planTier) != "" {
		if _, err := commerce.NormalizePlanTier(planTier); err != nil {
			logStructured("stripe_subscription_plan_tier_invalid", map[string]interface{}{
				"level":        "warn",
				"plan_tier":    planTier,
				"price_id":     priceID,
				"subscription": sub.ID,
			})
			planTier = ""
		}
	}

	state := mapSubscriptionState(sub.Status)
	now := time.Now()

	var canceledAt *time.Time
	if sub.CanceledAt > 0 {
		ts := time.Unix(sub.CanceledAt, 0)
		canceledAt = &ts
	}

	// Extract billing cycle day from anchor timestamp
	billingCycleStart := extractBillingCycleDay(sub.BillingCycleAnchor)

	_, err := s.db.Exec(`
		INSERT INTO subscriptions (subscription_id, customer_id, customer_email, status, plan_tier, price_id, bundle_key, billing_cycle_start, canceled_at, created_at, updated_at)
		VALUES ($1::varchar,$2::varchar,$3::varchar,$4::varchar,$5::varchar,$6::varchar,$7::varchar,$8::int,$9::timestamp,COALESCE((SELECT created_at FROM subscriptions WHERE subscription_id = $1::varchar), NOW()), NOW())
		ON CONFLICT (subscription_id) DO UPDATE SET
			customer_id = EXCLUDED.customer_id,
			customer_email = EXCLUDED.customer_email,
			status = EXCLUDED.status,
			plan_tier = COALESCE(NULLIF(EXCLUDED.plan_tier,''), subscriptions.plan_tier),
			price_id = COALESCE(NULLIF(EXCLUDED.price_id,''), subscriptions.price_id),
			bundle_key = COALESCE(NULLIF(EXCLUDED.bundle_key,''), subscriptions.bundle_key),
			billing_cycle_start = EXCLUDED.billing_cycle_start,
			canceled_at = EXCLUDED.canceled_at,
			updated_at = NOW()
	`, sub.ID, sub.Customer, sub.CustomerEmail, legacyStateLabel(state), planTier, priceID, bundleKey, billingCycleStart, canceledAt)
	if err != nil {
		return nil, err
	}

	user := chooseUserIdentity(userHint, sub)
	status := &shared.SubscriptionStatus{
		State:        state,
		UserIdentity: user,
		CachedAt:     timestamppb.New(now),
	}
	if sub.ID != "" {
		status.SubscriptionId = proto.String(sub.ID)
	}
	if priceID != "" {
		status.StripePriceId = proto.String(priceID)
	}
	if planTier != "" {
		status.PlanTier = proto.String(planTier)
	}
	if bundleKey != "" {
		status.BundleKey = proto.String(bundleKey)
	}
	if canceledAt != nil {
		status.CanceledAt = timestamppb.New(*canceledAt)
	}

	return status, nil
}
