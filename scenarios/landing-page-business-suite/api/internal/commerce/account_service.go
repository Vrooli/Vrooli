package commerce

import (
	"context"
	"database/sql"
	"strings"
	"sync"
	"time"

	entitlementclient "github.com/vrooli/vrooli/packages/entitlementclient-go"
	commonv1 "github.com/vrooli/vrooli/packages/proto/gen/go/common/v1"
	shared "github.com/vrooli/vrooli/packages/proto/gen/go/landing-page-business-suite/v1/shared"
	"google.golang.org/protobuf/proto"
	"google.golang.org/protobuf/types/known/timestamppb"
)

// Store is the context-aware persistence contract for subscription and
// credit lookups. It is satisfied by both *sql.DB and database.RoutedDB.
//
// seam: Store keeps account persistence independent of a concrete pool
// and allows HTTP requests to select Test Genie's lease-owned database.
type Store interface {
	QueryRowContext(context.Context, string, ...any) *sql.Row
	ExecContext(context.Context, string, ...any) (sql.Result, error)
}

// Service exposes subscription, credits, and entitlement helpers.
// PlanCatalog supplies immutable bundle and plan metadata to account policy.
type PlanCatalog interface {
	BundleKey() string
	GetPricingOverview() (*shared.PricingOverview, error)
	GetPlanByPriceID(string) (*shared.PlanOption, error)
}

// Runtime carries root-owned infrastructure policy into the account domain.
type Runtime struct {
	CacheTTL       time.Duration
	LeaseTTL       time.Duration
	LimitsService  LimitsServicer
	NormalizeEmail func(string) string
	Log            func(string, map[string]interface{})
}

const defaultCacheTTL = 60 * time.Second

type Service struct {
	db             Store
	planService    PlanCatalog
	bundleKey      string
	cacheTTL       time.Duration
	leaseTTL       time.Duration
	limitsSvc      LimitsServicer
	normalizeEmail func(string) string
	logf           func(string, map[string]interface{})
	cacheMutex     sync.RWMutex
	cache          map[string]subscriptionCacheEntry
}

type subscriptionCacheEntry struct {
	status    *shared.SubscriptionStatus
	expiresAt time.Time
}

type subscriptionRecord struct {
	id                     string
	status                 string
	source                 string
	externalSubscriptionID string
	planTier               string
	priceID                string
	bundleKey              string
	canceledAt             sql.NullTime
	updatedAt              time.Time
}

func NewService(db Store, planService PlanCatalog, runtime Runtime) *Service {
	if runtime.NormalizeEmail == nil {
		runtime.NormalizeEmail = normalizeEmail
	}
	if runtime.Log == nil {
		runtime.Log = func(string, map[string]interface{}) {}
	}
	if runtime.CacheTTL <= 0 {
		runtime.CacheTTL = defaultCacheTTL
	}
	if runtime.LeaseTTL <= 0 {
		runtime.LeaseTTL = 7 * 24 * time.Hour
	}
	return &Service{
		db: db, planService: planService, bundleKey: planService.BundleKey(),
		cacheTTL: runtime.CacheTTL, leaseTTL: runtime.LeaseTTL, limitsSvc: runtime.LimitsService, cache: make(map[string]subscriptionCacheEntry),
		normalizeEmail: runtime.NormalizeEmail, logf: runtime.Log,
	}
}

func normalizeEmail(value string) string { return strings.ToLower(strings.TrimSpace(value)) }

func (s *Service) GetSubscription(userIdentity string) (*shared.SubscriptionStatus, error) {
	return s.GetSubscriptionContext(context.Background(), userIdentity)
}

// GetSubscriptionContext reads subscription state using the caller's context.
func (s *Service) GetSubscriptionContext(ctx context.Context, userIdentity string) (*shared.SubscriptionStatus, error) {
	user := s.normalizeEmail(userIdentity)
	if user == "" {
		return &shared.SubscriptionStatus{
			State:        shared.SubscriptionState_SUBSCRIPTION_STATE_INACTIVE,
			UserIdentity: "",
			Message:      proto.String("user not provided"),
		}, nil
	}

	if cached, ok := s.getCachedSubscription(user); ok {
		return cached, nil
	}

	record, err := s.loadSubscriptionRecord(ctx, user)
	if err != nil {
		return nil, err
	}
	if record == nil {
		return &shared.SubscriptionStatus{State: shared.SubscriptionState_SUBSCRIPTION_STATE_INACTIVE, UserIdentity: user}, nil
	}
	s.reconcileSubscriptionPlan(ctx, record)
	result := subscriptionStatusFromRecord(user, record)
	s.cacheSubscription(user, result)
	return result, nil
}

