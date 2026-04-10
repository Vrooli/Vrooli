package main

import (
	"crypto/hmac"
	"crypto/sha256"
	"database/sql"
	"encoding/hex"
	"encoding/json"
	"errors"
	"fmt"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// ============================================================================
// Shared Test Harness for Cross-Concern Monetization Tests
// ============================================================================

// monetizationTestHarness bundles all services needed for cross-concern
// monetization tests (checkout → webhook → entitlement → download).
type monetizationTestHarness struct {
	db             *sql.DB
	stripeService  *StripeService
	accountService *AccountService
	authorizer     *DownloadAuthorizer
	webhookSecret  string
}

// signWebhookPayload generates a valid Stripe webhook signature header using HMAC-SHA256.
func (h *monetizationTestHarness) signWebhookPayload(payload []byte) string {
	timestamp := fmt.Sprintf("%d", time.Now().Unix())
	signedPayload := timestamp + "." + string(payload)
	mac := hmac.New(sha256.New, []byte(h.webhookSecret))
	mac.Write([]byte(signedPayload))
	signature := hex.EncodeToString(mac.Sum(nil))
	return "t=" + timestamp + ",v1=" + signature
}

// fireWebhook marshals an event and processes it through the webhook handler.
func (h *monetizationTestHarness) fireWebhook(t *testing.T, event map[string]interface{}) {
	t.Helper()
	payload, err := json.Marshal(event)
	require.NoError(t, err)
	sig := h.signWebhookPayload(payload)
	err = h.stripeService.HandleWebhook(payload, sig)
	require.NoError(t, err)
}

// setupMonetizationHarness creates all required tables and services for E2E tests.
func setupMonetizationHarness(t *testing.T, stripeServer *httptest.Server) *monetizationTestHarness {
	t.Helper()

	db := setupTestDB(t)
	t.Cleanup(func() { db.Close() })
	resetStripeTestData(t, db)

	_, err := db.Exec(`
		DROP TABLE IF EXISTS download_assets CASCADE;
		DROP TABLE IF EXISTS download_apps CASCADE;
		DROP TABLE IF EXISTS intro_coupon_usage CASCADE;
		DROP TABLE IF EXISTS intro_anomaly_log CASCADE;
		DROP TABLE IF EXISTS credit_transactions CASCADE;
		DROP TABLE IF EXISTS credit_wallets CASCADE;
		DROP TABLE IF EXISTS subscription_schedules CASCADE;
		DROP TABLE IF EXISTS subscriptions CASCADE;
		DROP TABLE IF EXISTS checkout_sessions CASCADE;
		DROP TABLE IF EXISTS users CASCADE;

		CREATE TABLE users (
			id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
			email VARCHAR(255) UNIQUE NOT NULL,
			stripe_customer_id VARCHAR(255),
			has_used_intro BOOLEAN DEFAULT FALSE,
			email_verified BOOLEAN DEFAULT FALSE,
			created_at TIMESTAMP DEFAULT NOW(),
			updated_at TIMESTAMP DEFAULT NOW(),
			last_login_at TIMESTAMP
		);
		CREATE TABLE checkout_sessions (
			id SERIAL PRIMARY KEY,
			session_id VARCHAR(255) UNIQUE NOT NULL,
			customer_email VARCHAR(255),
			customer_id VARCHAR(255),
			price_id VARCHAR(255),
			subscription_id VARCHAR(255),
			status VARCHAR(50) NOT NULL,
			session_type VARCHAR(50) DEFAULT 'subscription',
			amount_cents INTEGER,
			schedule_id VARCHAR(255),
			metadata JSONB DEFAULT '{}'::jsonb,
			created_at TIMESTAMP DEFAULT NOW(),
			updated_at TIMESTAMP DEFAULT NOW()
		);
		CREATE TABLE subscriptions (
			id SERIAL PRIMARY KEY,
			subscription_id VARCHAR(255) UNIQUE NOT NULL,
			customer_id VARCHAR(255),
			customer_email VARCHAR(255),
			status VARCHAR(50) NOT NULL,
			plan_tier VARCHAR(50),
			price_id VARCHAR(255),
			bundle_key VARCHAR(100),
			canceled_at TIMESTAMP,
			billing_cycle_start INTEGER DEFAULT 0,
			created_at TIMESTAMP DEFAULT NOW(),
			updated_at TIMESTAMP DEFAULT NOW()
		);
		CREATE TABLE subscription_schedules (
			id SERIAL PRIMARY KEY,
			schedule_id VARCHAR(255) UNIQUE NOT NULL,
			subscription_id VARCHAR(255),
			price_id VARCHAR(255),
			billing_interval VARCHAR(50),
			intro_enabled BOOLEAN DEFAULT FALSE,
			intro_amount_cents INTEGER DEFAULT 0,
			intro_periods INTEGER DEFAULT 0,
			normal_amount_cents INTEGER DEFAULT 0,
			next_billing_at TIMESTAMP,
			status VARCHAR(50) DEFAULT 'active',
			metadata JSONB DEFAULT '{}'::jsonb,
			created_at TIMESTAMP DEFAULT NOW(),
			updated_at TIMESTAMP DEFAULT NOW()
		);
		CREATE TABLE credit_wallets (
			id SERIAL PRIMARY KEY,
			customer_email VARCHAR(255) UNIQUE NOT NULL,
			balance_credits BIGINT DEFAULT 0,
			bonus_credits BIGINT DEFAULT 0,
			updated_at TIMESTAMP DEFAULT NOW()
		);
		CREATE TABLE credit_transactions (
			id SERIAL PRIMARY KEY,
			customer_email VARCHAR(255) NOT NULL,
			amount_credits BIGINT NOT NULL,
			transaction_type VARCHAR(50) NOT NULL,
			stripe_event_id VARCHAR(255) UNIQUE,
			metadata JSONB DEFAULT '{}'::jsonb,
			created_at TIMESTAMP DEFAULT NOW()
		);
		CREATE UNIQUE INDEX IF NOT EXISTS idx_credit_transactions_stripe_event_id
		ON credit_transactions(stripe_event_id) WHERE stripe_event_id IS NOT NULL;
		CREATE TABLE intro_coupon_usage (
			id SERIAL PRIMARY KEY,
			email VARCHAR(255) NOT NULL,
			stripe_customer_id VARCHAR(255),
			coupon_id VARCHAR(255),
			plan_tier VARCHAR(50),
			subscription_id VARCHAR(255),
			used_at TIMESTAMP DEFAULT NOW()
		);
		CREATE TABLE intro_anomaly_log (
			id SERIAL PRIMARY KEY,
			email VARCHAR(255),
			customer_id VARCHAR(255),
			coupon_id VARCHAR(255),
			anomaly_type VARCHAR(100),
			details JSONB DEFAULT '{}'::jsonb,
			created_at TIMESTAMP DEFAULT NOW()
		);
		CREATE TABLE download_apps (
			id SERIAL PRIMARY KEY,
			bundle_key VARCHAR(100) NOT NULL,
			app_key VARCHAR(100) NOT NULL,
			name VARCHAR(255) NOT NULL DEFAULT '',
			tagline TEXT,
			description TEXT,
			icon_url TEXT,
			screenshot_url TEXT,
			install_overview TEXT,
			install_steps JSONB DEFAULT '[]'::jsonb,
			storefronts JSONB DEFAULT '{}'::jsonb,
			metadata JSONB DEFAULT '{}'::jsonb,
			display_order INTEGER DEFAULT 0,
			update_api_key TEXT,
			created_at TIMESTAMP DEFAULT NOW(),
			updated_at TIMESTAMP DEFAULT NOW(),
			UNIQUE(bundle_key, app_key)
		);
		CREATE TABLE download_assets (
			id SERIAL PRIMARY KEY,
			bundle_key VARCHAR(100) NOT NULL,
			app_key VARCHAR(100) NOT NULL,
			platform VARCHAR(50) NOT NULL,
			artifact_url TEXT,
			artifact_source VARCHAR(50) DEFAULT 'direct',
			artifact_id BIGINT,
			release_version VARCHAR(50),
			release_notes TEXT,
			checksum VARCHAR(255),
			requires_entitlement BOOLEAN DEFAULT TRUE,
			metadata JSONB DEFAULT '{}'::jsonb,
			is_current BOOLEAN DEFAULT TRUE,
			variant_key VARCHAR(50) NOT NULL DEFAULT 'default',
			display_order INTEGER DEFAULT 0,
			created_at TIMESTAMP DEFAULT NOW(),
			updated_at TIMESTAMP DEFAULT NOW(),
			UNIQUE(bundle_key, app_key, platform, variant_key)
		)
	`)
	require.NoError(t, err)

	productID := upsertTestBundleProduct(t, db, "business_suite", "Business Suite", "prod_e2e", "production", 1000000, 0.001, "credits")
	insertBundlePrice(t, db, productID, "price_pro_e2e", "Pro Plan", "pro", "month", "usd", 2900, false, "", 0, 0, "", 1000000, 0, 1, 10, "none", sessionTypeSubscription, map[string]interface{}{})

	webhookSecret := "whsec_e2e_test"
	cfg := DefaultStripeTestConfig().WithKeys("pk_test", "sk_test", webhookSecret)
	service := ConfigureStripeService(t, db, cfg, stripeServer)

	accountSvc := NewAccountService(db, service.planService)
	downloadSvc := &DownloadService{db: db}
	authorizer := NewDownloadAuthorizer(downloadSvc, accountSvc, "business_suite")

	return &monetizationTestHarness{
		db:             db,
		stripeService:  service,
		accountService: accountSvc,
		authorizer:     authorizer,
		webhookSecret:  webhookSecret,
	}
}

// seedGatedAsset inserts a download asset that requires entitlement.
func (h *monetizationTestHarness) seedGatedAsset(t *testing.T, appKey, platform string) {
	t.Helper()
	_, err := h.db.Exec(`
		INSERT INTO download_apps (bundle_key, app_key, name)
		VALUES ($1, $2, $3)
		ON CONFLICT (bundle_key, app_key) DO NOTHING
	`, "business_suite", appKey, appKey)
	require.NoError(t, err)

	_, err = h.db.Exec(`
		INSERT INTO download_assets (bundle_key, app_key, platform, artifact_url, release_version, release_notes, checksum, requires_entitlement)
		VALUES ($1, $2, $3, $4, $5, '', '', TRUE)
		ON CONFLICT (bundle_key, app_key, platform, variant_key) DO UPDATE SET requires_entitlement = TRUE
	`, "business_suite", appKey, platform, "https://cdn.example.com/app.zip", "1.0.0")
	require.NoError(t, err)
}

// ============================================================================
// E2E Tests: Checkout → Webhook → Entitlement → Download
// ============================================================================

// TestE2E_Checkout_Webhook_Entitlement_Download_HappyPath verifies the complete
// monetization flow: checkout → webhook → subscription active → download authorized.
func TestE2E_Checkout_Webhook_Entitlement_Download_HappyPath(t *testing.T) {
	stripeServer := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		switch {
		case r.Method == http.MethodPost && r.URL.Path == "/v1/checkout/sessions":
			fmt.Fprint(w, `{
				"id":"cs_e2e_happy",
				"url":"https://checkout.stripe.test/cs_e2e_happy",
				"status":"open",
				"customer_email":"happy@example.com",
				"customer":"cus_e2e_happy",
				"subscription":"sub_e2e_happy",
				"amount_total":2900,
				"mode":"subscription",
				"currency":"usd"
			}`)
		case r.Method == http.MethodPost && r.URL.Path == "/v1/customers/cus_e2e_happy":
			fmt.Fprint(w, `{"id":"cus_e2e_happy"}`)
		default:
			http.NotFound(w, r)
		}
	}))
	defer stripeServer.Close()

	h := setupMonetizationHarness(t, stripeServer)
	h.seedGatedAsset(t, "desktop-app", "windows")

	// Step 1: Create checkout session
	session, err := h.stripeService.CreateCheckoutSession("price_pro_e2e", "/success", "/cancel", "happy@example.com")
	require.NoError(t, err)
	require.NotNil(t, session)

	// Step 2: Before webhook — download should be denied (no subscription)
	_, err = h.authorizer.Authorize("desktop-app", "windows", "happy@example.com")
	assert.True(t, errors.Is(err, ErrDownloadRequiresActiveSubscription) || strings.Contains(err.Error(), "subscription"),
		"expected download denied before payment, got: %v", err)

	// Step 3: Simulate checkout.session.completed webhook
	h.fireWebhook(t, map[string]interface{}{
		"id":   "evt_e2e_happy_checkout",
		"type": "checkout.session.completed",
		"data": map[string]interface{}{
			"object": map[string]interface{}{
				"id":             "cs_e2e_happy",
				"customer_email": "happy@example.com",
				"customer":       "cus_e2e_happy",
				"subscription":   "sub_e2e_happy",
				"amount_total":   2900,
			},
		},
	})

	// Step 4: Verify subscription is active in DB
	var subStatus string
	err = h.db.QueryRow(`SELECT status FROM subscriptions WHERE subscription_id = $1`, "sub_e2e_happy").Scan(&subStatus)
	require.NoError(t, err)
	assert.Equal(t, "active", subStatus)

	// Step 5: Download should now be authorized
	asset, err := h.authorizer.Authorize("desktop-app", "windows", "happy@example.com")
	require.NoError(t, err)
	assert.Equal(t, "windows", asset.Platform)
}

