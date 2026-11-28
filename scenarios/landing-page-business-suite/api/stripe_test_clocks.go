package main

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"net/http"
	"net/url"
	"regexp"
	"strings"
	"time"

	"landing-page-business-suite-api/internal/envx"
)

// stripeTestClock is the minimized Stripe Test Clock state needed by PY1.
// Customer and subscription data stay on Stripe; the replay retains only IDs
// and clock status in its evidence.
type stripeTestClock struct {
	ID       string `json:"id"`
	FrozenAt int64  `json:"frozen_time"`
	Status   string `json:"status"`
	Livemode bool   `json:"livemode"`
}

type stripeTestClockSubscription struct {
	CustomerID         string `json:"customer_id"`
	PaymentMethodID    string `json:"payment_method_id"`
	SubscriptionID     string `json:"subscription_id"`
	SubscriptionItemID string `json:"subscription_item_id,omitempty"`
	CurrentPeriodEnd   int64  `json:"current_period_end,omitempty"`
	SubscriptionState  string `json:"subscription_status"`
	CancelAtPeriodEnd  bool   `json:"cancel_at_period_end,omitempty"`
}

var py1StripeRunIDPattern = regexp.MustCompile(`^py1-[A-Za-z0-9_-]{1,96}$`)
var stripeTestClockIDPattern = regexp.MustCompile(`^clock_[A-Za-z0-9]+$`)

func (s *StripeService) createPY1TestClock(ctx context.Context, runID string, frozenAt int64) (*stripeTestClock, error) {
	if err := s.requirePY1StripeTestTarget(runID); err != nil {
		return nil, err
	}
	if frozenAt <= 0 {
		return nil, errors.New("Stripe Test Clock requires a positive frozen timestamp")
	}
	form := url.Values{}
	form.Set("frozen_time", fmt.Sprintf("%d", frozenAt))
	form.Set("name", "PY1 "+runID)
	headers := http.Header{}
	headers.Set("Idempotency-Key", runID+"-test-clock-create")
	body, err := s.doStripeRequestWithHeaders(ctx, http.MethodPost, "/v1/test_helpers/test_clocks", strings.NewReader(form.Encode()), "application/x-www-form-urlencoded", headers)
	if err != nil {
		return nil, errors.New("Stripe test clock creation failed")
	}
	var clock stripeTestClock
	if err := json.Unmarshal(body, &clock); err != nil || !stripeTestClockIDPattern.MatchString(clock.ID) || clock.Livemode || clock.FrozenAt != frozenAt {
		return nil, errors.New("Stripe returned an invalid test clock")
	}
	return &clock, nil
}

func (s *StripeService) advancePY1TestClock(ctx context.Context, runID, clockID string, frozenAt int64) (*stripeTestClock, error) {
	if err := s.requirePY1StripeTestTarget(runID); err != nil {
		return nil, err
	}
	clockID = strings.TrimSpace(clockID)
	if !stripeTestClockIDPattern.MatchString(clockID) || frozenAt <= 0 {
		return nil, errors.New("invalid Stripe Test Clock advance request")
	}
	form := url.Values{}
	form.Set("frozen_time", fmt.Sprintf("%d", frozenAt))
	headers := http.Header{}
	headers.Set("Idempotency-Key", runID+"-clock-advance-"+fmt.Sprintf("%d", frozenAt))
	body, err := s.doStripeRequestWithHeaders(ctx, http.MethodPost, "/v1/test_helpers/test_clocks/"+url.PathEscape(clockID)+"/advance", strings.NewReader(form.Encode()), "application/x-www-form-urlencoded", headers)
	if err != nil {
		return nil, errors.New("Stripe test clock advance failed")
	}
	var clock stripeTestClock
	if err := json.Unmarshal(body, &clock); err != nil || clock.ID != clockID || clock.Livemode || clock.FrozenAt != frozenAt {
		return nil, errors.New("Stripe returned an invalid advanced test clock")
	}
	return &clock, nil
}

