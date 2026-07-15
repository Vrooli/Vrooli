// Package stripe is the fully-simulated Stripe payments integration: in-process
// checkout session creation, a hand-rolled HMAC-SHA256 webhook verifier and
// dispatcher, subscription lifecycle handling, intro-pricing schedules, and
// credit top-ups. No stripe-go SDK is used; ids and amounts are simulated.
// Credentials resolve from the admin-configured payment_settings row, then the
// environment. The Connect handler in handlers/payments adapts this Service.
package stripe

import (
	"context"
	"crypto/hmac"
	"crypto/sha256"
	"database/sql"
	"encoding/hex"
	"encoding/json"
	"errors"
	"fmt"
	"landing-page-react-vite-api/internal/paymentsettings"
	"landing-page-react-vite-api/internal/plan"
	"log"
	"os"
	"strconv"
	"strings"
	"sync"
	"time"

	"google.golang.org/protobuf/types/known/timestamppb"

	landingv1 "github.com/vrooli/vrooli/packages/proto/gen/go/landing-page-react-vite/v1"
)

// Service handles the simulated Stripe payment integration.
type Service struct {
	db               *sql.DB
	plan             *plan.Service
	settings         *paymentsettings.Service
	checkoutCacheTTL time.Duration
	mu               sync.RWMutex
	runtimeConfig    runtimeConfig
}

type runtimeConfig struct {
	publishableKey string
	secretKey      string
	webhookSecret  string
	source         string
	hasPublishable bool
	hasSecret      bool
	hasWebhook     bool
}

const (
	sessionTypeCreditsTopup = "credits_topup"
	sessionTypeSubscription = "subscription"
)

// NewService constructs the stripe Service and loads the initial credentials.
func NewService(db *sql.DB, planSvc *plan.Service, settings *paymentsettings.Service) *Service {
	if planSvc == nil {
		planSvc = plan.NewService(db)
	}
	if settings == nil {
		settings = paymentsettings.NewService(db)
	}
	s := &Service{
		db:               db,
		plan:             planSvc,
		settings:         settings,
		checkoutCacheTTL: 60 * time.Second,
	}
	if err := s.RefreshConfig(context.Background()); err != nil {
		log.Printf("stripe: failed to initialize config: %v", err)
	}
	return s
}

// RefreshConfig reloads Stripe credentials from the database/environment.
func (s *Service) RefreshConfig(ctx context.Context) error {
	cfg, err := s.loadConfig(ctx)
	if err != nil {
		return err
	}
	s.mu.Lock()
	s.runtimeConfig = cfg
	s.mu.Unlock()
	return nil
}

func (s *Service) loadConfig(ctx context.Context) (runtimeConfig, error) {
	cfg := runtimeConfig{source: "env"}

	if s.settings != nil {
		record, err := s.settings.GetStripeSettings(ctx)
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
	}
	if !cfg.hasSecret {
		cfg.secretKey = "sk_test_placeholder"
	}
	if !cfg.hasWebhook {
		cfg.webhookSecret = "whsec_placeholder"
	}
	return cfg, nil
}

func (s *Service) getConfig() runtimeConfig {
	s.mu.RLock()
	defer s.mu.RUnlock()
	return s.runtimeConfig
}

func maskValue(value string) string {
	if value == "" || len(value) <= 6 {
		return value
	}
	return value[:4] + "…" + value[len(value)-2:]
}

// ConfigSnapshot returns a redacted view of the active Stripe configuration.
func (s *Service) ConfigSnapshot() *landingv1.StripeConfigSnapshot {
	cfg := s.getConfig()
	source := landingv1.ConfigSource_CONFIG_SOURCE_ENV
	if cfg.source == "database" {
		source = landingv1.ConfigSource_CONFIG_SOURCE_DATABASE
	}
	return &landingv1.StripeConfigSnapshot{
		PublishableKeyPreview: maskValue(cfg.publishableKey),
		PublishableKeySet:     cfg.hasPublishable,
		SecretKeySet:          cfg.hasSecret,
		WebhookSecretSet:      cfg.hasWebhook,
		Source:                source,
	}
}