// TestE2E_Checkout_StripeAPIError_ReturnsError verifies that Stripe API failures
// during checkout are properly surfaced.
func TestE2E_Checkout_StripeAPIError_ReturnsError(t *testing.T) {
	stripeServer := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusInternalServerError)
		fmt.Fprint(w, `{"error":{"message":"internal server error"}}`)
	}))
	defer stripeServer.Close()

	h := setupMonetizationHarness(t, stripeServer)

	_, err := h.stripeService.CreateCheckoutSession("price_pro_e2e", "/success", "/cancel", "error@example.com")
	assert.Error(t, err, "checkout should fail when Stripe API returns error")
}

// TestE2E_Webhook_OutOfOrder_InvoicePaid_BeforeCheckoutCompleted verifies correct
// final state when invoice.paid arrives before checkout.session.completed.
func TestE2E_Webhook_OutOfOrder_InvoicePaid_BeforeCheckoutCompleted(t *testing.T) {
	stripeServer := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		if r.Method == http.MethodPost && r.URL.Path == "/v1/customers/cus_ooo" {
			fmt.Fprint(w, `{"id":"cus_ooo"}`)
			return
		}
		http.NotFound(w, r)
	}))
	defer stripeServer.Close()

	h := setupMonetizationHarness(t, stripeServer)
	h.seedGatedAsset(t, "desktop-app", "windows")

	// Pre-create checkout session (simulates what CreateCheckoutSession would do)
	_, err := h.db.Exec(`
		INSERT INTO checkout_sessions (session_id, customer_email, price_id, status, session_type, amount_cents)
		VALUES ($1, $2, $3, $4, $5, $6)
	`, "cs_ooo", "ooo@example.com", "price_pro_e2e", "open", sessionTypeSubscription, 2900)
	require.NoError(t, err)

	// Step 1: Fire invoice.paid FIRST (out of order)
	h.fireWebhook(t, map[string]interface{}{
		"id":   "evt_ooo_invoice",
		"type": "invoice.paid",
		"data": map[string]interface{}{
			"object": map[string]interface{}{
				"subscription":   "sub_ooo",
				"customer":       "cus_ooo",
				"customer_email": "ooo@example.com",
				"lines": map[string]interface{}{
					"data": []interface{}{
						map[string]interface{}{
							"price": map[string]interface{}{
								"id": "price_pro_e2e",
							},
						},
					},
				},
			},
		},
	})

	// Subscription should exist with active status from invoice.paid
	var statusAfterInvoice string
	err = h.db.QueryRow(`SELECT status FROM subscriptions WHERE subscription_id = $1`, "sub_ooo").Scan(&statusAfterInvoice)
	require.NoError(t, err)
	assert.Equal(t, "active", statusAfterInvoice)

	// Step 2: Fire checkout.session.completed SECOND
	h.fireWebhook(t, map[string]interface{}{
		"id":   "evt_ooo_checkout",
		"type": "checkout.session.completed",
		"data": map[string]interface{}{
			"object": map[string]interface{}{
				"id":             "cs_ooo",
				"customer_email": "ooo@example.com",
				"customer":       "cus_ooo",
				"subscription":   "sub_ooo",
				"amount_total":   2900,
			},
		},
	})

	// Step 3: Verify final state is correct
	var finalStatus, planTier string
	err = h.db.QueryRow(`SELECT status, COALESCE(plan_tier, '') FROM subscriptions WHERE subscription_id = $1`, "sub_ooo").Scan(&finalStatus, &planTier)
	require.NoError(t, err)
	assert.Equal(t, "active", finalStatus, "subscription should be active regardless of event order")
	assert.Equal(t, "pro", planTier, "plan tier should be set")

	// Download should be authorized
	asset, err := h.authorizer.Authorize("desktop-app", "windows", "ooo@example.com")
	require.NoError(t, err)
	assert.Equal(t, "windows", asset.Platform)
}