func (s *StripeService) waitForPY1TestClock(ctx context.Context, runID, clockID string, interval time.Duration) (*stripeTestClock, error) {
	if err := s.requirePY1StripeTestTarget(runID); err != nil {
		return nil, err
	}
	clockID = strings.TrimSpace(clockID)
	if !stripeTestClockIDPattern.MatchString(clockID) {
		return nil, errors.New("invalid Stripe Test Clock ID")
	}
	if interval <= 0 {
		interval = time.Second
	}
	ticker := time.NewTicker(interval)
	defer ticker.Stop()
	for {
		body, err := s.doStripeRequest(ctx, http.MethodGet, "/v1/test_helpers/test_clocks/"+url.PathEscape(clockID), nil, "")
		if err != nil {
			return nil, errors.New("Stripe test clock status lookup failed")
		}
		var clock stripeTestClock
		if err := json.Unmarshal(body, &clock); err != nil || clock.ID != clockID || clock.Livemode {
			return nil, errors.New("Stripe returned an invalid test clock status")
		}
		switch clock.Status {
		case "ready":
			return &clock, nil
		case "advancing":
		case "internal_failure":
			return nil, errors.New("Stripe test clock entered internal_failure")
		default:
			return nil, errors.New("Stripe test clock returned an unknown status")
		}
		select {
		case <-ticker.C:
		case <-ctx.Done():
			return nil, ctx.Err()
		}
	}
}

// createPY1TestClockSubscription creates an isolated customer on the clock,
// attaches a Stripe test Visa, and creates the subscription through Stripe's
// real API. The customer and subscription carry the correlation metadata used
// by the signed webhook receipt path.
func (s *StripeService) createPY1TestClockSubscription(ctx context.Context, runID, clockID, priceID, outcome string) (*stripeTestClockSubscription, error) {
	if err := s.requirePY1StripeTestTarget(runID); err != nil {
		return nil, err
	}
	if !stripeTestClockIDPattern.MatchString(strings.TrimSpace(clockID)) || !regexp.MustCompile(`^price_[A-Za-z0-9]+$`).MatchString(strings.TrimSpace(priceID)) || !py1StripeOutcomeNameAllowed(outcome) {
		return nil, errors.New("invalid PY1 test clock subscription request")
	}
	values := url.Values{}
	values.Set("email", runID+"@example.test")
	values.Set("test_clock", clockID)
	values.Set("metadata[lpbs_py1_test_run_id]", runID)
	values.Set("metadata[lpbs_py1_test_outcome]", outcome)
	customerBytes, err := s.testClockPost(ctx, runID+"-"+outcome+"-customer-create", "/v1/customers", values)
	if err != nil {
		return nil, errors.New("Stripe test clock customer creation failed")
	}
	var customer struct {
		ID string `json:"id"`
	}
	if json.Unmarshal(customerBytes, &customer) != nil || !strings.HasPrefix(customer.ID, "cus_") {
		return nil, errors.New("Stripe returned an invalid test clock customer")
	}

	card := url.Values{}
	card.Set("type", "card")
	card.Set("card[number]", "4242424242424242")
	card.Set("card[exp_month]", "12")
	card.Set("card[exp_year]", "2035")
	card.Set("card[cvc]", "123")
	paymentBytes, err := s.testClockPost(ctx, runID+"-"+outcome+"-payment-method-create", "/v1/payment_methods", card)
	if err != nil {
		return nil, errors.New("Stripe test payment method creation failed")
	}
	var payment struct {
		ID string `json:"id"`
	}
	if json.Unmarshal(paymentBytes, &payment) != nil || !strings.HasPrefix(payment.ID, "pm_") {
		return nil, errors.New("Stripe returned an invalid test payment method")
	}
	attach := url.Values{}
	attach.Set("customer", customer.ID)
	if _, err := s.testClockPost(ctx, runID+"-"+outcome+"-payment-method-attach", "/v1/payment_methods/"+url.PathEscape(payment.ID)+"/attach", attach); err != nil {
		return nil, errors.New("Stripe test payment method attachment failed")
	}

	subscription := url.Values{}
	subscription.Set("customer", customer.ID)
	subscription.Set("items[0][price]", priceID)
	subscription.Set("default_payment_method", payment.ID)
	subscription.Set("payment_behavior", "error_if_incomplete")
	subscription.Set("metadata[lpbs_py1_test_run_id]", runID)
	subscription.Set("metadata[lpbs_py1_test_outcome]", outcome)
	subscriptionBytes, err := s.testClockPost(ctx, runID+"-"+outcome+"-subscription-create", "/v1/subscriptions", subscription)
	if err != nil {
		return nil, errors.New("Stripe test clock subscription creation failed")
	}
	var result struct {
		ID               string `json:"id"`
		Status           string `json:"status"`
		CurrentPeriodEnd int64  `json:"current_period_end"`
		Items            struct {
			Data []struct {
				ID string `json:"id"`
			} `json:"data"`
		} `json:"items"`
	}
	if json.Unmarshal(subscriptionBytes, &result) != nil || !strings.HasPrefix(result.ID, "sub_") || result.Status != "active" || result.CurrentPeriodEnd <= 0 || len(result.Items.Data) != 1 || !strings.HasPrefix(result.Items.Data[0].ID, "si_") {
		return nil, errors.New("Stripe returned a non-active test clock subscription")
	}
	return &stripeTestClockSubscription{CustomerID: customer.ID, PaymentMethodID: payment.ID, SubscriptionID: result.ID, SubscriptionItemID: result.Items.Data[0].ID, CurrentPeriodEnd: result.CurrentPeriodEnd, SubscriptionState: result.Status}, nil
}

