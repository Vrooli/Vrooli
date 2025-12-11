// Package entitlement provides subscription verification and feature gating.
//
// This package connects to the landing-page-business-suite entitlement service
// to verify user subscriptions and enforce tier-based limits on features like:
// - Workflow execution counts
// - Export watermarking
// - AI-powered features
// - Live recording
//
// # Architecture
//
// The entitlement system follows a cache-first approach:
// 1. Check local cache for valid entitlement
// 2. If expired or missing, fetch from remote service
// 3. Cache response for configured TTL
// 4. Fall back to default tier if service unavailable
//
// # Usage
//
//	svc := entitlement.NewService(cfg, log)
//	ent, err := svc.GetEntitlement(ctx, "user@example.com")
//	if svc.CanExecuteWorkflow(ctx, "user@example.com") {
//	    // proceed with execution
//	}
package entitlement

import (
	"time"
)

// Tier represents a subscription tier with its capabilities.
type Tier string

const (
	TierFree     Tier = "free"
	TierSolo     Tier = "solo"
	TierPro      Tier = "pro"
	TierStudio   Tier = "studio"
	TierBusiness Tier = "business"
)

// Status represents the subscription status.
type Status string

const (
	StatusActive   Status = "active"
	StatusTrialing Status = "trialing"
	StatusPastDue  Status = "past_due"
	StatusCanceled Status = "canceled"
	StatusInactive Status = "inactive"
)

// Entitlement represents a user's current subscription and capabilities.
type Entitlement struct {
	// UserIdentity is the email or customer ID used to look up entitlements.
	UserIdentity string `json:"user_identity"`

	// Status is the subscription status (active, trialing, inactive, etc.).
	Status Status `json:"status"`

	// Tier is the subscription tier (free, solo, pro, studio, business).
	Tier Tier `json:"tier"`

	// PriceID is the Stripe price ID if subscribed.
	PriceID string `json:"price_id,omitempty"`

	// Features is a list of feature flags enabled for this subscription.
	Features []string `json:"features,omitempty"`

	// Credits is the user's credit balance (for future use).
	Credits int64 `json:"credits,omitempty"`

	// FetchedAt is when this entitlement was fetched from the service.
	FetchedAt time.Time `json:"fetched_at"`

	// ExpiresAt is when this cached entitlement expires.
	ExpiresAt time.Time `json:"expires_at"`
}

// IsActive returns true if the subscription is in an active state.
func (e *Entitlement) IsActive() bool {
	return e.Status == StatusActive || e.Status == StatusTrialing
}

// IsExpired returns true if this cached entitlement has expired.
func (e *Entitlement) IsExpired() bool {
	return time.Now().After(e.ExpiresAt)
}

// HasFeature checks if a specific feature flag is enabled.
func (e *Entitlement) HasFeature(feature string) bool {
	for _, f := range e.Features {
		if f == feature {
			return true
		}
	}
	return false
}

// TierOrder returns a numeric order for tier comparison.
// Higher is better.
func (t Tier) Order() int {
	switch t {
	case TierBusiness:
		return 5
	case TierStudio:
		return 4
	case TierPro:
		return 3
	case TierSolo:
		return 2
	case TierFree:
		return 1
	default:
		return 0
	}
}

// AtLeast returns true if this tier is at least as high as the given tier.
func (t Tier) AtLeast(other Tier) bool {
	return t.Order() >= other.Order()
}

// entitlementResponse matches the response from landing-page-business-suite /api/v1/entitlements.
type entitlementResponse struct {
	Status       string        `json:"status"`
	PlanTier     string        `json:"plan_tier"`
	PriceID      string        `json:"price_id"`
	Features     []string      `json:"features"`
	Credits      *credits      `json:"credits"`
	Subscription *subscription `json:"subscription"`
}

type credits struct {
	BalanceCredits int64 `json:"balance_credits"`
}

type subscription struct {
	State         string `json:"state"`
	PlanTier      string `json:"plan_tier"`
	StripePriceID string `json:"stripe_price_id"`
}
