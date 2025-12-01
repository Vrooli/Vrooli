package main

import (
	"context"
	"crypto/hmac"
	"crypto/sha256"
	"database/sql"
	"encoding/hex"
	"encoding/json"
	"errors"
	"fmt"
	"os"
	"strconv"
	"strings"
	"time"
	"sync"
)

// StripeService handles Stripe payment integration
type StripeService struct {
	db                *sql.DB
	planService       *PlanService
	paymentSettings   *PaymentSettingsService
	checkoutCacheTTL  time.Duration
	mu                sync.RWMutex
	runtimeConfig     stripeRuntimeConfig
}

type stripeRuntimeConfig struct {
	publishableKey string
	secretKey      string
	webhookSecret  string
	source         string
	hasPublishable bool
	hasSecret      bool
	hasWebhook     bool
}

const (
	sessionTypeSubscription          = "subscription"
	sessionTypeCreditsTopup          = "credits_topup"
	sessionTypeSupporterContribution = "supporter_contribution"
)

// NewStripeService creates a new Stripe service instance.
func NewStripeService(db *sql.DB) *StripeService {
	return NewStripeServiceWithSettings(db, NewPlanService(db), NewPaymentSettingsService(db))
}

// NewStripeServiceWithSettings wires explicit plan/payment dependencies (used by server).
func NewStripeServiceWithSettings(db *sql.DB, planService *PlanService, paymentSettings *PaymentSettingsService) *StripeService {
	if planService == nil {
		planService = NewPlanService(db)
	}
	if paymentSettings == nil {
		paymentSettings = NewPaymentSettingsService(db)
	}

	service := &StripeService{
		db:               db,
		planService:      planService,
		paymentSettings:  paymentSettings,
		checkoutCacheTTL: 60 * time.Second,
	}

	if err := service.RefreshConfig(context.Background()); err != nil {
		logStructured("failed to initialize Stripe config", map[string]interface{}{
			"level": "warn",
			"error": err.Error(),
		})
	}

	return service
}

// RefreshConfig reloads Stripe credentials from DB/env.
func (s *StripeService) RefreshConfig(ctx context.Context) error {
	cfg, err := s.loadStripeConfig(ctx)
	if err != nil {
		return err
	}

	s.mu.Lock()
	s.runtimeConfig = cfg
	s.mu.Unlock()
	return nil
}

func (s *StripeService) loadStripeConfig(ctx context.Context) (stripeRuntimeConfig, error) {
	cfg := stripeRuntimeConfig{source: "env"}

	if s.paymentSettings != nil {
		record, err := s.paymentSettings.GetStripeSettings(ctx)
		if err != nil {
			return cfg, err
		}
		if record != nil {
			cfg.publishableKey = strings.TrimSpace(record.PublishableKey)
			cfg.secretKey = strings.TrimSpace(record.SecretKey)
			cfg.webhookSecret = strings.TrimSpace(record.WebhookSecret)
			cfg.source = "database"
			cfg.hasPublishable = cfg.publishableKey != ""
			cfg.hasSecret = cfg.secretKey != ""
			cfg.hasWebhook = cfg.webhookSecret != ""
			if cfg.hasPublishable && cfg.hasSecret {
				return cfg, nil
			}
		}
	}

	envPublishable := strings.TrimSpace(os.Getenv("STRIPE_PUBLISHABLE_KEY"))
	envSecret := strings.TrimSpace(os.Getenv("STRIPE_SECRET_KEY"))
	envWebhook := strings.TrimSpace(os.Getenv("STRIPE_WEBHOOK_SECRET"))

	cfg.publishableKey = envPublishable
	cfg.secretKey = envSecret
	cfg.webhookSecret = envWebhook
	cfg.hasPublishable = envPublishable != ""
	cfg.hasSecret = envSecret != ""
	cfg.hasWebhook = envWebhook != ""

	if !cfg.hasPublishable {
		cfg.publishableKey = "pk_test_placeholder"
		logStructured("STRIPE_PUBLISHABLE_KEY missing - using placeholder", map[string]interface{}{
			"level":   "warn",
			"message": "STRIPE_PUBLISHABLE_KEY not set; using placeholder for development",
		})
	}
	if !cfg.hasSecret {
		cfg.secretKey = "sk_test_placeholder"
		logStructured("STRIPE_SECRET_KEY missing - using placeholder", map[string]interface{}{
			"level":   "warn",
			"message": "STRIPE_SECRET_KEY not set; using placeholder for development",
		})
	}
	if !cfg.hasWebhook {
		cfg.webhookSecret = "whsec_placeholder"
		logStructured("STRIPE_WEBHOOK_SECRET missing - using placeholder", map[string]interface{}{
			"level":   "warn",
			"message": "STRIPE_WEBHOOK_SECRET not set; using placeholder for development",
		})
	}

	return cfg, nil
}

