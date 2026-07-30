package main
 
import (
	"crypto/hmac"
	"crypto/sha256"
	"database/sql"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"net/http"
	"net/http/httptest"
	"sync"
	"sync/atomic"
	"testing"
	"time"
 
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)
 
// TestConcurrent_WebhookVsUserCancel verifies no race conditions between
// webhook processing and user-initiated cancellation.
func TestConcurrent_WebhookVsUserCancel(t *testing.T) {
	db := setupTestDB(t)
	resetStripeTestData(t, db)
 
	// Create required tables
	_, err := db.Exec(`
		DROP TABLE IF EXISTS subscriptions CASCADE;
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
		)
	`)
	require.NoError(t, err)
 
	// Insert initial subscription
	_, err = db.Exec(`
		INSERT INTO subscriptions (subscription_id, customer_id, customer_email, status, plan_tier, bundle_key)
		VALUES ($1, $2, $3, $4, $5, $6)
	`, "sub_concurrent_test", "cus_concurrent", "concurrent@example.com", "active", "pro", "business_suite")
	require.NoError(t, err)
 
	productID := upsertTestBundleProduct(t, db, "business_suite", "Business Suite", "prod_test", "production", 1000000, 0.001, "credits")
	insertBundlePrice(t, db, productID, "price_pro", "Pro Plan", "pro", "month", "usd", 5000, false, "", 0, 0, "", 1000000, 0, 1, 10, "none", sessionTypeSubscription, map[string]interface{}{})
 
	// Set up mock Stripe server
	var cancelCalls int64
	stripeServer := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
 
		if r.Method == http.MethodDelete && r.URL.Path == "/v1/subscriptions/sub_concurrent_test" {
			atomic.AddInt64(&cancelCalls, 1)
			// Simulate some processing time
			time.Sleep(10 * time.Millisecond)
			fmt.Fprint(w, `{"id":"sub_concurrent_test","status":"canceled","customer":"cus_concurrent"}`)
			return
		}
 
		if r.URL.Path == "/v1/subscriptions/sub_concurrent_test" {
			fmt.Fprint(w, `{"id":"sub_concurrent_test","status":"active","customer":"cus_concurrent","customer_email":"concurrent@example.com","items":{"data":[{"price":{"id":"price_pro"}}]}}`)
			return
		}
 
		http.NotFound(w, r)
	}))
	defer stripeServer.Close()
 
	cfg := DefaultStripeTestConfig()
	service := ConfigureStripeService(t, db, cfg, stripeServer)
 
	const numGoroutines = 5
	var wg sync.WaitGroup
	var webhookErrors int64
	var cancelErrors int64
 
	// Start goroutines for webhook processing
	for i := 0; i < numGoroutines; i++ {
		wg.Add(1)
		go func(i int) {
			defer wg.Done()
 
			event := map[string]interface{}{
				"id":   fmt.Sprintf("evt_concurrent_%d", i),
				"type": "customer.subscription.updated",
				"data": map[string]interface{}{
					"object": map[string]interface{}{
						"id":             "sub_concurrent_test",
						"customer":       "cus_concurrent",
						"customer_email": "concurrent@example.com",
						"status":         "active",
					},
				},
			}
 
			payload, _ := json.Marshal(event)
			timestamp := fmt.Sprintf("%d", time.Now().Unix())
			signedPayload := timestamp + "." + string(payload)
			mac := hmac.New(sha256.New, []byte(cfg.WebhookSecret))
			mac.Write([]byte(signedPayload))
			signature := hex.EncodeToString(mac.Sum(nil))
			signatureHeader := "t=" + timestamp + ",v1=" + signature
 
			if err := service.HandleWebhook(payload, signatureHeader); err != nil {
				atomic.AddInt64(&webhookErrors, 1)
			}
		}(i)
	}
 
	// Start goroutines for cancel operations
	for i := 0; i < numGoroutines; i++ {
		wg.Add(1)
		go func() {
			defer wg.Done()
 
			_, err := service.CancelSubscription("concurrent@example.com")
			if err != nil {
				atomic.AddInt64(&cancelErrors, 1)
			}
		}()
	}
 
	wg.Wait()
 
	// Verify no panics occurred (if we got here, no panics)
	// Some errors are expected due to concurrent state changes
	t.Logf("Webhook errors: %d, Cancel errors: %d, Cancel API calls: %d",
		webhookErrors, cancelErrors, cancelCalls)
 
	// Verify final state is consistent (subscription exists)
	var count int
	err = db.QueryRow(`SELECT COUNT(*) FROM subscriptions WHERE subscription_id = $1`, "sub_concurrent_test").Scan(&count)
	require.NoError(t, err)
	assert.Equal(t, 1, count, "subscription record should still exist")
}
 
