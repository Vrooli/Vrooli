// Package commerce defines transport-safe subscription and entitlement contracts.
package commerce

import shared "github.com/vrooli/vrooli/packages/proto/gen/go/landing-page-business-suite/v1/shared"

// EntitlementPayload is used by bundled apps to unlock features.
type EntitlementPayload struct {
	Status            string                     `json:"status"`
	PlanTier          string                     `json:"plan_tier,omitempty"`
	PriceID           string                     `json:"price_id,omitempty"`
	Features          []string                   `json:"features,omitempty"`
	BillingCycleStart int                        `json:"billing_cycle_start,omitempty"`
	Credits           *shared.CreditsBalance     `json:"credits,omitempty"`
	Subscription      *shared.SubscriptionStatus `json:"subscription,omitempty"`
}

type CreditsEnvelope struct {
	Balance                  *shared.CreditsBalance `json:"balance"`
	DisplayCreditsLabel      string                 `json:"display_credits_label"`
	DisplayCreditsMultiplier float64                `json:"display_credits_multiplier"`
}