func (s *StripeService) getConfig() stripeRuntimeConfig {
	s.mu.RLock()
	defer s.mu.RUnlock()
	return s.runtimeConfig
}

// StripeConfigSnapshot exposes sanitized runtime config info.
type StripeConfigSnapshot struct {
	PublishableKeyPreview string `json:"publishable_key_preview,omitempty"`
	PublishableKeySet     bool   `json:"publishable_key_set"`
	SecretKeySet          bool   `json:"secret_key_set"`
	WebhookSecretSet      bool   `json:"webhook_secret_set"`
	Source                string `json:"source"`
}

func maskValue(value string) string {
	if value == "" {
		return ""
	}
	if len(value) <= 6 {
		return value
	}
	return value[:4] + "…" + value[len(value)-2:]
}

// ConfigSnapshot returns a redacted view of the active Stripe configuration.
func (s *StripeService) ConfigSnapshot() StripeConfigSnapshot {
	cfg := s.getConfig()
	return StripeConfigSnapshot{
		PublishableKeyPreview: maskValue(cfg.publishableKey),
		PublishableKeySet:     cfg.hasPublishable,
		SecretKeySet:          cfg.hasSecret,
		WebhookSecretSet:      cfg.hasWebhook,
		Source:                cfg.source,
	}
}

// CreateCheckoutSession creates a Stripe checkout session
// [REQ:STRIPE-ROUTES] POST /api/checkout/create endpoint
func (s *StripeService) CreateCheckoutSession(priceID string, successURL string, cancelURL string, customerEmail string) (map[string]interface{}, error) {
	cfg := s.getConfig()

	// [REQ:STRIPE-CONFIG] Uses Stripe keys from environment or admin settings
	if !cfg.hasSecret {
		return nil, errors.New("Stripe not configured - missing STRIPE_SECRET_KEY")
	}

	// NOTE: In production, this would call the actual Stripe API
	// For MVP, we return a mock session for testing
	sessionID := "cs_test_" + time.Now().Format("20060102150405")
	checkoutURL := "https://checkout.stripe.com/c/pay/" + sessionID

	session := map[string]interface{}{
		"id":              sessionID,
		"url":             checkoutURL,
		"customer":        customerEmail,
		"amount_total":    5000, // $50.00 in cents
		"currency":        "usd",
		"status":          "open",
		"created":         time.Now().Unix(),
		"success_url":     successURL,
		"cancel_url":      cancelURL,
		"publishable_key": cfg.publishableKey,
	}

	// Store session in database for later verification
	_, err := s.db.Exec(`
		INSERT INTO checkout_sessions (session_id, customer_email, price_id, status, created_at)
		VALUES ($1, $2, $3, $4, $5)
	`, sessionID, customerEmail, priceID, "open", time.Now())

	if err != nil {
		return nil, err
	}

	return session, nil
}

