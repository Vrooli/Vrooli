package main

import (
	"context"
	"sync"
	"testing"
)

func TestStripeService_AddCredits(t *testing.T) {
	db := setupTestDB(t)
	defer db.Close()

	resetStripeTestData(t, db)
	upsertTestBundleProduct(t, db, "business_suite", "Business Suite", "prod_credits", "production", 1000, 1, "credits")
	service := ConfigureStripeServiceSimple(t, db)

	t.Run("adds credits to new wallet", func(t *testing.T) {
		err := service.AddCredits("addcredits@example.com", 1000, "test_topup", "evt_add_1", nil)
		if err != nil {
			t.Fatalf("AddCredits failed: %v", err)
		}

		balance, err := service.GetBalance("addcredits@example.com")
		if err != nil {
			t.Fatalf("GetBalance failed: %v", err)
		}
		if balance != 1000 {
			t.Errorf("expected balance 1000, got %d", balance)
		}
	})

	t.Run("accumulates credits with different events", func(t *testing.T) {
		err := service.AddCredits("addcredits@example.com", 500, "test_topup", "evt_add_2", nil)
		if err != nil {
			t.Fatalf("AddCredits failed: %v", err)
		}

		balance, err := service.GetBalance("addcredits@example.com")
		if err != nil {
			t.Fatalf("GetBalance failed: %v", err)
		}
		if balance != 1500 {
			t.Errorf("expected balance 1500, got %d", balance)
		}
	})

	t.Run("is idempotent with same event ID", func(t *testing.T) {
		// Try to add with same event ID
		err := service.AddCredits("addcredits@example.com", 1000, "test_topup", "evt_add_1", nil)
		if err != nil {
			t.Fatalf("AddCredits failed: %v", err)
		}

		// Balance should not change
		balance, err := service.GetBalance("addcredits@example.com")
		if err != nil {
			t.Fatalf("GetBalance failed: %v", err)
		}
		if balance != 1500 {
			t.Errorf("expected balance 1500 (unchanged), got %d", balance)
		}
	})

	t.Run("skips for empty email", func(t *testing.T) {
		err := service.AddCredits("", 1000, "test_topup", "evt_empty", nil)
		if err != nil {
			t.Fatalf("AddCredits should not error for empty email: %v", err)
		}
	})

	t.Run("skips for zero amount", func(t *testing.T) {
		err := service.AddCredits("zero@example.com", 0, "test_topup", "evt_zero", nil)
		if err != nil {
			t.Fatalf("AddCredits should not error for zero amount: %v", err)
		}

		balance, err := service.GetBalance("zero@example.com")
		if err != nil {
			t.Fatalf("GetBalance failed: %v", err)
		}
		if balance != 0 {
			t.Errorf("expected balance 0, got %d", balance)
		}
	})

	t.Run("skips for negative amount", func(t *testing.T) {
		err := service.AddCredits("negative@example.com", -100, "test_topup", "evt_negative", nil)
		if err != nil {
			t.Fatalf("AddCredits should not error for negative amount: %v", err)
		}
	})

	t.Run("normalizes email case", func(t *testing.T) {
		err := service.AddCredits("CaseTest@Example.COM", 200, "test_topup", "evt_case", nil)
		if err != nil {
			t.Fatalf("AddCredits failed: %v", err)
		}

		// Query with different case
		balance, err := service.GetBalance("casetest@example.com")
		if err != nil {
			t.Fatalf("GetBalance failed: %v", err)
		}
		if balance != 200 {
			t.Errorf("expected balance 200, got %d", balance)
		}
	})

	t.Run("handles metadata", func(t *testing.T) {
		metadata := map[string]interface{}{
			"source":    "test",
			"plan_tier": "pro",
		}
		err := service.AddCredits("metadata@example.com", 100, "test_topup", "evt_metadata", metadata)
		if err != nil {
			t.Fatalf("AddCredits failed: %v", err)
		}

		// Verify metadata was stored
		var storedMeta string
		err = db.QueryRow(`SELECT metadata FROM credit_transactions WHERE stripe_event_id = 'evt_metadata'`).Scan(&storedMeta)
		if err != nil {
			t.Fatalf("failed to query metadata: %v", err)
		}
		if storedMeta == "" || storedMeta == "null" {
			t.Error("expected metadata to be stored")
		}
	})
}

