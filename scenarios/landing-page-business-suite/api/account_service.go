package main

import (
	"context"
	"database/sql"
	"strconv"
	"strings"
	"sync"
	"time"

	accountdomain "landing-page-business-suite-api/internal/account"
	"landing-page-business-suite-api/internal/envx"

	commonv1 "github.com/vrooli/vrooli/packages/proto/gen/go/common/v1"
	shared "github.com/vrooli/vrooli/packages/proto/gen/go/landing-page-business-suite/v1/shared"
	"google.golang.org/protobuf/proto"
	"google.golang.org/protobuf/types/known/timestamppb"
)

// AccountStore is the context-aware persistence contract for subscription and
// credit lookups. It is satisfied by both *sql.DB and database.RoutedDB.
//
// seam: AccountStore keeps account persistence independent of a concrete pool
// and allows HTTP requests to select Test Genie's lease-owned database.
type AccountStore interface {
	QueryRowContext(context.Context, string, ...any) *sql.Row
	ExecContext(context.Context, string, ...any) (sql.Result, error)
}

// AccountService exposes subscription, credits, and entitlement helpers.
type AccountService struct {
	db          AccountStore
	planService *PlanService
	bundleKey   string
	cacheTTL    time.Duration
	cacheMutex  sync.RWMutex
	cache       map[string]subscriptionCacheEntry
}

type (
	EntitlementPayload = accountdomain.EntitlementPayload
	CreditsEnvelope    = accountdomain.CreditsEnvelope
)

type subscriptionCacheEntry struct {
	status    *shared.SubscriptionStatus
	expiresAt time.Time
}

func NewAccountService(db AccountStore, planService *PlanService) *AccountService {
	return &AccountService{
		db:          db,
		planService: planService,
		bundleKey:   planService.BundleKey(),
		cacheTTL:    loadCacheTTL(),
		cache:       make(map[string]subscriptionCacheEntry),
	}
}

func loadCacheTTL() time.Duration {
	const defaultTTL = 60 * time.Second
	value := strings.TrimSpace(envx.Get("SUBSCRIPTION_CACHE_TTL_SECONDS"))
	if value == "" {
		return defaultTTL
	}

	seconds, err := strconv.Atoi(value)
	if err != nil || seconds <= 0 {
		return defaultTTL
	}

	return time.Duration(seconds) * time.Second
}

func (s *AccountService) GetSubscription(userIdentity string) (*shared.SubscriptionStatus, error) {
	return s.GetSubscriptionContext(context.Background(), userIdentity)
}

