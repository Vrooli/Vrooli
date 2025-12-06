package main

import (
	"database/sql"
	"os"
	"strconv"
	"strings"
	"sync"
	"time"

	lprvv1 "github.com/vrooli/vrooli/packages/proto/gen/go/landing-page-react-vite/v1"
	"google.golang.org/protobuf/proto"
	"google.golang.org/protobuf/types/known/structpb"
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
	Status       string                     `json:"status"`
	PlanTier     string                     `json:"plan_tier,omitempty"`
	PriceID      string                     `json:"price_id,omitempty"`
	Features     []string                   `json:"features,omitempty"`
	Credits      *lprvv1.CreditsBalance     `json:"credits,omitempty"`
	Subscription *lprvv1.SubscriptionStatus `json:"subscription,omitempty"`
}

type CreditsEnvelope struct {
	Balance                  *lprvv1.CreditsBalance `json:"balance"`
	DisplayCreditsLabel      string                 `json:"display_credits_label"`
	DisplayCreditsMultiplier float64                `json:"display_credits_multiplier"`
}

type subscriptionCacheEntry struct {
	status    *lprvv1.SubscriptionStatus
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

func (s *AccountService) GetSubscription(userIdentity string) (*lprvv1.SubscriptionStatus, error) {
	user := strings.TrimSpace(userIdentity)
	if user == "" {
		return &lprvv1.SubscriptionStatus{State: lprvv1.SubscriptionState_SUBSCRIPTION_STATE_INACTIVE, Message: "user not provided"}, nil
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
			return &lprvv1.SubscriptionStatus{State: lprvv1.SubscriptionState_SUBSCRIPTION_STATE_INACTIVE, UserIdentity: user}, nil
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
	result := &lprvv1.SubscriptionStatus{
		State:          state,
		SubscriptionId: subID,
		UserIdentity:   user,
		PlanTier:       planTier,
		StripePriceId:  priceID,
		BundleKey:      bundleKey,
		CachedAt:       timestamppb.New(updatedAt),
		CacheAgeMs:     cacheAge.Milliseconds(),
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
			Balance: &lprvv1.CreditsBalance{
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
	var balance lprvv1.CreditsBalance
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
		PlanTier:     subscription.PlanTier,
		PriceID:      subscription.StripePriceId,
		Credits:      flattenCredits(credits),
		Subscription: subscription,
	}

	if subscription.StripePriceId != "" {
		if plan, err := s.planService.GetPlanByPriceID(subscription.StripePriceId); err == nil {
			payload.Features = extractFeatureFlags(plan.Metadata)
		}
	}

	return payload, nil
}

func (s *AccountService) getCachedSubscription(user string) (*lprvv1.SubscriptionStatus, bool) {
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

	cached := proto.Clone(entry.status).(*lprvv1.SubscriptionStatus)
	s.cacheMutex.RUnlock()
	return cached, true
}

func (s *AccountService) cacheSubscription(user string, status *lprvv1.SubscriptionStatus) {
	if status == nil || s.cacheTTL <= 0 {
		return
	}

	entry := subscriptionCacheEntry{
		status:    &lprvv1.SubscriptionStatus{},
		expiresAt: time.Now().Add(s.cacheTTL),
	}
	*entry.status = *status

	s.cacheMutex.Lock()
	s.cache[user] = entry
	s.cacheMutex.Unlock()
}

func mapSubscriptionState(state string) lprvv1.SubscriptionState {
	switch strings.ToLower(strings.TrimSpace(state)) {
	case "active":
		return lprvv1.SubscriptionState_SUBSCRIPTION_STATE_ACTIVE
	case "trialing":
		return lprvv1.SubscriptionState_SUBSCRIPTION_STATE_TRIALING
	case "past_due", "past-due":
		return lprvv1.SubscriptionState_SUBSCRIPTION_STATE_PAST_DUE
	case "canceled", "cancelled":
		return lprvv1.SubscriptionState_SUBSCRIPTION_STATE_CANCELED
	default:
		return lprvv1.SubscriptionState_SUBSCRIPTION_STATE_INACTIVE
	}
}

func extractFeatureFlags(metadata map[string]*structpb.Value) []string {
	if metadata == nil {
		return nil
	}

	value, ok := metadata["features"]
	if !ok || value == nil || value.Kind == nil {
		return nil
	}

	list := value.GetListValue()
	if list == nil {
		return nil
	}

	var features []string
	for _, v := range list.Values {
		if str := v.GetStringValue(); str != "" {
			features = append(features, str)
		}
	}
	return features
}

func flattenCredits(resp *CreditsEnvelope) *lprvv1.CreditsBalance {
	if resp == nil {
		return nil
	}
	return resp.Balance
}

func legacyStateLabel(state lprvv1.SubscriptionState) string {
	switch state {
	case lprvv1.SubscriptionState_SUBSCRIPTION_STATE_ACTIVE:
		return "active"
	case lprvv1.SubscriptionState_SUBSCRIPTION_STATE_TRIALING:
		return "trialing"
	case lprvv1.SubscriptionState_SUBSCRIPTION_STATE_PAST_DUE:
		return "past_due"
	case lprvv1.SubscriptionState_SUBSCRIPTION_STATE_CANCELED:
		return "canceled"
	default:
		return "inactive"
	}
}
