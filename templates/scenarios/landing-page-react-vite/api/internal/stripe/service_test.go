package stripe_test

import (
	"crypto/hmac"
	"crypto/sha256"
	"database/sql"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"landing-page-react-vite-api/internal/paymentsettings"
	"landing-page-react-vite-api/internal/plan"
	"landing-page-react-vite-api/internal/stripe"
	"landing-page-react-vite-api/internal/testutil/billingfix"
	"landing-page-react-vite-api/internal/testutil/pgtest"
	"testing"
	"time"

	"github.com/stretchr/testify/require"

	landingv1 "github.com/vrooli/vrooli/packages/proto/gen/go/landing-page-react-vite/v1"
)

func newDB(t *testing.T) *sql.DB {
	t.Helper()
	db := pgtest.NewDB(t)
	pgtest.Apply(t, db, plan.Schema, paymentsettings.Schema, stripe.Schema)
	for _, table := range []string{
		"subscription_schedules", "subscriptions", "checkout_sessions",
		"credit_transactions", "credit_wallets", "bundle_prices", "bundle_products", "payment_settings",
	} {
		_, err := db.Exec("DELETE FROM " + table)
		require.NoError(t, err)
	}
	return db
}

func setStripeEnv(t *testing.T, pub, secret, webhook string) {
	t.Helper()
	t.Setenv("STRIPE_PUBLISHABLE_KEY", pub)
	t.Setenv("STRIPE_SECRET_KEY", secret)
	t.Setenv("STRIPE_WEBHOOK_SECRET", webhook)
}

func sign(payload []byte, timestamp, secret string) string {
	mac := hmac.New(sha256.New, []byte(secret))
	mac.Write([]byte(timestamp + "." + string(payload)))
	return "t=" + timestamp + ",v1=" + hex.EncodeToString(mac.Sum(nil))
}

func TestConfigSnapshot(t *testing.T) {
	db := newDB(t)
	setStripeEnv(t, "pk_test_123", "sk_test_123", "whsec_123")

	snapshot := stripe.NewService(db, nil, nil).ConfigSnapshot()
	require.True(t, snapshot.PublishableKeySet)
	require.NotEmpty(t, snapshot.PublishableKeyPreview)
	require.True(t, snapshot.SecretKeySet)
	require.True(t, snapshot.WebhookSecretSet)
}

func TestCreateCheckoutSession(t *testing.T) {
	db := newDB(t)
	setStripeEnv(t, "pk_test_valid", "sk_test_valid", "whsec_test_valid")
	productID := billingfix.UpsertBundleProduct(t, db, "business_suite", "Business Suite", "prod_business_suite", "production", 1_000_000, 0.001, "credits")
	billingfix.InsertBundlePrice(t, db, productID, "price_123", "Test Plan", "pro", "month", "usd",
		5000, true, "flat_amount", 100, 1, "test_intro_lookup", 1_000_000, 0, 1, 10, "none", "subscription", nil)

	svc := stripe.NewService(db, nil, nil)
	session, err := svc.CreateCheckoutSession("price_123", "/success", "/cancel", "test@example.com")
	require.NoError(t, err)
	require.Equal(t, "test@example.com", session.CustomerEmail)
	require.Equal(t, landingv1.CheckoutSessionStatus_CHECKOUT_SESSION_STATUS_OPEN, session.Status)

	var count int
	require.NoError(t, db.QueryRow(`SELECT COUNT(*) FROM checkout_sessions WHERE customer_email = $1`, "test@example.com").Scan(&count))
	require.Equal(t, 1, count)
}

func TestCreateCheckoutSessionRequiresSecret(t *testing.T) {
	db := newDB(t)
	// No STRIPE_SECRET_KEY in env and no DB settings -> placeholder, hasSecret false.
	t.Setenv("STRIPE_PUBLISHABLE_KEY", "pk_test_missing_secret")
	svc := stripe.NewService(db, nil, nil)
	_, err := svc.CreateCheckoutSession("price_missing_secret", "/ok", "/cancel", "no-secret@example.com")
	require.Error(t, err)
}

func TestVerifyWebhookSignature(t *testing.T) {
	db := newDB(t)
	setStripeEnv(t, "pk_test_valid", "sk_test_valid", "whsec_test_secret")
	svc := stripe.NewService(db, nil, nil)

	payload := []byte(`{"type":"checkout.session.completed","data":{}}`)
	ts := fmt.Sprintf("%d", time.Now().Unix())
	require.True(t, svc.VerifyWebhookSignature(payload, sign(payload, ts, "whsec_test_secret")))
	require.False(t, svc.VerifyWebhookSignature(payload, "t="+ts+",v1=invalid_signature"))
	require.False(t, svc.VerifyWebhookSignature(payload, ""))
}