// CreateCheckoutSession creates a simulated Stripe checkout session.
func (s *Service) CreateCheckoutSession(priceID, successURL, cancelURL, customerEmail string) (*landingv1.CheckoutSession, error) {
	cfg := s.getConfig()
	if !cfg.hasSecret {
		return nil, errors.New("Stripe not configured - missing STRIPE_SECRET_KEY")
	}

	sessionID := fmt.Sprintf("cs_test_%d", time.Now().UnixNano())
	session := &landingv1.CheckoutSession{
		SessionId:      sessionID,
		SessionKind:    landingv1.SessionKind_SESSION_KIND_SUBSCRIPTION,
		Status:         landingv1.CheckoutSessionStatus_CHECKOUT_SESSION_STATUS_OPEN,
		Url:            "https://checkout.stripe.com/c/pay/" + sessionID,
		PublishableKey: cfg.publishableKey,
		CustomerEmail:  customerEmail,
		StripePriceId:  priceID,
		AmountCents:    5000, // $50.00
		Currency:       "usd",
		SuccessUrl:     successURL,
		CancelUrl:      cancelURL,
		CreatedAt:      timestamppb.Now(),
	}

	if _, err := s.db.Exec(`
		INSERT INTO checkout_sessions (session_id, customer_email, price_id, status, created_at)
		VALUES ($1, $2, $3, $4, $5)`,
		sessionID, customerEmail, priceID, "open", time.Now()); err != nil {
		return nil, err
	}
	return session, nil
}

// VerifyWebhookSignature validates the simulated Stripe webhook signature
// (HMAC-SHA256 over "timestamp.payload", header format "t=…,v1=…").
func (s *Service) VerifyWebhookSignature(payload []byte, signature string) bool {
	cfg := s.getConfig()
	if !cfg.hasWebhook {
		return false
	}
	var timestamp, sig string
	for _, part := range strings.Split(signature, ",") {
		switch {
		case strings.HasPrefix(part, "t="):
			timestamp = strings.TrimPrefix(part, "t=")
		case strings.HasPrefix(part, "v1="):
			sig = strings.TrimPrefix(part, "v1=")
		}
	}
	if timestamp == "" || sig == "" {
		return false
	}
	mac := hmac.New(sha256.New, []byte(cfg.webhookSecret))
	mac.Write([]byte(timestamp + "." + string(payload)))
	expectedSig := hex.EncodeToString(mac.Sum(nil))
	return hmac.Equal([]byte(sig), []byte(expectedSig))
}

// HandleWebhook verifies and dispatches a simulated Stripe webhook event.
func (s *Service) HandleWebhook(body []byte, signature string) error {
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
		log.Printf("stripe: unhandled webhook event %q", eventType)
	}
	return nil
}