// TestE2E_Webhook_OutOfOrder_SubscriptionUpdated_BeforeCheckoutCompleted verifies
// that customer.subscription.updated arriving before checkout.session.completed
// still produces correct final state.
func TestE2E_Webhook_OutOfOrder_SubscriptionUpdated_BeforeCheckoutCompleted(t *testing.T) {
	stripeServer := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		if r.Method == http.MethodPost && r.URL.Path == "/v1/customers/cus_ooo2" {
			fmt.Fprint(w, `{"id":"cus_ooo2"}`)
			return
		}
		http.NotFound(w, r)
	}))
	defer stripeServer.Close()

	h := setupMonetizationHarness(t, stripeServer)

	// Pre-create checkout session
	_, err := h.db.Exec(`
		INSERT INTO checkout_sessions (session_id, customer_email, price_id, status, session_type, amount_cents)
		VALUES ($1, $2, $3, $4, $5, $6)
	`, "cs_ooo2", "ooo2@example.com", "price_pro_e2e", "open", sessionTypeSubscription, 2900)
	require.NoError(t, err)

	// Step 1: Fire subscription.updated FIRST
	h.fireWebhook(t, map[string]interface{}{
		"id":   "evt_ooo2_sub_updated",
		"type": "customer.subscription.updated",
		"data": map[string]interface{}{
			"object": map[string]interface{}{
				"id":             "sub_ooo2",
				"customer":       "cus_ooo2",
				"customer_email": "ooo2@example.com",
				"status":         "active",
			},
		},
	})

	// Subscription should exist from the updated event
	var countAfterUpdate int
	err = h.db.QueryRow(`SELECT COUNT(*) FROM subscriptions WHERE subscription_id = $1`, "sub_ooo2").Scan(&countAfterUpdate)
	require.NoError(t, err)
	assert.Equal(t, 1, countAfterUpdate, "subscription should be created by updated event via upsert")

	// Step 2: Fire checkout.session.completed SECOND
	h.fireWebhook(t, map[string]interface{}{
		"id":   "evt_ooo2_checkout",
		"type": "checkout.session.completed",
		"data": map[string]interface{}{
			"object": map[string]interface{}{
				"id":             "cs_ooo2",
				"customer_email": "ooo2@example.com",
				"customer":       "cus_ooo2",
				"subscription":   "sub_ooo2",
				"amount_total":   2900,
			},
		},
	})

	// Verify final state — should have full enrichment from checkout
	var finalStatus string
	err = h.db.QueryRow(`SELECT status FROM subscriptions WHERE subscription_id = $1`, "sub_ooo2").Scan(&finalStatus)
	require.NoError(t, err)
	assert.Equal(t, "active", finalStatus)

	// Should still only be one subscription row
	var finalCount int
	err = h.db.QueryRow(`SELECT COUNT(*) FROM subscriptions WHERE subscription_id = $1`, "sub_ooo2").Scan(&finalCount)
	require.NoError(t, err)
	assert.Equal(t, 1, finalCount, "upsert should prevent duplicate subscription rows")
}