func (s *Service) loadSubscriptionRecord(ctx context.Context, user string) (*subscriptionRecord, error) {
	row := s.db.QueryRowContext(ctx, `
		SELECT subscription_id, status, source, external_subscription_id, plan_tier, price_id, bundle_key, canceled_at, updated_at
		FROM subscriptions
		WHERE (customer_email = $1 OR customer_id = $1)
		ORDER BY updated_at DESC
		LIMIT 1
	`, user)

	record := &subscriptionRecord{}
	var source, externalSubscriptionID, planTier, priceID, bundleKey sql.NullString
	if err := row.Scan(
		&record.id,
		&record.status,
		&source,
		&externalSubscriptionID,
		&planTier,
		&priceID,
		&bundleKey,
		&record.canceledAt,
		&record.updatedAt,
	); err != nil {
		if err == sql.ErrNoRows {
			return nil, nil
		}
		return nil, err
	}
	record.planTier = planTier.String
	record.priceID = priceID.String
	record.bundleKey = bundleKey.String
	record.source = source.String
	record.externalSubscriptionID = externalSubscriptionID.String
	return record, nil
}

func (s *Service) reconcileSubscriptionPlan(ctx context.Context, record *subscriptionRecord) {
	storedPlanTier, storedBundleKey := record.planTier, record.bundleKey
	if record.planTier == "" && record.priceID != "" {
		if plan, err := s.planService.GetPlanByPriceID(record.priceID); err == nil {
			record.planTier = plan.PlanTier
			if record.bundleKey == "" {
				record.bundleKey = plan.BundleKey
			}
		}
	}
	if record.planTier == "" {
		return
	}
	if _, err := NormalizePlanTier(record.planTier); err != nil {
		s.logf("subscription_plan_tier_invalid", map[string]interface{}{
			"level":     "warn",
			"plan_tier": record.planTier,
			"price_id":  record.priceID,
		})
		record.planTier = ""
		return
	}
	if storedPlanTier != "" && storedBundleKey != "" {
		return
	}
	_, _ = s.db.ExecContext(ctx, `
		UPDATE subscriptions
		SET plan_tier = COALESCE(NULLIF($1,''), plan_tier),
			bundle_key = COALESCE(NULLIF($2,''), bundle_key),
			updated_at = NOW()
		WHERE subscription_id = $3
	`, record.planTier, record.bundleKey, record.id)
}

func subscriptionStatusFromRecord(user string, record *subscriptionRecord) *shared.SubscriptionStatus {
	result := &shared.SubscriptionStatus{
		State:          MapSubscriptionState(record.status),
		SubscriptionId: proto.String(record.id),
		UserIdentity:   user,
		CachedAt:       timestamppb.New(record.updatedAt),
		CacheAgeMs:     time.Since(record.updatedAt).Milliseconds(),
	}
	if record.planTier != "" {
		result.PlanTier = proto.String(record.planTier)
	}
	if record.priceID != "" {
		result.StripePriceId = proto.String(record.priceID)
	}
	if record.bundleKey != "" {
		result.BundleKey = proto.String(record.bundleKey)
	}
	if record.canceledAt.Valid {
		result.CanceledAt = timestamppb.New(record.canceledAt.Time)
	}
	return result
}

func (s *Service) GetCredits(userIdentity string) (*CreditsEnvelope, error) {
	return s.GetCreditsContext(context.Background(), userIdentity)
}

// GetCreditsContext reads credit balance using the caller's context.
func (s *Service) GetCreditsContext(ctx context.Context, userIdentity string) (*CreditsEnvelope, error) {
	userIdentity = s.normalizeEmail(userIdentity)
	if userIdentity == "" {
		return &CreditsEnvelope{
			Balance: &shared.CreditsBalance{
				CustomerEmail:  "",
				BalanceCredits: 0,
				BundleKey:      s.bundleKey,
				UpdatedAt:      timestamppb.Now(),
			},
			DisplayCreditsLabel:      "credits",
			DisplayCreditsMultiplier: 1.0,
		}, nil
	}

	query := `
		SELECT customer_email, balance_credits, bonus_credits, updated_at
		FROM credit_wallets
		WHERE customer_email = $1
		LIMIT 1
	`

	row := s.db.QueryRowContext(ctx, query, userIdentity)
	var balance shared.CreditsBalance
	var updatedAt time.Time
	if err := row.Scan(
		&balance.CustomerEmail,
		&balance.BalanceCredits,
		new(int64), // bonus credits (unused placeholder)
		&updatedAt,
	); err != nil {
		if err == sql.ErrNoRows {
			// No wallet yet; return zero values
			balance.CustomerEmail = userIdentity
		} else {
			return nil, err
		}
	}

	label := "credits"
	multiplier := 1.0

	pOverview, err := s.planService.GetPricingOverview()
	if err == nil {
		label = pOverview.Bundle.DisplayCreditsLabel
		multiplier = pOverview.Bundle.DisplayCreditsMultiplier
		balance.BundleKey = pOverview.Bundle.BundleKey
	} else {
		balance.BundleKey = s.bundleKey
	}

	if updatedAt.IsZero() {
		balance.UpdatedAt = timestamppb.Now()
	} else {
		balance.UpdatedAt = timestamppb.New(updatedAt)
	}

	return &CreditsEnvelope{
		Balance:                  &balance,
		DisplayCreditsLabel:      label,
		DisplayCreditsMultiplier: multiplier,
	}, nil
}

