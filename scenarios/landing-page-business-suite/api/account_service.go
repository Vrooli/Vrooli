package main

import (
	"database/sql"
	"os"
	"strconv"
	"strings"
	"sync"
	"time"

	commonv1 "github.com/vrooli/vrooli/packages/proto/gen/go/common/v1"
	landing_page_react_vite_v1 "github.com/vrooli/vrooli/packages/proto/gen/go/landing-page-react-vite/v1"
	"google.golang.org/protobuf/proto"
	"google.golang.org/protobuf/types/known/timestamppb"
)

// AccountService exposes subscription, credits, and entitlement helpers.
type AccountService struct {
	db          *sql.DB
	planService *PlanService
	bundleKey   string
	cacheTTL    time.Duration
	cacheMutex  sync.RWMutex
	cache       map[string]subscriptionCacheEntry
}

// EntitlementPayload is used by bundled apps to unlock features.
type EntitlementPayload struct {
	Status       string                                         `json:"status"`
	PlanTier     string                                         `json:"plan_tier,omitempty"`
	PriceID      string                                         `json:"price_id,omitempty"`
	Features     []string                                       `json:"features,omitempty"`
	Credits      *landing_page_react_vite_v1.CreditsBalance     `json:"credits,omitempty"`
	Subscription *landing_page_react_vite_v1.SubscriptionStatus `json:"subscription,omitempty"`
}

type CreditsEnvelope struct {
	Balance                  *landing_page_react_vite_v1.CreditsBalance `json:"balance"`
	DisplayCreditsLabel      string                                     `json:"display_credits_label"`
	DisplayCreditsMultiplier float64                                    `json:"display_credits_multiplier"`
}

type subscriptionCacheEntry struct {
	status    *landing_page_react_vite_v1.SubscriptionStatus
	expiresAt time.Time
}

func NewAccountService(db *sql.DB, planService *PlanService) *AccountService {
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
	value := strings.TrimSpace(os.Getenv("SUBSCRIPTION_CACHE_TTL_SECONDS"))
	if value == "" {
		return defaultTTL
	}

	seconds, err := strconv.Atoi(value)
	if err != nil || seconds <= 0 {
		return defaultTTL
	}

	return time.Duration(seconds) * time.Second
}

func (s *AccountService) GetSubscription(userIdentity string) (*landing_page_react_vite_v1.SubscriptionStatus, error) {
	user := strings.TrimSpace(userIdentity)
	if user == "" {
		return &landing_page_react_vite_v1.SubscriptionStatus{
			State:        landing_page_react_vite_v1.SubscriptionState_SUBSCRIPTION_STATE_INACTIVE,
			UserIdentity: "",
			Message:      proto.String("user not provided"),
		}, nil
	}

	if cached, ok := s.getCachedSubscription(user); ok {
		return cached, nil
	}

	query := `
		SELECT subscription_id, status, customer_email, plan_tier, price_id, bundle_key, canceled_at, updated_at
		FROM subscriptions
		WHERE (customer_email = $1 OR customer_id = $1)
		ORDER BY updated_at DESC
		LIMIT 1
	`

	row := s.db.QueryRow(query, user)
	var subID, status, customerEmail, planTier, priceID, bundleKey string
	var canceledAt sql.NullTime
	var updatedAt time.Time
	if err := row.Scan(
		&subID,
		&status,
		&customerEmail,
		&planTier,
		&priceID,
		&bundleKey,
		&canceledAt,
		&updatedAt,
	); err != nil {
		if err == sql.ErrNoRows {
			return &landing_page_react_vite_v1.SubscriptionStatus{State: landing_page_react_vite_v1.SubscriptionState_SUBSCRIPTION_STATE_INACTIVE, UserIdentity: user}, nil
		}
		return nil, err
	}

	if planTier == "" && priceID != "" {
		if plan, err := s.planService.GetPlanByPriceID(priceID); err == nil {
			planTier = plan.PlanTier
		}
	}

	state := mapSubscriptionState(status)
	cacheAge := time.Since(updatedAt)
	result := &landing_page_react_vite_v1.SubscriptionStatus{
		State:          state,
		SubscriptionId: proto.String(subID),
		UserIdentity:   user,
		CachedAt:       timestamppb.New(updatedAt),
		CacheAgeMs:     cacheAge.Milliseconds(),
	}
	if planTier != "" {
		result.PlanTier = proto.String(planTier)
	}
	if priceID != "" {
		result.StripePriceId = proto.String(priceID)
	}
	if bundleKey != "" {
		result.BundleKey = proto.String(bundleKey)
	}
	if canceledAt.Valid {
		result.CanceledAt = timestamppb.New(canceledAt.Time)
	}

	s.cacheSubscription(user, result)

	return result, nil
}