// TestE2E_Webhook_CustomerUpdated_InterleavedWithPayment verifies that email
// change webhook interleaved with payment events maintains data consistency.
func TestE2E_Webhook_CustomerUpdated_InterleavedWithPayment(t *testing.T) {
	stripeServer := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		if r.Method == http.MethodPost && strings.HasPrefix(r.URL.Path, "/v1/customers/") {
			fmt.Fprintf(w, `{"id":"%s"}`, strings.TrimPrefix(r.URL.Path, "/v1/customers/"))
			return
		}
		http.NotFound(w, r)
	}))
	defer stripeServer.Close()

	h := setupMonetizationHarness(t, stripeServer)

	oldEmail := "old-interleave@example.com"
	newEmail := "new-interleave@example.com"
	customerID := "cus_interleave"

	// Seed user and subscription with old email
	_, err := h.db.Exec(`INSERT INTO users (email, stripe_customer_id) VALUES ($1, $2)`, oldEmail, customerID)
	require.NoError(t, err)
	_, err = h.db.Exec(`
		INSERT INTO subscriptions (subscription_id, customer_id, customer_email, status, plan_tier, bundle_key)
		VALUES ($1, $2, $3, $4, $5, $6)
	`, "sub_interleave", customerID, oldEmail, "active", "pro", "business_suite")
	require.NoError(t, err)

	// Step 1: Fire customer.updated (email change)
	err = h.stripeService.handleCustomerUpdated(map[string]interface{}{
		"id":    customerID,
		"email": newEmail,
		"previous_attributes": map[string]interface{}{
			"email": oldEmail,
		},
	})
	require.NoError(t, err)

	// Step 2: Fire invoice.paid with the NEW email
	h.fireWebhook(t, map[string]interface{}{
		"id":   "evt_interleave_invoice",
		"type": "invoice.paid",
		"data": map[string]interface{}{
			"object": map[string]interface{}{
				"subscription":   "sub_interleave",
				"customer":       customerID,
				"customer_email": newEmail,
				"lines": map[string]interface{}{
					"data": []interface{}{
						map[string]interface{}{
							"price": map[string]interface{}{
								"id": "price_pro_e2e",
							},
						},
					},
				},
			},
		},
	})

	// Verify subscription email was migrated and status is active
	var subEmail, subStatus string
	err = h.db.QueryRow(`SELECT customer_email, status FROM subscriptions WHERE subscription_id = $1`, "sub_interleave").Scan(&subEmail, &subStatus)
	require.NoError(t, err)
	assert.Equal(t, newEmail, subEmail, "subscription should use new email")
	assert.Equal(t, "active", subStatus)

	// Verify user record was migrated
	var userEmail string
	err = h.db.QueryRow(`SELECT email FROM users WHERE stripe_customer_id = $1`, customerID).Scan(&userEmail)
	require.NoError(t, err)
	assert.Equal(t, newEmail, userEmail)
}

