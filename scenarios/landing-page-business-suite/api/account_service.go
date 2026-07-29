package main

import (
	"strconv"
	"strings"
	"time"

	accountdomain "landing-page-business-suite-api/internal/account"
	"landing-page-business-suite-api/internal/envx"

	commonv1 "github.com/vrooli/vrooli/packages/proto/gen/go/common/v1"
	shared "github.com/vrooli/vrooli/packages/proto/gen/go/landing-page-business-suite/v1/shared"
)

type AccountStore = accountdomain.Store
type AccountService = accountdomain.Service
type EntitlementPayload = accountdomain.EntitlementPayload
type CreditsEnvelope = accountdomain.CreditsEnvelope

func NewAccountService(db AccountStore, planService *PlanService) *AccountService {
	return accountdomain.NewService(db, planService, accountdomain.Runtime{
		CacheTTL: loadCacheTTL(), NormalizeEmail: NormalizeEmail, Log: logStructured,
	})
}

func loadCacheTTL() time.Duration {
	const defaultTTL = 60 * time.Second
	value := strings.TrimSpace(envx.Get("SUBSCRIPTION_CACHE_TTL_SECONDS"))
	if value == "" {
		return defaultTTL
	}
	seconds, err := strconv.Atoi(value)
	if err != nil || seconds <= 0 {
		return defaultTTL
	}
	return time.Duration(seconds) * time.Second
}

func mapSubscriptionState(state string) shared.SubscriptionState {
	return accountdomain.MapSubscriptionState(state)
}

func extractFeatureFlags(metadata map[string]*commonv1.JsonValue) []string {
	if metadata == nil {
		return nil
	}
	value := metadata["features"]
	if value == nil || value.Kind == nil {
		return nil
	}
	list, ok := value.Kind.(*commonv1.JsonValue_ListValue)
	if !ok || list.ListValue == nil {
		return nil
	}
	features := make([]string, 0, len(list.ListValue.Values))
	for _, item := range list.ListValue.Values {
		if stringValue, ok := item.Kind.(*commonv1.JsonValue_StringValue); ok && stringValue.StringValue != "" {
			features = append(features, stringValue.StringValue)
		}
	}
	return features
}

func legacyStateLabel(state shared.SubscriptionState) string {
	switch state {
	case shared.SubscriptionState_SUBSCRIPTION_STATE_ACTIVE:
		return "active"
	case shared.SubscriptionState_SUBSCRIPTION_STATE_TRIALING:
		return "trialing"
	case shared.SubscriptionState_SUBSCRIPTION_STATE_PAST_DUE:
		return "past_due"
	case shared.SubscriptionState_SUBSCRIPTION_STATE_CANCELED:
		return "canceled"
	default:
		return "inactive"
	}
}

var _ AccountServicer = (*AccountService)(nil)
