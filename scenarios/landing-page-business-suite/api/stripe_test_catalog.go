package main

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"net/url"
	"strings"

	"landing-page-business-suite-api/internal/envx"
)

// createStripeTestPrice is deliberately narrower than the general Stripe
// surface. It only creates an active, fixed-amount test price for an existing
// product, and requires both test mode and a stable idempotency key.
func (s *StripeService) createStripeTestPrice(ctx context.Context, productID, label string, amountCents int64, interval, idempotencyKey string) (string, error) {
	config := s.getConfig()
	if strings.ToLower(strings.TrimSpace(envx.Get("STRIPE_MODE"))) != "test" ||
		(!strings.HasPrefix(config.secretKey, "sk_test_") && !strings.HasPrefix(config.secretKey, "rk_test_")) {
		return "", fmt.Errorf("test catalog price creation requires a Stripe test-mode key")
	}
	if strings.TrimSpace(productID) == "" || strings.TrimSpace(label) == "" || amountCents <= 0 || strings.TrimSpace(idempotencyKey) == "" {
		return "", fmt.Errorf("product, label, positive amount, and idempotency key are required")
	}
	values := url.Values{
		"product":     {strings.TrimSpace(productID)},
		"currency":    {"usd"},
		"unit_amount": {fmt.Sprintf("%d", amountCents)},
		"nickname":    {strings.TrimSpace(label)},
	}
	if interval != "" {
		if interval != "month" && interval != "year" {
			return "", fmt.Errorf("unsupported recurring interval")
		}
		values.Set("recurring[interval]", interval)
	}
	headers := make(http.Header)
	headers.Set("Idempotency-Key", idempotencyKey)
	data, err := s.doStripeRequestWithHeaders(ctx, http.MethodPost, "/v1/prices", strings.NewReader(values.Encode()), "application/x-www-form-urlencoded", headers)
	if err != nil {
		return "", err
	}
	var response struct {
		ID       string `json:"id"`
		Active   bool   `json:"active"`
		Currency string `json:"currency"`
		Amount   int64  `json:"unit_amount"`
	}
	if err := json.Unmarshal(data, &response); err != nil {
		return "", fmt.Errorf("decode Stripe test price response")
	}
	if !strings.HasPrefix(response.ID, "price_") || !response.Active || response.Currency != "usd" || response.Amount != amountCents {
		return "", fmt.Errorf("Stripe returned an unexpected test price shape")
	}
	return response.ID, nil
}
