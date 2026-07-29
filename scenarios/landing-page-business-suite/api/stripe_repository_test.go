package main

import (
	"context"
	"database/sql"
	"testing"
	"time"
)

func TestNewStripeRepository(t *testing.T) {
	db := setupTestDB(t)

	repo := NewStripeRepository(db)
	if repo == nil {
		t.Fatal("NewStripeRepository returned nil")
	}
	if got := repo.LookupCustomerID(""); got != "" {
		t.Errorf("LookupCustomerID(empty) = %q, want empty", got)
	}
}

func TestStripeRepository_LookupCustomerID_Extended(t *testing.T) {
	db := setupTestDB(t)

	resetStripeTestData(t, db)
	repo := NewStripeRepository(db)

	// Insert a subscription to look up
	_, err := db.Exec(`
		INSERT INTO subscriptions (subscription_id, customer_id, customer_email, status, created_at, updated_at)
		VALUES ('sub_lookup_1', 'cus_lookup_123', 'lookup@example.com', 'active', NOW(), NOW())
	`)
	if err != nil {
		t.Fatalf("failed to insert test subscription: %v", err)
	}

	tests := []struct {
		name string
		user string
		want string
	}{
		{
			name: "returns empty for empty user",
			user: "",
			want: "",
		},
		{
			name: "finds customer by email",
			user: "lookup@example.com",
			want: "cus_lookup_123",
		},
		{
			name: "finds customer by email case-insensitive",
			user: "LOOKUP@EXAMPLE.COM",
			want: "cus_lookup_123",
		},
		{
			name: "finds customer by customer ID",
			user: "cus_lookup_123",
			want: "cus_lookup_123",
		},
		{
			name: "returns empty for non-existent user",
			user: "nonexistent@example.com",
			want: "",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := repo.LookupCustomerID(tt.user)
			if got != tt.want {
				t.Errorf("LookupCustomerID(%q) = %q, want %q", tt.user, got, tt.want)
			}
		})
	}
}

func TestStripeRepository_GetSubscriptionByUser(t *testing.T) {
	db := setupTestDB(t)

	resetStripeTestData(t, db)
	repo := NewStripeRepository(db)

	// Insert test subscriptions
	now := time.Now()
	_, err := db.Exec(`
		INSERT INTO subscriptions (subscription_id, customer_id, customer_email, status, plan_tier, price_id, bundle_key, billing_cycle_start, created_at, updated_at)
		VALUES
			('sub_get_1', 'cus_get_123', 'getuser@example.com', 'active', 'pro', 'price_pro', 'business_suite', 15, $1, $1),
			('sub_get_2', 'cus_get_123', 'getuser@example.com', 'canceled', 'starter', 'price_starter', 'business_suite', 1, $2, $2)
	`, now.Add(-time.Hour), now) // sub_get_2 is more recent
	if err != nil {
		t.Fatalf("failed to insert test subscriptions: %v", err)
	}

	t.Run("returns nil for non-existent user", func(t *testing.T) {
		rec, err := repo.GetSubscriptionByUser("nonexistent@example.com")
		if err != nil {
			t.Errorf("unexpected error: %v", err)
		}
		if rec != nil {
			t.Errorf("expected nil, got %+v", rec)
		}
	})

	t.Run("returns most recent subscription by email", func(t *testing.T) {
		rec, err := repo.GetSubscriptionByUser("getuser@example.com")
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
		if rec == nil {
			t.Fatal("expected subscription, got nil")
		}
		if rec.SubscriptionID != "sub_get_2" {
			t.Errorf("expected most recent subscription sub_get_2, got %s", rec.SubscriptionID)
		}
		if rec.Status != "canceled" {
			t.Errorf("expected status canceled, got %s", rec.Status)
		}
	})

	t.Run("returns subscription by customer ID", func(t *testing.T) {
		rec, err := repo.GetSubscriptionByUser("cus_get_123")
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
		if rec == nil {
			t.Fatal("expected subscription, got nil")
		}
		if rec.CustomerID != "cus_get_123" {
			t.Errorf("expected customer ID cus_get_123, got %s", rec.CustomerID)
		}
	})
}