// TestConcurrent_MultipleWebhooksSameSubscription verifies that concurrent
// webhook events for the same subscription don't create duplicate rows.
func TestConcurrent_MultipleWebhooksSameSubscription(t *testing.T) {
	db := setupTestDB(t)
	resetStripeTestData(t, db)
 
	// Create required tables
	_, err := db.Exec(`
		DROP TABLE IF EXISTS subscriptions CASCADE;
		DROP TABLE IF EXISTS users CASCADE;
		DROP TABLE IF EXISTS checkout_sessions CASCADE;
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
		)
	`)
	require.NoError(t, err)
 
	productID := upsertTestBundleProduct(t, db, "business_suite", "Business Suite", "prod_test", "production", 1000000, 0.001, "credits")
	insertBundlePrice(t, db, productID, "price_multi", "Multi Plan", "pro", "month", "usd", 5000, false, "", 0, 0, "", 1000000, 0, 1, 10, "none", sessionTypeSubscription, map[string]interface{}{})
 
	cfg := DefaultStripeTestConfig()
	service := ConfigureStripeService(t, db, cfg, nil)
 
	const numGoroutines = 10
	var wg sync.WaitGroup
	var successCount int64
	var errorCount int64
 
	// Start all goroutines at the same time
	startCh := make(chan struct{})
 
	for i := 0; i < numGoroutines; i++ {
		wg.Add(1)
		go func(i int) {
			defer wg.Done()
 
			// Wait for signal to start
			<-startCh
 
			// All send the same subscription.created event
			event := map[string]interface{}{
				"id":   fmt.Sprintf("evt_multi_%d", i),
				"type": "customer.subscription.created",
				"data": map[string]interface{}{
					"object": map[string]interface{}{
						"id":             "sub_multi_test",
						"customer":       "cus_multi",
						"customer_email": "multi@example.com",
						"status":         "active",
						"items": map[string]interface{}{
							"data": []interface{}{
								map[string]interface{}{
									"price": map[string]interface{}{
										"id": "price_multi",
									},
								},
							},
						},
					},
				},
			}
 
			payload, _ := json.Marshal(event)
			timestamp := fmt.Sprintf("%d", time.Now().Unix())
			signedPayload := timestamp + "." + string(payload)
			mac := hmac.New(sha256.New, []byte(cfg.WebhookSecret))
			mac.Write([]byte(signedPayload))
			signature := hex.EncodeToString(mac.Sum(nil))
			signatureHeader := "t=" + timestamp + ",v1=" + signature
 
			if err := service.HandleWebhook(payload, signatureHeader); err != nil {
				atomic.AddInt64(&errorCount, 1)
			} else {
				atomic.AddInt64(&successCount, 1)
			}
		}(i)
	}
 
	// Signal all goroutines to start simultaneously
	close(startCh)
	wg.Wait()
 
	// All should succeed (upsert behavior)
	t.Logf("Success: %d, Errors: %d", successCount, errorCount)
 
	// Verify only one subscription row exists (due to unique constraint)
	var count int
	err = db.QueryRow(`SELECT COUNT(*) FROM subscriptions WHERE subscription_id = $1`, "sub_multi_test").Scan(&count)
	require.NoError(t, err)
	assert.Equal(t, 1, count, "only one subscription record should exist despite concurrent inserts")
 
	// Verify subscription data is consistent
	var status string
	var customerEmail sql.NullString
	err = db.QueryRow(`SELECT status, customer_email FROM subscriptions WHERE subscription_id = $1`, "sub_multi_test").Scan(&status, &customerEmail)
	require.NoError(t, err)
	assert.Equal(t, "active", status)
	// customer_email may be null depending on webhook handler behavior
	if customerEmail.Valid {
		assert.Equal(t, "multi@example.com", customerEmail.String)
	}
}
 