func (s *StripeService) setPY1TestClockDefaultPaymentMethod(ctx context.Context, runID, subscriptionID, paymentMethodID, outcome string) error {
	if err := s.requirePY1StripeTestTarget(runID); err != nil {
		return err
	}
	if !strings.HasPrefix(strings.TrimSpace(subscriptionID), "sub_") || !strings.HasPrefix(strings.TrimSpace(paymentMethodID), "pm_") || (outcome != "renewal" && outcome != "past_due") {
		return errors.New("invalid PY1 test default payment method update")
	}
	values := url.Values{}
	values.Set("default_payment_method", paymentMethodID)
	values.Set("metadata[lpbs_py1_test_run_id]", runID)
	values.Set("metadata[lpbs_py1_test_outcome]", outcome)
	body, err := s.testClockPost(ctx, runID+"-default-payment-"+outcome, "/v1/subscriptions/"+url.PathEscape(subscriptionID), values)
	if err != nil {
		return errors.New("Stripe test default payment method update failed")
	}
	var result struct {
		ID                   string `json:"id"`
		DefaultPaymentMethod string `json:"default_payment_method"`
	}
	if json.Unmarshal(body, &result) != nil || result.ID != subscriptionID || result.DefaultPaymentMethod != paymentMethodID {
		return errors.New("Stripe returned an invalid default payment method update")
	}
	return nil
}

// createPY1DecliningRenewalPaymentMethod provisions Stripe's documented test
// card for a failed renewal. It is reachable only for the past_due test outcome.
func (s *StripeService) createPY1DecliningRenewalPaymentMethod(ctx context.Context, runID, customerID string) (string, error) {
	if err := s.requirePY1StripeTestTarget(runID); err != nil {
		return "", err
	}
	if !strings.HasPrefix(strings.TrimSpace(customerID), "cus_") {
		return "", errors.New("invalid PY1 test customer ID")
	}
	card := url.Values{}
	card.Set("type", "card")
	card.Set("card[number]", "4000000000000341")
	card.Set("card[exp_month]", "12")
	card.Set("card[exp_year]", "2035")
	card.Set("card[cvc]", "123")
	body, err := s.testClockPost(ctx, runID+"-past-due-failure-payment-method-create", "/v1/payment_methods", card)
	if err != nil {
		return "", errors.New("Stripe test decline payment method creation failed")
	}
	var payment struct {
		ID string `json:"id"`
	}
	if json.Unmarshal(body, &payment) != nil || !strings.HasPrefix(payment.ID, "pm_") {
		return "", errors.New("Stripe returned an invalid test decline payment method")
	}
	attach := url.Values{}
	attach.Set("customer", customerID)
	attached, err := s.testClockPost(ctx, runID+"-past-due-failure-payment-method-attach", "/v1/payment_methods/"+url.PathEscape(payment.ID)+"/attach", attach)
	if err != nil {
		return "", errors.New("Stripe test decline payment method attachment failed")
	}
	var result struct {
		ID       string `json:"id"`
		Customer string `json:"customer"`
	}
	if json.Unmarshal(attached, &result) != nil || result.ID != payment.ID || result.Customer != customerID {
		return "", errors.New("Stripe returned an invalid attached test decline payment method")
	}
	return payment.ID, nil
}

