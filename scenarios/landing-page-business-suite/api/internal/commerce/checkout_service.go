package commerce

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"net/url"
	"strings"

	lpbsv1 "github.com/vrooli/vrooli/packages/proto/gen/go/landing-page-business-suite/v1"
	shared "github.com/vrooli/vrooli/packages/proto/gen/go/landing-page-business-suite/v1/shared"
	"google.golang.org/protobuf/types/known/timestamppb"
)

const (
	checkoutSessionSubscription = "subscription"
	checkoutSessionCredits      = "credits_topup"
	checkoutSessionContribution = "supporter_contribution"
)

// CheckoutService owns normalized checkout construction, customer identity
// reuse, coupon selection, provider session creation, and durable local
// session recording. API composition supplies provider access and public-key
// presentation data through narrow callbacks.
type CheckoutService struct {
	store       StripeStore
	plans       *PlanService
	requester   StripeRequester
	introOffers *IntroOfferService
	introCoupon func(string) string
	publicKey   func() string
	logf        func(string, map[string]interface{})
}

type CheckoutServiceOptions struct {
	Store       StripeStore
	Plans       *PlanService
	Requester   StripeRequester
	IntroOffers *IntroOfferService
	IntroCoupon func(string) string
	PublicKey   func() string
	Log         func(string, map[string]interface{})
}

func NewCheckoutService(options CheckoutServiceOptions) *CheckoutService {
	return &CheckoutService{store: options.Store, plans: options.Plans, requester: options.Requester, introOffers: options.IntroOffers, introCoupon: options.IntroCoupon, publicKey: options.PublicKey, logf: options.Log}
}

func (s *CheckoutService) Create(ctx context.Context, priceID, successURL, cancelURL, customerEmail string) (*lpbsv1.CheckoutSession, error) {
	if s.plans == nil || s.store == nil || s.requester == nil {
		return nil, fmt.Errorf("checkout dependencies unavailable")
	}
	customerEmail = normalizeEmail(customerEmail)
	plan, err := s.plans.GetPlanByPriceID(priceID)
	if err != nil {
		return nil, fmt.Errorf("price %s not found: %w", priceID, err)
	}
	resolvedPriceID, err := s.ResolvePriceID(ctx, priceID)
	if err != nil {
		return nil, fmt.Errorf("resolve price %s: %w", priceID, err)
	}
	if successURL == "" {
		successURL = "/success"
	}
	if cancelURL == "" {
		cancelURL = "/cancel"
	}
	mode, kind, sessionType := checkoutShape(plan)
	values := url.Values{"mode": {mode}, "success_url": {successURL}, "cancel_url": {cancelURL}, "line_items[0][price]": {resolvedPriceID}, "line_items[0][quantity]": {"1"}, "metadata[bundle_key]": {plan.BundleKey}, "metadata[plan_tier]": {plan.PlanTier}}
	if customerEmail != "" {
		values.Set("customer_email", customerEmail)
		values.Set("metadata[user_identity]", customerEmail)
	}
	if customerID := NewAccountLinkService(s.store).LookupCustomerID(customerEmail); customerID != "" {
		values.Del("customer_email")
		values.Set("customer", customerID)
	}
	appliedCoupon := s.applyCoupon(ctx, values, plan, resolvedPriceID, customerEmail, mode)
	body, err := s.requester.Request(ctx, http.MethodPost, "/v1/checkout/sessions", strings.NewReader(values.Encode()), "application/x-www-form-urlencoded")
	if err != nil {
		return nil, fmt.Errorf("create checkout for price %s: %w", priceID, err)
	}
	var response stripeCheckoutResponse
	if err := json.Unmarshal(body, &response); err != nil {
		return nil, fmt.Errorf("decode stripe response: %w", err)
	}
	amount := response.AmountTotal
	if amount == 0 {
		amount = plan.AmountCents
	}
	if err := s.record(response, priceID, sessionType, amount, plan, appliedCoupon); err != nil {
		return nil, err
	}
	publicKey := ""
	if s.publicKey != nil {
		publicKey = s.publicKey()
	}
	return &lpbsv1.CheckoutSession{SessionId: response.ID, SessionKind: kind, Status: lpbsv1.CheckoutSessionStatus_CHECKOUT_SESSION_STATUS_OPEN, Url: response.URL, PublishableKey: publicKey, CustomerEmail: response.CustomerEmail, StripePriceId: priceID, AmountCents: amount, Currency: plan.Currency, SuccessUrl: successURL, CancelUrl: cancelURL, CreatedAt: timestamppb.Now()}, nil
}

type stripeCheckoutResponse struct {
	ID            string `json:"id"`
	URL           string `json:"url"`
	Status        string `json:"status"`
	Subscription  string `json:"subscription"`
	Customer      string `json:"customer"`
	CustomerEmail string `json:"customer_email"`
	AmountTotal   int64  `json:"amount_total"`
}