func TestStripeRepository_UpsertSubscription(t *testing.T) {
	db := setupTestDB(t)

	resetStripeTestData(t, db)
	repo := NewStripeRepository(db)

	t.Run("inserts new subscription", func(t *testing.T) {
		rec := &SubscriptionRecord{
			SubscriptionID:    "sub_upsert_new",
			CustomerID:        "cus_upsert_1",
			CustomerEmail:     "upsert@example.com",
			Status:            "active",
			PlanTier:          sql.NullString{String: "pro", Valid: true},
			PriceID:           sql.NullString{String: "price_pro", Valid: true},
			BundleKey:         sql.NullString{String: "business_suite", Valid: true},
			BillingCycleStart: 15,
		}

		err := repo.UpsertSubscription(rec)
		if err != nil {
			t.Fatalf("failed to insert: %v", err)
		}

		// Verify insertion
		var status string
		err = db.QueryRow(`SELECT status FROM subscriptions WHERE subscription_id = 'sub_upsert_new'`).Scan(&status)
		if err != nil {
			t.Fatalf("failed to query: %v", err)
		}
		if status != "active" {
			t.Errorf("expected status active, got %s", status)
		}
	})

	t.Run("updates existing subscription", func(t *testing.T) {
		rec := &SubscriptionRecord{
			SubscriptionID:    "sub_upsert_new",
			CustomerID:        "cus_upsert_1",
			CustomerEmail:     "upsert@example.com",
			Status:            "past_due",
			PlanTier:          sql.NullString{String: "pro", Valid: true},
			PriceID:           sql.NullString{String: "price_pro", Valid: true},
			BundleKey:         sql.NullString{String: "business_suite", Valid: true},
			BillingCycleStart: 15,
		}

		err := repo.UpsertSubscription(rec)
		if err != nil {
			t.Fatalf("failed to update: %v", err)
		}

		// Verify update
		var status string
		err = db.QueryRow(`SELECT status FROM subscriptions WHERE subscription_id = 'sub_upsert_new'`).Scan(&status)
		if err != nil {
			t.Fatalf("failed to query: %v", err)
		}
		if status != "past_due" {
			t.Errorf("expected status past_due, got %s", status)
		}
	})

	t.Run("handles canceled_at timestamp", func(t *testing.T) {
		cancelTime := time.Now()
		rec := &SubscriptionRecord{
			SubscriptionID: "sub_upsert_canceled",
			CustomerID:     "cus_upsert_2",
			CustomerEmail:  "canceled@example.com",
			Status:         "canceled",
			CanceledAt:     &cancelTime,
		}

		err := repo.UpsertSubscription(rec)
		if err != nil {
			t.Fatalf("failed to insert: %v", err)
		}

		// Verify canceled_at is set
		var canceledAt sql.NullTime
		err = db.QueryRow(`SELECT canceled_at FROM subscriptions WHERE subscription_id = 'sub_upsert_canceled'`).Scan(&canceledAt)
		if err != nil {
			t.Fatalf("failed to query: %v", err)
		}
		if !canceledAt.Valid {
			t.Error("expected canceled_at to be set")
		}
	})
}

