package commerce

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"net/url"
	"strings"
	"time"
)

// ExtractBillingCycleDay converts a Unix timestamp to a safe calendar day.
func ExtractBillingCycleDay(timestamp int64) int {
	if timestamp <= 0 {
		return 0
	}
	day := time.Unix(timestamp, 0).UTC().Day()
	if day > 28 {
		return 28
	}
	return day
}

// ChooseSubscriptionUserIdentity prefers a supplied identity, then the
// provider's email, and finally its customer ID.
func ChooseSubscriptionUserIdentity(userHint string, sub *StripeSubscription) string {
	if strings.TrimSpace(userHint) != "" {
		return userHint
	}
	if sub == nil {
		return ""
	}
	if strings.TrimSpace(sub.CustomerEmail) != "" {
		return sub.CustomerEmail
	}
	return strings.TrimSpace(sub.Customer)
}

// StripeProviderClient owns Stripe's subscription and customer read model.
// It is transport-only: subscription policy and persistence stay with the
// commerce workflows that compose this client.
type StripeProviderClient struct{ requester StripeRequester }

func NewStripeProviderClient(requester StripeRequester) *StripeProviderClient {
	return &StripeProviderClient{requester: requester}
}

type StripePriceRef struct {
	ID         string `json:"id"`
	Currency   string `json:"currency"`
	UnitAmount int64  `json:"unit_amount"`
	Recurring  struct {
		Interval string `json:"interval"`
	} `json:"recurring"`
	Metadata map[string]interface{} `json:"metadata"`
}

type StripeSubscription struct {
	ID                 string `json:"id"`
	Status             string `json:"status"`
	Customer           string `json:"customer"`
	CustomerEmail      string `json:"customer_email"`
	CancelAtPeriodEnd  bool   `json:"cancel_at_period_end"`
	CanceledAt         int64  `json:"canceled_at"`
	BillingCycleAnchor int64  `json:"billing_cycle_anchor"`
	Items              struct {
		Data []struct {
			Price StripePriceRef `json:"price"`
		} `json:"data"`
	} `json:"items"`
	Metadata map[string]interface{} `json:"metadata"`
}

type StripeCustomer struct {
	ID    string `json:"id"`
	Email string `json:"email"`
}

func (c *StripeProviderClient) FetchSubscription(ctx context.Context, subscriptionID string) (*StripeSubscription, error) {
	if c.requester == nil {
		return nil, fmt.Errorf("stripe requester unavailable")
	}
	body, err := c.requester.Request(ctx, http.MethodGet, "/v1/subscriptions/"+url.PathEscape(subscriptionID), nil, "")
	if err != nil {
		return nil, err
	}
	var resp StripeSubscription
	if err := json.Unmarshal(body, &resp); err != nil {
		return nil, err
	}
	if resp.ID == "" {
		return nil, fmt.Errorf("subscription %s not found", subscriptionID)
	}
	return &resp, nil
}

func (c *StripeProviderClient) FindCustomerByEmail(ctx context.Context, email string) (*StripeCustomer, error) {
	if strings.TrimSpace(email) == "" {
		return nil, nil
	}
	if c.requester == nil {
		return nil, fmt.Errorf("stripe requester unavailable")
	}
	params := url.Values{"query": {fmt.Sprintf(`email:"%s"`, email)}, "limit": {"1"}}
	body, err := c.requester.Request(ctx, http.MethodGet, "/v1/customers/search?"+params.Encode(), nil, "")
	if err != nil {
		return nil, err
	}
	var resp struct {
		Data []StripeCustomer `json:"data"`
	}
	if err := json.Unmarshal(body, &resp); err != nil {
		return nil, err
	}
	if len(resp.Data) == 0 {
		return nil, nil
	}
	return &resp.Data[0], nil
}

func (c *StripeProviderClient) LatestSubscriptionForCustomer(ctx context.Context, customerID string) (*StripeSubscription, error) {
	if strings.TrimSpace(customerID) == "" {
		return nil, nil
	}
	if c.requester == nil {
		return nil, fmt.Errorf("stripe requester unavailable")
	}
	params := url.Values{"customer": {customerID}, "limit": {"1"}, "status": {"all"}}
	body, err := c.requester.Request(ctx, http.MethodGet, "/v1/subscriptions?"+params.Encode(), nil, "")
	if err != nil {
		return nil, err
	}
	var resp struct {
		Data []StripeSubscription `json:"data"`
	}
	if err := json.Unmarshal(body, &resp); err != nil {
		return nil, err
	}
	if len(resp.Data) == 0 {
		return nil, nil
	}
	return &resp.Data[0], nil
}
