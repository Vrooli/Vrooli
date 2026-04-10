package main

import (
	"crypto/hmac"
	"crypto/sha256"
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

// TestHandleWebhook_MissingEventID_ReturnsError verifies that webhooks without
// an event ID are rejected to prevent unsafe processing.
func TestHandleWebhook_MissingEventID_ReturnsError(t *testing.T) {
	db := setupTestDB(t)
	defer db.Close()

	cfg := DefaultStripeTestConfig().WithKeys("pk_test_valid", "sk_test_valid", "whsec_test_secret")
	service := ConfigureStripeService(t, db, cfg, nil)

	tests := []struct {
		name    string
		event   map[string]interface{}
		wantErr string
	}{
		{
			name: "missing id field",
			event: map[string]interface{}{
				"type": "checkout.session.completed",
				"data": map[string]interface{}{
					"object": map[string]interface{}{
						"id": "cs_test_123",
					},
				},
			},
			wantErr: "missing or invalid event ID",
		},
		{
			name: "empty id field",
			event: map[string]interface{}{
				"id":   "",
				"type": "checkout.session.completed",
				"data": map[string]interface{}{
					"object": map[string]interface{}{
						"id": "cs_test_123",
					},
				},
			},
			wantErr: "missing or invalid event ID",
		},
		{
			name: "id is not string",
			event: map[string]interface{}{
				"id":   12345,
				"type": "checkout.session.completed",
				"data": map[string]interface{}{
					"object": map[string]interface{}{
						"id": "cs_test_123",
					},
				},
			},
			wantErr: "missing or invalid event ID",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			payload, _ := json.Marshal(tt.event)
			timestamp := fmt.Sprintf("%d", time.Now().Unix())
			signedPayload := timestamp + "." + string(payload)
			mac := hmac.New(sha256.New, []byte("whsec_test_secret"))
			mac.Write([]byte(signedPayload))
			signature := hex.EncodeToString(mac.Sum(nil))
			signatureHeader := "t=" + timestamp + ",v1=" + signature

			err := service.HandleWebhook(payload, signatureHeader)
			require.Error(t, err)
			assert.Contains(t, err.Error(), tt.wantErr)
		})
	}
}

// TestAddCredits_DuplicateEventID_ProcessesOnce verifies that the same Stripe
// event ID only results in credits being added once.
func TestAddCredits_DuplicateEventID_ProcessesOnce(t *testing.T) {
	db := setupTestDB(t)
	defer db.Close()
	resetStripeTestData(t, db)

	// Ensure credit tables exist with unique constraint
	_, err := db.Exec(`
		DROP TABLE IF EXISTS credit_transactions CASCADE;
		DROP TABLE IF EXISTS credit_wallets CASCADE;
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
			stripe_event_id VARCHAR(255) UNIQUE,
			metadata JSONB DEFAULT '{}'::jsonb,
			created_at TIMESTAMP DEFAULT NOW()
		);
		CREATE UNIQUE INDEX IF NOT EXISTS idx_credit_transactions_stripe_event_id
		ON credit_transactions(stripe_event_id) WHERE stripe_event_id IS NOT NULL;
	`)
	require.NoError(t, err)

	productID := upsertTestBundleProduct(t, db, "business_suite", "Business Suite", "prod_test", "production", 100, 0.001, "credits")
	insertBundlePrice(t, db, productID, "price_credits", "Credits Pack", "credits", "one_time", "usd", 1000, false, "", 0, 0, "", 100, 0, 1, 10, "none", sessionTypeCreditsTopup, map[string]interface{}{})

	service := requireTestStripeService(t, db)

	// Add credits first time
	err = service.addCredits("test@example.com", 100, "credit_topup", "evt_test_123", map[string]interface{}{
		"test": true,
	})
	require.NoError(t, err)

	// Check balance
	var balance int64
	err = db.QueryRow(`SELECT balance_credits FROM credit_wallets WHERE customer_email = $1`, "test@example.com").Scan(&balance)
	require.NoError(t, err)
	assert.Equal(t, int64(100), balance)

	// Try to add credits with same event ID
	err = service.addCredits("test@example.com", 100, "credit_topup", "evt_test_123", map[string]interface{}{
		"test": true,
	})
	require.NoError(t, err) // Should succeed (idempotent)

	// Balance should still be 100, not 200
	err = db.QueryRow(`SELECT balance_credits FROM credit_wallets WHERE customer_email = $1`, "test@example.com").Scan(&balance)
	require.NoError(t, err)
	assert.Equal(t, int64(100), balance, "duplicate event ID should not add credits twice")

	// Count transactions - should only be 1
	var txnCount int
	err = db.QueryRow(`SELECT COUNT(*) FROM credit_transactions WHERE customer_email = $1`, "test@example.com").Scan(&txnCount)
	require.NoError(t, err)
	assert.Equal(t, 1, txnCount, "should only have one transaction record")
}

