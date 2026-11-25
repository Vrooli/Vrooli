package main

import (
	"context"
	"crypto/hmac"
	"crypto/sha256"
	"database/sql"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	entitlementclient "github.com/vrooli/vrooli/packages/entitlementclient-go"
	"landing-page-business-suite-api/internal/commerce"
)

type webhookTopupWalletProbe struct {
	ctx             context.Context
	account, email  string
	amount          int64
	txnType, source string
	event           string
	metadata        map[string]interface{}
}

func (w *webhookTopupWalletProbe) AddCreditsForBusinessAccountContext(ctx context.Context, account, email string, amount int64, txnType, source, event string, metadata map[string]interface{}) error {
	w.ctx, w.account, w.email, w.amount = ctx, account, email, amount
	w.txnType, w.source, w.event, w.metadata = txnType, source, event, metadata
	return nil
}

func (*webhookTopupWalletProbe) BalanceForBusinessAccountContext(context.Context, string, string) (int64, error) {
	return 0, nil
}

// setupCheckoutAtomicitySchema restores the production schema before a test
// adds its temporary failure constraint. The fixture must not recreate a
// partial Stripe schema: setupTestDB already applies the embedded domain
// schemas, which keeps these tests aligned with runtime columns and indexes.
func setupCheckoutAtomicitySchema(t *testing.T, db *sql.DB) {
	t.Helper()
	if err := applyRuntimeSchema(db); err != nil {
		require.NoError(t, err)
	}
}

// buildCheckoutCompletedWebhook returns a signed checkout.session.completed
// payload together with the stripe signature header for HandleWebhook.
func buildCheckoutCompletedWebhook(t *testing.T, secret, eventID, sessionID, subscriptionID, customerEmail string, amountTotal int64) ([]byte, string) {
	t.Helper()
	body := map[string]interface{}{
		"id":   eventID,
		"type": "checkout.session.completed",
		"data": map[string]interface{}{
			"object": map[string]interface{}{
				"id":             sessionID,
				"customer":       "cus_atomicity",
				"customer_email": customerEmail,
				"subscription":   subscriptionID,
				"amount_total":   amountTotal,
			},
		},
	}
	payload, err := json.Marshal(body)
	require.NoError(t, err)
	timestamp := fmt.Sprintf("%d", time.Now().Unix())
	mac := hmac.New(sha256.New, []byte(secret))
	mac.Write([]byte(timestamp + "." + string(payload)))
	signature := hex.EncodeToString(mac.Sum(nil))
	return payload, "t=" + timestamp + ",v1=" + signature
}

// insertPendingSubscriptionCheckout seeds an open subscription checkout_session
// so that handleCheckoutCompleted can transition it to 'complete'.
func insertPendingSubscriptionCheckout(t *testing.T, db *sql.DB, sessionID, customerEmail, priceID string, amountCents int) {
	t.Helper()
	_, err := db.Exec(`
		INSERT INTO checkout_sessions (session_id, customer_email, price_id, status, session_type, amount_cents)
		VALUES ($1, $2, $3, 'pending', $4, $5)
	`, sessionID, customerEmail, priceID, sessionTypeSubscription, amountCents)
	require.NoError(t, err)
}

