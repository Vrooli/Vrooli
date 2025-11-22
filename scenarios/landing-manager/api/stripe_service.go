package main

import (
	"crypto/hmac"
	"crypto/sha256"
	"database/sql"
	"encoding/hex"
	"encoding/json"
	"errors"
	"os"
	"strings"
	"time"
)

// StripeService handles Stripe payment integration
type StripeService struct {
	db                *sql.DB
	publishableKey    string
	secretKey         string
	webhookSecret     string
	checkoutCacheTTL  time.Duration
}

// NewStripeService creates a new Stripe service instance
func NewStripeService(db *sql.DB) *StripeService {
	publishableKey := os.Getenv("STRIPE_PUBLISHABLE_KEY")
	secretKey := os.Getenv("STRIPE_SECRET_KEY")
	webhookSecret := os.Getenv("STRIPE_WEBHOOK_SECRET")

	if publishableKey == "" {
		logStructured("STRIPE_PUBLISHABLE_KEY missing", map[string]interface{}{
			"level":   "fatal",
			"message": "STRIPE_PUBLISHABLE_KEY environment variable is required",
		})
		panic("STRIPE_PUBLISHABLE_KEY environment variable is required")
	}
	if secretKey == "" {
		logStructured("STRIPE_SECRET_KEY missing", map[string]interface{}{
			"level":   "fatal",
			"message": "STRIPE_SECRET_KEY environment variable is required",
		})
		panic("STRIPE_SECRET_KEY environment variable is required")
	}
	if webhookSecret == "" {
		logStructured("STRIPE_WEBHOOK_SECRET missing", map[string]interface{}{
			"level":   "fatal",
			"message": "STRIPE_WEBHOOK_SECRET environment variable is required",
		})
		panic("STRIPE_WEBHOOK_SECRET environment variable is required")
	}

	return &StripeService{
		db:                db,
		publishableKey:    publishableKey,
		secretKey:         secretKey,
		webhookSecret:     webhookSecret,
		checkoutCacheTTL:  60 * time.Second,
	}
}

// CreateCheckoutSession creates a Stripe checkout session
// [REQ:STRIPE-ROUTES] POST /api/checkout/create endpoint
func (s *StripeService) CreateCheckoutSession(priceID string, successURL string, cancelURL string, customerEmail string) (map[string]interface{}, error) {
	// [REQ:STRIPE-CONFIG] Uses Stripe keys from environment
	if s.secretKey == "sk_test_placeholder" {
		return nil, errors.New("Stripe not configured - missing STRIPE_SECRET_KEY")
	}

	// NOTE: In production, this would call the actual Stripe API
	// For MVP, we return a mock session for testing
	sessionID := "cs_test_" + time.Now().Format("20060102150405")
	checkoutURL := "https://checkout.stripe.com/c/pay/" + sessionID

	session := map[string]interface{}{
		"id":            sessionID,
		"url":           checkoutURL,
		"customer":      customerEmail,
		"amount_total":  5000, // $50.00 in cents
		"currency":      "usd",
		"status":        "open",
		"created":       time.Now().Unix(),
		"success_url":   successURL,
		"cancel_url":    cancelURL,
		"publishable_key": s.publishableKey,
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
	// [REQ:STRIPE-CONFIG] Uses webhook secret from environment
	if s.webhookSecret == "whsec_placeholder" {
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
	mac := hmac.New(sha256.New, []byte(s.webhookSecret))
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

	// Update checkout session status
	_, err := s.db.Exec(`
		UPDATE checkout_sessions
		SET status = $1, subscription_id = $2, updated_at = $3
		WHERE session_id = $4
	`, "complete", subscriptionID, time.Now(), sessionID)

	if err != nil {
		return err
	}

	// Create or update subscription record
	if subscriptionID != "" {
		_, err = s.db.Exec(`
			INSERT INTO subscriptions (subscription_id, customer_email, status, created_at, updated_at)
			VALUES ($1, $2, $3, $4, $5)
			ON CONFLICT (subscription_id) DO UPDATE
			SET status = $3, updated_at = $5
		`, subscriptionID, customerEmail, "active", time.Now(), time.Now())

		if err != nil {
			return err
		}
	}

	logStructured("Checkout session completed", map[string]interface{}{
		"session_id":      sessionID,
		"customer_email":  customerEmail,
		"subscription_id": subscriptionID,
	})

	return nil
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
		"status":      status,
		"cached_at":   updatedAt,
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