// TestAddCredits_ConcurrentSameEvent_OnlyOneSucceeds verifies that concurrent
// webhook processing with the same event ID only credits the user once.
func TestAddCredits_ConcurrentSameEvent_OnlyOneSucceeds(t *testing.T) {
	db := setupTestDB(t)
	defer db.Close()
	resetStripeTestData(t, db)

	// Ensure credit tables exist with unique constraint
	_, err := db.Exec(`
		DROP TABLE IF EXISTS credit_transactions CASCADE;
		DROP TABLE IF EXISTS credit_wallets CASCADE;
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
			stripe_event_id VARCHAR(255) UNIQUE,
			metadata JSONB DEFAULT '{}'::jsonb,
			created_at TIMESTAMP DEFAULT NOW()
		);
		CREATE UNIQUE INDEX IF NOT EXISTS idx_credit_transactions_stripe_event_id
		ON credit_transactions(stripe_event_id) WHERE stripe_event_id IS NOT NULL;
	`)
	require.NoError(t, err)

	productID := upsertTestBundleProduct(t, db, "business_suite", "Business Suite", "prod_test", "production", 100, 0.001, "credits")
	insertBundlePrice(t, db, productID, "price_credits", "Credits Pack", "credits", "one_time", "usd", 1000, false, "", 0, 0, "", 100, 0, 1, 10, "none", sessionTypeCreditsTopup, map[string]interface{}{})

	service := requireTestStripeService(t, db)

	const numGoroutines = 10
	eventID := "evt_concurrent_test_123"
	creditsPerAttempt := int64(100)

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

			err := service.addCredits("concurrent@example.com", creditsPerAttempt, "credit_topup", eventID, map[string]interface{}{
				"goroutine": i,
			})
			if err != nil {
				atomic.AddInt64(&errorCount, 1)
			} else {
				atomic.AddInt64(&successCount, 1)
			}
		}(i)
	}

	// Signal all goroutines to start simultaneously
	close(startCh)
	wg.Wait()

	// All should succeed (idempotent behavior)
	assert.Equal(t, int64(numGoroutines), successCount, "all calls should succeed (idempotent)")
	assert.Equal(t, int64(0), errorCount, "no errors expected")

	// But only one credit should have been added
	var balance int64
	err = db.QueryRow(`SELECT balance_credits FROM credit_wallets WHERE customer_email = $1`, "concurrent@example.com").Scan(&balance)
	require.NoError(t, err)
	assert.Equal(t, creditsPerAttempt, balance, "only one credit addition should succeed despite concurrent attempts")

	// Verify only one transaction was created
	var txnCount int
	err = db.QueryRow(`SELECT COUNT(*) FROM credit_transactions WHERE stripe_event_id = $1`, eventID).Scan(&txnCount)
	require.NoError(t, err)
	assert.Equal(t, 1, txnCount, "only one transaction record should exist")
}