func TestStripeService_ConsumeCredits(t *testing.T) {
	db := setupTestDB(t)
	defer db.Close()

	resetStripeTestData(t, db)
	upsertTestBundleProduct(t, db, "business_suite", "Business Suite", "prod_consume", "production", 1000, 1, "credits")
	service := ConfigureStripeServiceSimple(t, db)
	ctx := context.Background()

	// Set up initial balance
	err := service.AddCredits("consume@example.com", 1000, "initial", "evt_initial", nil)
	if err != nil {
		t.Fatalf("failed to add initial credits: %v", err)
	}

	t.Run("consumes credits successfully", func(t *testing.T) {
		err := service.ConsumeCredits(ctx, "consume@example.com", 300, "api_call", nil)
		if err != nil {
			t.Fatalf("ConsumeCredits failed: %v", err)
		}

		balance, err := service.GetBalance("consume@example.com")
		if err != nil {
			t.Fatalf("GetBalance failed: %v", err)
		}
		if balance != 700 {
			t.Errorf("expected balance 700, got %d", balance)
		}
	})

	t.Run("fails for insufficient credits", func(t *testing.T) {
		err := service.ConsumeCredits(ctx, "consume@example.com", 1000, "api_call", nil)
		if err == nil {
			t.Fatal("expected error for insufficient credits")
		}
		if !containsHelper(err.Error(), "insufficient credits") {
			t.Errorf("expected insufficient credits error, got: %v", err)
		}
	})

	t.Run("fails for empty email", func(t *testing.T) {
		err := service.ConsumeCredits(ctx, "", 100, "api_call", nil)
		if err == nil {
			t.Fatal("expected error for empty email")
		}
	})

	t.Run("fails for zero amount", func(t *testing.T) {
		err := service.ConsumeCredits(ctx, "consume@example.com", 0, "api_call", nil)
		if err == nil {
			t.Fatal("expected error for zero amount")
		}
	})

	t.Run("fails for negative amount", func(t *testing.T) {
		err := service.ConsumeCredits(ctx, "consume@example.com", -100, "api_call", nil)
		if err == nil {
			t.Fatal("expected error for negative amount")
		}
	})

	t.Run("fails for non-existent wallet", func(t *testing.T) {
		err := service.ConsumeCredits(ctx, "nonexistent@example.com", 100, "api_call", nil)
		if err == nil {
			t.Fatal("expected error for non-existent wallet")
		}
		if !containsHelper(err.Error(), "no credit wallet found") {
			t.Errorf("expected 'no credit wallet found' error, got: %v", err)
		}
	})

	t.Run("normalizes email case", func(t *testing.T) {
		err := service.ConsumeCredits(ctx, "CONSUME@EXAMPLE.COM", 100, "api_call", nil)
		if err != nil {
			t.Fatalf("ConsumeCredits failed: %v", err)
		}

		balance, err := service.GetBalance("consume@example.com")
		if err != nil {
			t.Fatalf("GetBalance failed: %v", err)
		}
		if balance != 600 {
			t.Errorf("expected balance 600, got %d", balance)
		}
	})
}

func TestStripeService_ConsumeCreditsIdempotent(t *testing.T) {
	db := setupTestDB(t)
	defer db.Close()

	resetStripeTestData(t, db)
	upsertTestBundleProduct(t, db, "business_suite", "Business Suite", "prod_idem", "production", 1000, 1, "credits")
	service := ConfigureStripeServiceSimple(t, db)
	ctx := context.Background()

	// Set up initial balance
	err := service.AddCredits("idempotent@example.com", 1000, "initial", "evt_idem_initial", nil)
	if err != nil {
		t.Fatalf("failed to add initial credits: %v", err)
	}

	t.Run("consumes with idempotency key", func(t *testing.T) {
		err := service.ConsumeCreditsIdempotent(ctx, "idempotent@example.com", 200, "api_call", "idem_key_1", nil)
		if err != nil {
			t.Fatalf("ConsumeCreditsIdempotent failed: %v", err)
		}

		balance, err := service.GetBalance("idempotent@example.com")
		if err != nil {
			t.Fatalf("GetBalance failed: %v", err)
		}
		if balance != 800 {
			t.Errorf("expected balance 800, got %d", balance)
		}
	})

	t.Run("is idempotent with same key", func(t *testing.T) {
		err := service.ConsumeCreditsIdempotent(ctx, "idempotent@example.com", 200, "api_call", "idem_key_1", nil)
		if err != nil {
			t.Fatalf("ConsumeCreditsIdempotent failed: %v", err)
		}

		// Balance should not change
		balance, err := service.GetBalance("idempotent@example.com")
		if err != nil {
			t.Fatalf("GetBalance failed: %v", err)
		}
		if balance != 800 {
			t.Errorf("expected balance 800 (unchanged), got %d", balance)
		}
	})

	t.Run("consumes with different key", func(t *testing.T) {
		err := service.ConsumeCreditsIdempotent(ctx, "idempotent@example.com", 100, "api_call", "idem_key_2", nil)
		if err != nil {
			t.Fatalf("ConsumeCreditsIdempotent failed: %v", err)
		}

		balance, err := service.GetBalance("idempotent@example.com")
		if err != nil {
			t.Fatalf("GetBalance failed: %v", err)
		}
		if balance != 700 {
			t.Errorf("expected balance 700, got %d", balance)
		}
	})

	t.Run("works without idempotency key", func(t *testing.T) {
		err := service.ConsumeCreditsIdempotent(ctx, "idempotent@example.com", 50, "api_call", "", nil)
		if err != nil {
			t.Fatalf("ConsumeCreditsIdempotent failed: %v", err)
		}

		balance, err := service.GetBalance("idempotent@example.com")
		if err != nil {
			t.Fatalf("GetBalance failed: %v", err)
		}
		if balance != 650 {
			t.Errorf("expected balance 650, got %d", balance)
		}
	})
}

