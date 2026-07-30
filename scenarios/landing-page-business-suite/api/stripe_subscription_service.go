package main

import (
	"context"
	"database/sql"
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

	state := commerce.MapSubscriptionState(status)
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
	return s.subscriptionManagementService().Cancel(context.Background(), userIdentity)
}

// CreateBillingPortalSession creates a Stripe billing portal session for subscription management.
func (s *StripeService) CreateBillingPortalSession(ctx context.Context, userIdentity string, returnURL string) (*landing_page_business_suite_v1.BillingPortalResponse, error) {
	return s.subscriptionManagementService().CreateBillingPortalSession(ctx, userIdentity, returnURL)
}

// refreshSubscriptionFromStripe fetches subscription data from Stripe and updates local cache.
func (s *StripeService) refreshSubscriptionFromStripe(userIdentity string, currentSubscriptionID string) (*shared.SubscriptionStatus, error) {
	return s.subscriptionRefresher().Refresh(context.Background(), userIdentity, currentSubscriptionID)
}

// persistSubscriptionFromStripe saves subscription data from Stripe to local database.
func (s *StripeService) persistSubscriptionFromStripe(userHint string, sub *commerce.StripeSubscription) (*shared.SubscriptionStatus, error) {
	return commerce.NewSubscriptionPersistenceService(s.db, s.planService, logStructured).Persist(userHint, sub)
}
