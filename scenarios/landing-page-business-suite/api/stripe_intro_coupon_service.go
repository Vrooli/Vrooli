package main

import (
	"context"
	"database/sql"
	"errors"
	"fmt"
	"net/http"
	"net/url"
)

// checkIntroEligibility checks if a user is eligible for the intro pricing coupon.
// Returns true if the user has never used the intro offer before.
func (s *StripeService) checkIntroEligibility(ctx context.Context, email string) (bool, error) {
	email = NormalizeEmail(email)
	if email == "" {
		// No email means we can't track eligibility, so don't apply intro.
		return false, nil
	}

	var hasUsedIntro sql.NullBool
	err := s.db.QueryRowContext(ctx, `
		SELECT has_used_intro FROM users WHERE email = $1
	`, email).Scan(&hasUsedIntro)
	if err == sql.ErrNoRows {
		return true, nil
	}
	if err != nil {
		return false, fmt.Errorf("check intro eligibility: %w", err)
	}
	return !hasUsedIntro.Valid || !hasUsedIntro.Bool, nil
}

// markIntroUsed records a successfully redeemed intro offer and synchronizes
// Stripe customer metadata as a best-effort secondary audit trail.
func (s *StripeService) markIntroUsed(ctx context.Context, email, customerID, couponID, planTier, subscriptionID string) error {
	email = NormalizeEmail(email)
	if email == "" {
		return errors.New("email required to mark intro used")
	}

	tx, err := s.db.BeginTx(ctx, nil)
	if err != nil {
		return fmt.Errorf("begin transaction: %w", err)
	}
	defer func() { _ = tx.Rollback() }()

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

	if customerID != "" {
		stripeCustomerID := customerID
		introCouponID := couponID
		// #nosec G118 -- best-effort metadata sync receives immutable string copies.
		metadataCtx := context.WithoutCancel(ctx)
		go func(ctx context.Context, customerID, couponID string) {
			values := url.Values{}
			values.Set("metadata[has_used_intro]", "true")
			values.Set("metadata[intro_coupon_id]", couponID)
			_, updateErr := s.doStripeForm(ctx, http.MethodPost, "/v1/customers/"+url.PathEscape(customerID), values)
			if updateErr != nil {
				logStructuredError("stripe_customer_metadata_update_failed", map[string]interface{}{
					"customer_id": customerID,
					"error":       updateErr.Error(),
				})
			}
		}(metadataCtx, stripeCustomerID, introCouponID)
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

// isIntroCoupon reports whether couponID appears in the configured intro map.
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

// extractIntroCouponFromInvoice finds a configured intro coupon across Stripe's
// supported invoice discount representations.
func (s *StripeService) extractIntroCouponFromInvoice(obj map[string]interface{}) string {
	if discount, ok := obj["discount"].(map[string]interface{}); ok {
		if coupon, ok := discount["coupon"].(map[string]interface{}); ok {
			if couponID, ok := coupon["id"].(string); ok && s.isIntroCoupon(couponID) {
				return couponID
			}
		}
	}
	if discounts, ok := obj["discounts"].([]interface{}); ok {
		for _, discount := range discounts {
			if discountMap, ok := discount.(map[string]interface{}); ok {
				if coupon, ok := discountMap["coupon"].(map[string]interface{}); ok {
					if couponID, ok := coupon["id"].(string); ok && s.isIntroCoupon(couponID) {
						return couponID
					}
				}
			}
		}
	}
	if amounts, ok := obj["total_discount_amounts"].([]interface{}); ok {
		for _, amount := range amounts {
			if amountMap, ok := amount.(map[string]interface{}); ok {
				if discount, ok := amountMap["discount"].(map[string]interface{}); ok {
					if coupon, ok := discount["coupon"].(map[string]interface{}); ok {
						if couponID, ok := coupon["id"].(string); ok && s.isIntroCoupon(couponID) {
							return couponID
						}
					}
				}
			}
		}
	}
	return ""
}

// checkIntroCouponMapping returns whether couponID is configured and its tier.
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

// logIntroAnomaly forwards suspicious redemption events to the unified payment
// anomaly pipeline for operator review and configured notification delivery.
func (s *StripeService) logIntroAnomaly(email, customerID, couponID, anomalyType string, details map[string]interface{}) {
	if email == "" {
		return
	}
	if s.paymentAnomaly == nil {
		logStructuredError("intro_anomaly_log_insert_failed", map[string]interface{}{
			"email":        email,
			"customer_id":  customerID,
			"coupon_id":    couponID,
			"anomaly_type": anomalyType,
			"error":        "payment_anomaly_service_unavailable",
		})
		return
	}
	if _, err := s.paymentAnomaly.Log(context.Background(), PaymentAnomaly{
		Type:        anomalyType,
		Severity:    anomalySeverityWarn,
		Email:       email,
		CustomerID:  customerID,
		SubjectID:   couponID,
		SubjectKind: "intro_coupon",
		Details:     details,
	}); err != nil {
		logStructuredError("intro_anomaly_log_insert_failed", map[string]interface{}{
			"email":        email,
			"customer_id":  customerID,
			"coupon_id":    couponID,
			"anomaly_type": anomalyType,
			"error":        err.Error(),
		})
	}
}
