package readiness

import (
	"fmt"
	"strings"
	"time"
)

type TriggerKind string

const (
	TriggerMajorVersion       TriggerKind = "major_version_change"
	TriggerMonetizationChange TriggerKind = "monetization_changed"
	TriggerBrandVersion       TriggerKind = "brand_version_bump"
	TriggerNewRamp            TriggerKind = "new_ramp"
	TriggerListingEdit        TriggerKind = "storefront_listing_edited"
	TriggerPriceChange        TriggerKind = "price_changed"
	TriggerElapsedApproval    TriggerKind = "approval_elapsed"
)

type TriggerInput struct {
	Kind             TriggerKind
	PreviousValue    string
	CurrentValue     string
	LastApprovedAt   time.Time
	ApprovalLifetime time.Duration
}

func (t TriggerInput) Fired(now time.Time) bool {
	if strings.TrimSpace(t.PreviousValue) != strings.TrimSpace(t.CurrentValue) {
		return true
	}
	return t.Kind == TriggerElapsedApproval && !t.LastApprovedAt.IsZero() && t.ApprovalLifetime > 0 && !now.Before(t.LastApprovedAt.Add(t.ApprovalLifetime))
}

type Waiver struct {
	Reason string    `json:"reason"`
	Actor  string    `json:"actor"`
	Commit string    `json:"commit"`
	At     time.Time `json:"at"`
}

func (w Waiver) Validate() error {
	if strings.TrimSpace(w.Reason) == "" {
		return fmt.Errorf("waiver reason is required")
	}
	if strings.TrimSpace(w.Actor) == "" || strings.TrimSpace(w.Commit) == "" {
		return fmt.Errorf("waiver actor and commit are required")
	}
	if w.At.IsZero() {
		return fmt.Errorf("waiver timestamp is required")
	}
	return nil
}

type Acceptance struct {
	Reason           string    `json:"reason"`
	Actor            string    `json:"actor"`
	Commit           string    `json:"commit"`
	AcceptedAt       time.Time `json:"accepted_at"`
	ExpiresOnTrigger bool      `json:"expires_on_trigger"`
}

func (a Acceptance) Expired(triggerFired bool) bool { return a.ExpiresOnTrigger && triggerFired }
