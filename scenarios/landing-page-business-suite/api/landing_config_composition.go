package main

import (
	"context"

	"landing-page-business-suite-api/internal/commerce"
	"landing-page-business-suite-api/internal/delivery"
	"landing-page-business-suite-api/internal/experimentation"
	"landing-page-business-suite-api/internal/landing"
	"landing-page-business-suite-api/internal/logx"
)

func newLandingConfigService(configStore *experimentation.ConfigStore, planService *commerce.PlanService, downloadService *delivery.CatalogService, stripe *StripeService) *landing.LandingConfigService {
	service := landing.NewLandingConfigServiceWithConfigStore(configStore, planService, downloadService, landingIntroOfferLookup(stripe))
	service.UseEventLogger(logx.Info)
	return service
}

// landingIntroOfferLookup adapts Stripe's provider-specific coupon model at
// composition time. The content domain only receives display-safe offer data.
func landingIntroOfferLookup(stripe *StripeService) landing.IntroOfferLookup {
	if stripe == nil {
		return nil
	}
	return func(ctx context.Context, couponID string) (*landing.IntroOffer, error) {
		coupon, err := stripe.GetCoupon(ctx, couponID)
		if err != nil || coupon == nil {
			return nil, err
		}
		offer := &landing.IntroOffer{
			ID: coupon.ID, Duration: coupon.Duration, DurationInMonths: coupon.DurationInMonths,
			MaxRedemptions: coupon.MaxRedemptions, TimesRedeemed: coupon.TimesRedeemed,
			Valid: coupon.Valid, Created: coupon.Created, IsIntroCoupon: coupon.IsIntroCoupon,
		}
		if coupon.Name != "" {
			offer.Name = &coupon.Name
		}
		if coupon.AmountOff != nil {
			offer.AmountOff = *coupon.AmountOff
		}
		if coupon.PercentOff != nil {
			offer.PercentOff = *coupon.PercentOff
		}
		if coupon.Currency != "" {
			offer.Currency = &coupon.Currency
		}
		if coupon.RedeemBy != nil {
			offer.RedeemBy = *coupon.RedeemBy
		}
		if coupon.IntroTier != "" {
			offer.IntroTier = &coupon.IntroTier
		}
		return offer, nil
	}
}
