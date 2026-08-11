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
	"strings"
	"time"
)

const (
	OverrideTierSettingKey = "entitlement_override_tier"
	ApiSourceSettingKey    = "entitlement_api_source"
	LocalApiPortSettingKey = "entitlement_local_api_port"
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

// Feature constants for type-safe feature checks.
// These are the canonical feature strings that can appear in the Features array.
const (
	FeatureAI            = "ai"
	FeatureRecording     = "recording"
	FeatureWatermarkFree = "watermark_free"
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

	// BillingCycleStart is the day of month (1-28) when the billing cycle resets.
	// 0 means use calendar month (1st of each month).
	BillingCycleStart int `json:"billing_cycle_start,omitempty"`

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

// GetBillingPeriod returns start/end for the billing period containing t.
// Falls back to calendar month if BillingCycleStart is 0 or invalid.
func (e *Entitlement) GetBillingPeriod(t time.Time) (start, end time.Time) {
	day := e.BillingCycleStart
	if day < 1 || day > 28 {
		// Calendar month fallback
		year, month, _ := t.Date()
		start = time.Date(year, month, 1, 0, 0, 0, 0, t.Location())
		end = start.AddDate(0, 1, 0).Add(-time.Nanosecond)
		return
	}

	year, month, currentDay := t.Date()
	loc := t.Location()

	if currentDay >= day {
		// Period started this month
		start = time.Date(year, month, day, 0, 0, 0, 0, loc)
		end = time.Date(year, month+1, day, 0, 0, 0, 0, loc).Add(-time.Nanosecond)
	} else {
		// Period started last month
		start = time.Date(year, month-1, day, 0, 0, 0, 0, loc)
		end = time.Date(year, month, day, 0, 0, 0, 0, loc).Add(-time.Nanosecond)
	}
	return
}

// GetBillingMonth returns billing period identifier for DB queries (e.g., "2026-01-15").
func (e *Entitlement) GetBillingMonth(t time.Time) string {
	start, _ := e.GetBillingPeriod(t)
	return start.Format("2006-01-02")
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

// ParseTier normalizes a tier string into a Tier enum.
func ParseTier(value string) (Tier, bool) {
	normalized := strings.TrimSpace(strings.ToLower(value))
	switch normalized {
	case string(TierFree):
		return TierFree, true
	case string(TierSolo):
		return TierSolo, true
	case string(TierPro):
		return TierPro, true
	case string(TierStudio):
		return TierStudio, true
	case string(TierBusiness):
		return TierBusiness, true
	default:
		return "", false
	}
}

// entitlementResponse matches the response from landing-page-business-suite /api/v1/entitlements.
type entitlementResponse struct {
	Status            string                `json:"status"`
	PlanTier          string                `json:"plan_tier"`
	PriceID           string                `json:"price_id"`
	Features          []string              `json:"features"`
	BillingCycleStart int                   `json:"billing_cycle_start"`
	Credits           *credits              `json:"credits"`
	Subscription      *subscriptionIdentity `json:"subscription"`
}

type credits struct {
	CustomerEmail  string `json:"customer_email"`
	BalanceCredits int64  `json:"balance_credits"`
}

type subscriptionIdentity struct {
	UserIdentity string `json:"user_identity"`
}