// TestHandleCheckoutCompleted_SubscriptionInsertFailure_RollsBackStatus
// Injects a schema-level CHECK constraint that blocks the subscription UPSERT
// (it INSERTs with status='active'). Asserts that the checkout_sessions row
// remains 'pending' and no subscription row materializes — proving the
// UPDATE + INSERT are wrapped in a single transaction.
//
// Failure mechanism: CHECK (status <> 'active') on subscriptions.
// If the UPSERT in handleSubscriptionCompletion is ever relaxed so that the
// SET clause no longer writes status='active', update this constraint.
func TestHandleCheckoutCompleted_SubscriptionInsertFailure_RollsBackStatus(t *testing.T) {
	db := setupTestDB(t)
	resetStripeTestData(t, db)
	setupCheckoutAtomicitySchema(t, db)

	productID := upsertTestBundleProduct(t, db, "business_suite", "Business Suite", "prod_atomicity_sub_fail", "production", 1000000, 0.001, "credits")
	insertBundlePrice(t, db, productID, "price_atomicity_sub_fail", "Pro Plan", "pro", "month", "usd", 2900, false, "", 0, 0, "", 1000000, 0, 1, 10, "none", sessionTypeSubscription, map[string]interface{}{})

	insertPendingSubscriptionCheckout(t, db, "cs_atomicity_sub_fail", "atomicity1@example.com", "price_atomicity_sub_fail", 2900)

	_, err := db.Exec(`ALTER TABLE subscriptions ADD CONSTRAINT test_block_active_sub CHECK (status <> 'active')`)
	require.NoError(t, err)
	t.Cleanup(func() {
		_, _ = db.Exec(`ALTER TABLE subscriptions DROP CONSTRAINT IF EXISTS test_block_active_sub`)
	})

	service := ConfigureStripeService(t, db, DefaultStripeTestConfig().WithKeys("pk_test", "sk_test", "whsec_atomicity"), nil)

	payload, sig := buildCheckoutCompletedWebhook(t, "whsec_atomicity", "evt_sub_fail", "cs_atomicity_sub_fail", "sub_atomicity_fail", "atomicity1@example.com", 2900)

	err = service.HandleWebhook(payload, sig)
	require.Error(t, err, "webhook should surface the DB error for Stripe to retry")

	var sessionStatus string
	require.NoError(t, db.QueryRow(`SELECT status FROM checkout_sessions WHERE session_id = $1`, "cs_atomicity_sub_fail").Scan(&sessionStatus))
	assert.Equal(t, "pending", sessionStatus, "checkout_sessions.status must be rolled back when subscription INSERT fails")

	var subCount int
	require.NoError(t, db.QueryRow(`SELECT COUNT(*) FROM subscriptions WHERE subscription_id = $1`, "sub_atomicity_fail").Scan(&subCount))
	assert.Equal(t, 0, subCount, "no subscription row should exist after rollback")
}

// TestHandleCheckoutCompleted_ScheduleInsertFailure_RollsBackAll
// Uses an intro-enabled monthly plan so handleSubscriptionCompletion also
// calls createSubscriptionSchedule. Blocks the schedule UPSERT via a CHECK
// constraint and asserts that BOTH the subscription INSERT and the
// checkout_sessions UPDATE roll back.
func TestHandleCheckoutCompleted_ScheduleInsertFailure_RollsBackAll(t *testing.T) {
	db := setupTestDB(t)
	resetStripeTestData(t, db)
	setupCheckoutAtomicitySchema(t, db)

	productID := upsertTestBundleProduct(t, db, "business_suite", "Business Suite", "prod_atomicity_sched_fail", "production", 1000000, 0.001, "credits")
	insertBundlePrice(t, db, productID, "price_atomicity_sched_fail", "Pro Plan", "pro", "month", "usd", 2900, true, "flat_amount", 100, 1, "intro_atomicity_lookup", 1000000, 0, 1, 10, "none", sessionTypeSubscription, map[string]interface{}{})

	insertPendingSubscriptionCheckout(t, db, "cs_atomicity_sched_fail", "atomicity2@example.com", "price_atomicity_sched_fail", 2900)

	_, err := db.Exec(`ALTER TABLE subscription_schedules ADD CONSTRAINT test_block_active_sched CHECK (status <> 'active')`)
	require.NoError(t, err)
	t.Cleanup(func() {
		_, _ = db.Exec(`ALTER TABLE subscription_schedules DROP CONSTRAINT IF EXISTS test_block_active_sched`)
	})

	service := ConfigureStripeService(t, db, DefaultStripeTestConfig().WithKeys("pk_test", "sk_test", "whsec_atomicity"), nil)

	payload, sig := buildCheckoutCompletedWebhook(t, "whsec_atomicity", "evt_sched_fail", "cs_atomicity_sched_fail", "sub_atomicity_sched", "atomicity2@example.com", 2900)

	err = service.HandleWebhook(payload, sig)
	require.Error(t, err, "schedule failure must propagate so Stripe retries")

	var sessionStatus string
	require.NoError(t, db.QueryRow(`SELECT status FROM checkout_sessions WHERE session_id = $1`, "cs_atomicity_sched_fail").Scan(&sessionStatus))
	assert.Equal(t, "pending", sessionStatus, "checkout_sessions.status must be rolled back when schedule INSERT fails")

	var subCount int
	require.NoError(t, db.QueryRow(`SELECT COUNT(*) FROM subscriptions WHERE subscription_id = $1`, "sub_atomicity_sched").Scan(&subCount))
	assert.Equal(t, 0, subCount, "subscription INSERT must roll back with the schedule INSERT")

	var schedCount int
	require.NoError(t, db.QueryRow(`SELECT COUNT(*) FROM subscription_schedules WHERE subscription_id = $1`, "sub_atomicity_sched").Scan(&schedCount))
	assert.Equal(t, 0, schedCount, "no schedule row should exist after rollback")
}