func TestStripeService_GetBalance(t *testing.T) {
	db := setupTestDB(t)
	defer db.Close()

	resetStripeTestData(t, db)
	upsertTestBundleProduct(t, db, "business_suite", "Business Suite", "prod_balance", "production", 1000, 1, "credits")
	service := ConfigureStripeServiceSimple(t, db)

	t.Run("returns 0 for non-existent wallet", func(t *testing.T) {
		balance, err := service.GetBalance("nonexistent@example.com")
		if err != nil {
			t.Fatalf("GetBalance failed: %v", err)
		}
		if balance != 0 {
			t.Errorf("expected 0, got %d", balance)
		}
	})

	t.Run("returns correct balance", func(t *testing.T) {
		err := service.AddCredits("balance@example.com", 500, "test", "evt_balance", nil)
		if err != nil {
			t.Fatalf("AddCredits failed: %v", err)
		}

		balance, err := service.GetBalance("balance@example.com")
		if err != nil {
			t.Fatalf("GetBalance failed: %v", err)
		}
		if balance != 500 {
			t.Errorf("expected 500, got %d", balance)
		}
	})

	t.Run("fails for empty email", func(t *testing.T) {
		_, err := service.GetBalance("")
		if err == nil {
			t.Fatal("expected error for empty email")
		}
	})

	t.Run("normalizes email case", func(t *testing.T) {
		balance, err := service.GetBalance("BALANCE@EXAMPLE.COM")
		if err != nil {
			t.Fatalf("GetBalance failed: %v", err)
		}
		if balance != 500 {
			t.Errorf("expected 500, got %d", balance)
		}
	})
}

func TestStripeService_CreditsConcurrency(t *testing.T) {
	db := setupTestDB(t)
	defer db.Close()

	resetStripeTestData(t, db)
	upsertTestBundleProduct(t, db, "business_suite", "Business Suite", "prod_concurrent", "production", 1000, 1, "credits")
	service := ConfigureStripeServiceSimple(t, db)
	ctx := context.Background()

	// Set up initial balance
	err := service.AddCredits("concurrent@example.com", 10000, "initial", "evt_concurrent_init", nil)
	if err != nil {
		t.Fatalf("failed to add initial credits: %v", err)
	}

	t.Run("handles concurrent AddCredits with idempotency", func(t *testing.T) {
		var wg sync.WaitGroup
		eventID := "evt_concurrent_add"

		// Try to add same credits concurrently
		for i := 0; i < 5; i++ {
			wg.Add(1)
			go func() {
				defer wg.Done()
				_ = service.AddCredits("concurrent@example.com", 100, "concurrent_add", eventID, nil)
			}()
		}
		wg.Wait()

		// Should only add 100 once due to idempotency
		balance, err := service.GetBalance("concurrent@example.com")
		if err != nil {
			t.Fatalf("GetBalance failed: %v", err)
		}
		if balance != 10100 {
			t.Errorf("expected balance 10100 (10000 + 100 once), got %d", balance)
		}
	})

	t.Run("handles concurrent ConsumeCredits with idempotency", func(t *testing.T) {
		var wg sync.WaitGroup
		idempotencyKey := "idem_concurrent_consume"

		// Try to consume same credits concurrently
		for i := 0; i < 5; i++ {
			wg.Add(1)
			go func() {
				defer wg.Done()
				_ = service.ConsumeCreditsIdempotent(ctx, "concurrent@example.com", 50, "concurrent_consume", idempotencyKey, nil)
			}()
		}
		wg.Wait()

		// Should only consume 50 once due to idempotency
		balance, err := service.GetBalance("concurrent@example.com")
		if err != nil {
			t.Fatalf("GetBalance failed: %v", err)
		}
		if balance != 10050 {
			t.Errorf("expected balance 10050 (10100 - 50 once), got %d", balance)
		}
	})
}
