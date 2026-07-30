package main

import (
	"landing-page-business-suite-api/internal/commerce"

	shared "github.com/vrooli/vrooli/packages/proto/gen/go/landing-page-business-suite/v1/shared"
)

func NewAccountService(db commerce.Store, planService *commerce.PlanService) *commerce.Service {
	return newAccountService(db, planService)
}

func mapSubscriptionState(state string) shared.SubscriptionState {
	return commerce.MapSubscriptionState(state)
}

func legacyStateLabel(state shared.SubscriptionState) string {
	return commerce.SubscriptionStateLabel(state)
}