// VerifyWebhookSignature validates the Stripe webhook signature
// [REQ:STRIPE-SIG] Webhook signature verification
func (s *StripeService) VerifyWebhookSignature(payload []byte, signature string) bool {
	cfg := s.getConfig()
	// [REQ:STRIPE-CONFIG] Uses webhook secret from environment/admin settings
	if !cfg.hasWebhook {
		logStructured("Stripe webhook secret not configured", map[string]interface{}{"level": "warn"})
		return false
	}

	// Extract timestamp and signature from header
	// Format: t=timestamp,v1=signature
	parts := strings.Split(signature, ",")
	if len(parts) < 2 {
		return false
	}

	var timestamp, sig string
	for _, part := range parts {
		if strings.HasPrefix(part, "t=") {
			timestamp = strings.TrimPrefix(part, "t=")
		} else if strings.HasPrefix(part, "v1=") {
			sig = strings.TrimPrefix(part, "v1=")
		}
	}

	if timestamp == "" || sig == "" {
		return false
	}

	// Construct signed payload: timestamp.payload
	signedPayload := timestamp + "." + string(payload)

	// Compute HMAC-SHA256
	mac := hmac.New(sha256.New, []byte(cfg.webhookSecret))
	mac.Write([]byte(signedPayload))
	expectedSig := hex.EncodeToString(mac.Sum(nil))

	// Compare signatures
	return hmac.Equal([]byte(sig), []byte(expectedSig))
}

// HandleWebhook processes Stripe webhook events
// [REQ:STRIPE-ROUTES] POST /api/webhooks/stripe endpoint
func (s *StripeService) HandleWebhook(body []byte, signature string) error {
	// [REQ:STRIPE-SIG] Verify signature before processing
	if !s.VerifyWebhookSignature(body, signature) {
		return errors.New("invalid webhook signature")
	}

	var event map[string]interface{}
	if err := json.Unmarshal(body, &event); err != nil {
		return err
	}

	eventType, ok := event["type"].(string)
	if !ok {
		return errors.New("missing event type")
	}

	data, ok := event["data"].(map[string]interface{})
	if !ok {
		return errors.New("missing event data")
	}

	obj, ok := data["object"].(map[string]interface{})
	if !ok {
		return errors.New("missing event object")
	}

	// Handle different event types
	switch eventType {
	case "checkout.session.completed":
		return s.handleCheckoutCompleted(obj)
	case "customer.subscription.created":
		return s.handleSubscriptionCreated(obj)
	case "customer.subscription.updated":
		return s.handleSubscriptionUpdated(obj)
	case "customer.subscription.deleted":
		return s.handleSubscriptionDeleted(obj)
	default:
		logStructured("Unhandled webhook event", map[string]interface{}{
			"event_type": eventType,
		})
	}

	return nil
}

func (s *StripeService) handleCheckoutCompleted(obj map[string]interface{}) error {
	sessionID, ok := obj["id"].(string)
	if !ok {
		return errors.New("missing session id")
	}

	customerEmail, _ := obj["customer_email"].(string)
	subscriptionID, _ := obj["subscription"].(string)

	sessionRec, err := s.loadCheckoutSession(sessionID)
	if err != nil {
		return err
	}

	if sessionRec.Status == "complete" {
		logStructured("checkout.session.completed ignored (duplicate)", map[string]interface{}{
			"session_id": sessionID,
		})
		return nil
	}

	if _, err := s.db.Exec(`
		UPDATE checkout_sessions
		SET status = $1, subscription_id = $2, updated_at = $3
		WHERE session_id = $4
	`, "complete", subscriptionID, time.Now(), sessionID); err != nil {
		return err
	}

	var plan *PlanOption
	if sessionRec.PriceID.Valid {
		if p, planErr := s.planService.GetPlanByPriceID(sessionRec.PriceID.String); planErr == nil {
			plan = p
		} else {
			logStructured("plan metadata missing during checkout completion", map[string]interface{}{
				"price_id": sessionRec.PriceID.String,
				"error":    planErr.Error(),
			})
		}
	}

	amountCents := s.extractAmount(obj, sessionRec)

	switch {
	case plan != nil && plan.Kind == sessionTypeCreditsTopup:
		return s.handleCreditTopup(customerEmail, amountCents, plan, map[string]interface{}{
			"session_id": sessionID,
		})
	case plan != nil && plan.Kind == sessionTypeSupporterContribution:
		logStructured("supporter contribution received", map[string]interface{}{
			"session_id": sessionID,
			"email":      customerEmail,
			"amount":     amountCents,
		})
		return nil
	default:
		return s.handleSubscriptionCompletion(subscriptionID, customerEmail, plan, sessionRec, amountCents)
	}
}