func TestHandleWebhook_CheckoutCompletedTopupPassesRequestContextAndAttribution(t *testing.T) {
	db := setupTestDB(t)
	resetStripeTestData(t, db)
	setupCheckoutAtomicitySchema(t, db)

	productID := upsertTestBundleProduct(t, db, "business_suite", "Business Suite", "prod_context_topup", "test", 100, 1, "credits")
	insertBundlePrice(t, db, productID, "price_context_topup", "Credits", "credits", "one_time", "usd", 1000, false, "", 0, 0, "", 0, 0, 1, 1, "none", "credits_topup", map[string]interface{}{})
	_, err := db.Exec(`INSERT INTO checkout_sessions(session_id,customer_email,price_id,status,session_type,amount_cents,business_account_id) VALUES('cs_context_topup','buyer@example.test','price_context_topup','pending','credits_topup',1000,'acct_context_topup')`)
	require.NoError(t, err)

	plans := commerce.NewPlanServiceWithPlanStore(globalTestPlanStore)
	repository := commerce.NewStripeRepository(db)
	secret := "whsec_context_topup"
	type contextKey struct{}
	wantContextValue := "selected-request-pool"
	wallet := &webhookTopupWalletProbe{}
	topup := commerce.NewCreditTopupService(plans, wallet, nil)
	service := commerce.NewStripeWebhookService(commerce.StripeWebhookOptions{
		DB:          db,
		PlanService: plans,
		Config: func() commerce.StripeWebhookConfig {
			return commerce.StripeWebhookConfig{WebhookSecret: secret, HasWebhook: true}
		},
		LoadCheckoutSessionContext: func(ctx context.Context, id string) (*commerce.CheckoutSessionRecord, error) {
			if ctx.Value(contextKey{}) != wantContextValue {
				t.Errorf("checkout loader context value = %v", ctx.Value(contextKey{}))
			}
			return repository.LoadCheckoutSession(id)
		},
		ExtractAmount: func(obj map[string]interface{}, _ *commerce.CheckoutSessionRecord) int64 {
			amount, _ := obj["amount_total"].(float64)
			return int64(amount)
		},
		HandleCreditTopupContext: topup.ApplyContext,
	})
	body, signature := buildContextTopupEvent(t, secret, "evt_context_topup", "cs_context_topup")
	ctx := context.WithValue(context.Background(), contextKey{}, wantContextValue)
	require.NoError(t, service.HandleWebhookContext(ctx, body, signature))
	if wallet.ctx == nil || wallet.ctx.Value(contextKey{}) != wantContextValue {
		t.Fatalf("CreditTopupService.ApplyContext did not preserve request context: %v", wallet.ctx)
	}
	if wallet.account != "acct_context_topup" || wallet.email != "buyer@example.test" || wallet.amount != 1000 || wallet.txnType != "credit_topup" || wallet.source != "stripe" || wallet.event != "evt_context_topup" || wallet.metadata["business_account_id"] != "acct_context_topup" || wallet.metadata["session_id"] != "cs_context_topup" || wallet.metadata["livemode"] != false || wallet.metadata["payment_intent_id"] != "pi_context_topup" {
		t.Fatalf("contextual wallet attribution = %#v", wallet)
	}
}