// getBillingCycleStart retrieves billing cycle start for a user.
func (s *Service) BillingCycleStart(userIdentity string) int {
	return s.billingCycleStartContext(context.Background(), userIdentity)
}

func (s *Service) billingCycleStartContext(ctx context.Context, userIdentity string) int {
	userIdentity = s.normalizeEmail(userIdentity)
	if userIdentity == "" {
		return 0
	}

	var billingCycleStart int
	err := s.db.QueryRowContext(ctx, `
		SELECT COALESCE(billing_cycle_start, 0)
		FROM subscriptions
		WHERE (customer_email = $1 OR customer_id = $1)
		ORDER BY updated_at DESC
		LIMIT 1
	`, userIdentity).Scan(&billingCycleStart)
	if err != nil {
		return 0
	}
	return billingCycleStart
}

func (s *Service) GetEntitlements(userIdentity string) (*EntitlementPayload, error) {
	return s.GetEntitlementsContext(context.Background(), userIdentity)
}

// GetEntitlementStatus is the narrow delivery-facing entitlement seam. It
// avoids leaking the commerce payload into download authorization.
func (s *Service) GetEntitlementStatus(ctx context.Context, userIdentity string) (string, error) {
	entitlements, err := s.GetEntitlementsContext(ctx, userIdentity)
	if err != nil {
		return "", err
	}
	if entitlements == nil {
		return "", nil
	}
	return entitlements.Status, nil
}

// GetEntitlementsContext resolves subscription and credit state using the
// caller's context for all account persistence.
func (s *Service) GetEntitlementsContext(ctx context.Context, userIdentity string) (*EntitlementPayload, error) {
	userIdentity = s.normalizeEmail(userIdentity)
	subscription, err := s.GetSubscriptionContext(ctx, userIdentity)
	if err != nil {
		return nil, err
	}

	credits, err := s.GetCreditsContext(ctx, userIdentity)
	if err != nil {
		return nil, err
	}

	payload := &EntitlementPayload{
		Status:            SubscriptionStateLabel(subscription.State),
		PlanTier:          subscription.GetPlanTier(),
		PlanRank:          PlanRankForTier(subscription.GetPlanTier()),
		PriceID:           subscription.GetStripePriceId(),
		NotAfter:          time.Now().UTC().Add(s.leaseTTL),
		BillingCycleStart: s.billingCycleStartContext(ctx, userIdentity),
		Credits:           flattenCredits(credits),
		Subscription:      subscription,
	}
	if s.limitsSvc != nil && payload.PlanTier != "" {
		if configured, err := s.limitsSvc.GetTierLimits(ctx, payload.PlanTier); err == nil {
			for _, limit := range configured {
				if limit.AppBundleKey != nil && *limit.AppBundleKey != s.bundleKey {
					continue
				}
				payload.Limits = append(payload.Limits, entitlementclient.Limit{Key: limit.LimitKey, Value: limit.LimitValue, BundleKey: s.bundleKey})
			}
		} else {
			s.logf("entitlement_limits_unavailable", map[string]interface{}{"level": "warn", "plan_tier": payload.PlanTier, "error": err.Error()})
		}
	}

	if subscriptionRecordSource, externalID := s.subscriptionSourceAndExternalID(userIdentity, subscription); subscriptionRecordSource != "stripe" && externalID != "" {
		if resolver, ok := s.planService.(interface {
			GetPlanByExternalProductID(string) (*shared.PlanOption, error)
		}); ok {
			if plan, err := resolver.GetPlanByExternalProductID(externalID); err == nil {
				payload.Features = extractFeatureFlags(plan.Metadata)
			}
		}
	}
	if len(payload.Features) == 0 && subscription.GetStripePriceId() != "" {
		if plan, err := s.planService.GetPlanByPriceID(subscription.GetStripePriceId()); err == nil {
			payload.Features = extractFeatureFlags(plan.Metadata)
		}
	} else if payload.PlanTier != "" {
		// Store-issued subscriptions may not have a Stripe price. Resolve the
		// canonical feature flags by tier so non-Stripe subscribers do not fail
		// closed merely because the Stripe identifier is absent.
		if overview, err := s.planService.GetPricingOverview(); err == nil {
			plans := append(append([]*shared.PlanOption{}, overview.GetMonthly()...), overview.GetYearly()...)
			for _, plan := range plans {
				if strings.EqualFold(plan.GetPlanTier(), payload.PlanTier) {
					payload.Features = extractFeatureFlags(plan.Metadata)
					break
				}
			}
		}
	}

	return payload, nil
}