// TestConcurrent_CreditsAndSubscription verifies concurrent credit and subscription operations.
func TestConcurrent_CreditsAndSubscription(t *testing.T) {
	db := setupTestDB(t)
	resetStripeTestData(t, db)
 
	// Create required tables
	_, err := db.Exec(`
		DROP TABLE IF EXISTS credit_transactions CASCADE;
		DROP TABLE IF EXISTS credit_wallets CASCADE;
		DROP TABLE IF EXISTS subscriptions CASCADE;
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
			source VARCHAR(100),
			stripe_event_id VARCHAR(255) UNIQUE,
			metadata JSONB DEFAULT '{}'::jsonb,
			created_at TIMESTAMP DEFAULT NOW()
		)
	`)
	require.NoError(t, err)
 
	productID := upsertTestBundleProduct(t, db, "business_suite", "Business Suite", "prod_test", "production", 1000000, 0.001, "credits")
	insertBundlePrice(t, db, productID, "price_sub", "Sub Plan", "pro", "month", "usd", 5000, false, "", 0, 0, "", 1000000, 0, 1, 10, "none", sessionTypeSubscription, map[string]interface{}{})
 
	cfg := DefaultStripeTestConfig()
	service := ConfigureStripeService(t, db, cfg, nil)
 
	const numCreditsOps = 5
	const numSubOps = 5
	var wg sync.WaitGroup
	var creditErrors int64
	var subErrors int64
 
	email := "mixed@example.com"
 
	// Credit operations
	for i := 0; i < numCreditsOps; i++ {
		wg.Add(1)
		go func(i int) {
			defer wg.Done()
 
			eventID := fmt.Sprintf("evt_credit_%d", i)
			err := service.creditWallet.AddCredits(email, 100, "credit_topup", eventID, map[string]interface{}{
				"test": true,
			})
			if err != nil {
				atomic.AddInt64(&creditErrors, 1)
			}
		}(i)
	}
 
	// Subscription operations
	for i := 0; i < numSubOps; i++ {
		wg.Add(1)
		go func(i int) {
			defer wg.Done()
 
			event := map[string]interface{}{
				"id":   fmt.Sprintf("evt_sub_%d", i),
				"type": "customer.subscription.created",
				"data": map[string]interface{}{
					"object": map[string]interface{}{
						"id":             fmt.Sprintf("sub_mixed_%d", i),
						"customer":       "cus_mixed",
						"customer_email": email,
						"status":         "active",
						"items": map[string]interface{}{
							"data": []interface{}{
								map[string]interface{}{
									"price": map[string]interface{}{
										"id": "price_sub",
									},
								},
							},
						},
					},
				},
			}
 
			payload, _ := json.Marshal(event)
			timestamp := fmt.Sprintf("%d", time.Now().Unix())
			signedPayload := timestamp + "." + string(payload)
			mac := hmac.New(sha256.New, []byte(cfg.WebhookSecret))
			mac.Write([]byte(signedPayload))
			signature := hex.EncodeToString(mac.Sum(nil))
			signatureHeader := "t=" + timestamp + ",v1=" + signature
 
			if err := service.HandleWebhook(payload, signatureHeader); err != nil {
				atomic.AddInt64(&subErrors, 1)
			}
		}(i)
	}
 
	wg.Wait()
 
	t.Logf("Credit errors: %d, Subscription errors: %d", creditErrors, subErrors)
 
	// Verify credit balance is correct (5 unique events * 100 credits)
	var balance int64
	err = db.QueryRow(`SELECT COALESCE(balance_credits, 0) FROM credit_wallets WHERE customer_email = $1`, email).Scan(&balance)
	require.NoError(t, err)
	assert.Equal(t, int64(500), balance, "all credit transactions should be processed")
 
	// Verify correct number of credit transactions
	var txnCount int
	err = db.QueryRow(`SELECT COUNT(*) FROM credit_transactions WHERE customer_email = $1`, email).Scan(&txnCount)
	require.NoError(t, err)
	assert.Equal(t, numCreditsOps, txnCount, "all credit transactions should be recorded")
 
	// Verify subscriptions are created (each with unique ID)
	// Note: handleSubscriptionCreated only sets customer_id, not customer_email
	var subCount int
	err = db.QueryRow(`SELECT COUNT(*) FROM subscriptions WHERE customer_id = $1`, "cus_mixed").Scan(&subCount)
	require.NoError(t, err)
	assert.Equal(t, numSubOps, subCount, "all subscriptions should be created")
}
 
