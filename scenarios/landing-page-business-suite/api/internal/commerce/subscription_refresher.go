package commerce

import (
	"context"
	"strings"

	shared "github.com/vrooli/vrooli/packages/proto/gen/go/landing-page-business-suite/v1/shared"
)

// SubscriptionRefresher resolves a subscription from Stripe when the local
// projection is missing or stale, then delegates durable normalization to the
// shared persistence workflow.
type SubscriptionRefresher struct {
	provider    *StripeProviderClient
	persistence *SubscriptionPersistenceService
	logf        func(string, map[string]interface{})
}

func NewSubscriptionRefresher(store StripeStore, plans *PlanService, requester StripeRequester, logf func(string, map[string]interface{})) *SubscriptionRefresher {
	return &SubscriptionRefresher{provider: NewStripeProviderClient(requester), persistence: NewSubscriptionPersistenceService(store, plans, logf), logf: logf}
}

func (s *SubscriptionRefresher) Refresh(ctx context.Context, userIdentity, currentSubscriptionID string) (*shared.SubscriptionStatus, error) {
	if currentSubscriptionID != "" {
		if sub, err := s.provider.FetchSubscription(ctx, currentSubscriptionID); err == nil {
			return s.persistence.Persist(userIdentity, sub)
		} else {
			s.log("stripe fetch subscription failed", map[string]interface{}{"level": "warn", "id": currentSubscriptionID, "error": err.Error(), "user_id": userIdentity})
		}
	}
	if strings.HasPrefix(strings.TrimSpace(userIdentity), "sub_") {
		if sub, err := s.provider.FetchSubscription(ctx, userIdentity); err == nil {
			return s.persistence.Persist(userIdentity, sub)
		}
	}
	if strings.Contains(userIdentity, "@") {
		customer, err := s.provider.FindCustomerByEmail(ctx, userIdentity)
		if err != nil {
			return nil, err
		}
		if customer == nil {
			return nil, nil
		}
		sub, err := s.provider.LatestSubscriptionForCustomer(ctx, customer.ID)
		if err != nil {
			return nil, err
		}
		return s.persistence.Persist(userIdentity, sub)
	}
	if strings.TrimSpace(userIdentity) == "" {
		return nil, nil
	}
	sub, err := s.provider.LatestSubscriptionForCustomer(ctx, userIdentity)
	if err != nil {
		return nil, err
	}
	return s.persistence.Persist(userIdentity, sub)
}

func (s *SubscriptionRefresher) log(event string, fields map[string]interface{}) {
	if s.logf != nil {
		s.logf(event, fields)
	}
}