func TestStripeRepository_UpdateSubscriptionStatus(t *testing.T) {
	db := setupTestDB(t)

	resetStripeTestData(t, db)
	repo := NewStripeRepository(db)

	// Insert a subscription to update
	_, err := db.Exec(`
		INSERT INTO subscriptions (subscription_id, customer_id, customer_email, status, created_at, updated_at)
		VALUES ('sub_status_1', 'cus_status_1', 'status@example.com', 'active', NOW(), NOW())
	`)
	if err != nil {
		t.Fatalf("failed to insert: %v", err)
	}

	t.Run("updates status without canceled_at", func(t *testing.T) {
		err := repo.UpdateSubscriptionStatus("sub_status_1", "past_due", nil)
		if err != nil {
			t.Fatalf("failed to update: %v", err)
		}

		var status string
		err = db.QueryRow(`SELECT status FROM subscriptions WHERE subscription_id = 'sub_status_1'`).Scan(&status)
		if err != nil {
			t.Fatalf("failed to query: %v", err)
		}
		if status != "past_due" {
			t.Errorf("expected past_due, got %s", status)
		}
	})

	t.Run("updates status with canceled_at", func(t *testing.T) {
		cancelTime := time.Now()
		err := repo.UpdateSubscriptionStatus("sub_status_1", "canceled", &cancelTime)
		if err != nil {
			t.Fatalf("failed to update: %v", err)
		}

		var status string
		var canceledAt sql.NullTime
		err = db.QueryRow(`SELECT status, canceled_at FROM subscriptions WHERE subscription_id = 'sub_status_1'`).Scan(&status, &canceledAt)
		if err != nil {
			t.Fatalf("failed to query: %v", err)
		}
		if status != "canceled" {
			t.Errorf("expected canceled, got %s", status)
		}
		if !canceledAt.Valid {
			t.Error("expected canceled_at to be set")
		}
	})
}

func TestStripeRepository_GetSubscriptionPlanTier(t *testing.T) {
	db := setupTestDB(t)

	resetStripeTestData(t, db)
	repo := NewStripeRepository(db)

	// Insert subscriptions with different plan tiers
	_, err := db.Exec(`
		INSERT INTO subscriptions (subscription_id, customer_email, status, plan_tier, created_at, updated_at)
		VALUES
			('sub_tier_1', 'tier1@example.com', 'active', 'pro', NOW(), NOW()),
			('sub_tier_2', 'tier2@example.com', 'active', NULL, NOW(), NOW())
	`)
	if err != nil {
		t.Fatalf("failed to insert: %v", err)
	}

	tests := []struct {
		name           string
		subscriptionID string
		want           string
	}{
		{
			name:           "returns plan tier for existing subscription",
			subscriptionID: "sub_tier_1",
			want:           "pro",
		},
		{
			name:           "returns empty for null plan tier",
			subscriptionID: "sub_tier_2",
			want:           "",
		},
		{
			name:           "returns empty for non-existent subscription",
			subscriptionID: "sub_nonexistent",
			want:           "",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := repo.GetSubscriptionPlanTier(tt.subscriptionID)
			if got != tt.want {
				t.Errorf("GetSubscriptionPlanTier(%q) = %q, want %q", tt.subscriptionID, got, tt.want)
			}
		})
	}
}