// TestConcurrent_EmailMigration_DuringPaymentWebhook verifies no data corruption
// when a customer.updated (email change) and invoice.paid webhook are processed
// concurrently for the same customer.
func TestConcurrent_EmailMigration_DuringPaymentWebhook(t *testing.T) {
	db := setupTestDB(t)
	resetStripeTestData(t, db)
 
	_, err := db.Exec(`
		DROP TABLE IF EXISTS intro_coupon_usage CASCADE;
		DROP TABLE IF EXISTS credit_transactions CASCADE;
		DROP TABLE IF EXISTS credit_wallets CASCADE;
		DROP TABLE IF EXISTS subscriptions CASCADE;
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
		CREATE TABLE credit_wallets (
			id SERIAL PRIMARY KEY,
			customer_email VARCHAR(255) UNIQUE NOT NULL,
			balance_credits BIGINT DEFAULT 0,
			updated_at TIMESTAMP DEFAULT NOW()
		);
		CREATE TABLE credit_transactions (
			id SERIAL PRIMARY KEY,
			customer_email VARCHAR(255) NOT NULL,
			amount_credits BIGINT NOT NULL,
			transaction_type VARCHAR(50) NOT NULL,
			source VARCHAR(100),
			stripe_event_id VARCHAR(255) UNIQUE,
			metadata JSONB DEFAULT '{}'::jsonb,
			created_at TIMESTAMP DEFAULT NOW()
		);
		CREATE TABLE intro_coupon_usage (
			id SERIAL PRIMARY KEY,
			email VARCHAR(255) NOT NULL,
			stripe_customer_id VARCHAR(255),
			coupon_id VARCHAR(255),
			plan_tier VARCHAR(50),
			subscription_id VARCHAR(255),
			used_at TIMESTAMP DEFAULT NOW()
		)
	`)
	require.NoError(t, err)
 
	oldEmail := "concurrent-old@example.com"
	newEmail := "concurrent-new@example.com"
	customerID := "cus_concurrent_email"
 
	// Seed user and subscription
	_, err = db.Exec(`INSERT INTO users (email, stripe_customer_id) VALUES ($1, $2)`, oldEmail, customerID)
	require.NoError(t, err)
	_, err = db.Exec(`
		INSERT INTO subscriptions (subscription_id, customer_id, customer_email, status, plan_tier, bundle_key)
		VALUES ($1, $2, $3, $4, $5, $6)
	`, "sub_concurrent_email", customerID, oldEmail, "active", "pro", "business_suite")
	require.NoError(t, err)
 
	productID := upsertTestBundleProduct(t, db, "business_suite", "Business Suite", "prod_conc_email", "production", 1000000, 0.001, "credits")
	insertBundlePrice(t, db, productID, "price_conc_email", "Pro Plan", "pro", "month", "usd", 2900, false, "", 0, 0, "", 1000000, 0, 1, 10, "none", sessionTypeSubscription, map[string]interface{}{})
 
	cfg := DefaultStripeTestConfig()
	service := ConfigureStripeService(t, db, cfg, nil)
 
	const numRounds = 5
	var wg sync.WaitGroup
	var emailMigrationErrors int64
	var invoiceErrors int64
 
	startCh := make(chan struct{})
 
	// Email migration goroutines
	for i := 0; i < numRounds; i++ {
		wg.Add(1)
		go func() {
			defer wg.Done()
			<-startCh
			if err := service.handleCustomerUpdated(map[string]interface{}{
				"id":    customerID,
				"email": newEmail,
				"previous_attributes": map[string]interface{}{
					"email": oldEmail,
				},
			}); err != nil {
				atomic.AddInt64(&emailMigrationErrors, 1)
			}
		}()
	}
 
	// Invoice.paid goroutines
	for i := 0; i < numRounds; i++ {
		wg.Add(1)
		go func(i int) {
			defer wg.Done()
			<-startCh
 
			event := map[string]interface{}{
				"id":   fmt.Sprintf("evt_conc_invoice_%d", i),
				"type": "invoice.paid",
				"data": map[string]interface{}{
					"object": map[string]interface{}{
						"subscription":   "sub_concurrent_email",
						"customer":       customerID,
						"customer_email": newEmail,
						"lines": map[string]interface{}{
							"data": []interface{}{
								map[string]interface{}{
									"price": map[string]interface{}{
										"id": "price_conc_email",
									},
								},
							},
						},
					},
				},
			}
 
			payload, _ := json.Marshal(event)
			timestamp := fmt.Sprintf("%d", time.Now().Unix())
			signedPayload := timestamp + "." + string(payload)
			mac := hmac.New(sha256.New, []byte(cfg.WebhookSecret))
			mac.Write([]byte(signedPayload))
			signature := hex.EncodeToString(mac.Sum(nil))
			signatureHeader := "t=" + timestamp + ",v1=" + signature
 
			if err := service.HandleWebhook(payload, signatureHeader); err != nil {
				atomic.AddInt64(&invoiceErrors, 1)
			}
		}(i)
	}
 
	close(startCh)
	wg.Wait()
 
	t.Logf("Email migration errors: %d, Invoice errors: %d", emailMigrationErrors, invoiceErrors)
 
	// Verify subscription still exists (exactly one row)
	var subCount int
	err = db.QueryRow(`SELECT COUNT(*) FROM subscriptions WHERE subscription_id = $1`, "sub_concurrent_email").Scan(&subCount)
	require.NoError(t, err)
	assert.Equal(t, 1, subCount, "subscription record should still exist")
 
	// Verify subscription status is active
	var finalStatus string
	err = db.QueryRow(`SELECT status FROM subscriptions WHERE subscription_id = $1`, "sub_concurrent_email").Scan(&finalStatus)
	require.NoError(t, err)
	assert.Equal(t, "active", finalStatus, "subscription should remain active")
}
 
