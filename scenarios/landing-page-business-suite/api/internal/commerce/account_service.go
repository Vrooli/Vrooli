package commerce

import (
	"context"
	"database/sql"
	"fmt"
	"sort"
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
	QueryContext(context.Context, string, ...any) (*sql.Rows, error)
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

// GetSubscriptionContext reads subscription state using the caller's context.
func (s *Service) GetSubscriptionContext(ctx context.Context, userIdentity string) (*shared.SubscriptionStatus, error) {
	user := s.normalizeEmail(userIdentity)
	return s.getSubscriptionContext(ctx, user, "")
}

// GetSubscriptionForBusinessAccountContext resolves only the selected LPBS
// account. The email is retained as a provider lookup hint, never as the
// commercial isolation key.
func (s *Service) GetSubscriptionForBusinessAccountContext(ctx context.Context, userIdentity, businessAccountID string) (*shared.SubscriptionStatus, error) {
	return s.getSubscriptionContext(ctx, s.normalizeEmail(userIdentity), strings.TrimSpace(businessAccountID))
}

func (s *Service) getSubscriptionContext(ctx context.Context, user, businessAccountID string) (*shared.SubscriptionStatus, error) {
	if user == "" {
		return &shared.SubscriptionStatus{
			State:        shared.SubscriptionState_SUBSCRIPTION_STATE_INACTIVE,
			UserIdentity: "",
			Message:      proto.String("user not provided"),
		}, nil
	}

	cacheKey := user
	if businessAccountID != "" {
		cacheKey = businessAccountID + "\x00" + user
	}
	if cached, ok := s.getCachedSubscription(cacheKey); ok {
		return cached, nil
	}

	record, err := s.loadSubscriptionRecordForBusinessAccount(ctx, user, businessAccountID)
	if err != nil {
		return nil, err
	}
	if record == nil {
		return &shared.SubscriptionStatus{State: shared.SubscriptionState_SUBSCRIPTION_STATE_INACTIVE, UserIdentity: user}, nil
	}
	s.reconcileSubscriptionPlan(ctx, record)
	result := subscriptionStatusFromRecord(user, record)
	s.cacheSubscription(cacheKey, result)
	return result, nil
}

func (s *Service) loadSubscriptionRecordForBusinessAccount(ctx context.Context, user, businessAccountID string) (*subscriptionRecord, error) {
	query := `
		SELECT subscription_id, status, source, external_subscription_id, plan_tier, price_id, bundle_key, canceled_at, updated_at
		FROM subscriptions
		WHERE (customer_email = $1 OR customer_id = $1)`
	args := []any{user}
	if strings.TrimSpace(businessAccountID) != "" {
		query = `
		SELECT subscription_id, status, source, external_subscription_id, plan_tier, price_id, bundle_key, canceled_at, updated_at
		FROM subscriptions
		WHERE business_account_id = $1 AND (customer_email = $2 OR customer_id = $2)`
		args = []any{strings.TrimSpace(businessAccountID), user}
	}
	query += ` ORDER BY CASE WHEN status IN ('active','trialing','past_due') THEN 0 ELSE 1 END, updated_at DESC LIMIT 1`
	row := s.db.QueryRowContext(ctx, `
		`+query, args...)

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

// GetCreditsForBusinessAccountContext reports only attributed account lots.
// Cached wallet balances are not funding provenance, and this path never
// falls back to the person-email wallet for a selected business account.
func (s *Service) GetCreditsForBusinessAccountContext(ctx context.Context, userIdentity, businessAccountID string) (*CreditsEnvelope, error) {
	userIdentity = s.normalizeEmail(userIdentity)
	businessAccountID = strings.TrimSpace(businessAccountID)
	if userIdentity == "" || businessAccountID == "" {
		return nil, ErrBusinessAccountRequired
	}
	var balance int64
	err := s.db.QueryRowContext(ctx, `
		SELECT COALESCE(SUM(l.remaining), 0)
		FROM credit_funding_lots l
		JOIN business_account_credit_wallets w ON w.business_account_id = l.business_account_id
		WHERE w.business_account_id = $1
		  AND w.customer_email = $2
		  AND l.status = 'open'
		  AND l.settlement_state = 'settled'
		  AND l.livemode IS NOT NULL
		  AND (l.spendable_after IS NULL OR l.spendable_after <= CURRENT_TIMESTAMP)
		  AND (l.expires_at IS NULL OR l.expires_at > CURRENT_TIMESTAMP)
	`, businessAccountID, userIdentity).Scan(&balance)
	if err != nil {
		return nil, err
	}
	credits := shared.CreditsBalance{CustomerEmail: userIdentity, BalanceCredits: balance}
	return s.decorateCredits(credits, time.Time{}), nil
}

func (s *Service) decorateCredits(balance shared.CreditsBalance, updatedAt time.Time) *CreditsEnvelope {
	label := "credits"
	multiplier := 1.0
	if pOverview, err := s.planService.GetPricingOverview(); err == nil && pOverview != nil && pOverview.Bundle != nil {
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
	return &CreditsEnvelope{Balance: &balance, DisplayCreditsLabel: label, DisplayCreditsMultiplier: multiplier}
}

// getBillingCycleStart retrieves billing cycle start for a user.
func (s *Service) billingCycleStartForBusinessAccountContext(ctx context.Context, userIdentity, businessAccountID string) int {
	userIdentity = s.normalizeEmail(userIdentity)
	if userIdentity == "" {
		return 0
	}

	var billingCycleStart int
	query := `
		SELECT COALESCE(billing_cycle_start, 0)
		FROM subscriptions
		WHERE (customer_email = $1 OR customer_id = $1)`
	args := []any{userIdentity}
	if strings.TrimSpace(businessAccountID) != "" {
		query = `
		SELECT COALESCE(billing_cycle_start, 0)
		FROM subscriptions
		WHERE business_account_id = $1 AND (customer_email = $2 OR customer_id = $2)`
		args = []any{strings.TrimSpace(businessAccountID), userIdentity}
	}
	query += ` ORDER BY updated_at DESC LIMIT 1`
	err := s.db.QueryRowContext(ctx, query, args...).Scan(&billingCycleStart)
	if err != nil {
		return 0
	}
	return billingCycleStart
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
	return s.getEntitlementsContext(ctx, userIdentity, "")
}

// GetEntitlementsForBusinessAccountContext keeps desktop entitlement issuance
// bound to the selected LPBS account all the way through the commerce read.
func (s *Service) GetEntitlementsForBusinessAccountContext(ctx context.Context, userIdentity, businessAccountID string) (*EntitlementPayload, error) {
	return s.getEntitlementsContext(ctx, s.normalizeEmail(userIdentity), strings.TrimSpace(businessAccountID))
}

func (s *Service) getEntitlementsContext(ctx context.Context, userIdentity, businessAccountID string) (*EntitlementPayload, error) {
	subscription, err := s.GetSubscriptionForBusinessAccountContext(ctx, userIdentity, businessAccountID)
	if err != nil {
		return nil, err
	}

	// Subscription-only callers do not obtain credit authority without an
	// authenticated account selection. Their subscription entitlements remain.
	credits := s.decorateCredits(shared.CreditsBalance{CustomerEmail: userIdentity}, time.Time{})
	if businessAccountID != "" {
		credits, err = s.GetCreditsForBusinessAccountContext(ctx, userIdentity, businessAccountID)
		if err != nil {
			return nil, err
		}
	}

	payload := &EntitlementPayload{
		UserIdentity:      userIdentity,
		Status:            SubscriptionStateLabel(subscription.State),
		PlanTier:          subscription.GetPlanTier(),
		PlanRank:          PlanRankForTier(subscription.GetPlanTier()),
		PriceID:           subscription.GetStripePriceId(),
		NotAfter:          time.Now().UTC().Add(s.leaseTTL),
		BillingCycleStart: s.billingCycleStartForBusinessAccountContext(ctx, userIdentity, businessAccountID),
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

	appEntitlements, scoped, err := s.deriveAppEntitlements(ctx, userIdentity, businessAccountID)
	if err != nil {
		return nil, fmt.Errorf("derive effective app entitlements: %w", err)
	}
	if scoped {
		payload.AppEntitlements = appEntitlements
		payload.Features = nil
		payload.Limits = nil
		// These legacy summary fields remain for account display. Authorization
		// in relying apps uses each signed app record when scoped records exist.
		for _, grant := range appEntitlements {
			payload.Features = append(payload.Features, grant.Features...)
			payload.Limits = append(payload.Limits, grant.Limits...)
		}
		payload.Features = uniqueSortedStrings(payload.Features)
		payload.Limits = mergeUniqueLimits(payload.Limits)
		payload.Status, payload.PlanRank = summarizeEffectiveAppGrants(appEntitlements)
		if tier := planTierForRank(payload.PlanRank); tier != "" {
			payload.PlanTier = tier
		}
	}

	if subscriptionRecordSource, externalID := s.subscriptionSourceAndExternalIDForBusinessAccount(userIdentity, businessAccountID, subscription); subscriptionRecordSource != "stripe" && externalID != "" {
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

func summarizeEffectiveAppGrants(grants []entitlementclient.AppEntitlement) (string, int32) {
	status := "inactive"
	statusPriority := entitlementStatusPriority(status)
	var rank int32
	for _, grant := range grants {
		if grant.PlanRank > rank {
			rank = grant.PlanRank
		}
		if priority := entitlementStatusPriority(grant.Status); priority > statusPriority {
			status = grant.Status
			statusPriority = priority
		}
	}
	return status, rank
}

func planTierForRank(rank int32) string {
	for tier, candidate := range allowedPlanTiers {
		if candidate == rank {
			return tier
		}
	}
	return ""
}

type appEntitlementDeclaration struct {
	features []string
	limits   []entitlementclient.Limit
}

// deriveAppEntitlements computes the signed effective grant set from every
// currently usable subscription on the selected account. If any active plan
// is scoped while another cannot be resolved to a scope declaration, refuse
// to issue a partial scoped lease.
func (s *Service) deriveAppEntitlements(ctx context.Context, userIdentity, businessAccountID string) ([]entitlementclient.AppEntitlement, bool, error) {
	if s == nil || s.db == nil || s.planService == nil {
		return nil, false, nil
	}
	query := `SELECT status, plan_tier, price_id, bundle_key, source, external_subscription_id FROM subscriptions WHERE (customer_email = $1 OR customer_id = $1) AND status IN ('active','trialing','past_due')`
	args := []any{userIdentity}
	if strings.TrimSpace(businessAccountID) != "" {
		query = `SELECT status, plan_tier, price_id, bundle_key, source, external_subscription_id FROM subscriptions WHERE business_account_id = $1 AND (customer_email = $2 OR customer_id = $2) AND status IN ('active','trialing','past_due')`
		args = []any{strings.TrimSpace(businessAccountID), userIdentity}
	}
	query += ` ORDER BY updated_at DESC, subscription_id`
	rows, err := s.db.QueryContext(ctx, query, args...)
	if err != nil {
		return nil, false, err
	}
	defer rows.Close()

	// App keys are scenario identifiers, but the signed grant is also scoped
	// to the purchased bundle. Keep those scopes separate when an account owns
	// plans from more than one bundle; merging by app key would let one bundle's
	// features/limits ride on the other bundle's grant.
	type appBundleKey struct{ app, bundle string }
	merged := make(map[appBundleKey]entitlementclient.AppEntitlement)
	unscopedPlans := 0
	scopedPlans := 0
	for rows.Next() {
		var status, planTier, priceID, bundleKey, source, externalID sql.NullString
		if err := rows.Scan(&status, &planTier, &priceID, &bundleKey, &source, &externalID); err != nil {
			return nil, false, err
		}
		plan, err := s.resolvePlanForSubscription(priceID.String, source.String, externalID.String)
		if err != nil {
			return nil, false, err
		}
		if plan == nil {
			unscopedPlans++
			continue
		}
		declarations, declared, err := parseAppEntitlementDeclarations(plan.Metadata)
		if err != nil {
			return nil, false, fmt.Errorf("plan %s app entitlement metadata: %w", priceID.String, err)
		}
		if !declared {
			unscopedPlans++
			continue
		}
		scopedPlans++
		planTierValue := strings.TrimSpace(planTier.String)
		if planTierValue == "" {
			planTierValue = plan.GetPlanTier()
		}
		planRank := plan.GetPlanRank()
		if planRank == 0 && planTierValue != "" {
			planRank = PlanRankForTier(planTierValue)
		}
		for appKey, declaration := range declarations {
			grant := entitlementclient.AppEntitlement{
				AppKey: appKey, BundleKey: strings.TrimSpace(bundleKey.String), Status: strings.ToLower(strings.TrimSpace(status.String)),
				PlanRank: planRank, Features: append([]string(nil), declaration.features...), Limits: append([]entitlementclient.Limit(nil), declaration.limits...),
			}
			if s.limitsSvc != nil && planTierValue != "" {
				configured, err := s.limitsSvc.GetTierLimits(ctx, planTierValue)
				if err != nil {
					return nil, false, fmt.Errorf("load limits for plan %s: %w", planTierValue, err)
				}
				for _, limit := range configured {
					if limit.AppBundleKey != nil && *limit.AppBundleKey != s.bundleKey {
						continue
					}
					grant.Limits = append(grant.Limits, entitlementclient.Limit{Key: limit.LimitKey, Value: limit.LimitValue, BundleKey: s.bundleKey})
				}
			}
			grant.Limits = mergeUniqueLimits(grant.Limits)
			key := appBundleKey{app: appKey, bundle: grant.BundleKey}
			merged[key] = mergeAppEntitlement(merged[key], grant)
		}
	}
	if err := rows.Err(); err != nil {
		return nil, false, err
	}
	if scopedPlans == 0 {
		return nil, false, nil
	}
	if unscopedPlans > 0 {
		return nil, false, fmt.Errorf("%d active subscription plan(s) lack app entitlement declarations", unscopedPlans)
	}
	keys := make([]appBundleKey, 0, len(merged))
	for key := range merged {
		keys = append(keys, key)
	}
	sort.Slice(keys, func(i, j int) bool {
		if keys[i].app != keys[j].app {
			return keys[i].app < keys[j].app
		}
		return keys[i].bundle < keys[j].bundle
	})
	result := make([]entitlementclient.AppEntitlement, 0, len(keys))
	for _, key := range keys {
		grant := merged[key]
		grant.Limits = mergeUniqueLimits(grant.Limits)
		grant.Features = uniqueSortedStrings(grant.Features)
		result = append(result, grant)
	}
	return result, true, nil
}

func (s *Service) resolvePlanForSubscription(priceID, source, externalID string) (*shared.PlanOption, error) {
	if strings.TrimSpace(priceID) != "" {
		plan, err := s.planService.GetPlanByPriceID(strings.TrimSpace(priceID))
		if err != nil {
			return nil, nil
		}
		if plan != nil {
			return plan, nil
		}
	}
	if strings.ToLower(strings.TrimSpace(source)) != "stripe" && strings.TrimSpace(externalID) != "" {
		resolver, ok := s.planService.(interface {
			GetPlanByExternalProductID(string) (*shared.PlanOption, error)
		})
		if ok {
			plan, err := resolver.GetPlanByExternalProductID(strings.TrimSpace(externalID))
			if err == nil {
				return plan, err
			}
		}
	}
	return nil, nil
}

func mergeAppEntitlement(existing, next entitlementclient.AppEntitlement) entitlementclient.AppEntitlement {
	if existing.AppKey == "" {
		return next
	}
	if next.PlanRank > existing.PlanRank {
		existing.PlanRank = next.PlanRank
	}
	if entitlementStatusPriority(next.Status) > entitlementStatusPriority(existing.Status) {
		existing.Status = next.Status
	}
	existing.Features = append(existing.Features, next.Features...)
	existing.Limits = append(existing.Limits, next.Limits...)
	if existing.BundleKey == "" {
		existing.BundleKey = next.BundleKey
	}
	return existing
}

func entitlementStatusPriority(status string) int {
	switch strings.ToLower(strings.TrimSpace(status)) {
	case "active":
		return 3
	case "trialing":
		return 2
	case "past_due":
		return 1
	default:
		return 0
	}
}

func mergeUniqueLimits(limits []entitlementclient.Limit) []entitlementclient.Limit {
	merged := make(map[string]entitlementclient.Limit, len(limits))
	for _, limit := range limits {
		key := strings.TrimSpace(limit.Key)
		if key == "" {
			continue
		}
		limit.Key = key
		current, ok := merged[key]
		if !ok || (limit.Value == -1 && current.Value != -1) || (limit.Value != -1 && current.Value != -1 && limit.Value > current.Value) {
			merged[key] = limit
		}
	}
	keys := make([]string, 0, len(merged))
	for key := range merged {
		keys = append(keys, key)
	}
	sort.Strings(keys)
	result := make([]entitlementclient.Limit, 0, len(keys))
	for _, key := range keys {
		result = append(result, merged[key])
	}
	return result
}

func uniqueSortedStrings(values []string) []string {
	seen := make(map[string]struct{}, len(values))
	result := make([]string, 0, len(values))
	for _, value := range values {
		value = strings.TrimSpace(value)
		if value == "" {
			continue
		}
		if _, ok := seen[value]; ok {
			continue
		}
		seen[value] = struct{}{}
		result = append(result, value)
	}
	sort.Strings(result)
	return result
}

func parseAppEntitlementDeclarations(metadata map[string]*commonv1.JsonValue) (map[string]appEntitlementDeclaration, bool, error) {
	value := metadata["app_entitlements"]
	if value == nil {
		return nil, false, nil
	}
	object, ok := jsonValueToAny(value).(map[string]any)
	if !ok {
		return nil, true, fmt.Errorf("app_entitlements must be an object keyed by app_key")
	}
	declarations := make(map[string]appEntitlementDeclaration, len(object))
	for rawAppKey, rawValue := range object {
		appKey := strings.TrimSpace(rawAppKey)
		if appKey == "" || appKey != rawAppKey {
			return nil, true, fmt.Errorf("app key must be canonical and nonempty")
		}
		entry, ok := rawValue.(map[string]any)
		if !ok {
			return nil, true, fmt.Errorf("app %s declaration must be an object", appKey)
		}
		declaration := appEntitlementDeclaration{}
		if featureValue, exists := entry["features"]; exists {
			list, ok := featureValue.([]any)
			if !ok {
				return nil, true, fmt.Errorf("app %s features must be a list", appKey)
			}
			for _, item := range list {
				text, ok := item.(string)
				if !ok || strings.TrimSpace(text) == "" || strings.TrimSpace(text) != text {
					return nil, true, fmt.Errorf("app %s feature must be a canonical nonempty string", appKey)
				}
				declaration.features = append(declaration.features, text)
			}
		}
		if limitsValue, exists := entry["limits"]; exists {
			limits, ok := limitsValue.(map[string]any)
			if !ok {
				return nil, true, fmt.Errorf("app %s limits must be an object", appKey)
			}
			for key, amount := range limits {
				if strings.TrimSpace(key) == "" || strings.TrimSpace(key) != key {
					return nil, true, fmt.Errorf("app %s limit key must be canonical and nonempty", appKey)
				}
				var value int64
				switch number := amount.(type) {
				case int64:
					value = number
				case float64:
					if number != float64(int64(number)) {
						return nil, true, fmt.Errorf("app %s limit %s must be an integer", appKey, key)
					}
					value = int64(number)
				default:
					return nil, true, fmt.Errorf("app %s limit %s must be an integer", appKey, key)
				}
				if value < -1 {
					return nil, true, fmt.Errorf("app %s limit %s must be -1 (unlimited) or nonnegative", appKey, key)
				}
				declaration.limits = append(declaration.limits, entitlementclient.Limit{Key: key, Value: value})
			}
		}
		declaration.features = uniqueSortedStrings(declaration.features)
		declarations[appKey] = declaration
	}
	if len(declarations) == 0 {
		return nil, true, fmt.Errorf("app_entitlements must declare at least one app")
	}
	return declarations, true, nil
}

func (s *Service) subscriptionSourceAndExternalIDForBusinessAccount(userIdentity, businessAccountID string, subscription *shared.SubscriptionStatus) (string, string) {
	// SubscriptionStatus intentionally remains source-neutral and compact for
	// existing API consumers. The source-specific identifier is resolved from
	// the authoritative row only when the lease is assembled.
	record, err := s.loadSubscriptionRecordForBusinessAccount(context.Background(), userIdentity, businessAccountID)
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
