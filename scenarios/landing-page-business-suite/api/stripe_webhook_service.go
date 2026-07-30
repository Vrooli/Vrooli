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
	"strconv"
	"strings"
	"time"

	shared "github.com/vrooli/vrooli/packages/proto/gen/go/landing-page-business-suite/v1/shared"
	"landing-page-business-suite-api/internal/commerce"
)

// webhookTimestampTolerance defines the maximum age of webhook timestamps
// to prevent replay attacks. Stripe recommends 5 minutes.
const webhookTimestampTolerance = 5 * time.Minute

// --- StripeWebhookService Interface Implementation ---
// This file contains webhook signature verification and event handling.

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

	// Validate timestamp is within acceptable window to prevent replay attacks
	timestampInt, err := strconv.ParseInt(timestamp, 10, 64)
	if err != nil {
		logStructuredError("webhook_timestamp_invalid", map[string]interface{}{
			"timestamp": timestamp,
			"error":     err.Error(),
		})
		return false
	}
	eventTime := time.Unix(timestampInt, 0)
	age := time.Since(eventTime)
	if age < 0 {
		age = -age // Handle future timestamps (clock skew)
	}
	if age > webhookTimestampTolerance {
		logStructuredError("webhook_timestamp_out_of_range", map[string]interface{}{
			"timestamp":    timestamp,
			"event_time":   eventTime.Format(time.RFC3339),
			"age_seconds":  age.Seconds(),
			"tolerance_ms": webhookTimestampTolerance.Milliseconds(),
		})
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

	// Extract event ID for idempotency - required for safe webhook processing
	eventID, ok := event["id"].(string)
	if !ok || eventID == "" {
		logStructuredError("webhook_missing_event_id", map[string]interface{}{
			"event_type": eventType,
		})
		return errors.New("missing or invalid event ID - cannot process webhook safely")
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
		return s.handleCheckoutCompleted(obj, eventID)
	case "customer.subscription.created":
		return s.handleSubscriptionCreated(obj)
	case "customer.subscription.updated":
		return s.handleSubscriptionUpdated(obj)
	case "customer.subscription.deleted":
		return s.handleSubscriptionDeleted(obj)
	case "customer.updated":
		return s.handleCustomerUpdated(obj)
	case "invoice.paid":
		return s.handleInvoicePaid(obj)
	case "invoice.payment_failed":
		return s.handleInvoicePaymentFailed(obj)
	default:
		logStructured("Unhandled webhook event", map[string]interface{}{
			"event_type": eventType,
		})
	}

	return nil
}

func (s *StripeService) handleCheckoutCompleted(obj map[string]interface{}, stripeEventID string) error {
	sessionID, ok := obj["id"].(string)
	if !ok {
		return errors.New("missing session id")
	}

	customerEmail, _ := obj["customer_email"].(string)
	customerID, _ := obj["customer"].(string)
	subscriptionID, _ := obj["subscription"].(string)

	// Normalize email for consistency
	customerEmail = NormalizeEmail(customerEmail)

	// Link user account to Stripe customer (creates user if not exists)
	if customerEmail != "" && customerID != "" {
		if err := commerce.NewAccountLinkService(s.db).LinkUserToStripeCustomer(customerEmail, customerID); err != nil {
			logStructuredError("link_stripe_customer_failed", map[string]interface{}{
				"email":       customerEmail,
				"customer_id": customerID,
				"error":       err.Error(),
			})
			// Continue - don't fail checkout for this
		} else {
			logStructured("stripe_customer_linked", map[string]interface{}{
				"level":       "info",
				"email":       customerEmail,
				"customer_id": customerID,
			})
		}
	}

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

	var plan *commerce.PlanOption
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
	case plan != nil && plan.Kind == shared.PlanKind_PLAN_KIND_CREDITS_TOPUP:
		if _, err := s.db.Exec(`
			UPDATE checkout_sessions
			SET status = $1, subscription_id = $2, customer_id = $3, customer_email = $4, updated_at = $5
			WHERE session_id = $6
		`, "complete", subscriptionID, customerID, customerEmail, time.Now(), sessionID); err != nil {
			return err
		}
		return s.handleCreditTopup(customerEmail, amountCents, plan, stripeEventID, map[string]interface{}{
			"session_id": sessionID,
		})
	case plan != nil && plan.Kind == shared.PlanKind_PLAN_KIND_SUPPORTER_CONTRIBUTION:
		if _, err := s.db.Exec(`
			UPDATE checkout_sessions
			SET status = $1, subscription_id = $2, customer_id = $3, customer_email = $4, updated_at = $5
			WHERE session_id = $6
		`, "complete", subscriptionID, customerID, customerEmail, time.Now(), sessionID); err != nil {
			return err
		}
		logStructured("supporter contribution received", map[string]interface{}{
			"session_id": sessionID,
			"email":      customerEmail,
			"amount":     amountCents,
		})
		return nil
	default:
		return WithTransaction(context.Background(), s.db, nil, func(tx *sql.Tx) error {
			if _, err := tx.Exec(`
				UPDATE checkout_sessions
				SET status = $1, subscription_id = $2, customer_id = $3, customer_email = $4, updated_at = $5
				WHERE session_id = $6
			`, "complete", subscriptionID, customerID, customerEmail, time.Now(), sessionID); err != nil {
				return err
			}
			return s.handleSubscriptionCompletion(tx, subscriptionID, customerID, customerEmail, plan, sessionRec, amountCents)
		})
	}
}