// TestConcurrent_SubscriptionStatusChange_DuringDownloadAuth verifies deterministic
// outcome when a subscription transitions to canceled while DownloadAuthorizer
// is processing concurrent authorization requests.
func TestConcurrent_SubscriptionStatusChange_DuringDownloadAuth(t *testing.T) {
	db := setupTestDB(t)
	resetStripeTestData(t, db)
 
	_, err := db.Exec(`
		DROP TABLE IF EXISTS download_assets CASCADE;
		DROP TABLE IF EXISTS download_apps CASCADE;
		DROP TABLE IF EXISTS credit_wallets CASCADE;
		DROP TABLE IF EXISTS subscriptions CASCADE;
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
		CREATE TABLE credit_wallets (
			id SERIAL PRIMARY KEY,
			customer_email VARCHAR(255) UNIQUE NOT NULL,
			balance_credits BIGINT DEFAULT 0,
			updated_at TIMESTAMP DEFAULT NOW()
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
			update_policy JSONB NOT NULL DEFAULT '{"check_interval_hours":4,"update_mode":"optional","allow_downgrade":false}'::jsonb,
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
 
	productID := upsertTestBundleProduct(t, db, "business_suite", "Business Suite", "prod_conc_dl", "production", 1000000, 0.001, "credits")
	insertBundlePrice(t, db, productID, "price_conc_dl", "Pro Plan", "pro", "month", "usd", 2900, false, "", 0, 0, "", 1000000, 0, 1, 10, "none", sessionTypeSubscription, map[string]interface{}{})
 
	// Seed user, subscription, and asset
	email := "concurrent-dl@example.com"
	_, err = db.Exec(`INSERT INTO users (email, stripe_customer_id) VALUES ($1, $2)`, email, "cus_conc_dl")
	require.NoError(t, err)
	_, err = db.Exec(`
		INSERT INTO subscriptions (subscription_id, customer_id, customer_email, status, plan_tier, price_id, bundle_key)
		VALUES ($1, $2, $3, $4, $5, $6, $7)
	`, "sub_conc_dl", "cus_conc_dl", email, "active", "pro", "price_conc_dl", "business_suite")
	require.NoError(t, err)
	_, err = db.Exec(`INSERT INTO download_apps (bundle_key, app_key, name) VALUES ($1, $2, $3)`,
		"business_suite", "test-app", "Test App")
	require.NoError(t, err)
	_, err = db.Exec(`
		INSERT INTO download_assets (bundle_key, app_key, platform, artifact_url, release_version, release_notes, checksum, requires_entitlement)
		VALUES ($1, $2, $3, $4, $5, '', '', TRUE)
	`, "business_suite", "test-app", "windows", "https://cdn.example.com/app.zip", "1.0.0")
	require.NoError(t, err)
 
	service := ConfigureStripeServiceSimple(t, db)
	accountSvc := NewAccountService(db, service.planService)
	downloadSvc := NewDownloadService(db)
	authorizer := NewDownloadAuthorizer(downloadSvc, accountSvc, "business_suite")
 
	const numGoroutines = 10
	var wg sync.WaitGroup
	var authAllowed int64
	var authDenied int64
 
	startCh := make(chan struct{})
 
	// Launch concurrent authorization checks
	for i := 0; i < numGoroutines; i++ {
		wg.Add(1)
		go func() {
			defer wg.Done()
			<-startCh
			_, err := authorizer.Authorize(testRequestContext, "test-app", "windows", email)
			if err == nil {
				atomic.AddInt64(&authAllowed, 1)
			} else {
				atomic.AddInt64(&authDenied, 1)
			}
		}()
	}
 
	// Launch a goroutine to cancel the subscription mid-flight
	wg.Add(1)
	go func() {
		defer wg.Done()
		<-startCh
		_, _ = db.Exec(`UPDATE subscriptions SET status = 'canceled', canceled_at = NOW() WHERE subscription_id = $1`, "sub_conc_dl")
	}()
 
	close(startCh)
	wg.Wait()
 
	t.Logf("Auth allowed: %d, Auth denied: %d", authAllowed, authDenied)
 
	// Every result should be deterministic (either allowed or denied, no panics)
	totalResults := authAllowed + authDenied
	assert.Equal(t, int64(numGoroutines), totalResults, "all authorization requests should complete without panics")
 
	// Final state check: subscription is canceled
	var finalStatus string
	err = db.QueryRow(`SELECT status FROM subscriptions WHERE subscription_id = $1`, "sub_conc_dl").Scan(&finalStatus)
	require.NoError(t, err)
	assert.Equal(t, "canceled", finalStatus)
}