type checkoutSessionRecord struct {
	SessionID   string
	Status      string
	PriceID     sql.NullString
	SessionType sql.NullString
	AmountCents sql.NullInt64
	ScheduleID  sql.NullString
}

func (s *StripeService) loadCheckoutSession(sessionID string) (*checkoutSessionRecord, error) {
	record := &checkoutSessionRecord{}
	err := s.db.QueryRow(`
		SELECT session_id, status, price_id, session_type, amount_cents, schedule_id
		FROM checkout_sessions
		WHERE session_id = $1
	`, sessionID).Scan(
		&record.SessionID,
		&record.Status,
		&record.PriceID,
		&record.SessionType,
		&record.AmountCents,
		&record.ScheduleID,
	)
	if err != nil {
		return nil, err
	}
	return record, nil
}

func (s *StripeService) extractAmount(obj map[string]interface{}, session *checkoutSessionRecord) int64 {
	if amount := s.parseStripeAmount(obj["amount_total"]); amount != 0 {
		return amount
	}
	if amount := s.parseStripeAmount(obj["amount"]); amount != 0 {
		return amount
	}
	if session != nil && session.AmountCents.Valid {
		return session.AmountCents.Int64
	}
	return 0
}

func (s *StripeService) parseStripeAmount(value interface{}) int64 {
	switch v := value.(type) {
	case float64:
		return int64(v)
	case int64:
		return v
	case json.Number:
		if parsed, err := v.Int64(); err == nil {
			return parsed
		}
	case string:
		if parsed, err := strconv.ParseInt(v, 10, 64); err == nil {
			return parsed
		}
	}
	return 0
}

func (s *StripeService) handleSubscriptionCompletion(subscriptionID, customerEmail string, plan *PlanOption, session *checkoutSessionRecord, amountCents int64) error {
	if plan == nil {
		// Without plan metadata we cannot create enriched entries
		return nil
	}

	if subscriptionID == "" {
		return errors.New("subscription id required for subscription completion")
	}

	if amountCents == 0 {
		amountCents = plan.AmountCents
	}

	now := time.Now()
	_, err := s.db.Exec(`
		INSERT INTO subscriptions (subscription_id, customer_email, status, plan_tier, price_id, bundle_key, created_at, updated_at)
		VALUES ($1, $2, $3, $4, $5, $6, $7, $8)
		ON CONFLICT (subscription_id) DO UPDATE
		SET status = $3, customer_email = $2, plan_tier = $4, price_id = $5, bundle_key = $6, updated_at = $8
	`, subscriptionID, customerEmail, "active", plan.PlanTier, plan.StripePriceID, plan.BundleKey, now, now)
	if err != nil {
		return err
	}

	if plan.IntroEnabled && plan.BillingInterval == "month" {
		scheduleID, err := s.createSubscriptionSchedule(subscriptionID, plan, amountCents)
		if err != nil {
			return err
		}
		if scheduleID != "" && session != nil {
			if _, err := s.db.Exec(`
				UPDATE checkout_sessions
				SET schedule_id = $1
				WHERE session_id = $2
			`, scheduleID, session.SessionID); err != nil {
				return err
			}
		}
	}

	logStructured("Checkout session completed", map[string]interface{}{
		"session_id":      session.SessionID,
		"customer_email":  customerEmail,
		"subscription_id": subscriptionID,
		"plan_tier":       plan.PlanTier,
		"price_id":        plan.StripePriceID,
		"session_type":    sessionTypeSubscription,
	})

	return nil
}