// GetSubscriptionContext reads subscription state using the caller's context.
func (s *AccountService) GetSubscriptionContext(ctx context.Context, userIdentity string) (*shared.SubscriptionStatus, error) {
	user := NormalizeEmail(userIdentity)
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

	query := `
		SELECT subscription_id, status, customer_email, plan_tier, price_id, bundle_key, billing_cycle_start, canceled_at, updated_at
		FROM subscriptions
		WHERE (customer_email = $1 OR customer_id = $1)
		ORDER BY updated_at DESC
		LIMIT 1
	`

	row := s.db.QueryRowContext(ctx, query, user)
	var subID, status, customerEmail string
	var planTier, priceID, bundleKey sql.NullString
	var billingCycleStart int
	var canceledAt sql.NullTime
	var updatedAt time.Time
	if err := row.Scan(
		&subID,
		&status,
		&customerEmail,
		&planTier,
		&priceID,
		&bundleKey,
		&billingCycleStart,
		&canceledAt,
		&updatedAt,
	); err != nil {
		if err == sql.ErrNoRows {
			return &shared.SubscriptionStatus{State: shared.SubscriptionState_SUBSCRIPTION_STATE_INACTIVE, UserIdentity: user}, nil
		}
		return nil, err
	}

	planTierStr := planTier.String
	priceIDStr := priceID.String
	bundleKeyStr := bundleKey.String

	if planTierStr == "" && priceIDStr != "" {
		if plan, err := s.planService.GetPlanByPriceID(priceIDStr); err == nil {
			planTierStr = plan.PlanTier
			if bundleKeyStr == "" {
				bundleKeyStr = plan.BundleKey
			}
		}
	}
	if planTierStr != "" {
		if _, err := normalizePlanTier(planTierStr); err != nil {
			logStructured("subscription_plan_tier_invalid", map[string]interface{}{
				"level":     "warn",
				"plan_tier": planTierStr,
				"price_id":  priceIDStr,
			})
			planTierStr = ""
		} else if planTier.String == "" || bundleKey.String == "" {
			_, _ = s.db.ExecContext(ctx, `
				UPDATE subscriptions
				SET plan_tier = COALESCE(NULLIF($1,''), plan_tier),
					bundle_key = COALESCE(NULLIF($2,''), bundle_key),
					updated_at = NOW()
				WHERE subscription_id = $3
			`, planTierStr, bundleKeyStr, subID)
		}
	}

	state := mapSubscriptionState(status)
	cacheAge := time.Since(updatedAt)
	result := &shared.SubscriptionStatus{
		State:          state,
		SubscriptionId: proto.String(subID),
		UserIdentity:   user,
		CachedAt:       timestamppb.New(updatedAt),
		CacheAgeMs:     cacheAge.Milliseconds(),
	}
	if planTierStr != "" {
		result.PlanTier = proto.String(planTierStr)
	}
	if priceIDStr != "" {
		result.StripePriceId = proto.String(priceIDStr)
	}
	if bundleKeyStr != "" {
		result.BundleKey = proto.String(bundleKeyStr)
	}
	if canceledAt.Valid {
		result.CanceledAt = timestamppb.New(canceledAt.Time)
	}

	s.cacheSubscription(user, result)

	return result, nil
}

func (s *AccountService) GetCredits(userIdentity string) (*CreditsEnvelope, error) {
	return s.GetCreditsContext(context.Background(), userIdentity)
}

// GetCreditsContext reads credit balance using the caller's context.
func (s *AccountService) GetCreditsContext(ctx context.Context, userIdentity string) (*CreditsEnvelope, error) {
	userIdentity = NormalizeEmail(userIdentity)
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
func (s *AccountService) getBillingCycleStart(userIdentity string) int {
	return s.getBillingCycleStartContext(context.Background(), userIdentity)
}

func (s *AccountService) getBillingCycleStartContext(ctx context.Context, userIdentity string) int {
	userIdentity = NormalizeEmail(userIdentity)
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

func (s *AccountService) GetEntitlements(userIdentity string) (*EntitlementPayload, error) {
	return s.GetEntitlementsContext(context.Background(), userIdentity)
}

// GetEntitlementsContext resolves subscription and credit state using the
// caller's context for all account persistence.
func (s *AccountService) GetEntitlementsContext(ctx context.Context, userIdentity string) (*EntitlementPayload, error) {
	userIdentity = NormalizeEmail(userIdentity)
	subscription, err := s.GetSubscriptionContext(ctx, userIdentity)
	if err != nil {
		return nil, err
	}

	credits, err := s.GetCreditsContext(ctx, userIdentity)
	if err != nil {
		return nil, err
	}

	payload := &EntitlementPayload{
		Status:            legacyStateLabel(subscription.State),
		PlanTier:          subscription.GetPlanTier(),
		PriceID:           subscription.GetStripePriceId(),
		BillingCycleStart: s.getBillingCycleStartContext(ctx, userIdentity),
		Credits:           flattenCredits(credits),
		Subscription:      subscription,
	}

	if subscription.GetStripePriceId() != "" {
		if plan, err := s.planService.GetPlanByPriceID(subscription.GetStripePriceId()); err == nil {
			payload.Features = extractFeatureFlags(plan.Metadata)
		}
	}

	return payload, nil
}

func (s *AccountService) getCachedSubscription(user string) (*shared.SubscriptionStatus, bool) {
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

func (s *AccountService) cacheSubscription(user string, status *shared.SubscriptionStatus) {
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

func mapSubscriptionState(state string) shared.SubscriptionState {
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

func legacyStateLabel(state shared.SubscriptionState) string {
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

// Compile-time interface check for AccountServicer
var _ AccountServicer = (*AccountService)(nil)
