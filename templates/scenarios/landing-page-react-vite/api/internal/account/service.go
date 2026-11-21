// Package account exposes the caller-scoped views over the billing tables:
// cached subscription status, credit-wallet balance (with bundle display
// labelling), and computed entitlements (status, tier, feature flags, credits).
// It composes the plan service and reads the subscription/credit tables owned by
// the stripe domain; it owns no tables of its own.
package account

import (
	"database/sql"
	"landing-page-react-vite-api/internal/jsonval"
	"landing-page-react-vite-api/internal/plan"
	"os"
	"strconv"
	"strings"
	"sync"
	"time"

	landingv1 "github.com/vrooli/vrooli/packages/proto/gen/go/landing-page-react-vite/v1"
	"google.golang.org/protobuf/proto"
	"google.golang.org/protobuf/types/known/timestamppb"
)

// Service exposes subscription, credits, and entitlement helpers.
type Service struct {
	db          *sql.DB
	planService *plan.Service
	bundleKey   string
	cacheTTL    time.Duration
	cacheMutex  sync.RWMutex
	cache       map[string]subscriptionCacheEntry
}

type subscriptionCacheEntry struct {
	status    *landingv1.SubscriptionStatus
	expiresAt time.Time
}

// CreditsEnvelope carries a wallet balance plus the bundle-derived display
// labelling used to render credit amounts.
type CreditsEnvelope struct {
	Balance                  *landingv1.CreditsBalance
	DisplayCreditsLabel      string
	DisplayCreditsMultiplier float64
}

// Entitlements is the computed access payload bundled apps read to unlock
// features.
type Entitlements struct {
	Status       string
	PlanTier     string
	PriceID      string
	Features     []string
	Credits      *landingv1.CreditsBalance
	Subscription *landingv1.SubscriptionStatus
}