func (s *StripeService) createSubscriptionSchedule(subscriptionID string, plan *PlanOption, amountCents int64) (string, error) {
	if plan == nil || subscriptionID == "" {
		return "", nil
	}

	if amountCents == 0 {
		amountCents = plan.AmountCents
	}

	scheduleID := fmt.Sprintf("sched_%d", time.Now().UnixNano())
	nextBilling := time.Now().Add(s.billingIntervalDuration(plan.BillingInterval))

	meta := map[string]interface{}{
		"plan_rank":          plan.PlanRank,
		"intro_enabled":      plan.IntroEnabled,
		"intro_periods":      plan.IntroPeriods,
		"billing_interval":   plan.BillingInterval,
		"subscription_price": plan.AmountCents,
	}
	metaBytes, _ := json.Marshal(meta)

	_, err := s.db.Exec(`
		INSERT INTO subscription_schedules (
			schedule_id, subscription_id, price_id, billing_interval,
			intro_enabled, intro_amount_cents, intro_periods, normal_amount_cents,
			next_billing_at, status, metadata, created_at, updated_at
		) VALUES ($1,$2,$3,$4,$5,$6,$7,$8,$9,'active',$10,NOW(),NOW())
		ON CONFLICT (schedule_id) DO UPDATE SET
			subscription_id = EXCLUDED.subscription_id,
			price_id = EXCLUDED.price_id,
			billing_interval = EXCLUDED.billing_interval,
			intro_enabled = EXCLUDED.intro_enabled,
			intro_amount_cents = EXCLUDED.intro_amount_cents,
			intro_periods = EXCLUDED.intro_periods,
			normal_amount_cents = EXCLUDED.normal_amount_cents,
			next_billing_at = EXCLUDED.next_billing_at,
			status = 'active',
			metadata = EXCLUDED.metadata,
			updated_at = NOW()
	`, scheduleID, subscriptionID, plan.StripePriceID, plan.BillingInterval,
		plan.IntroEnabled, plan.IntroAmountCents, plan.IntroPeriods, amountCents,
		nextBilling, string(metaBytes))
	if err != nil {
		return "", err
	}

	return scheduleID, nil
}

func (s *StripeService) billingIntervalDuration(interval string) time.Duration {
	switch interval {
	case "year":
		return 365 * 24 * time.Hour
	case "month":
		return 30 * 24 * time.Hour
	default:
		return 30 * 24 * time.Hour
	}
}

func (s *StripeService) handleCreditTopup(customerEmail string, amountCents int64, plan *PlanOption, metadata map[string]interface{}) error {
	if customerEmail == "" {
		return errors.New("customer email required for credit top-up")
	}

	if amountCents == 0 {
		amountCents = plan.AmountCents
	}

	bundle, err := s.planService.GetBundleProduct()
	if err != nil {
		return err
	}

	if amountCents == 0 || bundle == nil {
		return nil
	}

	credits := (bundle.CreditsPerUSD * amountCents) / 100
	if credits <= 0 {
		return nil
	}

	if metadata == nil {
		metadata = make(map[string]interface{})
	}
	metadata["price_id"] = plan.StripePriceID
	metadata["session_type"] = sessionTypeCreditsTopup

	return s.addCredits(customerEmail, credits, "credit_topup", metadata)
}

func (s *StripeService) addCredits(customerEmail string, amount int64, txnType string, metadata map[string]interface{}) error {
	if customerEmail == "" || amount <= 0 {
		return nil
	}

	_, err := s.db.Exec(`
		INSERT INTO credit_wallets (customer_email, balance_credits, updated_at)
		VALUES ($1, $2, NOW())
		ON CONFLICT (customer_email) DO UPDATE
		SET balance_credits = credit_wallets.balance_credits + $2, updated_at = NOW()
	`, customerEmail, amount)
	if err != nil {
		return err
	}

	metaBytes, _ := json.Marshal(metadata)
	_, err = s.db.Exec(`
		INSERT INTO credit_transactions (customer_email, amount_credits, transaction_type, metadata, created_at)
		VALUES ($1, $2, $3, $4, NOW())
	`, customerEmail, amount, txnType, string(metaBytes))
	return err
}

func (s *StripeService) handleSubscriptionCreated(obj map[string]interface{}) error {
	subscriptionID, ok := obj["id"].(string)
	if !ok {
		return errors.New("missing subscription id")
	}

	status, _ := obj["status"].(string)
	customerID, _ := obj["customer"].(string)

	_, err := s.db.Exec(`
		INSERT INTO subscriptions (subscription_id, customer_id, status, created_at, updated_at)
		VALUES ($1, $2, $3, $4, $5)
		ON CONFLICT (subscription_id) DO UPDATE
		SET status = $3, updated_at = $5
	`, subscriptionID, customerID, status, time.Now(), time.Now())

	return err
}