func TestHandleWebhook_CheckoutCompletedTopupRequiresContextualHandler(t *testing.T) {
	db := setupTestDB(t)
	resetStripeTestData(t, db)
	setupCheckoutAtomicitySchema(t, db)

	productID := upsertTestBundleProduct(t, db, "business_suite", "Business Suite", "prod_context_topup_missing", "test", 100, 1, "credits")
	insertBundlePrice(t, db, productID, "price_context_topup_missing", "Credits", "credits", "one_time", "usd", 1000, false, "", 0, 0, "", 0, 0, 1, 1, "none", "credits_topup", map[string]interface{}{})
	_, err := db.Exec(`INSERT INTO checkout_sessions(session_id,customer_email,price_id,status,session_type,amount_cents,business_account_id) VALUES('cs_context_topup_missing','buyer@example.test','price_context_topup_missing','pending','credits_topup',1000,'acct_context_topup')`)
	require.NoError(t, err)
	plans := commerce.NewPlanServiceWithPlanStore(globalTestPlanStore)
	repository := commerce.NewStripeRepository(db)
	secret := "whsec_context_topup_missing"
	service := commerce.NewStripeWebhookService(commerce.StripeWebhookOptions{
		DB:          db,
		PlanService: plans,
		Config: func() commerce.StripeWebhookConfig {
			return commerce.StripeWebhookConfig{WebhookSecret: secret, HasWebhook: true}
		},
		LoadCheckoutSession: repository.LoadCheckoutSession,
		ExtractAmount: func(obj map[string]interface{}, _ *commerce.CheckoutSessionRecord) int64 {
			amount, _ := obj["amount_total"].(float64)
			return int64(amount)
		},
	})
	body, signature := buildContextTopupEvent(t, secret, "evt_context_topup_missing", "cs_context_topup_missing")
	err = service.HandleWebhookContext(context.Background(), body, signature)
	require.ErrorContains(t, err, "credit top-up handler unavailable")
	var status string
	require.NoError(t, db.QueryRow(`SELECT status FROM checkout_sessions WHERE session_id='cs_context_topup_missing'`).Scan(&status))
	assert.Equal(t, "pending", status, "a missing contextual credit handler must fail before checkout completion")
}

func buildContextTopupEvent(t *testing.T, secret, eventID, sessionID string) ([]byte, string) {
	t.Helper()
	body, err := json.Marshal(map[string]interface{}{
		"id": eventID, "type": "checkout.session.completed", "livemode": false,
		"data": map[string]interface{}{"object": map[string]interface{}{
			"id": sessionID, "customer_email": "buyer@example.test", "payment_status": "paid",
			"amount_total": float64(1000), "currency": "usd", "payment_intent": "pi_context_topup",
		}},
	})
	require.NoError(t, err)
	timestamp := fmt.Sprintf("%d", time.Now().Unix())
	mac := hmac.New(sha256.New, []byte(secret))
	_, _ = mac.Write([]byte(timestamp + "." + string(body)))
	return body, fmt.Sprintf("t=%s,v1=%s", timestamp, hex.EncodeToString(mac.Sum(nil)))
}