// schedulePY1TestClockCancellation schedules cancellation at the subscription
// boundary. Advancing the clock must still produce the signed deletion event;
// this method does not manufacture a cancellation receipt.
func (s *StripeService) schedulePY1TestClockCancellation(ctx context.Context, runID, subscriptionID string) (*stripeTestClockSubscription, error) {
	if err := s.requirePY1StripeTestTarget(runID); err != nil {
		return nil, err
	}
	subscriptionID = strings.TrimSpace(subscriptionID)
	if !strings.HasPrefix(subscriptionID, "sub_") {
		return nil, errors.New("invalid PY1 test subscription ID")
	}
	values := url.Values{}
	values.Set("cancel_at_period_end", "true")
	values.Set("metadata[lpbs_py1_test_run_id]", runID)
	values.Set("metadata[lpbs_py1_test_outcome]", "cancel")
	body, err := s.testClockPost(ctx, runID+"-schedule-cancel", "/v1/subscriptions/"+url.PathEscape(subscriptionID), values)
	if err != nil {
		return nil, errors.New("Stripe test subscription cancellation scheduling failed")
	}
	var result struct {
		ID                string `json:"id"`
		Status            string `json:"status"`
		CancelAtPeriodEnd bool   `json:"cancel_at_period_end"`
	}
	if json.Unmarshal(body, &result) != nil || result.ID != subscriptionID || result.Status != "active" || !result.CancelAtPeriodEnd {
		return nil, errors.New("Stripe returned an invalid scheduled test cancellation")
	}
	return &stripeTestClockSubscription{SubscriptionID: result.ID, SubscriptionState: result.Status, CancelAtPeriodEnd: true}, nil
}

// changePY1TestClockSubscriptionPrice updates one existing test subscription
// item and retains Stripe's invoice/proration behavior. Its invoice webhook,
// not this API response, is the durable billing result.
func (s *StripeService) changePY1TestClockSubscriptionPrice(ctx context.Context, runID, subscriptionID, itemID, priceID, outcome string) (*stripeTestClockSubscription, error) {
	if err := s.requirePY1StripeTestTarget(runID); err != nil {
		return nil, err
	}
	if !strings.HasPrefix(strings.TrimSpace(subscriptionID), "sub_") || !strings.HasPrefix(strings.TrimSpace(itemID), "si_") || !regexp.MustCompile(`^price_[A-Za-z0-9]+$`).MatchString(strings.TrimSpace(priceID)) || (outcome != "upgrade" && outcome != "downgrade") {
		return nil, errors.New("invalid PY1 test subscription price change")
	}
	values := url.Values{}
	values.Set("items[0][id]", itemID)
	values.Set("items[0][price]", priceID)
	values.Set("proration_behavior", "always_invoice")
	values.Set("metadata[lpbs_py1_test_run_id]", runID)
	values.Set("metadata[lpbs_py1_test_outcome]", outcome)
	body, err := s.testClockPost(ctx, runID+"-"+outcome, "/v1/subscriptions/"+url.PathEscape(subscriptionID), values)
	if err != nil {
		return nil, errors.New("Stripe test subscription price change failed")
	}
	var result struct {
		ID     string `json:"id"`
		Status string `json:"status"`
	}
	if json.Unmarshal(body, &result) != nil || result.ID != subscriptionID || result.Status != "active" {
		return nil, errors.New("Stripe returned an invalid updated test subscription")
	}
	return &stripeTestClockSubscription{SubscriptionID: result.ID, SubscriptionState: result.Status}, nil
}

func py1StripeOutcomeNameAllowed(name string) bool {
	for _, allowed := range stripeOutcomeNames {
		if name == allowed {
			return true
		}
	}
	return false
}

func (s *StripeService) testClockPost(ctx context.Context, runID, path string, values url.Values) ([]byte, error) {
	headers := http.Header{}
	headers.Set("Idempotency-Key", runID)
	return s.doStripeRequestWithHeaders(ctx, http.MethodPost, path, strings.NewReader(values.Encode()), "application/x-www-form-urlencoded", headers)
}

func (s *StripeService) requirePY1StripeTestTarget(runID string) error {
	if s == nil || !stripeOutcomeEnvironmentAllowed(runtimeEnvironment()) || !strings.EqualFold(strings.TrimSpace(envx.Get("STRIPE_MODE")), "test") {
		return errors.New("Stripe Test Clock operations require an explicit Stripe test target")
	}
	if !py1StripeRunIDPattern.MatchString(strings.TrimSpace(runID)) {
		return errors.New("Stripe Test Clock operations require a PY1 run ID")
	}
	config := s.getConfig()
	if !config.hasSecret || (!strings.HasPrefix(config.secretKey, "sk_test_") && !strings.HasPrefix(config.secretKey, "rk_test_")) {
		return errors.New("Stripe Test Clock operations require an authority-resolved test key")
	}
	return nil
}
