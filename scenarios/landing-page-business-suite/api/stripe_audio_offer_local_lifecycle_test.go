package main

import (
	"context"
	"crypto/hmac"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"github.com/stretchr/testify/require"
	"landing-page-business-suite-api/internal/commerce"
)

// This test drives LPBS checkout, subscription persistence, webhook, and
// entitlement code through a loopback provider. It exercises a simulated $10
// monthly Audio-only offer and a fixed fake-provider proration amount without
// contacting Stripe or changing a configured/live billing account.
func TestAudioOnlyOfferLocalProviderLifecycle(t *testing.T) {
	db := setupTestDB(t)
	resetStripeTestData(t, db)
	setupCheckoutAtomicitySchema(t, db)
	productID := upsertTestBundleProduct(t, db, "business_suite", "Business Suite", "prod_local_lifecycle", "production", 1000000, 0.001, "credits")
	insertBundlePrice(t, db, productID, "price_AudioMonthly", "Audio Tools", "pro", "month", "usd", 1000, false, "", 0, 0, "", 0, 0, 1, 10, "none", sessionTypeSubscription, map[string]interface{}{
		"app_entitlements": map[string]interface{}{"audio-tools": map[string]interface{}{"features": []interface{}{"transcribe"}, "limits": map[string]interface{}{"ai_credits": int64(100)}}},
	})
	insertBundlePrice(t, db, productID, "price_BundleMonthly", "Business Suite", "pro", "month", "usd", 2900, false, "", 0, 0, "", 0, 0, 2, 20, "none", sessionTypeSubscription, map[string]interface{}{
		"app_entitlements": map[string]interface{}{
			"audio-tools": map[string]interface{}{"features": []interface{}{"transcribe", "enhanced_export"}, "limits": map[string]interface{}{"ai_credits": int64(200)}},
			"web-console": map[string]interface{}{"features": []interface{}{"agents"}},
		},
	})
	const accountID, email, secret = "acct_audio_lifecycle", "audio-lifecycle@example.invalid", "whsec_audio_lifecycle"
	seedBusinessAccountCreditEmail(t, db, accountID, email)

	var checkoutCount, updateCount, cancelCount int
	var prorationAmount int64
	currentPrice := "price_AudioMonthly"
	provider := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		switch r.Method + " " + r.URL.Path {
		case "POST /v1/checkout/sessions":
			if err := r.ParseForm(); err != nil {
				t.Errorf("parse checkout form: %v", err)
				http.Error(w, "bad form", 400)
				return
			}
			checkoutCount++
			if r.Form.Get("line_items[0][price]") != "price_AudioMonthly" || r.Form.Get("mode") != "subscription" || r.Form.Get("subscription_data[metadata][business_account_id]") != accountID {
				t.Errorf("checkout did not use the configured Audio-only account-scoped subscription: %v", r.Form)
				http.Error(w, "unexpected checkout", 400)
				return
			}
			_, _ = fmt.Fprint(w, `{"id":"cs_audio_local","url":"http://checkout.invalid/session","status":"open","subscription":"sub_audio_local","customer":"cus_audio_local","customer_email":"audio-lifecycle@example.invalid","amount_total":1000}`)
		case "POST /v1/subscriptions/sub_audio_local":
			if err := r.ParseForm(); err != nil {
				t.Errorf("parse subscription form: %v", err)
				http.Error(w, "bad form", 400)
				return
			}
			if r.Form.Get("items[0][price]") != "" {
				updateCount++
				if r.Form.Get("items[0][id]") != "si_audio_local" || r.Form.Get("items[0][price]") != "price_BundleMonthly" || r.Form.Get("proration_behavior") != "always_invoice" {
					t.Errorf("upgrade omitted provider proration behavior: %v", r.Form)
					http.Error(w, "unexpected upgrade", 400)
					return
				}
				// Fixed simulated invoice amount reported by the loopback provider.
				prorationAmount = 1700
				currentPrice = "price_BundleMonthly"
				_, _ = fmt.Fprintf(w, `{"id":"sub_audio_local","status":"active","latest_invoice":{"id":"in_proration_local","amount_due":%d,"currency":"usd"}}`, prorationAmount)
				return
			}
			if r.Form.Get("cancel_at_period_end") == "true" {
				cancelCount++
			}
			_, _ = fmt.Fprint(w, `{"id":"sub_audio_local","status":"active","cancel_at_period_end":true}`)
		case "GET /v1/subscriptions/sub_audio_local":
			_, _ = fmt.Fprintf(w, `{"id":"sub_audio_local","customer":"cus_atomicity","customer_email":"audio-lifecycle@example.invalid","status":"active","metadata":{"business_account_id":"acct_audio_lifecycle","bundle_key":"business_suite","plan_tier":"pro"},"items":{"data":[{"id":"si_audio_local","price":{"id":%q}}]}}`, currentPrice)
		case "GET /v1/customers/search":
			_, _ = fmt.Fprint(w, `{"data":[{"id":"cus_atomicity","email":"audio-lifecycle@example.invalid"}]}`)
		default:
			t.Errorf("unexpected local provider request %s %s", r.Method, r.URL.Path)
			http.NotFound(w, r)
		}
	}))
	defer provider.Close()
	service := ConfigureStripeService(t, db, DefaultStripeTestConfig().WithKeys("pk_test_local", "sk_test_local", secret).WithMockServer(provider), provider)

	created, err := service.CreateCheckoutSessionForBusinessAccount("price_AudioMonthly", "/success", "/cancel", email, accountID)
	require.NoError(t, err)
	require.Equal(t, "cs_audio_local", created.SessionId)
	payload, signature := buildCheckoutCompletedWebhook(t, secret, "evt_audio_checkout_local", "cs_audio_local", "sub_audio_local", email, 1000)
	require.NoError(t, service.HandleWebhook(payload, signature))
	accounts := commerce.NewService(db, globalTestPlanStore, commerce.Runtime{})
	accounts.SetCacheTTL(0)
	assertAudioGrant := func(wantStatus string, allowBundle bool, wantFeatures ...string) {
		t.Helper()
		got, getErr := accounts.GetEntitlementsForBusinessAccountContext(context.Background(), email, accountID)
		require.NoError(t, getErr)
		var audio []string
		var audioStatus string
		audioPresent := false
		var web []string
		for _, grant := range got.AppEntitlements {
			if grant.AppKey == "audio-tools" {
				audioPresent = true
				audio = grant.Features
				audioStatus = grant.Status
			}
			if grant.AppKey == "web-console" {
				web = grant.Features
			}
			if !allowBundle {
				require.NotEqual(t, "web-console", grant.AppKey, "Audio-only lifecycle must not grant Web Console")
			}
		}
		if len(wantFeatures) == 0 {
			require.False(t, audioPresent, "cancellation must remove the Audio Tools entitlement")
		} else {
			require.True(t, audioPresent, "expected Audio Tools entitlement")
			require.ElementsMatch(t, wantFeatures, audio)
			require.Equal(t, wantStatus, audioStatus)
		}
		if allowBundle {
			require.ElementsMatch(t, []string{"agents"}, web, "bundle upgrade should add the separately declared Web Console grant")
		} else {
			require.Empty(t, web)
		}
	}
	assertAudioGrant("active", false, "transcribe")

	// Use the owner operation with the loopback HTTP client. The response comes
	// from the fake provider and exposes its proration invoice amount; the signed
	// invoice webhook below remains the durable billing projection.
	t.Setenv("LPBS_ENVIRONMENT", "development")
	t.Setenv("VROOLI_ENVIRONMENT", "")
	t.Setenv("STRIPE_MODE", "test")
	_, err = service.changePY1TestClockSubscriptionPrice(context.Background(), "py1-audio-bundle-run", "sub_audio_local", "si_audio_local", "price_BundleMonthly", "upgrade")
	require.NoError(t, err)
	require.EqualValues(t, 1700, prorationAmount, "loopback provider must report the configured simulated upgrade invoice amount")
	upgrade, err := json.Marshal(map[string]interface{}{"id": "evt_audio_upgrade_local", "type": "customer.subscription.updated", "data": map[string]interface{}{"object": map[string]interface{}{
		"id": "sub_audio_local", "customer": "cus_audio_local", "customer_email": email, "status": "active",
		"metadata": map[string]interface{}{"business_account_id": accountID, "plan_tier": "pro", "bundle_key": "business_suite"},
		"items":    map[string]interface{}{"data": []interface{}{map[string]interface{}{"id": "si_audio_local", "price": map[string]interface{}{"id": "price_BundleMonthly"}}}},
	}}})
	require.NoError(t, err)
	require.NoError(t, handleLocalLifecycleEvent(service, secret, upgrade))
	proration, err := json.Marshal(map[string]interface{}{"id": "evt_audio_proration_local", "type": "invoice.paid", "data": map[string]interface{}{"object": map[string]interface{}{
		"id": "in_proration_local", "subscription": "sub_audio_local", "customer": "cus_audio_local", "customer_email": email,
		"amount_paid": prorationAmount, "currency": "usd", "billing_reason": "subscription_update",
	}}})
	require.NoError(t, err)
	require.NoError(t, handleLocalLifecycleEvent(service, secret, proration))
	assertAudioGrant("active", true, "enhanced_export", "transcribe")

	// A failed renewal moves the grant to the shared past_due state; the paid
	// recovery event restores active access. No entitlement is inferred from
	// provider transport success alone.
	failed, _ := json.Marshal(map[string]interface{}{"id": "evt_audio_failed_local", "type": "invoice.payment_failed", "data": map[string]interface{}{"object": map[string]interface{}{
		"id": "in_failed_local", "subscription": "sub_audio_local", "customer": "cus_audio_local", "customer_email": email, "currency": "usd",
	}}})
	require.NoError(t, handleLocalLifecycleEvent(service, secret, failed))
	assertAudioGrant("past_due", true, "enhanced_export", "transcribe")
	recovered, _ := json.Marshal(map[string]interface{}{"id": "evt_audio_recovered_local", "type": "invoice.paid", "data": map[string]interface{}{"object": map[string]interface{}{
		"id": "in_recovered_local", "subscription": "sub_audio_local", "customer": "cus_audio_local", "customer_email": email, "amount_paid": 2900, "currency": "usd", "billing_reason": "subscription_cycle",
	}}})
	require.NoError(t, handleLocalLifecycleEvent(service, secret, recovered))
	assertAudioGrant("active", true, "enhanced_export", "transcribe")

	cancelResult, err := service.subscriptionManagementService().Cancel(context.Background(), email)
	require.NoError(t, err)
	require.Equal(t, "active", commerce.SubscriptionStateLabel(cancelResult.GetState()), "scheduled cancellation must retain access until Stripe terminates the subscription")
	require.Nil(t, cancelResult.GetCanceledAt(), "scheduled cancellation is not a terminal cancellation timestamp")
	require.Contains(t, cancelResult.GetMessage(), "access remains active")
	assertAudioGrant("active", true, "enhanced_export", "transcribe")

	// Access ends only when the provider delivers authoritative termination.
	deleted, err := json.Marshal(map[string]interface{}{"id": "evt_audio_deleted_local", "type": "customer.subscription.deleted", "data": map[string]interface{}{"object": map[string]interface{}{
		"id": "sub_audio_local", "customer": "cus_audio_local", "customer_email": email, "status": "canceled",
		"metadata": map[string]interface{}{"business_account_id": accountID, "plan_tier": "pro", "bundle_key": "business_suite"},
	}}})
	require.NoError(t, err)
	require.NoError(t, handleLocalLifecycleEvent(service, secret, deleted))
	assertAudioGrant("canceled", false)
	require.Equal(t, 1, checkoutCount)
	require.Equal(t, 1, updateCount)
	require.Equal(t, 1, cancelCount)
}

func handleLocalLifecycleEvent(service *StripeService, secret string, payload []byte) error {
	timestamp := fmt.Sprint(time.Now().Unix())
	mac := hmac.New(sha256.New, []byte(secret))
	_, _ = mac.Write([]byte(timestamp + "."))
	_, _ = mac.Write(payload)
	signature := "t=" + timestamp + ",v1=" + hex.EncodeToString(mac.Sum(nil))
	return service.HandleWebhook(payload, signature)
}
