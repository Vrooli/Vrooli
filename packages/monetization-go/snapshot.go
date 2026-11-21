package monetization

import (
	"strings"
	"time"

	entitlementclient "github.com/vrooli/vrooli/packages/entitlementclient-go"
)

// Snapshot is the shared display and local-capacity view of a verified lease.
// It does not grant access: security decisions use Gate against the verified
// lease, and Class A usage is reserved by its trusted authority.
type Snapshot struct {
	UserIdentity      string                             `json:"user_identity"`
	Status            SubscriptionStatus                 `json:"status"`
	Tier              string                             `json:"tier"`
	PlanRank          int32                              `json:"plan_rank"`
	PriceID           string                             `json:"price_id,omitempty"`
	Features          []string                           `json:"features,omitempty"`
	Limits            []entitlementclient.Limit          `json:"limits,omitempty"`
	AppEntitlements   []entitlementclient.AppEntitlement `json:"app_entitlements,omitempty"`
	Credits           int64                              `json:"credits,omitempty"`
	BillingCycleStart int                                `json:"billing_cycle_start,omitempty"`
	FetchedAt         time.Time                          `json:"fetched_at"`
	ExpiresAt         time.Time                          `json:"expires_at"`
}

// SnapshotFromLease adapts signed lease claims into the shared view consumed
// by settings and local-capacity code. Callers must pass a lease returned by
// entitlementclient after signature and expiry verification.
func SnapshotFromLease(lease entitlementclient.Payload, fetchedAt time.Time) Snapshot {
	if fetchedAt.IsZero() {
		fetchedAt = time.Now().UTC()
	}
	status := SubscriptionStatus(strings.ToLower(strings.TrimSpace(lease.Status)))
	tier := strings.ToLower(strings.TrimSpace(lease.PlanTier))
	if tier == "" {
		tier = string(PlanFree)
	}
	if status == "" {
		status = StatusInactive
	}
	snapshot := Snapshot{
		UserIdentity:      lease.UserIdentity,
		Status:            status,
		Tier:              tier,
		PlanRank:          lease.PlanRank,
		PriceID:           lease.PriceID,
		Features:          append([]string(nil), lease.Features...),
		Limits:            append([]entitlementclient.Limit(nil), lease.Limits...),
		AppEntitlements:   cloneAppEntitlements(lease.AppEntitlements),
		BillingCycleStart: lease.BillingCycleStart,
		FetchedAt:         fetchedAt,
		ExpiresAt:         lease.NotAfter,
	}
	if balance, ok := lease.Credits.(map[string]any); ok {
		if value, ok := balance["balance_credits"].(float64); ok {
			snapshot.Credits = int64(value)
		}
	}
	return snapshot
}

func cloneAppEntitlements(values []entitlementclient.AppEntitlement) []entitlementclient.AppEntitlement {
	if len(values) == 0 {
		return nil
	}
	cloned := make([]entitlementclient.AppEntitlement, len(values))
	for i, value := range values {
		cloned[i] = value
		cloned[i].Features = append([]string(nil), value.Features...)
		cloned[i].Limits = append([]entitlementclient.Limit(nil), value.Limits...)
	}
	return cloned
}

func (s *Snapshot) IsActive() bool {
	return s != nil && (s.Status == StatusActive || s.Status == StatusTrialing)
}

// IsExpired reports whether this display snapshot's signed lease boundary has
// passed. Authorization still belongs to entitlementclient verification and
// Gate; this projection is for status/UI compatibility only.
func (s *Snapshot) IsExpired() bool {
	return s == nil || !time.Now().Before(s.ExpiresAt)
}

// HasFeatureForApp returns the feature in an exact app grant when scoped
// grants are present. A scoped snapshot without an app key fails closed.
func (s *Snapshot) HasFeatureForApp(appKey, feature string) bool {
	if s == nil {
		return false
	}
	payload := entitlementclient.Payload{Features: s.Features, AppEntitlements: s.AppEntitlements}
	return HasFeatureForApp(payload, appKey, feature)
}

func (s *Snapshot) LimitValueForApp(appKey, key, bundleKey string) (int64, bool) {
	if s == nil {
		return 0, false
	}
	return LimitForApp(entitlementclient.Payload{Limits: s.Limits, AppEntitlements: s.AppEntitlements}, appKey, key, bundleKey)
}

func (s *Snapshot) GetBillingPeriod(at time.Time) (start, end time.Time) {
	if s == nil {
		day := 0
		return billingPeriod(at, day)
	}
	return billingPeriod(at, s.BillingCycleStart)
}

func (s *Snapshot) GetBillingMonth(at time.Time) string {
	start, _ := s.GetBillingPeriod(at)
	return start.Format("2006-01-02")
}

func billingPeriod(at time.Time, cycleDay int) (start, end time.Time) {
	if cycleDay < 1 || cycleDay > 28 {
		year, month, _ := at.Date()
		start = time.Date(year, month, 1, 0, 0, 0, 0, at.Location())
		return start, start.AddDate(0, 1, 0).Add(-time.Nanosecond)
	}
	year, month, day := at.Date()
	if day >= cycleDay {
		start = time.Date(year, month, cycleDay, 0, 0, 0, 0, at.Location())
		end = time.Date(year, month+1, cycleDay, 0, 0, 0, 0, at.Location()).Add(-time.Nanosecond)
	} else {
		start = time.Date(year, month-1, cycleDay, 0, 0, 0, 0, at.Location())
		end = time.Date(year, month, cycleDay, 0, 0, 0, 0, at.Location()).Add(-time.Nanosecond)
	}
	return start, end
}