func (s *StripeService) handleSubscriptionCompletion(tx *sql.Tx, subscriptionID, customerID, customerEmail string, plan *commerce.PlanOption, session *checkoutSessionRecord, amountCents int64) error {
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
	_, err := tx.Exec(`
		INSERT INTO subscriptions (subscription_id, customer_id, customer_email, status, plan_tier, price_id, bundle_key, created_at, updated_at)
		VALUES ($1, $2, $3, $4, $5, $6, $7, $8, $9)
		ON CONFLICT (subscription_id) DO UPDATE
		SET status = $4, customer_email = $3, customer_id = $2, plan_tier = $5, price_id = $6, bundle_key = $7, updated_at = $9
	`, subscriptionID, customerID, customerEmail, "active", plan.PlanTier, plan.StripePriceId, plan.BundleKey, now, now)
	if err != nil {
		return err
	}

	if plan.IntroEnabled && plan.BillingInterval == shared.BillingInterval_BILLING_INTERVAL_MONTH {
		scheduleID, err := s.createSubscriptionSchedule(tx, subscriptionID, plan, amountCents)
		if err != nil {
			return err
		}
		if scheduleID != "" && session != nil {
			if _, err := tx.Exec(`
				UPDATE checkout_sessions
				SET schedule_id = $1
				WHERE session_id = $2
			`, scheduleID, session.SessionID); err != nil {
				return err
			}
		}
	}

	meta := map[string]interface{}{
		"customer_email":  customerEmail,
		"customer_id":     customerID,
		"subscription_id": subscriptionID,
		"plan_tier":       plan.PlanTier,
		"price_id":        plan.StripePriceId,
		"session_type":    sessionTypeSubscription,
	}
	if session != nil {
		meta["session_id"] = session.SessionID
	}
	logStructured("Checkout session completed", meta)

	return nil
}

func (s *StripeService) createSubscriptionSchedule(tx *sql.Tx, subscriptionID string, plan *commerce.PlanOption, amountCents int64) (string, error) {
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
		"billing_interval":   commerce.BillingIntervalLabel(plan.BillingInterval),
		"subscription_price": plan.AmountCents,
	}
	metaBytes, _ := json.Marshal(meta)

	_, err := tx.Exec(`
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
	`, scheduleID, subscriptionID, plan.StripePriceId, commerce.BillingIntervalLabel(plan.BillingInterval),
		plan.IntroEnabled, plan.IntroAmountCents, plan.IntroPeriods, amountCents,
		nextBilling, string(metaBytes))
	if err != nil {
		return "", err
	}

	return scheduleID, nil
}

