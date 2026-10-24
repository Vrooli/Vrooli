package payments_test

import (
	"bytes"
	"context"
	"crypto/hmac"
	"crypto/sha256"
	"database/sql"
	"encoding/hex"
	"encoding/json"
	"landing-page-react-vite-api/internal/paymentsettings"
	"landing-page-react-vite-api/internal/plan"
	"landing-page-react-vite-api/internal/stripe"
	"landing-page-react-vite-api/internal/testutil/billingfix"
	"landing-page-react-vite-api/internal/testutil/pgtest"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"connectrpc.com/connect"
	"github.com/gorilla/mux"
	"github.com/stretchr/testify/require"

	landingv1 "github.com/vrooli/vrooli/packages/proto/gen/go/landing-page-react-vite/v1"

	paymentsH "landing-page-react-vite-api/handlers/payments"
)

func setup(t *testing.T) (*paymentsH.Deps, *sql.DB, http.Handler) {
	t.Helper()
	t.Setenv("STRIPE_PUBLISHABLE_KEY", "pk_test_handlers")
	t.Setenv("STRIPE_SECRET_KEY", "sk_test_handlers")
	t.Setenv("STRIPE_WEBHOOK_SECRET", "whsec_handlers")

	db := pgtest.NewDB(t)
	pgtest.Apply(t, db, plan.Schema, paymentsettings.Schema, stripe.Schema)
	for _, table := range []string{
		"subscription_schedules", "subscriptions", "checkout_sessions",
		"credit_transactions", "credit_wallets", "bundle_prices", "bundle_products", "payment_settings",
	} {
		_, err := db.Exec("DELETE FROM " + table)
		require.NoError(t, err)
	}

	planSvc := plan.NewService(db)
	settings := paymentsettings.NewService(db)
	stripeSvc := stripe.NewService(db, planSvc, settings)
	deps := &paymentsH.Deps{Stripe: stripeSvc, Plan: planSvc, PaymentSettings: settings}

	router := mux.NewRouter()
	paymentsH.Module(*deps).Mount(router)
	return deps, db, router
}

func sign(payload []byte, timestamp string) string {
	mac := hmac.New(sha256.New, []byte("whsec_handlers"))
	mac.Write([]byte(timestamp + "." + string(payload)))
	return "t=" + timestamp + ",v1=" + hex.EncodeToString(mac.Sum(nil))
}

func TestCreateCheckoutSessionValidation(t *testing.T) {
	deps, _, _ := setup(t)
	h := paymentsH.NewConnectHandler(*deps)
	_, err := h.CreateCheckoutSession(context.Background(), connect.NewRequest(&landingv1.CreateCheckoutSessionRequest{PriceId: ""}))
	require.Error(t, err)
	require.Equal(t, connect.CodeInvalidArgument, connect.CodeOf(err))
}

func TestCheckoutAndWebhookEndToEnd(t *testing.T) {
	deps, db, router := setup(t)
	productID := billingfix.UpsertBundleProduct(t, db, "business_suite", "Business Suite", "prod_handlers", "production", 1_000_000, 0.001, "credits")
	billingfix.InsertBundlePrice(t, db, productID, "price_handlers_sub", "Handlers Plan", "pro", "month", "usd",
		5000, true, "flat_amount", 100, 1, "intro_lookup", 1_000_000, 0, 1, 10, "none", "subscription",
		map[string]interface{}{"features": []string{"Handlers coverage"}})

	h := paymentsH.NewConnectHandler(*deps)
	resp, err := h.CreateCheckoutSession(context.Background(), connect.NewRequest(&landingv1.CreateCheckoutSessionRequest{
		PriceId: "price_handlers_sub", CustomerEmail: "handler@example.com", SuccessUrl: "/ok", CancelUrl: "/cancel",
	}))
	require.NoError(t, err)
	sessionID := resp.Msg.Session.SessionId

	event := map[string]interface{}{"type": "checkout.session.completed", "data": map[string]interface{}{
		"object": map[string]interface{}{"id": sessionID, "customer_email": "handler@example.com", "subscription": "sub_handlers_123", "amount_total": 5000},
	}}
	payload, _ := json.Marshal(event)
	req := httptest.NewRequest(http.MethodPost, "/api/v1/webhooks/stripe", bytes.NewReader(payload))
	req.Header.Set("Stripe-Signature", sign(payload, "1700000000"))
	rec := httptest.NewRecorder()
	router.ServeHTTP(rec, req)
	require.Equal(t, http.StatusOK, rec.Code, rec.Body.String())

	var status string
	require.NoError(t, db.QueryRow(`SELECT status FROM subscriptions WHERE subscription_id = $1`, "sub_handlers_123").Scan(&status))
	require.Equal(t, "active", status)

	verify, err := h.VerifySubscription(context.Background(), connect.NewRequest(&landingv1.VerifySubscriptionRequest{UserIdentity: "handler@example.com"}))
	require.NoError(t, err)
	require.Equal(t, landingv1.SubscriptionState_SUBSCRIPTION_STATE_ACTIVE, verify.Msg.Status.State)
}

