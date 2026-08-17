// Package monetization provides the shared client-side paid-surface
// primitives. It treats every lease as untrusted input until entitlementclient
// has verified its signature against LPBS's published JWKS.
package monetization

import (
	"context"
	"errors"
	"strings"
	"time"

	credentialclient "github.com/vrooli/vrooli/packages/credentialclient-go"
	entitlementclient "github.com/vrooli/vrooli/packages/entitlementclient-go"
)

const (
	// ReasonAllowed identifies a decision granted by a verified active lease.
	ReasonAllowed = "allowed"
	// ReasonPastDue identifies an allowed decision with a billing warning.
	ReasonPastDue = "past_due"
	// ReasonFeatureMissing identifies a verified lease without the requested feature.
	ReasonFeatureMissing = "feature_missing"
	// ReasonRankInsufficient identifies a verified lease below a required rank.
	ReasonRankInsufficient = "plan_rank_insufficient"
	// ReasonSubscriptionInactive identifies a lease that is not usable for paid access.
	ReasonSubscriptionInactive = "subscription_inactive"
	// ReasonLeaseUnavailable identifies a gate that has no current verified lease.
	ReasonLeaseUnavailable = "lease_unavailable"
	// ReasonLeaseExpired identifies a cached lease whose signed validity window ended.
	ReasonLeaseExpired = "lease_expired"
)

// Decision is the complete result of a paid-surface check. Its fields are
// derived from a verified lease or from an explicit free-tier fallback; the
// caller must not treat local decisions as an authoritative security boundary.
type Decision struct {
	Allowed     bool
	Reason      string
	Warning     bool
	UpgradePath string
	Limit       int64
	LimitFound  bool
	Lease       entitlementclient.Payload
}

// Gate owns shared lease retrieval and feature/limit decisions. The resolver
// holds only a short-lived access token in memory; the durable refresh
// credential remains in the credential authority.
type Gate struct {
	Entitlements *entitlementclient.Client
	Session      *credentialclient.ConsumerSessionResolver
	BundleKey    string
	UpgradePath  string
}

// NewGate constructs a gate that uses the supplied verified-lease client and
// shared device session. The client must point at the LPBS authority for the
// same issuer as the session resolver.
func NewGate(entitlements *entitlementclient.Client, session *credentialclient.ConsumerSessionResolver, bundleKey string) *Gate {
	return &Gate{Entitlements: entitlements, Session: session, BundleKey: strings.TrimSpace(bundleKey), UpgradePath: "/settings/subscription"}
}

// Lease resolves a lease for identity. A still-valid cached lease is returned
// during transport failure by entitlementclient; no network call is required
// to preserve a valid paid decision offline.
func (g *Gate) Lease(ctx context.Context, identity string) (entitlementclient.Payload, error) {
	if g == nil || g.Entitlements == nil {
		return entitlementclient.Payload{}, entitlementclient.ErrLeaseUnavailable
	}
	if accessToken := AccessTokenFromContext(ctx); accessToken != "" {
		return g.Entitlements.GetWithAccess(ctx, identity, accessToken)
	}
	return g.Entitlements.Get(ctx, identity)
}

// Feature evaluates a feature against the signed lease. The optional minimum
// rank is authoritative only when supplied by the scenario declaration; the
// rank value itself comes from the verified lease, never local pricing config.
func (g *Gate) Feature(ctx context.Context, identity, feature string, minPlanRank int32) Decision {
	payload, err := g.Lease(ctx, identity)
	if err != nil {
		return fallbackDecision(err, g.upgradePath())
	}
	decision := StatusDecision(payload, g.upgradePath())
	if !decision.Allowed {
		return decision
	}
	if strings.TrimSpace(feature) != "" && !HasFeature(payload, feature) {
		decision.Allowed = false
		decision.Reason = ReasonFeatureMissing
		decision.Warning = false
		return decision
	}
	if minPlanRank > 0 && !AtLeastRank(payload, minPlanRank) {
		decision.Allowed = false
		decision.Reason = ReasonRankInsufficient
		decision.Warning = false
	}
	return decision
}

// Meter evaluates a declared limit and returns the server-authoritative value.
// Class A callers must still execute the operation through LPBS; this method
// is only a local UX decision and cannot reserve or spend credits.
func (g *Gate) Meter(ctx context.Context, identity, limitKey string) Decision {
	payload, err := g.Lease(ctx, identity)
	if err != nil {
		return fallbackDecision(err, g.upgradePath())
	}
	decision := StatusDecision(payload, g.upgradePath())
	if !decision.Allowed {
		return decision
	}
	decision.Limit, decision.LimitFound = Limit(payload, limitKey, g.BundleKey)
	if !decision.LimitFound {
		decision.Allowed = false
		decision.Reason = ReasonFeatureMissing
	}
	return decision
}

