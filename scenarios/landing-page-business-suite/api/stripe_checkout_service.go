package main

import (
	"context"

	lpbsv1 "github.com/vrooli/vrooli/packages/proto/gen/go/landing-page-business-suite/v1"
	"landing-page-business-suite-api/internal/commerce"
)

// StripePriceInfo is the legacy API projection for typed price verification.
type StripePriceInfo struct {
	ID          string `json:"id"`
	LookupKey   string `json:"lookup_key,omitempty"`
	Currency    string `json:"currency"`
	AmountCents int64  `json:"amount_cents"`
	Interval    string `json:"interval,omitempty"`
	Active      bool   `json:"active"`
	ProductName string `json:"product_name,omitempty"`
}

func (s *StripeService) checkoutService() *commerce.CheckoutService {
	return commerce.NewCheckoutService(commerce.CheckoutServiceOptions{Store: s.db, Plans: s.planService, Requester: stripeCouponRequester{service: s}, IntroOffers: s.introOfferService(), IntroCoupon: s.introCouponForTier, PublicKey: s.publishableKey, Log: logStructured})
}

func (s *StripeService) introCouponForTier(tier string) string {
	s.mu.RLock()
	defer s.mu.RUnlock()
	return s.introCouponConfig.GetCouponForTier(tier)
}

func (s *StripeService) publishableKey() string { return s.getConfig().publishableKey }

func (s *StripeService) VerifyStripePriceTyped(key string) (*StripePriceInfo, error) {
	result, err := s.VerifyStripePrice(key)
	if err != nil {
		return nil, err
	}
	return &StripePriceInfo{ID: result["id"].(string), LookupKey: result["lookup_key"].(string), Currency: result["currency"].(string), AmountCents: result["amount_cents"].(int64), Interval: result["interval"].(string), Active: result["active"].(bool), ProductName: result["product"].(string)}, nil
}

func (s *StripeService) resolveStripePriceID(key string) (string, error) {
	return s.checkoutService().ResolvePriceID(context.Background(), key)
}

func (s *StripeService) VerifyStripePrice(key string) (map[string]interface{}, error) {
	resolved, err := s.resolveStripePriceID(key)
	if err != nil {
		return nil, err
	}
	price, err := s.FetchStripePriceDetails(context.Background(), resolved)
	if err != nil {
		return nil, err
	}
	return map[string]interface{}{"id": price.PriceID, "lookup_key": price.LookupKey, "currency": price.Currency, "amount_cents": price.AmountCents, "interval": price.Interval, "active": price.Active, "product": price.ProductName}, nil
}

// CreateCheckoutSession keeps the established service contract while the
// commerce package owns the provider-neutral workflow.
// [REQ:STRIPE-ROUTES] POST /api/checkout/create endpoint
func (s *StripeService) CreateCheckoutSession(priceID, successURL, cancelURL, customerEmail string) (*lpbsv1.CheckoutSession, error) {
	return s.checkoutService().Create(context.Background(), priceID, successURL, cancelURL, customerEmail)
}