func TestHandleWebhookCheckoutCompleted(t *testing.T) {
	db := newDB(t)
	setStripeEnv(t, "pk_test_valid", "sk_test_valid", "whsec_test_secret")
	productID := billingfix.UpsertBundleProduct(t, db, "business_suite", "Business Suite", "prod_business_suite", "production", 1_000_000, 0.001, "credits")
	billingfix.InsertBundlePrice(t, db, productID, "price_123", "Test Plan", "pro", "month", "usd",
		5000, true, "flat_amount", 100, 1, "test_intro_lookup", 1_000_000, 0, 1, 10, "none", "subscription", nil)

	_, err := db.Exec(`INSERT INTO checkout_sessions (session_id, customer_email, price_id, status) VALUES ($1,$2,$3,$4)`,
		"cs_test_123", "test@example.com", "price_123", "open")
	require.NoError(t, err)

	svc := stripe.NewService(db, nil, nil)
	event := map[string]interface{}{
		"type": "checkout.session.completed",
		"data": map[string]interface{}{"object": map[string]interface{}{
			"id": "cs_test_123", "customer_email": "test@example.com", "subscription": "sub_123",
		}},
	}
	payload, _ := json.Marshal(event)
	require.NoError(t, svc.HandleWebhook(payload, sign(payload, "1234567890", "whsec_test_secret")))

	var status, subscriptionID string
	require.NoError(t, db.QueryRow(`SELECT status, subscription_id FROM checkout_sessions WHERE session_id = $1`, "cs_test_123").Scan(&status, &subscriptionID))
	require.Equal(t, "complete", status)
	require.Equal(t, "sub_123", subscriptionID)

	var subStatus string
	require.NoError(t, db.QueryRow(`SELECT status FROM subscriptions WHERE subscription_id = $1`, "sub_123").Scan(&subStatus))
	require.Equal(t, "active", subStatus)
}

func TestVerifySubscription(t *testing.T) {
	db := newDB(t)
	setStripeEnv(t, "pk_test_valid", "sk_test_valid", "whsec_test_valid")
	_, err := db.Exec(`INSERT INTO subscriptions (subscription_id, customer_email, status) VALUES ($1,$2,$3)`,
		"sub_test_123", "active@example.com", "active")
	require.NoError(t, err)

	svc := stripe.NewService(db, nil, nil)
	result, err := svc.VerifySubscription("active@example.com")
	require.NoError(t, err)
	require.Equal(t, landingv1.SubscriptionState_SUBSCRIPTION_STATE_ACTIVE, result.State)
	require.NotNil(t, result.CachedAt)

	missing, err := svc.VerifySubscription("nonexistent@example.com")
	require.NoError(t, err)
	require.Equal(t, landingv1.SubscriptionState_SUBSCRIPTION_STATE_INACTIVE, missing.State)
}

func TestCancelSubscription(t *testing.T) {
	db := newDB(t)
	setStripeEnv(t, "pk_test_valid", "sk_test_valid", "whsec_test_valid")
	_, err := db.Exec(`INSERT INTO subscriptions (subscription_id, customer_email, status) VALUES ($1,$2,$3)`,
		"sub_cancel_test", "cancel@example.com", "active")
	require.NoError(t, err)

	svc := stripe.NewService(db, nil, nil)
	result, err := svc.CancelSubscription("cancel@example.com")
	require.NoError(t, err)
	require.NotNil(t, result.SubscriptionId)
	require.Equal(t, "sub_cancel_test", *result.SubscriptionId)
	require.Equal(t, landingv1.SubscriptionState_SUBSCRIPTION_STATE_CANCELED, result.State)

	var status string
	var canceledAt *time.Time
	require.NoError(t, db.QueryRow(`SELECT status, canceled_at FROM subscriptions WHERE subscription_id = $1`, "sub_cancel_test").Scan(&status, &canceledAt))
	require.Equal(t, "canceled", status)
	require.NotNil(t, canceledAt)

	_, err = svc.CancelSubscription("nonexistent@example.com")
	require.Error(t, err)
}

func TestVerifySubscriptionCacheWarning(t *testing.T) {
	db := newDB(t)
	setStripeEnv(t, "pk_test_valid", "sk_test_valid", "whsec_test_valid")
	staleTime := time.Now().Add(-120 * time.Second)
	_, err := db.Exec(`INSERT INTO subscriptions (subscription_id, customer_email, status, updated_at) VALUES ($1,$2,$3,$4)`,
		"sub_stale", "stale@example.com", "active", staleTime)
	require.NoError(t, err)

	result, err := stripe.NewService(db, nil, nil).VerifySubscription("stale@example.com")
	require.NoError(t, err)
	require.GreaterOrEqual(t, result.CacheAgeMs, int64(60000))
}