func (s *AccountService) GetCredits(userIdentity string) (*CreditsEnvelope, error) {
	if strings.TrimSpace(userIdentity) == "" {
		return &CreditsEnvelope{
			Balance: &landing_page_react_vite_v1.CreditsBalance{
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

	row := s.db.QueryRow(query, userIdentity)
	var balance landing_page_react_vite_v1.CreditsBalance
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

func (s *AccountService) GetEntitlements(userIdentity string) (*EntitlementPayload, error) {
	subscription, err := s.GetSubscription(userIdentity)
	if err != nil {
		return nil, err
	}

	credits, err := s.GetCredits(userIdentity)
	if err != nil {
		return nil, err
	}

payload := &EntitlementPayload{
	Status:       legacyStateLabel(subscription.State),
	PlanTier:     subscription.GetPlanTier(),
	PriceID:      subscription.GetStripePriceId(),
	Credits:      flattenCredits(credits),
	Subscription: subscription,
}

	if subscription.GetStripePriceId() != "" {
		if plan, err := s.planService.GetPlanByPriceID(subscription.GetStripePriceId()); err == nil {
			payload.Features = extractFeatureFlags(plan.Metadata)
		}
	}

	return payload, nil
}

func (s *AccountService) getCachedSubscription(user string) (*landing_page_react_vite_v1.SubscriptionStatus, bool) {
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

	cached := proto.Clone(entry.status).(*landing_page_react_vite_v1.SubscriptionStatus)
	s.cacheMutex.RUnlock()
	return cached, true
}

func (s *AccountService) cacheSubscription(user string, status *landing_page_react_vite_v1.SubscriptionStatus) {
	if status == nil || s.cacheTTL <= 0 {
		return
	}

	entry := subscriptionCacheEntry{
		status:    &landing_page_react_vite_v1.SubscriptionStatus{},
		expiresAt: time.Now().Add(s.cacheTTL),
	}
	*entry.status = *status

	s.cacheMutex.Lock()
	s.cache[user] = entry
	s.cacheMutex.Unlock()
}

func mapSubscriptionState(state string) landing_page_react_vite_v1.SubscriptionState {
	switch strings.ToLower(strings.TrimSpace(state)) {
	case "active":
		return landing_page_react_vite_v1.SubscriptionState_SUBSCRIPTION_STATE_ACTIVE
	case "trialing":
		return landing_page_react_vite_v1.SubscriptionState_SUBSCRIPTION_STATE_TRIALING
	case "past_due", "past-due":
		return landing_page_react_vite_v1.SubscriptionState_SUBSCRIPTION_STATE_PAST_DUE
	case "canceled", "cancelled":
		return landing_page_react_vite_v1.SubscriptionState_SUBSCRIPTION_STATE_CANCELED
	default:
		return landing_page_react_vite_v1.SubscriptionState_SUBSCRIPTION_STATE_INACTIVE
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

func flattenCredits(resp *CreditsEnvelope) *landing_page_react_vite_v1.CreditsBalance {
	if resp == nil {
		return nil
	}
	return resp.Balance
}

func legacyStateLabel(state landing_page_react_vite_v1.SubscriptionState) string {
	switch state {
	case landing_page_react_vite_v1.SubscriptionState_SUBSCRIPTION_STATE_ACTIVE:
		return "active"
	case landing_page_react_vite_v1.SubscriptionState_SUBSCRIPTION_STATE_TRIALING:
		return "trialing"
	case landing_page_react_vite_v1.SubscriptionState_SUBSCRIPTION_STATE_PAST_DUE:
		return "past_due"
	case landing_page_react_vite_v1.SubscriptionState_SUBSCRIPTION_STATE_CANCELED:
		return "canceled"
	default:
		return "inactive"
	}
}