// TestHandleCheckoutCompleted_SubscriptionBranch_HappyPath_UsesSingleTx
// Covers both the non-scheduled (flat monthly) and scheduled (intro) plans,
// asserting that all expected rows materialise when the transaction commits.
func TestHandleCheckoutCompleted_SubscriptionBranch_HappyPath_UsesSingleTx(t *testing.T) {
	t.Run("non-scheduled plan", func(t *testing.T) {
		db := setupTestDB(t)
		resetStripeTestData(t, db)
		setupCheckoutAtomicitySchema(t, db)

		productID := upsertTestBundleProduct(t, db, "business_suite", "Business Suite", "prod_happy_flat", "production", 1000000, 0.001, "credits")
		insertBundlePrice(t, db, productID, "price_happy_flat", "Pro Plan", "pro", "month", "usd", 2900, false, "", 0, 0, "", 1000000, 0, 1, 10, "none", sessionTypeSubscription, map[string]interface{}{})

		insertPendingSubscriptionCheckout(t, db, "cs_happy_flat", "happy1@example.com", "price_happy_flat", 2900)

		service := ConfigureStripeService(t, db, DefaultStripeTestConfig().WithKeys("pk_test", "sk_test", "whsec_happy"), nil)
		payload, sig := buildCheckoutCompletedWebhook(t, "whsec_happy", "evt_happy_flat", "cs_happy_flat", "sub_happy_flat", "happy1@example.com", 2900)
		require.NoError(t, service.HandleWebhook(payload, sig))

		var sessionStatus string
		require.NoError(t, db.QueryRow(`SELECT status FROM checkout_sessions WHERE session_id = $1`, "cs_happy_flat").Scan(&sessionStatus))
		assert.Equal(t, "complete", sessionStatus)

		var subStatus, planTier string
		require.NoError(t, db.QueryRow(`SELECT status, plan_tier FROM subscriptions WHERE subscription_id = $1`, "sub_happy_flat").Scan(&subStatus, &planTier))
		assert.Equal(t, "active", subStatus)
		assert.Equal(t, "pro", planTier)

		var schedCount int
		require.NoError(t, db.QueryRow(`SELECT COUNT(*) FROM subscription_schedules WHERE subscription_id = $1`, "sub_happy_flat").Scan(&schedCount))
		assert.Equal(t, 0, schedCount, "flat plan should not create a schedule row")
	})

	t.Run("scheduled intro plan", func(t *testing.T) {
		db := setupTestDB(t)
		resetStripeTestData(t, db)
		setupCheckoutAtomicitySchema(t, db)

		productID := upsertTestBundleProduct(t, db, "business_suite", "Business Suite", "prod_happy_intro", "production", 1000000, 0.001, "credits")
		insertBundlePrice(t, db, productID, "price_happy_intro", "Pro Plan", "pro", "month", "usd", 2900, true, "flat_amount", 100, 1, "intro_happy_lookup", 1000000, 0, 1, 10, "none", sessionTypeSubscription, map[string]interface{}{})

		insertPendingSubscriptionCheckout(t, db, "cs_happy_intro", "happy2@example.com", "price_happy_intro", 2900)

		service := ConfigureStripeService(t, db, DefaultStripeTestConfig().WithKeys("pk_test", "sk_test", "whsec_happy"), nil)
		payload, sig := buildCheckoutCompletedWebhook(t, "whsec_happy", "evt_happy_intro", "cs_happy_intro", "sub_happy_intro", "happy2@example.com", 2900)
		require.NoError(t, service.HandleWebhook(payload, sig))

		var sessionStatus, scheduleID string
		require.NoError(t, db.QueryRow(`SELECT status, COALESCE(schedule_id,'') FROM checkout_sessions WHERE session_id = $1`, "cs_happy_intro").Scan(&sessionStatus, &scheduleID))
		assert.Equal(t, "complete", sessionStatus)
		assert.NotEmpty(t, scheduleID, "checkout_sessions.schedule_id must be written inside the tx")

		var subStatus string
		require.NoError(t, db.QueryRow(`SELECT status FROM subscriptions WHERE subscription_id = $1`, "sub_happy_intro").Scan(&subStatus))
		assert.Equal(t, "active", subStatus)

		var schedStatus string
		require.NoError(t, db.QueryRow(`SELECT status FROM subscription_schedules WHERE subscription_id = $1`, "sub_happy_intro").Scan(&schedStatus))
		assert.Equal(t, "active", schedStatus)
	})
}