func (s *StripeService) billingIntervalDuration(interval shared.BillingInterval) time.Duration {
	switch interval {
	case shared.BillingInterval_BILLING_INTERVAL_YEAR:
		return 365 * 24 * time.Hour
	case shared.BillingInterval_BILLING_INTERVAL_MONTH:
		return 30 * 24 * time.Hour
	default:
		return 30 * 24 * time.Hour
	}
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
	if payload, err := json.Marshal(obj); err == nil {
		var sub stripeSubscription
		if err := json.Unmarshal(payload, &sub); err == nil && sub.ID != "" {
			if _, persistErr := s.persistSubscriptionFromStripe("", &sub); persistErr == nil {
				return nil
			}
		}
	}

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

	now := time.Now()
	if payload, err := json.Marshal(obj); err == nil {
		var sub stripeSubscription
		if err := json.Unmarshal(payload, &sub); err == nil && sub.ID != "" {
			sub.Status = "canceled"
			sub.CanceledAt = now.Unix()
			if _, persistErr := s.persistSubscriptionFromStripe("", &sub); persistErr == nil {
				return nil
			}
		}
	}

	_, err := s.db.Exec(`
		UPDATE subscriptions
		SET status = $1, canceled_at = $2, updated_at = $3
		WHERE subscription_id = $4
	`, "canceled", now, now, subscriptionID)

	return err
}

func (s *StripeService) extractInvoicePriceID(obj map[string]interface{}) string {
	// Invoice price may live under lines.data[0].price.id
	lines, ok := obj["lines"].(map[string]interface{})
	if !ok {
		return ""
	}
	rawData, ok := lines["data"].([]interface{})
	if !ok || len(rawData) == 0 {
		return ""
	}
	first, ok := rawData[0].(map[string]interface{})
	if !ok {
		return ""
	}
	price, ok := first["price"].(map[string]interface{})
	if !ok {
		return ""
	}
	if id, ok := price["id"].(string); ok {
		return id
	}
	return ""
}

func (s *StripeService) persistInvoiceStatus(subscriptionID, customerID, customerEmail, priceID, status string) error {
	if subscriptionID == "" {
		return nil
	}

	planTier := ""
	bundleKey := s.planService.BundleKey()
	if priceID != "" {
		if plan, err := s.planService.GetPlanByPriceID(priceID); err == nil {
			planTier = plan.PlanTier
			if plan.BundleKey != "" {
				bundleKey = plan.BundleKey
			}
		} else {
			logStructured("stripe_plan_lookup_failed", map[string]interface{}{
				"level":    "warn",
				"price_id": priceID,
				"error":    err.Error(),
			})
		}
	}

	if strings.TrimSpace(planTier) == "" {
		var current sql.NullString
		if err := s.db.QueryRow(`SELECT plan_tier FROM subscriptions WHERE subscription_id = $1`, subscriptionID).Scan(&current); err == nil && current.Valid {
			planTier = current.String
		}
	}
	if strings.TrimSpace(planTier) == "" && strings.TrimSpace(priceID) != "" {
		if inferred, ok := commerce.DetectTierToken(priceID); ok {
			planTier = inferred
		}
	}
	if strings.TrimSpace(planTier) != "" {
		if _, err := commerce.NormalizePlanTier(planTier); err != nil {
			logStructured("stripe_subscription_plan_tier_invalid", map[string]interface{}{
				"level":        "warn",
				"plan_tier":    planTier,
				"price_id":     priceID,
				"subscription": subscriptionID,
			})
			planTier = ""
		}
	}

	now := time.Now()
	_, err := s.db.Exec(`
		INSERT INTO subscriptions (subscription_id, customer_id, customer_email, status, plan_tier, price_id, bundle_key, created_at, updated_at)
		VALUES ($1::varchar,$2::varchar,$3::varchar,$4::varchar,$5::varchar,$6::varchar,$7::varchar,COALESCE((SELECT created_at FROM subscriptions WHERE subscription_id = $1::varchar), $8::timestamp), $8::timestamp)
		ON CONFLICT (subscription_id) DO UPDATE SET
			customer_id = COALESCE(EXCLUDED.customer_id, subscriptions.customer_id),
			customer_email = COALESCE(EXCLUDED.customer_email, subscriptions.customer_email),
			status = $4,
			plan_tier = COALESCE(NULLIF($5,''), subscriptions.plan_tier),
			price_id = COALESCE(NULLIF($6,''), subscriptions.price_id),
			bundle_key = COALESCE(NULLIF($7,''), subscriptions.bundle_key),
			updated_at = $8
	`, subscriptionID, customerID, customerEmail, status, planTier, priceID, bundleKey, now)
	return err
}

