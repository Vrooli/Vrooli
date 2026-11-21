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
	// ReasonMeterUndeclared identifies a meter absent from the committed inventory or app declaration.
	ReasonMeterUndeclared = "meter_undeclared"
	// ReasonMeterRequiresAuthority identifies a Class A meter queried through an offline-only path.
	ReasonMeterRequiresAuthority = "meter_requires_authority"
	// ReasonMeterInventoryUnavailable identifies invalid embedded declaration data.
	ReasonMeterInventoryUnavailable = "meter_inventory_unavailable"
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
	AppKey       string
	UpgradePath  string
}

// NewGate constructs a gate that uses the supplied verified-lease client and
// shared device session. The client must point at the LPBS authority for the
// same issuer as the session resolver.
func NewGate(entitlements *entitlementclient.Client, session *credentialclient.ConsumerSessionResolver, bundleKey string) *Gate {
	return &Gate{Entitlements: entitlements, Session: session, BundleKey: strings.TrimSpace(bundleKey), UpgradePath: "/settings/subscription"}
}

// NewGateForApp constructs a gate bound to a declared app key. Meter checks
// then require the meter to be present in the generated inventory for this app
// and bundle.
func NewGateForApp(entitlements *entitlementclient.Client, session *credentialclient.ConsumerSessionResolver, bundleKey, appKey string) *Gate {
	gate := NewGate(entitlements, session, bundleKey)
	gate.AppKey = strings.TrimSpace(appKey)
	return gate
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
	return FeatureDecisionForApp(payload, g.appKey(), feature, minPlanRank, decision)
}

// Meter evaluates a declared limit and returns the server-authoritative value.
// Class A callers must still execute the operation through LPBS; this method
// is only a local UX decision and cannot reserve or spend credits.
func (g *Gate) Meter(ctx context.Context, identity, limitKey string) Decision {
	if decision, denied := g.meterDeclaration(limitKey, false); denied {
		return decision
	}
	payload, err := g.Lease(ctx, identity)
	if err != nil {
		return fallbackDecision(err, g.upgradePath())
	}
	decision := StatusDecisionForApp(payload, g.appKey(), g.upgradePath())
	if !decision.Allowed {
		return decision
	}
	decision.Limit, decision.LimitFound = LimitForApp(payload, g.appKey(), limitKey, g.BundleKey)
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
	return FeatureDecisionForApp(payload, g.appKey(), feature, minPlanRank, decision)
}

// CachedMeter evaluates a local Class B limit without touching the network.
func (g *Gate) CachedMeter(identity, limitKey string) Decision {
	return g.CachedMeterAt(identity, limitKey, time.Now().UTC())
}

// CachedMeterAt is the deterministic offline limit decision boundary.
func (g *Gate) CachedMeterAt(identity, limitKey string, now time.Time) Decision {
	if decision, denied := g.meterDeclaration(limitKey, true); denied {
		return decision
	}
	payload, err := g.cachedLeaseAt(identity, now)
	if err != nil {
		return fallbackDecision(err, g.upgradePath())
	}
	decision := StatusDecisionForApp(payload, g.appKey(), g.upgradePath())
	if !decision.Allowed {
		return decision
	}
	decision.Limit, decision.LimitFound = LimitForApp(payload, g.appKey(), limitKey, g.BundleKey)
	if !decision.LimitFound {
		decision.Allowed = false
		decision.Reason = ReasonFeatureMissing
	}
	return decision
}