func TestStripeRepository_CheckoutSessionOperations(t *testing.T) {
	db := setupTestDB(t)

	resetStripeTestData(t, db)
	repo := NewStripeRepository(db)

	t.Run("UpsertCheckoutSession inserts new session", func(t *testing.T) {
		rec := &CheckoutSessionRecord{
			SessionID:     "cs_new_session",
			CustomerEmail: sql.NullString{String: "session@example.com", Valid: true},
			CustomerID:    sql.NullString{String: "cus_session_1", Valid: true},
			PriceID:       sql.NullString{String: "price_pro", Valid: true},
			Status:        "open",
			SessionType:   "subscription",
			AmountCents:   sql.NullInt64{Int64: 5000, Valid: true},
			Metadata:      map[string]interface{}{"plan_tier": "pro"},
		}

		err := repo.UpsertCheckoutSession(rec)
		if err != nil {
			t.Fatalf("failed to insert: %v", err)
		}

		// Verify insertion
		loaded, err := repo.LoadCheckoutSession("cs_new_session")
		if err != nil {
			t.Fatalf("failed to load: %v", err)
		}
		if loaded == nil {
			t.Fatal("expected session, got nil")
		}
		if loaded.Status != "open" {
			t.Errorf("expected status open, got %s", loaded.Status)
		}
		if loaded.SessionType != "subscription" {
			t.Errorf("expected session type subscription, got %s", loaded.SessionType)
		}
	})

	t.Run("UpsertCheckoutSession updates existing session", func(t *testing.T) {
		rec := &CheckoutSessionRecord{
			SessionID:      "cs_new_session",
			CustomerEmail:  sql.NullString{String: "session@example.com", Valid: true},
			SubscriptionID: sql.NullString{String: "sub_from_session", Valid: true},
			Status:         "complete",
			SessionType:    "subscription",
		}

		err := repo.UpsertCheckoutSession(rec)
		if err != nil {
			t.Fatalf("failed to update: %v", err)
		}

		loaded, err := repo.LoadCheckoutSession("cs_new_session")
		if err != nil {
			t.Fatalf("failed to load: %v", err)
		}
		if loaded.Status != "complete" {
			t.Errorf("expected status complete, got %s", loaded.Status)
		}
		if !loaded.SubscriptionID.Valid || loaded.SubscriptionID.String != "sub_from_session" {
			t.Errorf("expected subscription_id to be updated")
		}
	})

	t.Run("LoadCheckoutSession returns nil for non-existent session", func(t *testing.T) {
		loaded, err := repo.LoadCheckoutSession("cs_nonexistent")
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
		if loaded != nil {
			t.Errorf("expected nil, got %+v", loaded)
		}
	})

	t.Run("UpdateCheckoutSessionSchedule updates schedule_id", func(t *testing.T) {
		err := repo.UpdateCheckoutSessionSchedule("cs_new_session", "sched_123")
		if err != nil {
			t.Fatalf("failed to update schedule: %v", err)
		}

		loaded, err := repo.LoadCheckoutSession("cs_new_session")
		if err != nil {
			t.Fatalf("failed to load: %v", err)
		}
		if !loaded.ScheduleID.Valid || loaded.ScheduleID.String != "sched_123" {
			t.Errorf("expected schedule_id sched_123, got %v", loaded.ScheduleID)
		}
	})
}

func TestStripeRepository_CreditOperations(t *testing.T) {
	db := setupTestDB(t)

	resetStripeTestData(t, db)
	repo := NewStripeRepository(db)

	t.Run("AddCreditsWithIdempotency adds credits", func(t *testing.T) {
		wasProcessed, err := repo.AddCreditsWithIdempotency(
			"credits@example.com",
			1000,
			"test_topup",
			"evt_credit_1",
			map[string]interface{}{"source": "test"},
		)
		if err != nil {
			t.Fatalf("failed to add credits: %v", err)
		}
		if !wasProcessed {
			t.Error("expected wasProcessed to be true for new event")
		}

		// Verify balance
		balance, err := repo.GetCreditBalance("credits@example.com")
		if err != nil {
			t.Fatalf("failed to get balance: %v", err)
		}
		if balance != 1000 {
			t.Errorf("expected balance 1000, got %d", balance)
		}
	})

	t.Run("AddCreditsWithIdempotency is idempotent", func(t *testing.T) {
		// Try to add the same event again
		wasProcessed, err := repo.AddCreditsWithIdempotency(
			"credits@example.com",
			1000,
			"test_topup",
			"evt_credit_1", // Same event ID
			map[string]interface{}{"source": "test"},
		)
		if err != nil {
			t.Fatalf("failed: %v", err)
		}
		if wasProcessed {
			t.Error("expected wasProcessed to be false for duplicate event")
		}

		// Balance should still be 1000
		balance, err := repo.GetCreditBalance("credits@example.com")
		if err != nil {
			t.Fatalf("failed to get balance: %v", err)
		}
		if balance != 1000 {
			t.Errorf("expected balance 1000 (unchanged), got %d", balance)
		}
	})

	t.Run("AddCreditsWithIdempotency accumulates with different events", func(t *testing.T) {
		wasProcessed, err := repo.AddCreditsWithIdempotency(
			"credits@example.com",
			500,
			"test_topup",
			"evt_credit_2", // Different event ID
			nil,
		)
		if err != nil {
			t.Fatalf("failed: %v", err)
		}
		if !wasProcessed {
			t.Error("expected wasProcessed to be true for new event")
		}

		balance, err := repo.GetCreditBalance("credits@example.com")
		if err != nil {
			t.Fatalf("failed to get balance: %v", err)
		}
		if balance != 1500 {
			t.Errorf("expected balance 1500, got %d", balance)
		}
	})

	t.Run("GetCreditBalance returns 0 for non-existent user", func(t *testing.T) {
		balance, err := repo.GetCreditBalance("nonexistent@example.com")
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
		if balance != 0 {
			t.Errorf("expected 0, got %d", balance)
		}
	})

	t.Run("GetCreditBalance normalizes email", func(t *testing.T) {
		balance, err := repo.GetCreditBalance("CREDITS@EXAMPLE.COM")
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
		if balance != 1500 {
			t.Errorf("expected 1500 (case-insensitive lookup), got %d", balance)
		}
	})
}