// TestWebhook_CreditTopup_Idempotent verifies end-to-end webhook idempotency
// for credit topup scenarios.
func TestWebhook_CreditTopup_Idempotent(t *testing.T) {
	db := setupTestDB(t)
	defer db.Close()
	resetStripeTestData(t, db)

	// Create all required tables
	_, err := db.Exec(`
		DROP TABLE IF EXISTS credit_transactions CASCADE;
		DROP TABLE IF EXISTS credit_wallets CASCADE;
		DROP TABLE IF EXISTS checkout_sessions CASCADE;
		DROP TABLE IF EXISTS users CASCADE;

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
			stripe_event_id VARCHAR(255) UNIQUE,
			metadata JSONB DEFAULT '{}'::jsonb,
			created_at TIMESTAMP DEFAULT NOW()
		);
		CREATE UNIQUE INDEX IF NOT EXISTS idx_credit_transactions_stripe_event_id
		ON credit_transactions(stripe_event_id) WHERE stripe_event_id IS NOT NULL;

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
	`)
	require.NoError(t, err)

	// Set up bundle with credits_topup plan
	productID := upsertTestBundleProduct(t, db, "business_suite", "Business Suite", "prod_test", "production", 100, 0.001, "credits")
	insertBundlePrice(t, db, productID, "price_credits_topup", "Credits Pack", "credits", "one_time", "usd", 1000, false, "", 0, 0, "", 100, 0, 1, 10, "none", sessionTypeCreditsTopup, map[string]interface{}{})

	// Insert initial checkout session
	_, err = db.Exec(`
		INSERT INTO checkout_sessions (session_id, customer_email, price_id, status, session_type, amount_cents)
		VALUES ($1, $2, $3, $4, $5, $6)
	`, "cs_credits_test_123", "topup@example.com", "price_credits_topup", "open", sessionTypeCreditsTopup, 1000)
	require.NoError(t, err)

	stripeServer := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		fmt.Fprint(w, `{"id":"cs_credits_test_123","status":"complete"}`)
	}))
	defer stripeServer.Close()

	cfg := DefaultStripeTestConfig().WithKeys("pk_test_valid", "sk_test_valid", "whsec_test_secret")
	service := ConfigureStripeService(t, db, cfg, stripeServer)

	event := map[string]interface{}{
		"id":   "evt_credits_123",
		"type": "checkout.session.completed",
		"data": map[string]interface{}{
			"object": map[string]interface{}{
				"id":             "cs_credits_test_123",
				"customer_email": "topup@example.com",
				"customer":       "cus_credits_123",
				"amount_total":   1000,
			},
		},
	}

	payload, _ := json.Marshal(event)
	timestamp := fmt.Sprintf("%d", time.Now().Unix())
	signedPayload := timestamp + "." + string(payload)
	mac := hmac.New(sha256.New, []byte("whsec_test_secret"))
	mac.Write([]byte(signedPayload))
	signature := hex.EncodeToString(mac.Sum(nil))
	signatureHeader := "t=" + timestamp + ",v1=" + signature

	// Process webhook first time
	err = service.HandleWebhook(payload, signatureHeader)
	require.NoError(t, err)

	// Check credits were added
	// Calculation: (creditsPerUSD * amountCents) / 100 = (100 * 1000) / 100 = 1000 credits
	var balance int64
	err = db.QueryRow(`SELECT balance_credits FROM credit_wallets WHERE customer_email = $1`, "topup@example.com").Scan(&balance)
	require.NoError(t, err)
	assert.Equal(t, int64(1000), balance)

	// Process same webhook again (simulate retry)
	err = service.HandleWebhook(payload, signatureHeader)
	require.NoError(t, err)

	// Credits should not have doubled (should still be 1000, not 2000)
	err = db.QueryRow(`SELECT balance_credits FROM credit_wallets WHERE customer_email = $1`, "topup@example.com").Scan(&balance)
	require.NoError(t, err)
	assert.Equal(t, int64(1000), balance, "webhook retry should not add credits twice")
}
