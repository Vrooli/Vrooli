package main

import (
	"strconv"
	"strings"
	"time"

	"landing-page-business-suite-api/internal/commerce"
	"landing-page-business-suite-api/internal/envx"

	shared "github.com/vrooli/vrooli/packages/proto/gen/go/landing-page-business-suite/v1/shared"
)

type (
	AccountStore       = commerce.Store
	AccountService     = commerce.Service
	EntitlementPayload = commerce.EntitlementPayload
	CreditsEnvelope    = commerce.CreditsEnvelope
)

func NewAccountService(db AccountStore, planService *PlanService) *AccountService {
	return commerce.NewService(db, planService, commerce.Runtime{
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
	return commerce.MapSubscriptionState(state)
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
