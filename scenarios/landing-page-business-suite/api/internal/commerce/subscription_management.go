package commerce

import (
	"context"
	"database/sql"
	"encoding/json"
	"errors"
	"fmt"
	"net/http"
	"net/url"
	"strings"
	"time"

	lpbsv1 "github.com/vrooli/vrooli/packages/proto/gen/go/landing-page-business-suite/v1"
	"google.golang.org/protobuf/proto"
	"google.golang.org/protobuf/types/known/timestamppb"
)

// SubscriptionManagementService owns provider-backed cancellation and billing
// portal creation. It deliberately updates local state only after Stripe
// confirms cancellation, so a failed provider request cannot hide a charge.
type SubscriptionManagementService struct {
	store     StripeStore
	requester StripeRequester
	provider  *StripeProviderClient
	logf      func(string, map[string]interface{})
}

func NewSubscriptionManagementService(store StripeStore, requester StripeRequester, logf func(string, map[string]interface{})) *SubscriptionManagementService {
	return &SubscriptionManagementService{store: store, requester: requester, provider: NewStripeProviderClient(requester), logf: logf}
}

func (s *SubscriptionManagementService) Cancel(ctx context.Context, userIdentity string) (*lpbsv1.CancelSubscriptionResponse, error) {
	if s.store == nil || s.requester == nil {
		return nil, errors.New("subscription management dependencies unavailable")
	}
	var subscriptionID, status string
	var customerID sql.NullString
	err := s.store.QueryRow(`
		SELECT subscription_id, status, customer_id FROM subscriptions
		WHERE (customer_email = $1 OR customer_id = $1) AND status IN ('active', 'trialing')
		ORDER BY created_at DESC LIMIT 1
	`, userIdentity).Scan(&subscriptionID, &status, &customerID)
	if err == sql.ErrNoRows {
		return nil, errors.New("no active subscription found")
	}
	if err != nil {
		return nil, err
	}
	values := url.Values{"cancel_at_period_end": {"true"}}
	if _, err := s.requester.Request(ctx, http.MethodPost, "/v1/subscriptions/"+url.PathEscape(subscriptionID), strings.NewReader(values.Encode()), "application/x-www-form-urlencoded"); err != nil {
		s.log("failed_to_cancel_subscription_on_stripe", map[string]interface{}{"id": subscriptionID, "user": userIdentity, "error": err.Error()})
		return nil, fmt.Errorf("failed to cancel subscription with Stripe: %w", err)
	}
	now := time.Now()
	if _, err := s.store.Exec(`UPDATE subscriptions SET status = $1, canceled_at = $2, updated_at = $3 WHERE subscription_id = $4`, "canceled", now, now, subscriptionID); err != nil {
		return nil, err
	}
	return &lpbsv1.CancelSubscriptionResponse{SubscriptionId: proto.String(subscriptionID), State: MapSubscriptionState("canceled"), CanceledAt: timestamppb.New(now), Message: proto.String("Subscription canceled successfully")}, nil
}

func (s *SubscriptionManagementService) CreateBillingPortalSession(ctx context.Context, userIdentity, returnURL string) (*lpbsv1.BillingPortalResponse, error) {
	if s.store == nil || s.requester == nil {
		return nil, errors.New("subscription management dependencies unavailable")
	}
	user := strings.TrimSpace(userIdentity)
	if user == "" {
		return nil, errors.New("user identity is required")
	}
	customerID := NewAccountLinkService(s.store).LookupCustomerID(user)
	if customerID == "" {
		if strings.Contains(user, "@") {
			customer, err := s.provider.FindCustomerByEmail(ctx, user)
			if err != nil {
				return nil, err
			}
			if customer != nil {
				customerID = customer.ID
			}
		} else {
			customerID = user
		}
	}
	if customerID == "" {
		return nil, errors.New("no Stripe customer found for user")
	}
	values := url.Values{"customer": {customerID}}
	if strings.TrimSpace(returnURL) != "" {
		values.Set("return_url", returnURL)
	}
	body, err := s.requester.Request(ctx, http.MethodPost, "/v1/billing_portal/sessions", strings.NewReader(values.Encode()), "application/x-www-form-urlencoded")
	if err != nil {
		return nil, err
	}
	var response struct {
		URL string `json:"url"`
	}
	if err := json.Unmarshal(body, &response); err != nil {
		return nil, err
	}
	if response.URL == "" {
		return nil, errors.New("Stripe portal URL not returned")
	}
	return &lpbsv1.BillingPortalResponse{Url: response.URL}, nil
}

func (s *SubscriptionManagementService) log(event string, fields map[string]interface{}) {
	if s.logf != nil {
		s.logf(event, fields)
	}
}
