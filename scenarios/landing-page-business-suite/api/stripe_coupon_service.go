package main

import (
	"context"
	"io"

	"landing-page-business-suite-api/internal/commerce"
	"landing-page-business-suite-api/internal/logx"
)

// stripeCouponRequester is the root composition adapter for authenticated
// Stripe transport. Coupon policy and DTO mapping live in internal/commerce.
type stripeCouponRequester struct{ service *StripeService }

func (r stripeCouponRequester) Request(ctx context.Context, method, path string, body io.Reader, contentType string) ([]byte, error) {
	return r.service.doStripeRequest(ctx, method, path, body, contentType)
}

func (s *StripeService) couponProvider() *commerce.StripeCouponProvider {
	return commerce.NewStripeCouponProvider(stripeCouponRequester{service: s}, s.planService, s.GetIntroCouponMap)
}

func (s *StripeService) subscriptionManagementService() *commerce.SubscriptionManagementService {
	return commerce.NewSubscriptionManagementService(s.db, stripeCouponRequester{service: s}, logx.Error)
}

func (s *StripeService) subscriptionRefresher() *commerce.SubscriptionRefresher {
	return commerce.NewSubscriptionRefresher(s.db, s.planService, stripeCouponRequester{service: s}, logx.Info)
}

func (s *StripeService) ListCoupons(ctx context.Context) ([]commerce.Coupon, error) {
	return s.couponProvider().ListCoupons(ctx)
}

func (s *StripeService) GetCoupon(ctx context.Context, couponID string) (*commerce.Coupon, error) {
	return s.couponProvider().GetCoupon(ctx, couponID)
}

func (s *StripeService) CreateCoupon(ctx context.Context, input commerce.CreateCouponInput) (*commerce.Coupon, error) {
	return s.couponProvider().CreateCoupon(ctx, input)
}

func (s *StripeService) UpdateCoupon(ctx context.Context, couponID string, input commerce.UpdateCouponInput) (*commerce.Coupon, error) {
	return s.couponProvider().UpdateCoupon(ctx, couponID, input)
}

func (s *StripeService) DeleteCoupon(ctx context.Context, couponID string) error {
	return s.couponProvider().DeleteCoupon(ctx, couponID)
}

func (s *StripeService) GetCouponImportPreview(ctx context.Context) (*commerce.CouponImportPreview, error) {
	return s.couponProvider().GetCouponImportPreview(ctx)
}

// GetIntroCouponMap exposes an immutable configuration snapshot to generated
// coupon transport and the commerce provider implementation.
func (s *StripeService) GetIntroCouponMap() map[string]string {
	s.mu.RLock()
	defer s.mu.RUnlock()
	if !s.introCouponConfig.Enabled || s.introCouponConfig.CouponMap == nil {
		return nil
	}
	result := make(map[string]string, len(s.introCouponConfig.CouponMap))
	for tier, couponID := range s.introCouponConfig.CouponMap {
		result[tier] = couponID
	}
	return result
}
