package main

import (
	"context"

	"landing-page-business-suite-api/internal/commerce"
	"landing-page-business-suite-api/internal/logx"
)

func (s *StripeService) introOfferService() *commerce.IntroOfferService {
	return commerce.NewIntroOfferService(s.db, stripeCouponRequester{service: s}, logx.Error)
}

// checkIntroEligibility delegates durable offer policy to commerce.
func (s *StripeService) checkIntroEligibility(ctx context.Context, email string) (bool, error) {
	return s.introOfferService().Eligible(ctx, email)
}

// markIntroUsed records a successfully redeemed intro offer and synchronizes
// Stripe customer metadata as a best-effort secondary audit trail.
func (s *StripeService) markIntroUsed(ctx context.Context, email, customerID, couponID, planTier, subscriptionID string) error {
	return s.introOfferService().MarkUsed(ctx, email, customerID, couponID, planTier, subscriptionID)
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
		logx.Error("intro_anomaly_log_insert_failed", map[string]interface{}{
			"email":        email,
			"customer_id":  customerID,
			"coupon_id":    couponID,
			"anomaly_type": anomalyType,
			"error":        "payment_anomaly_service_unavailable",
		})
		return
	}
	if _, err := s.paymentAnomaly.Log(context.Background(), commerce.PaymentAnomaly{
		Type:        anomalyType,
		Severity:    "warn",
		Email:       email,
		CustomerID:  customerID,
		SubjectID:   couponID,
		SubjectKind: "intro_coupon",
		Details:     details,
	}); err != nil {
		logx.Error("intro_anomaly_log_insert_failed", map[string]interface{}{
			"email":        email,
			"customer_id":  customerID,
			"coupon_id":    couponID,
			"anomaly_type": anomalyType,
			"error":        err.Error(),
		})
	}
}
