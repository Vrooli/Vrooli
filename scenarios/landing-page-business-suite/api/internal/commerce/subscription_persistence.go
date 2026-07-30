package commerce

import (
	"database/sql"
	"fmt"
	"strings"
	"time"

	shared "github.com/vrooli/vrooli/packages/proto/gen/go/landing-page-business-suite/v1/shared"
	"google.golang.org/protobuf/proto"
	"google.golang.org/protobuf/types/known/timestamppb"
)

// SubscriptionPersistenceService translates a provider subscription into the
// durable local subscription projection used by account and webhook flows.
type SubscriptionPersistenceService struct {
	repository *StripeRepository
	plans      *PlanService
	logf       func(string, map[string]interface{})
}

func NewSubscriptionPersistenceService(store StripeStore, plans *PlanService, logf func(string, map[string]interface{})) *SubscriptionPersistenceService {
	return &SubscriptionPersistenceService{repository: NewStripeRepository(store), plans: plans, logf: logf}
}

func (s *SubscriptionPersistenceService) Persist(userHint string, sub *StripeSubscription) (*shared.SubscriptionStatus, error) {
	if sub == nil {
		return nil, nil
	}
	if s.repository == nil || s.plans == nil {
		return nil, fmt.Errorf("subscription persistence dependencies unavailable")
	}

	priceID := ""
	if len(sub.Items.Data) > 0 {
		priceID = sub.Items.Data[0].Price.ID
	}
	planTier, bundleKey := metadataString(sub.Metadata, "plan_tier"), s.plans.BundleKey()
	if value := metadataString(sub.Metadata, "bundle_key"); value != "" {
		bundleKey = value
	}
	if priceID != "" {
		if plan, err := s.plans.GetPlanByPriceID(priceID); err == nil {
			if plan.PlanTier != "" {
				planTier = plan.PlanTier
			}
			if plan.BundleKey != "" {
				bundleKey = plan.BundleKey
			}
		} else {
			s.log("stripe_plan_lookup_failed", map[string]interface{}{"level": "warn", "price_id": priceID, "error": err.Error()})
		}
	}
	if strings.TrimSpace(planTier) == "" && strings.TrimSpace(priceID) != "" {
		if inferred, ok := DetectTierToken(priceID); ok {
			planTier = inferred
		}
	}
	if strings.TrimSpace(planTier) != "" {
		if _, err := NormalizePlanTier(planTier); err != nil {
			s.log("stripe_subscription_plan_tier_invalid", map[string]interface{}{"level": "warn", "plan_tier": planTier, "price_id": priceID, "subscription": sub.ID})
			planTier = ""
		}
	}
	state := MapSubscriptionState(sub.Status)
	var canceledAt *time.Time
	if sub.CanceledAt > 0 {
		value := time.Unix(sub.CanceledAt, 0)
		canceledAt = &value
	}
	if err := s.repository.UpsertSubscription(&SubscriptionRecord{
		SubscriptionID: sub.ID, CustomerID: sub.Customer, CustomerEmail: sub.CustomerEmail,
		Status: SubscriptionStateLabel(state), PlanTier: nullableString(planTier), PriceID: nullableString(priceID), BundleKey: nullableString(bundleKey),
		BillingCycleStart: ExtractBillingCycleDay(sub.BillingCycleAnchor), CanceledAt: canceledAt,
	}); err != nil {
		return nil, err
	}

	status := &shared.SubscriptionStatus{State: state, UserIdentity: ChooseSubscriptionUserIdentity(userHint, sub), CachedAt: timestamppb.Now()}
	if sub.ID != "" {
		status.SubscriptionId = proto.String(sub.ID)
	}
	if priceID != "" {
		status.StripePriceId = proto.String(priceID)
	}
	if planTier != "" {
		status.PlanTier = proto.String(planTier)
	}
	if bundleKey != "" {
		status.BundleKey = proto.String(bundleKey)
	}
	if canceledAt != nil {
		status.CanceledAt = timestamppb.New(*canceledAt)
	}
	return status, nil
}

func metadataString(metadata map[string]interface{}, key string) string {
	value, _ := metadata[key].(string)
	return value
}

func nullableString(value string) sql.NullString {
	return sql.NullString{String: value, Valid: value != ""}
}

func (s *SubscriptionPersistenceService) log(event string, fields map[string]interface{}) {
	if s.logf != nil {
		s.logf(event, fields)
	}
}