func (g *Gate) meterDeclaration(limitKey string, offline bool) (Decision, bool) {
	upgradePath := g.upgradePath()
	if g == nil {
		return fallbackDecision(entitlementclient.ErrLeaseUnavailable, upgradePath), true
	}
	meter, ok, err := findCommittedMeter(limitKey, g.BundleKey, g.AppKey)
	if err != nil {
		return Decision{Reason: ReasonMeterInventoryUnavailable, UpgradePath: upgradePath}, true
	}
	if !ok {
		return Decision{Reason: ReasonMeterUndeclared, UpgradePath: upgradePath}, true
	}
	if offline && meter.Class != "B" {
		return Decision{Reason: ReasonMeterRequiresAuthority, UpgradePath: upgradePath}, true
	}
	return Decision{}, false
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

// FeatureDecision applies feature and rank requirements to an already
// verified lease decision. Callers that already have a lease (for example an
// inbound request middleware) can use this without performing a second
// authority lookup.
func FeatureDecision(payload entitlementclient.Payload, feature string, minPlanRank int32, decision Decision) Decision {
	if len(payload.AppEntitlements) > 0 {
		decision.Allowed = false
		decision.Reason = ReasonFeatureMissing
		decision.Warning = false
		return decision
	}
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

// FeatureDecisionForApp applies the signed effective grant for exactly one
// app. Once app-scoped records are present, legacy global features and rank
// are ignored so a single-app offer cannot unlock sibling apps.
func FeatureDecisionForApp(payload entitlementclient.Payload, appKey, feature string, minPlanRank int32, decision Decision) Decision {
	grant, scoped := appEntitlement(payload, appKey)
	if !scoped {
		return FeatureDecision(payload, feature, minPlanRank, decision)
	}
	decision = appStatusDecision(payload, grant, decision.UpgradePath)
	if !decision.Allowed {
		return decision
	}
	if strings.TrimSpace(feature) != "" && !containsValue(grant.Features, feature) {
		decision.Allowed = false
		decision.Reason = ReasonFeatureMissing
		decision.Warning = false
		return decision
	}
	if minPlanRank > 0 && grant.PlanRank < minPlanRank {
		decision.Allowed = false
		decision.Reason = ReasonRankInsufficient
		decision.Warning = false
	}
	return decision
}

// StatusDecisionForApp uses the app's effective signed status when the lease
// contains scoped grants; legacy leases retain their original global status.
func StatusDecisionForApp(payload entitlementclient.Payload, appKey, upgradePath string) Decision {
	grant, scoped := appEntitlement(payload, appKey)
	if !scoped {
		return StatusDecision(payload, upgradePath)
	}
	return appStatusDecision(payload, grant, upgradePath)
}

func appStatusDecision(payload entitlementclient.Payload, grant entitlementclient.AppEntitlement, upgradePath string) Decision {
	decision := Decision{Lease: payload, UpgradePath: upgradePath}
	if strings.TrimSpace(grant.AppKey) == "" {
		decision.Reason = ReasonFeatureMissing
		return decision
	}
	switch strings.ToLower(strings.TrimSpace(grant.Status)) {
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

func appEntitlement(payload entitlementclient.Payload, appKey string) (entitlementclient.AppEntitlement, bool) {
	if len(payload.AppEntitlements) == 0 {
		return entitlementclient.AppEntitlement{}, false
	}
	appKey = strings.TrimSpace(appKey)
	if appKey == "" {
		return entitlementclient.AppEntitlement{}, true
	}
	for _, grant := range payload.AppEntitlements {
		if grant.AppKey == appKey {
			return grant, true
		}
	}
	return entitlementclient.AppEntitlement{}, true
}

// HasFeature checks a feature in a lease that entitlementclient has already
// verified. It does not authenticate or authorize a cost-bearing operation.
func HasFeature(payload entitlementclient.Payload, feature string) bool {
	if len(payload.AppEntitlements) > 0 {
		return false
	}
	feature = strings.TrimSpace(feature)
	for _, candidate := range payload.Features {
		if candidate == feature {
			return true
		}
	}
	return false
}

// HasFeatureForApp scopes a feature to one app when app grants exist. Empty
// app identity fails closed for scoped leases.
func HasFeatureForApp(payload entitlementclient.Payload, appKey, feature string) bool {
	grant, scoped := appEntitlement(payload, appKey)
	if !scoped {
		return HasFeature(payload, feature)
	}
	return grant.AppKey != "" && containsValue(grant.Features, feature)
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
	if len(payload.AppEntitlements) > 0 {
		return 0, false
	}
	for _, limit := range payload.Limits {
		if limit.Key == key && (limit.BundleKey == "" || limit.BundleKey == bundle) {
			return limit.Value, true
		}
	}
	return 0, false
}

// LimitForApp scopes meter limits to the exact app grant when the lease is
// app-scoped. The legacy bundle-only format remains readable during migration.
func LimitForApp(payload entitlementclient.Payload, appKey, key, bundle string) (int64, bool) {
	grant, scoped := appEntitlement(payload, appKey)
	if !scoped {
		return Limit(payload, key, bundle)
	}
	if grant.AppKey == "" {
		return 0, false
	}
	for _, limit := range grant.Limits {
		if limit.Key == key && (limit.BundleKey == "" || limit.BundleKey == bundle) {
			return limit.Value, true
		}
	}
	return 0, false
}

func containsValue(values []string, wanted string) bool {
	wanted = strings.TrimSpace(wanted)
	for _, value := range values {
		if strings.TrimSpace(value) == wanted {
			return true
		}
	}
	return false
}

func (g *Gate) upgradePath() string {
	if g == nil || strings.TrimSpace(g.UpgradePath) == "" {
		return "/settings/subscription"
	}
	return g.UpgradePath
}

func (g *Gate) appKey() string {
	if g == nil {
		return ""
	}
	return g.AppKey
}

func fallbackDecision(err error, upgradePath string) Decision {
	reason := ReasonLeaseUnavailable
	if errors.Is(err, entitlementclient.ErrLeaseExpired) {
		reason = ReasonLeaseExpired
	}
	return Decision{Allowed: false, Reason: reason, UpgradePath: upgradePath}
}