func checkoutShape(plan *PlanOption) (string, lpbsv1.SessionKind, string) {
	if plan.Kind == shared.PlanKind_PLAN_KIND_CREDITS_TOPUP {
		return "payment", lpbsv1.SessionKind_SESSION_KIND_CREDITS_TOPUP, checkoutSessionCredits
	}
	if plan.Kind == shared.PlanKind_PLAN_KIND_SUPPORTER_CONTRIBUTION {
		return "payment", lpbsv1.SessionKind_SESSION_KIND_SUPPORTER_CONTRIBUTION, checkoutSessionContribution
	}
	return "subscription", lpbsv1.SessionKind_SESSION_KIND_SUBSCRIPTION, checkoutSessionSubscription
}

func (s *CheckoutService) applyCoupon(ctx context.Context, values url.Values, plan *PlanOption, resolvedPriceID, email, mode string) string {
	if mode != "subscription" || email == "" || s.introOffers == nil {
		return ""
	}
	if couponID := s.plans.GetCouponForPlan(resolvedPriceID); couponID != "" {
		eligible, err := s.introOffers.Eligible(ctx, email)
		if err != nil {
			s.log("coupon_eligibility_check_failed", map[string]interface{}{"email": email, "error": err.Error()})
			return ""
		}
		if !eligible {
			return ""
		}
		values.Set("discounts[0][coupon]", couponID)
		values.Set("metadata[coupon_applied]", couponID)
		values.Set("metadata[coupon_source]", "plan_mapping")
		s.log("plan_coupon_applying", map[string]interface{}{"level": "info", "email": email, "coupon_id": couponID, "price_id": resolvedPriceID, "source": "plan_mapping"})
		return couponID
	}
	if plan.BillingInterval != shared.BillingInterval_BILLING_INTERVAL_MONTH || s.introCoupon == nil {
		return ""
	}
	eligible, err := s.introOffers.Eligible(ctx, email)
	if err != nil {
		s.log("intro_eligibility_check_failed", map[string]interface{}{"email": email, "error": err.Error()})
		return ""
	}
	if !eligible {
		return ""
	}
	couponID := s.introCoupon(plan.PlanTier)
	if couponID == "" {
		return ""
	}
	values.Set("discounts[0][coupon]", couponID)
	values.Set("metadata[intro_coupon_applied]", couponID)
	values.Set("metadata[coupon_source]", "tier_config")
	s.log("intro_coupon_applying", map[string]interface{}{"level": "info", "email": email, "coupon_id": couponID, "plan_tier": plan.PlanTier, "source": "tier_config"})
	return couponID
}

// ResolvePriceID accepts either a Stripe price ID or a lookup key.
func (s *CheckoutService) ResolvePriceID(ctx context.Context, key string) (string, error) {
	trimmed := strings.TrimSpace(key)
	if trimmed == "" {
		return "", fmt.Errorf("stripe price id is required")
	}
	if strings.HasPrefix(trimmed, "price_") {
		return trimmed, nil
	}
	values := url.Values{"lookup_keys[]": {trimmed}, "limit": {"1"}, "expand[]": {"data.product"}}
	body, err := s.requester.Request(ctx, http.MethodGet, "/v1/prices?"+values.Encode(), nil, "")
	if err != nil {
		return "", err
	}
	var response struct {
		Data []struct {
			ID string `json:"id"`
		} `json:"data"`
	}
	if err := json.Unmarshal(body, &response); err != nil {
		return "", fmt.Errorf("decode stripe price lookup: %w", err)
	}
	if len(response.Data) == 0 || strings.TrimSpace(response.Data[0].ID) == "" {
		return "", fmt.Errorf("no stripe price found for lookup key %s", trimmed)
	}
	return response.Data[0].ID, nil
}

func (s *CheckoutService) record(response stripeCheckoutResponse, priceID, sessionType string, amount int64, plan *PlanOption, couponID string) error {
	metadata := map[string]interface{}{"bundle_key": plan.BundleKey, "plan_tier": plan.PlanTier, "kind": plan.Kind.String()}
	if couponID != "" {
		metadata["intro_coupon_applied"] = couponID
	}
	encoded, _ := json.Marshal(metadata)
	_, err := s.store.Exec(`INSERT INTO checkout_sessions (session_id, customer_email, customer_id, price_id, subscription_id, status, session_type, amount_cents, metadata, created_at, updated_at) VALUES ($1,$2,$3,$4,$5,$6,$7,$8,$9,NOW(),NOW()) ON CONFLICT (session_id) DO UPDATE SET customer_email = EXCLUDED.customer_email, customer_id = EXCLUDED.customer_id, price_id = EXCLUDED.price_id, subscription_id = EXCLUDED.subscription_id, status = EXCLUDED.status, session_type = EXCLUDED.session_type, amount_cents = EXCLUDED.amount_cents, metadata = EXCLUDED.metadata, updated_at = NOW()`, response.ID, response.CustomerEmail, response.Customer, priceID, response.Subscription, response.Status, sessionType, amount, string(encoded))
	return err
}

func (s *CheckoutService) log(event string, fields map[string]interface{}) {
	if s.logf != nil {
		s.logf(event, fields)
	}
}