func TestScopedOffersFlowThroughCheckoutWebhookAndLeaseEntitlements(t *testing.T) {
	db := setupTestDB(t)
	resetStripeTestData(t, db)
	setupCheckoutAtomicitySchema(t, db)

	productID := upsertTestBundleProduct(t, db, "business_suite", "Business Suite", "prod_scoped_offer", "production", 1000000, 0.001, "credits")
	insertBundlePrice(t, db, productID, "price_audio_only", "Audio Tools", "pro", "month", "usd", 1000, false, "", 0, 0, "", 0, 0, 1, 10, "none", sessionTypeSubscription, map[string]interface{}{
		"app_entitlements": map[string]interface{}{
			"audio-tools": map[string]interface{}{"features": []interface{}{"transcribe"}, "limits": map[string]interface{}{"ai_credits": int64(100)}},
		},
	})
	insertBundlePrice(t, db, productID, "price_audio_plus", "Audio Tools Plus", "pro", "month", "usd", 1500, false, "", 0, 0, "", 0, 0, 1, 15, "none", sessionTypeSubscription, map[string]interface{}{
		"app_entitlements": map[string]interface{}{
			"audio-tools": map[string]interface{}{"features": []interface{}{"transcribe", "enhanced_export"}, "limits": map[string]interface{}{"ai_credits": int64(150)}},
		},
	})
	insertBundlePrice(t, db, productID, "price_full_bundle", "Business Suite", "pro", "month", "usd", 3000, false, "", 0, 0, "", 0, 0, 1, 20, "none", sessionTypeSubscription, map[string]interface{}{
		"app_entitlements": map[string]interface{}{
			"audio-tools": map[string]interface{}{"features": []interface{}{"export"}, "limits": map[string]interface{}{"ai_credits": int64(200)}},
			"web-console": map[string]interface{}{"features": []interface{}{"agents"}},
		},
	})

	accountID, email := "acct_scoped_offer", "scoped-offer@example.com"
	if _, err := db.Exec(`INSERT INTO business_account_credit_wallets (business_account_id,customer_email,balance_credits,bonus_credits) VALUES ($1,$2,0,0)`, accountID, email); err != nil {
		t.Fatal(err)
	}
	checkoutIDs := map[string][2]string{
		"price_audio_only":  {"cs_audio_only", "sub_audio_only"},
		"price_full_bundle": {"cs_full_bundle", "sub_full_bundle"},
	}
	stripeMock := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodPost || r.URL.Path != "/v1/checkout/sessions" {
			http.NotFound(w, r)
			return
		}
		if err := r.ParseForm(); err != nil {
			http.Error(w, err.Error(), http.StatusBadRequest)
			return
		}
		ids, ok := checkoutIDs[r.Form.Get("line_items[0][price]")]
		if !ok || r.Form.Get("subscription_data[metadata][business_account_id]") != accountID {
			http.Error(w, "checkout price or account binding missing", http.StatusBadRequest)
			return
		}
		w.Header().Set("Content-Type", "application/json")
		_, _ = fmt.Fprintf(w, `{"id":%q,"url":"https://checkout.stripe.test/%s","status":"open","subscription":%q,"customer":"cus_atomicity","customer_email":%q,"amount_total":1000}`, ids[0], ids[0], ids[1], email)
	}))
	defer stripeMock.Close()
	service := ConfigureStripeService(t, db, DefaultStripeTestConfig().WithKeys("pk_test", "sk_test", "whsec_scoped_offer").WithMockServer(stripeMock), stripeMock)
	for i, checkout := range []struct{ session, subscription, price string }{
		{"cs_audio_only", "sub_audio_only", "price_audio_only"},
		{"cs_full_bundle", "sub_full_bundle", "price_full_bundle"},
	} {
		created, err := service.CreateCheckoutSessionForBusinessAccount(checkout.price, "/success", "/cancel", email, accountID)
		require.NoError(t, err)
		assert.Equal(t, checkout.session, created.SessionId)
		payload, signature := buildCheckoutCompletedWebhook(t, "whsec_scoped_offer", fmt.Sprintf("evt_scoped_checkout_%d", i), checkout.session, checkout.subscription, email, 1000)
		require.NoError(t, service.HandleWebhook(payload, signature))
	}

	accounts := commerce.NewService(db, globalTestPlanStore, commerce.Runtime{})
	effective, err := accounts.GetEntitlementsForBusinessAccountContext(t.Context(), email, accountID)
	require.NoError(t, err)
	grants := make(map[string]entitlementclient.AppEntitlement)
	for _, grant := range effective.AppEntitlements {
		grants[grant.AppKey] = grant
	}
	assert.ElementsMatch(t, []string{"export", "transcribe"}, grants["audio-tools"].Features)
	assert.ElementsMatch(t, []string{"agents"}, grants["web-console"].Features)
	assert.Equal(t, int64(200), grants["audio-tools"].Limits[0].Value, "overlapping offers must use the greatest allowance, not stack limits")
	assert.Equal(t, "active", grants["audio-tools"].Status)
	_, audioPresent := grants["audio-tools"]
	_, webPresent := grants["web-console"]
	assert.True(t, audioPresent, "standalone app checkout and bundle offer retain the app grant")
	assert.True(t, webPresent, "bundle checkout expands to its declared app member")

	sendSubscriptionEvent := func(eventID, eventType string, object map[string]interface{}) {
		t.Helper()
		payload, err := json.Marshal(map[string]interface{}{"id": eventID, "type": eventType, "data": map[string]interface{}{"object": object}})
		require.NoError(t, err)
		timestamp := fmt.Sprintf("%d", time.Now().Unix())
		mac := hmac.New(sha256.New, []byte("whsec_scoped_offer"))
		mac.Write([]byte(timestamp + "." + string(payload)))
		signature := "t=" + timestamp + ",v1=" + hex.EncodeToString(mac.Sum(nil))
		require.NoError(t, service.HandleWebhook(payload, signature))
	}
	sendSubscriptionEvent("evt_scoped_audio_upgrade", "customer.subscription.updated", map[string]interface{}{
		"id": "sub_audio_only", "customer": "cus_atomicity", "customer_email": email, "status": "active",
		"metadata": map[string]interface{}{"business_account_id": accountID, "plan_tier": "pro", "bundle_key": "business_suite"},
		"items":    map[string]interface{}{"data": []interface{}{map[string]interface{}{"price": map[string]interface{}{"id": "price_audio_plus"}}}},
	})
	effective, err = accounts.GetEntitlementsForBusinessAccountContext(t.Context(), email, accountID)
	require.NoError(t, err)
	for _, grant := range effective.AppEntitlements {
		if grant.AppKey == "audio-tools" {
			assert.Contains(t, grant.Features, "enhanced_export", "subscription update must derive the upgraded app plan")
		}
	}

	sendSubscriptionEvent("evt_scoped_bundle_deleted", "customer.subscription.deleted", map[string]interface{}{
		"id": "sub_full_bundle", "customer": "cus_atomicity", "customer_email": email, "status": "canceled",
		"metadata": map[string]interface{}{"business_account_id": accountID, "plan_tier": "pro", "bundle_key": "business_suite"},
		"items":    map[string]interface{}{"data": []interface{}{map[string]interface{}{"price": map[string]interface{}{"id": "price_full_bundle"}}}},
	})

	effective, err = accounts.GetEntitlementsForBusinessAccountContext(t.Context(), email, accountID)
	require.NoError(t, err)
	assert.Len(t, effective.AppEntitlements, 1)
	assert.Equal(t, "audio-tools", effective.AppEntitlements[0].AppKey)
	assert.ElementsMatch(t, []string{"transcribe", "enhanced_export"}, effective.AppEntitlements[0].Features)
	assert.Equal(t, int64(150), effective.AppEntitlements[0].Limits[0].Value)

	sendSubscriptionEvent("evt_scoped_audio_downgrade", "customer.subscription.updated", map[string]interface{}{
		"id": "sub_audio_only", "customer": "cus_atomicity", "customer_email": email, "status": "active",
		"metadata": map[string]interface{}{"business_account_id": accountID, "plan_tier": "pro", "bundle_key": "business_suite"},
		"items":    map[string]interface{}{"data": []interface{}{map[string]interface{}{"price": map[string]interface{}{"id": "price_audio_only"}}}},
	})
	effective, err = accounts.GetEntitlementsForBusinessAccountContext(t.Context(), email, accountID)
	require.NoError(t, err)
	assert.Len(t, effective.AppEntitlements, 1)
	assert.ElementsMatch(t, []string{"transcribe"}, effective.AppEntitlements[0].Features)
	assert.Equal(t, int64(100), effective.AppEntitlements[0].Limits[0].Value)
}