func (s *StripeService) handleInvoicePaid(obj map[string]interface{}) error {
	subscriptionID, _ := obj["subscription"].(string)
	customerID, _ := obj["customer"].(string)
	customerEmail, _ := obj["customer_email"].(string)
	priceID := s.extractInvoicePriceID(obj)
	billingReason, _ := obj["billing_reason"].(string)

	if err := s.persistInvoiceStatus(subscriptionID, customerID, customerEmail, priceID, "active"); err != nil {
		return err
	}

	// Check if this is a subscription creation with an intro coupon
	// billing_reason == "subscription_create" indicates first invoice
	if billingReason == "subscription_create" && customerEmail != "" {
		couponID := s.extractIntroCouponFromInvoice(obj)
		if couponID != "" {
			// Payment-time eligibility re-check: verify user is still eligible
			// This catches cases where eligibility changed between checkout and payment
			eligible, eligErr := s.checkIntroEligibility(context.Background(), customerEmail)
			if eligErr != nil {
				logStructuredError("payment_time_eligibility_check_failed", map[string]interface{}{
					"email":           customerEmail,
					"customer_id":     customerID,
					"coupon_id":       couponID,
					"subscription_id": subscriptionID,
					"error":           eligErr.Error(),
				})
				// Continue processing - eligibility check failure shouldn't block payment
			} else if !eligible {
				// User was ineligible at payment time but coupon was already applied
				// Log anomaly for admin review
				s.logIntroAnomaly(customerEmail, customerID, couponID, "ineligible_at_payment", map[string]interface{}{
					"subscription_id": subscriptionID,
					"price_id":        priceID,
					"billing_reason":  billingReason,
				})
			}

			// Determine plan tier from price
			planTier := ""
			if priceID != "" {
				if plan, err := s.planService.GetPlanByPriceID(priceID); err == nil {
					planTier = plan.PlanTier
				}
			}

			if err := s.markIntroUsed(context.Background(), customerEmail, customerID, couponID, planTier, subscriptionID); err != nil {
				logStructuredError("mark_intro_used_failed", map[string]interface{}{
					"email":           customerEmail,
					"customer_id":     customerID,
					"coupon_id":       couponID,
					"subscription_id": subscriptionID,
					"error":           err.Error(),
				})
				// Don't fail the webhook - intro tracking is non-critical
			}
		}
	}

	// Proactively refresh from Stripe if we have an ID to backfill details
	if subscriptionID != "" {
		if refreshed, err := s.refreshSubscriptionFromStripe(customerEmail, subscriptionID); err == nil && refreshed != nil {
			return nil
		}
	}
	return nil
}

func (s *StripeService) handleInvoicePaymentFailed(obj map[string]interface{}) error {
	subscriptionID, _ := obj["subscription"].(string)
	customerID, _ := obj["customer"].(string)
	customerEmail, _ := obj["customer_email"].(string)
	priceID := s.extractInvoicePriceID(obj)

	if err := s.persistInvoiceStatus(subscriptionID, customerID, customerEmail, priceID, "past_due"); err != nil {
		return err
	}
	return nil
}