// TestE2E_Webhook_UserDeletedBetweenEvents verifies graceful handling when a user
// is deleted after checkout but before the webhook is processed.
func TestE2E_Webhook_UserDeletedBetweenEvents(t *testing.T) {
	h := setupMonetizationHarness(t, nil)

	// Pre-create checkout session and user
	_, err := h.db.Exec(`
		INSERT INTO checkout_sessions (session_id, customer_email, price_id, status, session_type, amount_cents)
		VALUES ($1, $2, $3, $4, $5, $6)
	`, "cs_deleted_user", "deleted@example.com", "price_pro_e2e", "open", sessionTypeSubscription, 2900)
	require.NoError(t, err)

	_, err = h.db.Exec(`INSERT INTO users (email, stripe_customer_id) VALUES ($1, $2)`, "deleted@example.com", "cus_deleted")
	require.NoError(t, err)

	// Delete user before webhook arrives
	_, err = h.db.Exec(`DELETE FROM users WHERE email = $1`, "deleted@example.com")
	require.NoError(t, err)

	// Webhook should still succeed — handleCheckoutCompleted re-creates user via linkUserToStripeCustomer
	h.fireWebhook(t, map[string]interface{}{
		"id":   "evt_deleted_user",
		"type": "checkout.session.completed",
		"data": map[string]interface{}{
			"object": map[string]interface{}{
				"id":             "cs_deleted_user",
				"customer_email": "deleted@example.com",
				"customer":       "cus_deleted",
				"subscription":   "sub_deleted_user",
				"amount_total":   2900,
			},
		},
	})

	// Subscription should be created
	var subStatus string
	err = h.db.QueryRow(`SELECT status FROM subscriptions WHERE subscription_id = $1`, "sub_deleted_user").Scan(&subStatus)
	require.NoError(t, err)
	assert.Equal(t, "active", subStatus)
}