// TestHandleCheckoutCompleted_RetryAfterRollback_Succeeds simulates Stripe's
// automatic webhook retry after a rolled-back failure: inject failure, fire,
// observe rollback, clear the constraint, fire again, observe success. This
// exercises the ON CONFLICT safety net the fix relies on for healing.
func TestHandleCheckoutCompleted_RetryAfterRollback_Succeeds(t *testing.T) {
	db := setupTestDB(t)
	resetStripeTestData(t, db)
	setupCheckoutAtomicitySchema(t, db)

	productID := upsertTestBundleProduct(t, db, "business_suite", "Business Suite", "prod_retry_after_rb", "production", 1000000, 0.001, "credits")
	insertBundlePrice(t, db, productID, "price_retry_after_rb", "Pro Plan", "pro", "month", "usd", 2900, false, "", 0, 0, "", 1000000, 0, 1, 10, "none", sessionTypeSubscription, map[string]interface{}{})

	insertPendingSubscriptionCheckout(t, db, "cs_retry_after_rb", "retry-rb@example.com", "price_retry_after_rb", 2900)

	_, err := db.Exec(`ALTER TABLE subscriptions ADD CONSTRAINT test_block_active_retry CHECK (status <> 'active')`)
	require.NoError(t, err)

	service := ConfigureStripeService(t, db, DefaultStripeTestConfig().WithKeys("pk_test", "sk_test", "whsec_retry"), nil)
	payload, sig := buildCheckoutCompletedWebhook(t, "whsec_retry", "evt_retry_after_rb", "cs_retry_after_rb", "sub_retry_after_rb", "retry-rb@example.com", 2900)

	require.Error(t, service.HandleWebhook(payload, sig), "first attempt must fail")

	var sessionStatus string
	require.NoError(t, db.QueryRow(`SELECT status FROM checkout_sessions WHERE session_id = $1`, "cs_retry_after_rb").Scan(&sessionStatus))
	require.Equal(t, "pending", sessionStatus)

	_, err = db.Exec(`ALTER TABLE subscriptions DROP CONSTRAINT test_block_active_retry`)
	require.NoError(t, err)

	// Stripe replay of the same webhook: same event id, same payload, same signature window.
	payload2, sig2 := buildCheckoutCompletedWebhook(t, "whsec_retry", "evt_retry_after_rb", "cs_retry_after_rb", "sub_retry_after_rb", "retry-rb@example.com", 2900)
	require.NoError(t, service.HandleWebhook(payload2, sig2), "retry after clearing the failure must succeed")

	require.NoError(t, db.QueryRow(`SELECT status FROM checkout_sessions WHERE session_id = $1`, "cs_retry_after_rb").Scan(&sessionStatus))
	assert.Equal(t, "complete", sessionStatus)

	var subStatus string
	require.NoError(t, db.QueryRow(`SELECT status FROM subscriptions WHERE subscription_id = $1`, "sub_retry_after_rb").Scan(&subStatus))
	assert.Equal(t, "active", subStatus)
}
