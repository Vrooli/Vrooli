// Package commerce defines transport-safe subscription and entitlement contracts.
package commerce

import (
	"time"

	entitlementclient "github.com/vrooli/vrooli/packages/entitlementclient-go"
)

import shared "github.com/vrooli/vrooli/packages/proto/gen/go/landing-page-business-suite/v1/shared"

// EntitlementPayload is used by bundled apps to unlock features.
type EntitlementPayload struct {
	UserIdentity      string                     `json:"user_identity,omitempty"`
	Status            string                     `json:"status"`
	PlanTier          string                     `json:"plan_tier,omitempty"`
	PlanRank          int32                      `json:"plan_rank,omitempty"`
	PriceID           string                     `json:"price_id,omitempty"`
	Features          []string                   `json:"features,omitempty"`
	Limits            []entitlementclient.Limit  `json:"limits,omitempty"`
	NotAfter          time.Time                  `json:"not_after,omitempty"`
	Lease             string                     `json:"lease,omitempty"`
	BillingCycleStart int                        `json:"billing_cycle_start,omitempty"`
	Credits           *shared.CreditsBalance     `json:"credits,omitempty"`
	Subscription      *shared.SubscriptionStatus `json:"subscription,omitempty"`
}

type CreditsEnvelope struct {
	Balance                  *shared.CreditsBalance `json:"balance"`
	DisplayCreditsLabel      string                 `json:"display_credits_label"`
	DisplayCreditsMultiplier float64                `json:"display_credits_multiplier"`
}