func (s *Service) handleCheckoutCompleted(obj map[string]interface{}) error {
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
		return nil
	}
	if _, err := s.db.Exec(`
		UPDATE checkout_sessions SET status = $1, subscription_id = $2, updated_at = $3 WHERE session_id = $4`,
		"complete", subscriptionID, time.Now(), sessionID); err != nil {
		return err
	}

	var planOption *landingv1.PlanOption
	if sessionRec.PriceID.Valid {
		if p, planErr := s.plan.GetPlanByPriceID(sessionRec.PriceID.String); planErr == nil {
			planOption = p
		} else {
			log.Printf("stripe: plan metadata missing for price %s: %v", sessionRec.PriceID.String, planErr)
		}
	}
	amountCents := s.extractAmount(obj, sessionRec)

	switch {
	case planOption != nil && planOption.Kind == landingv1.PlanKind_PLAN_KIND_CREDITS_TOPUP:
		return s.handleCreditTopup(customerEmail, amountCents, planOption, map[string]interface{}{"session_id": sessionID})
	case planOption != nil && planOption.Kind == landingv1.PlanKind_PLAN_KIND_SUPPORTER_CONTRIBUTION:
		log.Printf("stripe: supporter contribution session=%s email=%s amount=%d", sessionID, customerEmail, amountCents)
		return nil
	default:
		return s.handleSubscriptionCompletion(subscriptionID, customerEmail, planOption, sessionRec, amountCents)
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

func (s *Service) loadCheckoutSession(sessionID string) (*checkoutSessionRecord, error) {
	record := &checkoutSessionRecord{}
	err := s.db.QueryRow(`
		SELECT session_id, status, price_id, session_type, amount_cents, schedule_id
		FROM checkout_sessions WHERE session_id = $1`, sessionID).Scan(
		&record.SessionID, &record.Status, &record.PriceID, &record.SessionType,
		&record.AmountCents, &record.ScheduleID)
	if err != nil {
		return nil, err
	}
	return record, nil
}

func (s *Service) extractAmount(obj map[string]interface{}, session *checkoutSessionRecord) int64 {
	if amount := parseStripeAmount(obj["amount_total"]); amount != 0 {
		return amount
	}
	if amount := parseStripeAmount(obj["amount"]); amount != 0 {
		return amount
	}
	if session != nil && session.AmountCents.Valid {
		return session.AmountCents.Int64
	}
	return 0
}

func parseStripeAmount(value interface{}) int64 {
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

func (s *Service) handleSubscriptionCompletion(subscriptionID, customerEmail string, planOption *landingv1.PlanOption, session *checkoutSessionRecord, amountCents int64) error {
	if planOption == nil {
		return nil
	}
	if subscriptionID == "" {
		return errors.New("subscription id required for subscription completion")
	}
	if amountCents == 0 {
		amountCents = planOption.AmountCents
	}

	now := time.Now()
	if _, err := s.db.Exec(`
		INSERT INTO subscriptions (subscription_id, customer_email, status, plan_tier, price_id, bundle_key, created_at, updated_at)
		VALUES ($1, $2, $3, $4, $5, $6, $7, $8)
		ON CONFLICT (subscription_id) DO UPDATE
		SET status = $3, customer_email = $2, plan_tier = $4, price_id = $5, bundle_key = $6, updated_at = $8`,
		subscriptionID, customerEmail, "active", planOption.PlanTier, planOption.StripePriceId, planOption.BundleKey, now, now); err != nil {
		return err
	}

	if planOption.IntroEnabled && planOption.BillingInterval == landingv1.BillingInterval_BILLING_INTERVAL_MONTH {
		scheduleID, err := s.createSubscriptionSchedule(subscriptionID, planOption, amountCents)
		if err != nil {
			return err
		}
		if scheduleID != "" && session != nil {
			if _, err := s.db.Exec(`UPDATE checkout_sessions SET schedule_id = $1 WHERE session_id = $2`, scheduleID, session.SessionID); err != nil {
				return err
			}
		}
	}
	return nil
}

func (s *Service) createSubscriptionSchedule(subscriptionID string, planOption *landingv1.PlanOption, amountCents int64) (string, error) {
	if planOption == nil || subscriptionID == "" {
		return "", nil
	}
	if amountCents == 0 {
		amountCents = planOption.AmountCents
	}
	scheduleID := fmt.Sprintf("sched_%d", time.Now().UnixNano())
	interval := plan.BillingIntervalString(planOption.BillingInterval)
	nextBilling := time.Now().Add(billingIntervalDuration(interval))

	meta := map[string]interface{}{
		"plan_rank":          planOption.PlanRank,
		"intro_enabled":      planOption.IntroEnabled,
		"intro_periods":      planOption.IntroPeriods,
		"billing_interval":   interval,
		"subscription_price": planOption.AmountCents,
	}
	metaBytes, _ := json.Marshal(meta)

	if _, err := s.db.Exec(`
		INSERT INTO subscription_schedules (
			schedule_id, subscription_id, price_id, billing_interval,
			intro_enabled, intro_amount_cents, intro_periods, normal_amount_cents,
			next_billing_at, status, metadata, created_at, updated_at
		) VALUES ($1,$2,$3,$4,$5,$6,$7,$8,$9,'active',$10,NOW(),NOW())
		ON CONFLICT (schedule_id) DO UPDATE SET
			subscription_id = EXCLUDED.subscription_id, price_id = EXCLUDED.price_id,
			billing_interval = EXCLUDED.billing_interval, intro_enabled = EXCLUDED.intro_enabled,
			intro_amount_cents = EXCLUDED.intro_amount_cents, intro_periods = EXCLUDED.intro_periods,
			normal_amount_cents = EXCLUDED.normal_amount_cents, next_billing_at = EXCLUDED.next_billing_at,
			status = 'active', metadata = EXCLUDED.metadata, updated_at = NOW()`,
		scheduleID, subscriptionID, planOption.StripePriceId, interval,
		planOption.IntroEnabled, planOption.IntroAmountCents, planOption.IntroPeriods, amountCents,
		nextBilling, string(metaBytes)); err != nil {
		return "", err
	}
	return scheduleID, nil
}

func billingIntervalDuration(interval string) time.Duration {
	if interval == "year" {
		return 365 * 24 * time.Hour
	}
	return 30 * 24 * time.Hour
}

func (s *Service) handleCreditTopup(customerEmail string, amountCents int64, planOption *landingv1.PlanOption, metadata map[string]interface{}) error {
	if customerEmail == "" {
		return errors.New("customer email required for credit top-up")
	}
	if amountCents == 0 {
		amountCents = planOption.AmountCents
	}
	bundle, err := s.plan.GetBundleProduct()
	if err != nil {
		return err
	}
	if amountCents == 0 || bundle == nil {
		return nil
	}
	credits := (bundle.CreditsPerUsd * amountCents) / 100
	if credits <= 0 {
		return nil
	}
	if metadata == nil {
		metadata = make(map[string]interface{})
	}
	metadata["price_id"] = planOption.StripePriceId
	metadata["session_type"] = sessionTypeCreditsTopup
	return s.addCredits(customerEmail, credits, "credit_topup", metadata)
}

func (s *Service) addCredits(customerEmail string, amount int64, txnType string, metadata map[string]interface{}) error {
	if customerEmail == "" || amount <= 0 {
		return nil
	}
	if _, err := s.db.Exec(`
		INSERT INTO credit_wallets (customer_email, balance_credits, updated_at)
		VALUES ($1, $2, NOW())
		ON CONFLICT (customer_email) DO UPDATE
		SET balance_credits = credit_wallets.balance_credits + $2, updated_at = NOW()`,
		customerEmail, amount); err != nil {
		return err
	}
	metaBytes, _ := json.Marshal(metadata)
	_, err := s.db.Exec(`
		INSERT INTO credit_transactions (customer_email, amount_credits, transaction_type, metadata, created_at)
		VALUES ($1, $2, $3, $4, NOW())`,
		customerEmail, amount, txnType, string(metaBytes))
	return err
}

func (s *Service) handleSubscriptionCreated(obj map[string]interface{}) error {
	subscriptionID, ok := obj["id"].(string)
	if !ok {
		return errors.New("missing subscription id")
	}
	status, _ := obj["status"].(string)
	customerID, _ := obj["customer"].(string)
	_, err := s.db.Exec(`
		INSERT INTO subscriptions (subscription_id, customer_id, status, created_at, updated_at)
		VALUES ($1, $2, $3, $4, $5)
		ON CONFLICT (subscription_id) DO UPDATE SET status = $3, updated_at = $5`,
		subscriptionID, customerID, status, time.Now(), time.Now())
	return err
}

func (s *Service) handleSubscriptionUpdated(obj map[string]interface{}) error {
	subscriptionID, ok := obj["id"].(string)
	if !ok {
		return errors.New("missing subscription id")
	}
	status, _ := obj["status"].(string)
	_, err := s.db.Exec(`UPDATE subscriptions SET status = $1, updated_at = $2 WHERE subscription_id = $3`,
		status, time.Now(), subscriptionID)
	return err
}

func (s *Service) handleSubscriptionDeleted(obj map[string]interface{}) error {
	subscriptionID, ok := obj["id"].(string)
	if !ok {
		return errors.New("missing subscription id")
	}
	_, err := s.db.Exec(`UPDATE subscriptions SET status = $1, canceled_at = $2, updated_at = $3 WHERE subscription_id = $4`,
		"canceled", time.Now(), time.Now(), subscriptionID)
	return err
}

// VerifySubscription checks the cached subscription status for a user identity.
func (s *Service) VerifySubscription(userIdentity string) (*landingv1.SubscriptionStatus, error) {
	var status string
	var canceledAt *time.Time
	var updatedAt time.Time
	err := s.db.QueryRow(`
		SELECT status, canceled_at, updated_at FROM subscriptions
		WHERE customer_email = $1 OR customer_id = $1
		ORDER BY created_at DESC LIMIT 1`, userIdentity).Scan(&status, &canceledAt, &updatedAt)
	if err == sql.ErrNoRows {
		message := "No subscription found"
		return &landingv1.SubscriptionStatus{
			State:        landingv1.SubscriptionState_SUBSCRIPTION_STATE_INACTIVE,
			UserIdentity: userIdentity,
			Message:      &message,
		}, nil
	}
	if err != nil {
		return nil, err
	}

	cacheAge := time.Since(updatedAt)
	if cacheAge > s.checkoutCacheTTL {
		log.Printf("stripe: subscription cache stale for %s (age %dms)", userIdentity, cacheAge.Milliseconds())
	}

	result := &landingv1.SubscriptionStatus{
		State:        mapState(status),
		UserIdentity: userIdentity,
		CachedAt:     timestamppb.New(updatedAt),
		CacheAgeMs:   cacheAge.Milliseconds(),
	}
	if canceledAt != nil {
		result.CanceledAt = timestamppb.New(*canceledAt)
	}
	return result, nil
}

// CancelSubscription cancels an active subscription for a user identity.
func (s *Service) CancelSubscription(userIdentity string) (*landingv1.CancelSubscriptionResponse, error) {
	var subscriptionID, status string
	err := s.db.QueryRow(`
		SELECT subscription_id, status FROM subscriptions
		WHERE (customer_email = $1 OR customer_id = $1) AND status IN ('active', 'trialing')
		ORDER BY created_at DESC LIMIT 1`, userIdentity).Scan(&subscriptionID, &status)
	if err == sql.ErrNoRows {
		return nil, errors.New("no active subscription found")
	}
	if err != nil {
		return nil, err
	}

	now := time.Now()
	if _, err := s.db.Exec(`UPDATE subscriptions SET status = $1, canceled_at = $2, updated_at = $3 WHERE subscription_id = $4`,
		"canceled", now, now, subscriptionID); err != nil {
		return nil, err
	}

	message := "Subscription canceled successfully"
	return &landingv1.CancelSubscriptionResponse{
		SubscriptionId: &subscriptionID,
		State:          landingv1.SubscriptionState_SUBSCRIPTION_STATE_CANCELED,
		CanceledAt:     timestamppb.New(now),
		Message:        &message,
	}, nil
}

func mapState(status string) landingv1.SubscriptionState {
	switch status {
	case "active":
		return landingv1.SubscriptionState_SUBSCRIPTION_STATE_ACTIVE
	case "trialing":
		return landingv1.SubscriptionState_SUBSCRIPTION_STATE_TRIALING
	case "past_due":
		return landingv1.SubscriptionState_SUBSCRIPTION_STATE_PAST_DUE
	case "canceled":
		return landingv1.SubscriptionState_SUBSCRIPTION_STATE_CANCELED
	default:
		return landingv1.SubscriptionState_SUBSCRIPTION_STATE_INACTIVE
	}
}