// NewService constructs the account Service, reading the subscription cache TTL
// from SUBSCRIPTION_CACHE_TTL_SECONDS (default 60s).
func NewService(db *sql.DB, planService *plan.Service) *Service {
	return &Service{
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

// GetSubscription returns the caller's cached subscription status, refreshing
// from the database once the per-user cache entry expires.
func (s *Service) GetSubscription(userIdentity string) (*landingv1.SubscriptionStatus, error) {
	user := strings.TrimSpace(userIdentity)
	if user == "" {
		message := "user not provided"
		return &landingv1.SubscriptionStatus{State: landingv1.SubscriptionState_SUBSCRIPTION_STATE_INACTIVE, Message: &message}, nil
	}

	if cached, ok := s.getCachedSubscription(user); ok {
		return cached, nil
	}

	var subID, status, customerEmail, planTier, priceID, bundleKey string
	var canceledAt sql.NullTime
	var updatedAt time.Time
	err := s.db.QueryRow(`
		SELECT subscription_id, status, customer_email, plan_tier, price_id, bundle_key, canceled_at, updated_at
		FROM subscriptions
		WHERE (customer_email = $1 OR customer_id = $1)
		ORDER BY updated_at DESC LIMIT 1
	`, user).Scan(&subID, &status, &customerEmail, &planTier, &priceID, &bundleKey, &canceledAt, &updatedAt)
	if err == sql.ErrNoRows {
		return &landingv1.SubscriptionStatus{State: landingv1.SubscriptionState_SUBSCRIPTION_STATE_INACTIVE, UserIdentity: user}, nil
	}
	if err != nil {
		return nil, err
	}

	if planTier == "" && priceID != "" {
		if p, err := s.planService.GetPlanByPriceID(priceID); err == nil {
			planTier = p.PlanTier
		}
	}

	cacheAge := time.Since(updatedAt)
	result := &landingv1.SubscriptionStatus{
		State:          mapSubscriptionState(status),
		SubscriptionId: &subID,
		UserIdentity:   user,
		PlanTier:       &planTier,
		StripePriceId:  &priceID,
		BundleKey:      &bundleKey,
		CachedAt:       timestamppb.New(updatedAt),
		CacheAgeMs:     cacheAge.Milliseconds(),
	}
	if canceledAt.Valid {
		result.CanceledAt = timestamppb.New(canceledAt.Time)
	}

	s.cacheSubscription(user, result)
	return result, nil
}

// GetCredits returns the caller's wallet balance plus bundle display labelling.
func (s *Service) GetCredits(userIdentity string) (*CreditsEnvelope, error) {
	if strings.TrimSpace(userIdentity) == "" {
		return &CreditsEnvelope{
			Balance: &landingv1.CreditsBalance{
				CustomerEmail:  "",
				BalanceCredits: 0,
				BundleKey:      s.bundleKey,
				UpdatedAt:      timestamppb.Now(),
			},
			DisplayCreditsLabel:      "credits",
			DisplayCreditsMultiplier: 1.0,
		}, nil
	}

	var balance landingv1.CreditsBalance
	var updatedAt time.Time
	err := s.db.QueryRow(`
		SELECT customer_email, balance_credits, bonus_credits, updated_at
		FROM credit_wallets WHERE customer_email = $1 LIMIT 1
	`, userIdentity).Scan(&balance.CustomerEmail, &balance.BalanceCredits, new(int64), &updatedAt)
	if err != nil {
		if err == sql.ErrNoRows {
			balance.CustomerEmail = userIdentity
		} else {
			return nil, err
		}
	}

	label := "credits"
	multiplier := 1.0
	if overview, err := s.planService.GetPricingOverview(); err == nil {
		label = overview.Bundle.DisplayCreditsLabel
		multiplier = overview.Bundle.DisplayCreditsMultiplier
		balance.BundleKey = overview.Bundle.BundleKey
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

// GetEntitlements computes the caller's access payload (status/tier/features +
// credits) by composing the subscription status, credit wallet, and plan
// metadata.
func (s *Service) GetEntitlements(userIdentity string) (*Entitlements, error) {
	subscription, err := s.GetSubscription(userIdentity)
	if err != nil {
		return nil, err
	}
	credits, err := s.GetCredits(userIdentity)
	if err != nil {
		return nil, err
	}

	priceID := deref(subscription.StripePriceId)
	payload := &Entitlements{
		Status:       legacyStateLabel(subscription.State),
		PlanTier:     deref(subscription.PlanTier),
		PriceID:      priceID,
		Credits:      credits.Balance,
		Subscription: subscription,
	}

	if priceID != "" {
		if p, err := s.planService.GetPlanByPriceID(priceID); err == nil {
			payload.Features = jsonval.StringSlice(p.Metadata["features"])
		}
	}
	return payload, nil
}

func deref(p *string) string {
	if p == nil {
		return ""
	}
	return *p
}

func (s *Service) getCachedSubscription(user string) (*landingv1.SubscriptionStatus, bool) {
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
	cached := proto.Clone(entry.status).(*landingv1.SubscriptionStatus)
	s.cacheMutex.RUnlock()
	return cached, true
}

func (s *Service) cacheSubscription(user string, status *landingv1.SubscriptionStatus) {
	if status == nil || s.cacheTTL <= 0 {
		return
	}
	s.cacheMutex.Lock()
	s.cache[user] = subscriptionCacheEntry{
		status:    proto.Clone(status).(*landingv1.SubscriptionStatus),
		expiresAt: time.Now().Add(s.cacheTTL),
	}
	s.cacheMutex.Unlock()
}

func mapSubscriptionState(state string) landingv1.SubscriptionState {
	switch strings.ToLower(strings.TrimSpace(state)) {
	case "active":
		return landingv1.SubscriptionState_SUBSCRIPTION_STATE_ACTIVE
	case "trialing":
		return landingv1.SubscriptionState_SUBSCRIPTION_STATE_TRIALING
	case "past_due", "past-due":
		return landingv1.SubscriptionState_SUBSCRIPTION_STATE_PAST_DUE
	case "canceled", "cancelled":
		return landingv1.SubscriptionState_SUBSCRIPTION_STATE_CANCELED
	default:
		return landingv1.SubscriptionState_SUBSCRIPTION_STATE_INACTIVE
	}
}

func legacyStateLabel(state landingv1.SubscriptionState) string {
	switch state {
	case landingv1.SubscriptionState_SUBSCRIPTION_STATE_ACTIVE:
		return "active"
	case landingv1.SubscriptionState_SUBSCRIPTION_STATE_TRIALING:
		return "trialing"
	case landingv1.SubscriptionState_SUBSCRIPTION_STATE_PAST_DUE:
		return "past_due"
	case landingv1.SubscriptionState_SUBSCRIPTION_STATE_CANCELED:
		return "canceled"
	default:
		return "inactive"
	}
}