func (s *Service) subscriptionSourceAndExternalID(userIdentity string, subscription *shared.SubscriptionStatus) (string, string) {
	// SubscriptionStatus intentionally remains source-neutral and compact for
	// existing API consumers. The source-specific identifier is resolved from
	// the authoritative row only when the lease is assembled.
	record, err := s.loadSubscriptionRecord(context.Background(), userIdentity)
	if err != nil || record == nil {
		return "stripe", ""
	}
	return strings.ToLower(strings.TrimSpace(record.source)), strings.TrimSpace(record.externalSubscriptionID)
}

func (s *Service) getCachedSubscription(user string) (*shared.SubscriptionStatus, bool) {
	s.cacheMutex.RLock()
	entry, ok := s.cache[user]
	if !ok {
		s.cacheMutex.RUnlock()
		return nil, false
	}

	if time.Now().After(entry.expiresAt) {
		s.cacheMutex.RUnlock()
		s.cacheMutex.Lock()
		delete(s.cache, user)
		s.cacheMutex.Unlock()
		return nil, false
	}

	cached := proto.Clone(entry.status).(*shared.SubscriptionStatus)
	s.cacheMutex.RUnlock()
	return cached, true
}

// SetCacheTTL updates cache behavior for tests and controlled runtime tuning.
func (s *Service) SetCacheTTL(ttl time.Duration) {
	s.cacheMutex.Lock()
	defer s.cacheMutex.Unlock()
	s.cacheTTL = ttl
	s.cache = make(map[string]subscriptionCacheEntry)
}

// BundleKey reports the configured commercial bundle.
func (s *Service) BundleKey() string { return s.bundleKey }

func (s *Service) cacheSubscription(user string, status *shared.SubscriptionStatus) {
	if status == nil || s.cacheTTL <= 0 {
		return
	}

	entry := subscriptionCacheEntry{
		status:    &shared.SubscriptionStatus{},
		expiresAt: time.Now().Add(s.cacheTTL),
	}
	entry.status = proto.Clone(status).(*shared.SubscriptionStatus)

	s.cacheMutex.Lock()
	s.cache[user] = entry
	s.cacheMutex.Unlock()
}

func MapSubscriptionState(state string) shared.SubscriptionState {
	switch strings.ToLower(strings.TrimSpace(state)) {
	case "active":
		return shared.SubscriptionState_SUBSCRIPTION_STATE_ACTIVE
	case "trialing":
		return shared.SubscriptionState_SUBSCRIPTION_STATE_TRIALING
	case "past_due", "past-due":
		return shared.SubscriptionState_SUBSCRIPTION_STATE_PAST_DUE
	case "canceled", "cancelled":
		return shared.SubscriptionState_SUBSCRIPTION_STATE_CANCELED
	default:
		return shared.SubscriptionState_SUBSCRIPTION_STATE_INACTIVE
	}
}

func extractFeatureFlags(metadata map[string]*commonv1.JsonValue) []string {
	if metadata == nil {
		return nil
	}

	value, ok := metadata["features"]
	if !ok || value == nil || value.Kind == nil {
		return nil
	}

	listVal, ok := value.Kind.(*commonv1.JsonValue_ListValue)
	if !ok || listVal.ListValue == nil {
		return nil
	}

	var features []string
	for _, v := range listVal.ListValue.Values {
		if strVal, ok := v.Kind.(*commonv1.JsonValue_StringValue); ok && strVal.StringValue != "" {
			features = append(features, strVal.StringValue)
		}
	}
	return features
}

func flattenCredits(resp *CreditsEnvelope) *shared.CreditsBalance {
	if resp == nil {
		return nil
	}
	return resp.Balance
}

// SubscriptionStateLabel returns the stable persistence label for a typed
// subscription state. Stripe reconciliation and account reads share this
// commerce-owned mapping.
func SubscriptionStateLabel(state shared.SubscriptionState) string {
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