func TestWebhookCreditTopup(t *testing.T) {
	// Credit top-ups convert against the configured default bundle, so point the
	// plan service at the seeded credits bundle.
	t.Setenv("BUNDLE_KEY", "credits_bundle")
	t.Setenv("BUNDLE_ENVIRONMENT", "production")
	deps, db, router := setup(t)
	productID := billingfix.UpsertBundleProduct(t, db, "credits_bundle", "Credits Bundle", "prod_credits", "production", 1000, 1, "credits")
	billingfix.InsertBundlePrice(t, db, productID, "price_credits_topup", "Credits Topup", "credits", "one_time", "usd",
		9900, false, "", 0, 0, "", 0, 0, 1, 0, "", "credits_topup", nil)

	h := paymentsH.NewConnectHandler(*deps)
	resp, err := h.CreateCheckoutSession(context.Background(), connect.NewRequest(&landingv1.CreateCheckoutSessionRequest{
		PriceId: "price_credits_topup", CustomerEmail: "credits@example.com", SuccessUrl: "/ok", CancelUrl: "/cancel",
	}))
	require.NoError(t, err)

	event := map[string]interface{}{"type": "checkout.session.completed", "data": map[string]interface{}{
		"object": map[string]interface{}{"id": resp.Msg.Session.SessionId, "customer_email": "credits@example.com", "subscription": "", "amount_total": 9900},
	}}
	payload, _ := json.Marshal(event)
	req := httptest.NewRequest(http.MethodPost, "/api/v1/webhooks/stripe", bytes.NewReader(payload))
	req.Header.Set("Stripe-Signature", sign(payload, "1700000001"))
	rec := httptest.NewRecorder()
	router.ServeHTTP(rec, req)
	require.Equal(t, http.StatusOK, rec.Code, rec.Body.String())

	var balance int64
	require.NoError(t, db.QueryRow(`SELECT balance_credits FROM credit_wallets WHERE customer_email = $1`, "credits@example.com").Scan(&balance))
	require.Greater(t, balance, int64(0))
}

func TestWebhookSubscriptionLifecycle(t *testing.T) {
	_, db, router := setup(t)
	_, err := db.Exec(`INSERT INTO subscriptions (subscription_id, status, created_at, updated_at) VALUES ($1,$2,NOW(),NOW())`, "sub_lifecycle", "active")
	require.NoError(t, err)

	for i, evt := range []struct {
		eventType string
		payload   map[string]interface{}
	}{
		{"customer.subscription.updated", map[string]interface{}{"id": "sub_lifecycle", "status": "past_due"}},
		{"customer.subscription.deleted", map[string]interface{}{"id": "sub_lifecycle", "status": "canceled"}},
	} {
		body := map[string]interface{}{"type": evt.eventType, "data": map[string]interface{}{"object": evt.payload}}
		raw, _ := json.Marshal(body)
		req := httptest.NewRequest(http.MethodPost, "/api/v1/webhooks/stripe", bytes.NewReader(raw))
		req.Header.Set("Stripe-Signature", sign(raw, time.Now().Format("150405")))
		rec := httptest.NewRecorder()
		router.ServeHTTP(rec, req)
		require.Equalf(t, http.StatusOK, rec.Code, "event %d (%s): %s", i, evt.eventType, rec.Body.String())
	}

	var status string
	var canceledAt *time.Time
	require.NoError(t, db.QueryRow(`SELECT status, canceled_at FROM subscriptions WHERE subscription_id = $1`, "sub_lifecycle").Scan(&status, &canceledAt))
	require.Equal(t, "canceled", status)
	require.NotNil(t, canceledAt)
}