func TestStripeRepository_IntroCouponOperations(t *testing.T) {
	db := setupTestDB(t)

	resetStripeTestData(t, db)
	repo := NewStripeRepository(db)
	ctx := context.Background()

	t.Run("CheckIntroEligibility returns true for new user", func(t *testing.T) {
		eligible, err := repo.CheckIntroEligibility(ctx, "newuser@example.com")
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
		if !eligible {
			t.Error("expected new user to be eligible")
		}
	})

	t.Run("MarkIntroUsed marks user as having used intro", func(t *testing.T) {
		err := repo.MarkIntroUsed(ctx, "introuser@example.com", "cus_intro", "coupon_intro", "pro", "sub_intro")
		if err != nil {
			t.Fatalf("failed to mark intro used: %v", err)
		}

		// Verify user is no longer eligible
		eligible, err := repo.CheckIntroEligibility(ctx, "introuser@example.com")
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
		if eligible {
			t.Error("expected user to no longer be eligible after using intro")
		}
	})

	t.Run("CheckIntroEligibility is case-insensitive", func(t *testing.T) {
		eligible, err := repo.CheckIntroEligibility(ctx, "INTROUSER@EXAMPLE.COM")
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
		if eligible {
			t.Error("expected case-insensitive check to find used intro")
		}
	})
}

func TestStripeRepository_UserOperations(t *testing.T) {
	db := setupTestDB(t)

	resetStripeTestData(t, db)
	repo := NewStripeRepository(db)

	t.Run("LinkUserToStripeCustomer creates user record", func(t *testing.T) {
		err := repo.LinkUserToStripeCustomer("linkuser@example.com", "cus_link_123")
		if err != nil {
			t.Fatalf("failed to link: %v", err)
		}

		// Verify user was created
		var customerID string
		err = db.QueryRow(`SELECT stripe_customer_id FROM users WHERE email = 'linkuser@example.com'`).Scan(&customerID)
		if err != nil {
			t.Fatalf("failed to query: %v", err)
		}
		if customerID != "cus_link_123" {
			t.Errorf("expected cus_link_123, got %s", customerID)
		}
	})

	t.Run("LinkUserToStripeCustomer updates existing user", func(t *testing.T) {
		err := repo.LinkUserToStripeCustomer("linkuser@example.com", "cus_link_456")
		if err != nil {
			t.Fatalf("failed to update: %v", err)
		}

		var customerID string
		err = db.QueryRow(`SELECT stripe_customer_id FROM users WHERE email = 'linkuser@example.com'`).Scan(&customerID)
		if err != nil {
			t.Fatalf("failed to query: %v", err)
		}
		if customerID != "cus_link_456" {
			t.Errorf("expected cus_link_456, got %s", customerID)
		}
	})

	t.Run("GetOldEmailForCustomer finds email by customer ID", func(t *testing.T) {
		email, err := repo.GetOldEmailForCustomer("cus_link_456")
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
		if email != "linkuser@example.com" {
			t.Errorf("expected linkuser@example.com, got %s", email)
		}
	})

	t.Run("GetOldEmailForCustomer returns empty for non-existent customer", func(t *testing.T) {
		email, err := repo.GetOldEmailForCustomer("cus_nonexistent")
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
		if email != "" {
			t.Errorf("expected empty string, got %s", email)
		}
	})
}