// handleCustomerUpdated handles the customer.updated webhook event.
// When a user changes their email in the Stripe billing portal, this updates
// all local records to prevent orphaned data.
func (s *StripeService) handleCustomerUpdated(obj map[string]interface{}) error {
	customerID, ok := obj["id"].(string)
	if !ok || customerID == "" {
		return errors.New("missing customer id in customer.updated event")
	}

	newEmail, _ := obj["email"].(string)
	newEmail = NormalizeEmail(newEmail)
	if newEmail == "" {
		// No email to update
		return nil
	}

	// Get the previous email from the previous_attributes if available
	var oldEmail string
	if prevAttrs, ok := obj["previous_attributes"].(map[string]interface{}); ok {
		if prevEmail, ok := prevAttrs["email"].(string); ok {
			oldEmail = NormalizeEmail(prevEmail)
		}
	}

	// If we don't have a previous email, look it up from our database
	if oldEmail == "" {
		err := s.db.QueryRow(`
			SELECT customer_email FROM subscriptions
			WHERE customer_id = $1
			ORDER BY updated_at DESC
			LIMIT 1
		`, customerID).Scan(&oldEmail)
		if err != nil && err != sql.ErrNoRows {
			return fmt.Errorf("lookup old email for customer %s: %w", customerID, err)
		}
		oldEmail = NormalizeEmail(oldEmail)
	}

	// If emails are the same or we don't have an old email, nothing to update
	if oldEmail == "" || oldEmail == newEmail {
		return nil
	}

	logStructured("customer_email_migration_starting", map[string]interface{}{
		"level":       "info",
		"customer_id": customerID,
		"old_email":   oldEmail,
		"new_email":   newEmail,
	})

	// Use transaction to update all tables atomically
	tx, err := s.db.Begin()
	if err != nil {
		return fmt.Errorf("begin email migration transaction: %w", err)
	}
	defer func() { _ = tx.Rollback() }()

	// Update subscriptions table
	_, err = tx.Exec(`
		UPDATE subscriptions
		SET customer_email = $1, updated_at = NOW()
		WHERE customer_id = $2 OR customer_email = $3
	`, newEmail, customerID, oldEmail)
	if err != nil {
		return fmt.Errorf("migrate subscriptions email: %w", err)
	}

	// Update users table
	_, err = tx.Exec(`
		UPDATE users
		SET email = $1, updated_at = NOW()
		WHERE email = $2 OR stripe_customer_id = $3
	`, newEmail, oldEmail, customerID)
	if err != nil {
		return fmt.Errorf("migrate users email: %w", err)
	}

	// Update credit_wallets table
	_, err = tx.Exec(`
		UPDATE credit_wallets
		SET customer_email = $1, updated_at = NOW()
		WHERE customer_email = $2
	`, newEmail, oldEmail)
	if err != nil {
		return fmt.Errorf("migrate credit_wallets email: %w", err)
	}

	// Update credit_transactions table
	_, err = tx.Exec(`
		UPDATE credit_transactions
		SET customer_email = $1
		WHERE customer_email = $2
	`, newEmail, oldEmail)
	if err != nil {
		return fmt.Errorf("migrate credit_transactions email: %w", err)
	}

	// Update intro_coupon_usage table
	_, err = tx.Exec(`
		UPDATE intro_coupon_usage
		SET email = $1
		WHERE email = $2 OR stripe_customer_id = $3
	`, newEmail, oldEmail, customerID)
	if err != nil {
		return fmt.Errorf("migrate intro_coupon_usage email: %w", err)
	}

	// Update checkout_sessions table
	_, err = tx.Exec(`
		UPDATE checkout_sessions
		SET customer_email = $1, updated_at = NOW()
		WHERE customer_email = $2 OR customer_id = $3
	`, newEmail, oldEmail, customerID)
	if err != nil {
		return fmt.Errorf("migrate checkout_sessions email: %w", err)
	}

	if err := tx.Commit(); err != nil {
		return fmt.Errorf("commit email migration transaction: %w", err)
	}

	logStructured("customer_email_migration_completed", map[string]interface{}{
		"level":       "info",
		"customer_id": customerID,
		"old_email":   oldEmail,
		"new_email":   newEmail,
	})

	return nil
}