// CachedFeature evaluates a feature using only the lease previously verified
// by this process. It is the explicit offline path for local, Class B work.
func (g *Gate) CachedFeature(identity, feature string, minPlanRank int32) Decision {
	return g.CachedFeatureAt(identity, feature, minPlanRank, time.Now().UTC())
}

// CachedFeatureAt is the deterministic offline feature decision boundary.
func (g *Gate) CachedFeatureAt(identity, feature string, minPlanRank int32, now time.Time) Decision {
	payload, err := g.cachedLeaseAt(identity, now)
	if err != nil {
		return fallbackDecision(err, g.upgradePath())
	}
	decision := StatusDecision(payload, g.upgradePath())
	if !decision.Allowed {
		return decision
	}
	if strings.TrimSpace(feature) != "" && !HasFeature(payload, feature) {
		decision.Allowed = false
		decision.Reason = ReasonFeatureMissing
		return decision
	}
	if minPlanRank > 0 && !AtLeastRank(payload, minPlanRank) {
		decision.Allowed = false
		decision.Reason = ReasonRankInsufficient
	}
	return decision
}

// CachedMeter evaluates a local Class B limit without touching the network.
func (g *Gate) CachedMeter(identity, limitKey string) Decision {
	return g.CachedMeterAt(identity, limitKey, time.Now().UTC())
}

// CachedMeterAt is the deterministic offline limit decision boundary.
func (g *Gate) CachedMeterAt(identity, limitKey string, now time.Time) Decision {
	payload, err := g.cachedLeaseAt(identity, now)
	if err != nil {
		return fallbackDecision(err, g.upgradePath())
	}
	decision := StatusDecision(payload, g.upgradePath())
	if !decision.Allowed {
		return decision
	}
	decision.Limit, decision.LimitFound = Limit(payload, limitKey, g.BundleKey)
	if !decision.LimitFound {
		decision.Allowed = false
		decision.Reason = ReasonFeatureMissing
	}
	return decision
}

func (g *Gate) cachedLeaseAt(identity string, now time.Time) (entitlementclient.Payload, error) {
	if g == nil || g.Entitlements == nil {
		return entitlementclient.Payload{}, entitlementclient.ErrLeaseUnavailable
	}
	return g.Entitlements.CachedAt(identity, now)
}

// StatusDecision maps the LPBS subscription status from a verified lease to a
// common decision. A past-due lease remains usable with a warning; cancellation
// and unknown statuses do not grant paid access.
func StatusDecision(payload entitlementclient.Payload, upgradePath string) Decision {
	decision := Decision{Lease: payload, UpgradePath: upgradePath}
	switch strings.ToLower(strings.TrimSpace(payload.Status)) {
	case "active", "trialing":
		decision.Allowed = true
		decision.Reason = ReasonAllowed
	case "past_due":
		decision.Allowed = true
		decision.Warning = true
		decision.Reason = ReasonPastDue
	default:
		decision.Reason = ReasonSubscriptionInactive
	}
	return decision
}

// HasFeature checks a feature in a lease that entitlementclient has already
// verified. It does not authenticate or authorize a cost-bearing operation.
func HasFeature(payload entitlementclient.Payload, feature string) bool {
	feature = strings.TrimSpace(feature)
	for _, candidate := range payload.Features {
		if candidate == feature {
			return true
		}
	}
	return false
}

// AtLeastRank compares a requested plan rank with the verified lease rank.
// Plan names are deliberately ignored so pricing changes do not require a
// client-side tier ladder.
func AtLeastRank(payload entitlementclient.Payload, rank int32) bool {
	return payload.PlanRank >= rank
}

// Limit returns a bundle-matching limit from a verified lease. A missing limit
// is returned as found=false rather than guessed from local configuration.
func Limit(payload entitlementclient.Payload, key, bundle string) (int64, bool) {
	for _, limit := range payload.Limits {
		if limit.Key == key && (limit.BundleKey == "" || limit.BundleKey == bundle) {
			return limit.Value, true
		}
	}
	return 0, false
}

func (g *Gate) upgradePath() string {
	if g == nil || strings.TrimSpace(g.UpgradePath) == "" {
		return "/settings/subscription"
	}
	return g.UpgradePath
}

func fallbackDecision(err error, upgradePath string) Decision {
	reason := ReasonLeaseUnavailable
	if errors.Is(err, entitlementclient.ErrLeaseExpired) {
		reason = ReasonLeaseExpired
	}
	return Decision{Allowed: false, Reason: reason, UpgradePath: upgradePath}
}