func TestStripeRepository_MigrateCustomerEmail_Extended(t *testing.T) {
	db := setupTestDB(t)

	resetStripeTestData(t, db)
	repo := NewStripeRepository(db)
	ctx := context.Background()

	// Set up test data
	_, err := db.Exec(`
		INSERT INTO users (email, stripe_customer_id, created_at, updated_at) VALUES ('old@example.com', 'cus_migrate', NOW(), NOW());
		INSERT INTO subscriptions (subscription_id, customer_id, customer_email, status, created_at, updated_at) VALUES ('sub_migrate', 'cus_migrate', 'old@example.com', 'active', NOW(), NOW());
		INSERT INTO credit_wallets (customer_email, balance_credits, updated_at) VALUES ('old@example.com', 500, NOW());
	`)
	if err != nil {
		t.Fatalf("failed to set up test data: %v", err)
	}

	t.Run("migrates email across all tables", func(t *testing.T) {
		err := repo.MigrateCustomerEmail(ctx, "old@example.com", "new@example.com", "cus_migrate")
		if err != nil {
			t.Fatalf("failed to migrate: %v", err)
		}

		// Verify users table
		var count int
		err = db.QueryRow(`SELECT COUNT(*) FROM users WHERE email = 'new@example.com'`).Scan(&count)
		if err != nil || count != 1 {
			t.Errorf("expected user email to be migrated")
		}

		// Verify subscriptions table
		err = db.QueryRow(`SELECT COUNT(*) FROM subscriptions WHERE customer_email = 'new@example.com'`).Scan(&count)
		if err != nil || count != 1 {
			t.Errorf("expected subscription email to be migrated")
		}

		// Verify credit_wallets table
		err = db.QueryRow(`SELECT COUNT(*) FROM credit_wallets WHERE customer_email = 'new@example.com'`).Scan(&count)
		if err != nil || count != 1 {
			t.Errorf("expected credit wallet email to be migrated")
		}

		// Verify old email no longer exists
		err = db.QueryRow(`SELECT COUNT(*) FROM users WHERE email = 'old@example.com'`).Scan(&count)
		if err != nil || count != 0 {
			t.Errorf("expected old email to be removed from users")
		}
	})
}

func TestStripeRepository_UpsertSubscriptionFromInvoice(t *testing.T) {
	db := setupTestDB(t)

	resetStripeTestData(t, db)
	repo := NewStripeRepository(db)

	t.Run("inserts new subscription from invoice", func(t *testing.T) {
		err := repo.UpsertSubscriptionFromInvoice(
			"sub_invoice_1",
			"cus_invoice_1",
			"invoice@example.com",
			"price_pro",
			"active",
			"pro",
			"business_suite",
		)
		if err != nil {
			t.Fatalf("failed to upsert: %v", err)
		}

		var status, planTier string
		err = db.QueryRow(`SELECT status, plan_tier FROM subscriptions WHERE subscription_id = 'sub_invoice_1'`).Scan(&status, &planTier)
		if err != nil {
			t.Fatalf("failed to query: %v", err)
		}
		if status != "active" || planTier != "pro" {
			t.Errorf("unexpected values: status=%s, planTier=%s", status, planTier)
		}
	})

	t.Run("updates existing subscription from invoice", func(t *testing.T) {
		err := repo.UpsertSubscriptionFromInvoice(
			"sub_invoice_1",
			"cus_invoice_1",
			"invoice@example.com",
			"price_enterprise",
			"past_due",
			"enterprise",
			"business_suite",
		)
		if err != nil {
			t.Fatalf("failed to upsert: %v", err)
		}

		var status, planTier string
		err = db.QueryRow(`SELECT status, plan_tier FROM subscriptions WHERE subscription_id = 'sub_invoice_1'`).Scan(&status, &planTier)
		if err != nil {
			t.Fatalf("failed to query: %v", err)
		}
		if status != "past_due" || planTier != "enterprise" {
			t.Errorf("unexpected values after update: status=%s, planTier=%s", status, planTier)
		}
	})
}