func TestStripeSettingsRoundTrip(t *testing.T) {
	deps, _, _ := setup(t)
	h := paymentsH.NewConnectHandler(*deps)
	ctx := context.Background()

	pub, secret, webhook, dash := "pk_live_handlers", "sk_live_handlers", "whsec_live_handlers", "https://dashboard.stripe.com/test"
	_, err := h.UpdateStripeSettings(ctx, connect.NewRequest(&landingv1.UpdateStripeSettingsRequest{
		PublishableKey: &pub, SecretKey: &secret, WebhookSecret: &webhook, DashboardUrl: &dash,
	}))
	require.NoError(t, err)

	got, err := h.GetStripeSettings(ctx, connect.NewRequest(&landingv1.GetStripeSettingsRequest{}))
	require.NoError(t, err)
	require.True(t, got.Msg.Snapshot.PublishableKeySet)
	require.True(t, got.Msg.Snapshot.SecretKeySet)
	require.True(t, got.Msg.Snapshot.WebhookSecretSet)
	require.Equal(t, dash, got.Msg.Settings.GetDashboardUrl())
}

func TestVerifyAndCancelSubscription(t *testing.T) {
	deps, db, _ := setup(t)
	now := time.Now()
	_, err := db.Exec(`INSERT INTO subscriptions (subscription_id, customer_email, status, created_at, updated_at) VALUES ($1,$2,$3,$4,$4)`,
		"sub_cancelme", "cancelme@example.com", "active", now)
	require.NoError(t, err)

	h := paymentsH.NewConnectHandler(*deps)
	ctx := context.Background()
	verify, err := h.VerifySubscription(ctx, connect.NewRequest(&landingv1.VerifySubscriptionRequest{UserIdentity: "cancelme@example.com"}))
	require.NoError(t, err)
	require.Equal(t, landingv1.SubscriptionState_SUBSCRIPTION_STATE_ACTIVE, verify.Msg.Status.State)

	_, err = h.CancelSubscription(ctx, connect.NewRequest(&landingv1.CancelSubscriptionRequest{UserIdentity: "cancelme@example.com"}))
	require.NoError(t, err)

	var status string
	require.NoError(t, db.QueryRow(`SELECT status FROM subscriptions WHERE subscription_id = $1`, "sub_cancelme").Scan(&status))
	require.Equal(t, "canceled", status)
}

func TestGetPricingHandler(t *testing.T) {
	deps, db, _ := setup(t)
	t.Setenv("BUNDLE_KEY", "business_suite")
	t.Setenv("BUNDLE_ENVIRONMENT", "production")
	productID := billingfix.UpsertBundleProduct(t, db, "business_suite", "Business Suite", "prod_pricing", "production", 1_000_000, 0.01, "credits")
	billingfix.InsertBundlePrice(t, db, productID, "price_pricing_m", "Monthly", "pro", "month", "usd",
		4999, false, "none", 0, 0, "k", 1_000_000, 0, 1, 10, "none", "subscription", nil)

	// Rebuild plan service so it observes the BUNDLE_* env set above.
	deps.Plan = plan.NewService(db)
	h := paymentsH.NewConnectHandler(*deps)
	resp, err := h.GetPricing(context.Background(), connect.NewRequest(&landingv1.GetPricingRequest{}))
	require.NoError(t, err)
	require.Equal(t, "business_suite", resp.Msg.Pricing.Bundle.BundleKey)
	require.Len(t, resp.Msg.Pricing.Monthly, 1)
}