func (s *StripeService) handleSubscriptionUpdated(obj map[string]interface{}) error {
	subscriptionID, ok := obj["id"].(string)
	if !ok {
		return errors.New("missing subscription id")
	}

	status, _ := obj["status"].(string)

	// [REQ:SUB-CACHE] Cache invalidation - update subscription status
	_, err := s.db.Exec(`
		UPDATE subscriptions
		SET status = $1, updated_at = $2
		WHERE subscription_id = $3
	`, status, time.Now(), subscriptionID)

	return err
}

func (s *StripeService) handleSubscriptionDeleted(obj map[string]interface{}) error {
	subscriptionID, ok := obj["id"].(string)
	if !ok {
		return errors.New("missing subscription id")
	}

	_, err := s.db.Exec(`
		UPDATE subscriptions
		SET status = $1, canceled_at = $2, updated_at = $3
		WHERE subscription_id = $4
	`, "canceled", time.Now(), time.Now(), subscriptionID)

	return err
}

// VerifySubscription checks subscription status for a user
// [REQ:SUB-VERIFY] GET /api/subscription/verify endpoint
func (s *StripeService) VerifySubscription(userIdentity string) (map[string]interface{}, error) {
	var status string
	var canceledAt *time.Time
	var updatedAt time.Time

	// Query subscription by customer email or customer ID
	err := s.db.QueryRow(`
		SELECT status, canceled_at, updated_at
		FROM subscriptions
		WHERE customer_email = $1 OR customer_id = $1
		ORDER BY created_at DESC
		LIMIT 1
	`, userIdentity).Scan(&status, &canceledAt, &updatedAt)

	if err == sql.ErrNoRows {
		return map[string]interface{}{
			"status":  "inactive",
			"message": "No subscription found",
		}, nil
	}

	if err != nil {
		return nil, err
	}

	// [REQ:SUB-CACHE] Check cache freshness (max 60s lag from webhook)
	cacheAge := time.Since(updatedAt)
	if cacheAge > s.checkoutCacheTTL {
		logStructured("Subscription cache stale", map[string]interface{}{
			"level":         "warn",
			"user_identity": userIdentity,
			"cache_age_ms":  cacheAge.Milliseconds(),
		})
	}

	result := map[string]interface{}{
		"status":       status,
		"cached_at":    updatedAt,
		"cache_age_ms": cacheAge.Milliseconds(),
	}

	if canceledAt != nil {
		result["canceled_at"] = *canceledAt
	}

	return result, nil
}

// CancelSubscription cancels an active subscription
// [REQ:SUB-CANCEL] POST /api/subscription/cancel endpoint
func (s *StripeService) CancelSubscription(userIdentity string) (map[string]interface{}, error) {
	var subscriptionID string
	var status string

	// Find active subscription
	err := s.db.QueryRow(`
		SELECT subscription_id, status
		FROM subscriptions
		WHERE (customer_email = $1 OR customer_id = $1)
		AND status IN ('active', 'trialing')
		ORDER BY created_at DESC
		LIMIT 1
	`, userIdentity).Scan(&subscriptionID, &status)

	if err == sql.ErrNoRows {
		return nil, errors.New("no active subscription found")
	}

	if err != nil {
		return nil, err
	}

	// NOTE: In production, this would call Stripe API to cancel the subscription
	// For MVP, we update the database directly
	now := time.Now()
	_, err = s.db.Exec(`
		UPDATE subscriptions
		SET status = $1, canceled_at = $2, updated_at = $3
		WHERE subscription_id = $4
	`, "canceled", now, now, subscriptionID)

	if err != nil {
		return nil, err
	}

	return map[string]interface{}{
		"subscription_id": subscriptionID,
		"status":          "canceled",
		"canceled_at":     now,
		"message":         "Subscription canceled successfully",
	}, nil
}
