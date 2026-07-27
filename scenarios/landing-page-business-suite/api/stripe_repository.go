package main

import (
	"context"
	"database/sql"
	"encoding/json"
	"errors"
	"time"
)

// StripeRepository handles all database operations for Stripe-related data.
// It provides a clean separation between business logic and data access.
type StripeRepository struct {
	db StripeStore
}

// StripeStore is the transaction-capable persistence boundary for Stripe data.
// RoutedDB implements this interface so request paths can use test-specific pools.
type StripeStore interface {
	QueryRow(string, ...any) *sql.Row
	QueryRowContext(context.Context, string, ...any) *sql.Row
	Exec(string, ...any) (sql.Result, error)
	Begin() (*sql.Tx, error)
	BeginTx(context.Context, *sql.TxOptions) (*sql.Tx, error)
}

// NewStripeRepository creates a new repository with the given database connection.
func NewStripeRepository(db StripeStore) *StripeRepository {
	return &StripeRepository{db: db}
}

// SubscriptionRecord represents a subscription stored in the database.
type SubscriptionRecord struct {
	SubscriptionID    string
	CustomerID        string
	CustomerEmail     string
	Status            string
	PlanTier          sql.NullString
	PriceID           sql.NullString
	BundleKey         sql.NullString
	BillingCycleStart int
	CanceledAt        *time.Time
	CreatedAt         time.Time
	UpdatedAt         time.Time
}

// CheckoutSessionRecord represents a checkout session stored in the database.
type CheckoutSessionRecord struct {
	SessionID      string
	CustomerEmail  sql.NullString
	CustomerID     sql.NullString
	PriceID        sql.NullString
	SubscriptionID sql.NullString
	Status         string
	SessionType    string
	AmountCents    sql.NullInt64
	ScheduleID     sql.NullString
	Metadata       map[string]interface{}
	CreatedAt      time.Time
	UpdatedAt      time.Time
}

// CreditWalletRecord represents a credit wallet in the database.
type CreditWalletRecord struct {
	ID             int64
	CustomerEmail  string
	BalanceCredits int64
	BonusCredits   int64
	UpdatedAt      time.Time
}

// CreditTransactionRecord represents a credit transaction in the database.
type CreditTransactionRecord struct {
	ID              int64
	CustomerEmail   string
	AmountCredits   int64
	TransactionType string
	Source          sql.NullString
	StripeEventID   sql.NullString
	Metadata        map[string]interface{}
	CreatedAt       time.Time
}

// --- Subscription Operations ---

// LookupCustomerID finds the customer ID for a user (by email or customer ID).
func (r *StripeRepository) LookupCustomerID(user string) string {
	if user == "" {
		return ""
	}
	normalizedUser := NormalizeEmail(user)
	var customerID sql.NullString
	err := r.db.QueryRow(`
		SELECT customer_id
		FROM subscriptions
		WHERE customer_email = $1 OR customer_id = $1
		ORDER BY updated_at DESC
		LIMIT 1
	`, normalizedUser).Scan(&customerID)
	if err != nil || !customerID.Valid {
		return ""
	}
	return customerID.String
}

// GetSubscriptionByUser finds the most recent subscription for a user.
func (r *StripeRepository) GetSubscriptionByUser(userIdentity string) (*SubscriptionRecord, error) {
	normalizedUser := NormalizeEmail(userIdentity)
	var rec SubscriptionRecord
	var canceledAt sql.NullTime

	err := r.db.QueryRow(`
		SELECT subscription_id, customer_id, customer_email, status, plan_tier, price_id, bundle_key,
			   COALESCE(billing_cycle_start, 0), canceled_at, created_at, updated_at
		FROM subscriptions
		WHERE customer_email = $1 OR customer_id = $1
		ORDER BY updated_at DESC
		LIMIT 1
	`, normalizedUser).Scan(
		&rec.SubscriptionID, &rec.CustomerID, &rec.CustomerEmail, &rec.Status,
		&rec.PlanTier, &rec.PriceID, &rec.BundleKey, &rec.BillingCycleStart,
		&canceledAt, &rec.CreatedAt, &rec.UpdatedAt,
	)
	if err == sql.ErrNoRows {
		return nil, nil
	}
	if err != nil {
		return nil, err
	}
	if canceledAt.Valid {
		rec.CanceledAt = &canceledAt.Time
	}
	return &rec, nil
}

// UpsertSubscription inserts or updates a subscription record.
func (r *StripeRepository) UpsertSubscription(rec *SubscriptionRecord) error {
	_, err := r.db.Exec(`
		INSERT INTO subscriptions (subscription_id, customer_id, customer_email, status, plan_tier, price_id, bundle_key, billing_cycle_start, canceled_at, created_at, updated_at)
		VALUES ($1::varchar,$2::varchar,$3::varchar,$4::varchar,$5::varchar,$6::varchar,$7::varchar,$8::int,$9::timestamp,COALESCE((SELECT created_at FROM subscriptions WHERE subscription_id = $1::varchar), NOW()), NOW())
		ON CONFLICT (subscription_id) DO UPDATE SET
			customer_id = EXCLUDED.customer_id,
			customer_email = EXCLUDED.customer_email,
			status = EXCLUDED.status,
			plan_tier = COALESCE(NULLIF(EXCLUDED.plan_tier,''), subscriptions.plan_tier),
			price_id = COALESCE(NULLIF(EXCLUDED.price_id,''), subscriptions.price_id),
			bundle_key = COALESCE(NULLIF(EXCLUDED.bundle_key,''), subscriptions.bundle_key),
			billing_cycle_start = EXCLUDED.billing_cycle_start,
			canceled_at = EXCLUDED.canceled_at,
			updated_at = NOW()
	`, rec.SubscriptionID, rec.CustomerID, rec.CustomerEmail, rec.Status,
		nullString(rec.PlanTier), nullString(rec.PriceID), nullString(rec.BundleKey),
		rec.BillingCycleStart, rec.CanceledAt)
	return err
}

// UpdateSubscriptionStatus updates just the status of a subscription.
func (r *StripeRepository) UpdateSubscriptionStatus(subscriptionID, status string, canceledAt *time.Time) error {
	_, err := r.db.Exec(`
		UPDATE subscriptions
		SET status = $1, canceled_at = $2, updated_at = NOW()
		WHERE subscription_id = $3
	`, status, canceledAt, subscriptionID)
	return err
}

// GetSubscriptionPlanTier returns the plan tier for a subscription.
func (r *StripeRepository) GetSubscriptionPlanTier(subscriptionID string) string {
	var planTier sql.NullString
	err := r.db.QueryRow(`SELECT plan_tier FROM subscriptions WHERE subscription_id = $1`, subscriptionID).Scan(&planTier)
	if err != nil || !planTier.Valid {
		return ""
	}
	return planTier.String
}

// --- Checkout Session Operations ---

// UpsertCheckoutSession inserts or updates a checkout session.
func (r *StripeRepository) UpsertCheckoutSession(rec *CheckoutSessionRecord) error {
	metadataJSON, _ := json.Marshal(rec.Metadata)
	_, err := r.db.Exec(`
		INSERT INTO checkout_sessions (session_id, customer_email, customer_id, price_id, subscription_id, status, session_type, amount_cents, schedule_id, metadata, created_at, updated_at)
		VALUES ($1,$2,$3,$4,$5,$6,$7,$8,$9,$10::jsonb, NOW(), NOW())
		ON CONFLICT (session_id) DO UPDATE SET
			customer_email = COALESCE(EXCLUDED.customer_email, checkout_sessions.customer_email),
			customer_id = COALESCE(EXCLUDED.customer_id, checkout_sessions.customer_id),
			price_id = COALESCE(EXCLUDED.price_id, checkout_sessions.price_id),
			subscription_id = COALESCE(EXCLUDED.subscription_id, checkout_sessions.subscription_id),
			status = EXCLUDED.status,
			session_type = COALESCE(EXCLUDED.session_type, checkout_sessions.session_type),
			amount_cents = COALESCE(EXCLUDED.amount_cents, checkout_sessions.amount_cents),
			schedule_id = COALESCE(EXCLUDED.schedule_id, checkout_sessions.schedule_id),
			metadata = COALESCE(EXCLUDED.metadata, checkout_sessions.metadata),
			updated_at = NOW()
	`, rec.SessionID, rec.CustomerEmail, rec.CustomerID, rec.PriceID, rec.SubscriptionID,
		rec.Status, rec.SessionType, rec.AmountCents, rec.ScheduleID, string(metadataJSON))
	return err
}

// LoadCheckoutSession retrieves a checkout session by ID.
func (r *StripeRepository) LoadCheckoutSession(sessionID string) (*CheckoutSessionRecord, error) {
	var rec CheckoutSessionRecord
	var metadataBytes []byte
	err := r.db.QueryRow(`
		SELECT session_id, customer_email, customer_id, price_id, subscription_id, status,
			   COALESCE(session_type, 'subscription'), COALESCE(amount_cents, 0), schedule_id, metadata
		FROM checkout_sessions
		WHERE session_id = $1
	`, sessionID).Scan(
		&rec.SessionID, &rec.CustomerEmail, &rec.CustomerID, &rec.PriceID,
		&rec.SubscriptionID, &rec.Status, &rec.SessionType, &rec.AmountCents,
		&rec.ScheduleID, &metadataBytes,
	)
	if err == sql.ErrNoRows {
		return nil, nil
	}
	if err != nil {
		return nil, err
	}
	if len(metadataBytes) > 0 {
		_ = json.Unmarshal(metadataBytes, &rec.Metadata) // Best-effort decode, empty map is fine
	}
	return &rec, nil
}

// UpdateCheckoutSessionSchedule updates the schedule_id for a session.
func (r *StripeRepository) UpdateCheckoutSessionSchedule(sessionID, scheduleID string) error {
	_, err := r.db.Exec(`
		UPDATE checkout_sessions SET schedule_id = $1, updated_at = NOW() WHERE session_id = $2
	`, scheduleID, sessionID)
	return err
}

// --- Credit Operations ---

// AddCreditsWithIdempotency adds credits to a wallet with idempotency protection.
// Returns (wasProcessed, error). If wasProcessed is false, the event was already processed.
func (r *StripeRepository) AddCreditsWithIdempotency(customerEmail string, amount int64, txnType, stripeEventID string, metadata map[string]interface{}) (bool, error) {
	tx, err := r.db.Begin()
	if err != nil {
		return false, err
	}
	defer func() { _ = tx.Rollback() }()

	metadataJSON, _ := json.Marshal(metadata)

	// Insert transaction first - unique index on stripe_event_id fails duplicates
	result, err := tx.Exec(`
		INSERT INTO credit_transactions (customer_email, amount_credits, transaction_type, stripe_event_id, metadata, created_at)
		VALUES ($1, $2, $3, $4, $5::jsonb, NOW())
		ON CONFLICT (stripe_event_id) DO NOTHING
	`, NormalizeEmail(customerEmail), amount, txnType, stripeEventID, string(metadataJSON))
	if err != nil {
		return false, err
	}

	rowsAffected, err := result.RowsAffected()
	if err != nil {
		return false, err
	}

	// If no rows affected, event was already processed
	if rowsAffected == 0 {
		return false, nil
	}

	// Update wallet balance
	_, err = tx.Exec(`
		INSERT INTO credit_wallets (customer_email, balance_credits, updated_at)
		VALUES ($1, $2, NOW())
		ON CONFLICT (customer_email) DO UPDATE SET
			balance_credits = credit_wallets.balance_credits + EXCLUDED.balance_credits,
			updated_at = NOW()
	`, NormalizeEmail(customerEmail), amount)
	if err != nil {
		return false, err
	}

	if err := tx.Commit(); err != nil {
		return false, err
	}
	return true, nil
}

// GetCreditBalance returns the credit balance for a customer.
func (r *StripeRepository) GetCreditBalance(customerEmail string) (int64, error) {
	var balance int64
	err := r.db.QueryRow(`
		SELECT COALESCE(balance_credits, 0) FROM credit_wallets WHERE customer_email = $1
	`, NormalizeEmail(customerEmail)).Scan(&balance)
	if err == sql.ErrNoRows {
		return 0, nil
	}
	return balance, err
}

// --- Intro Coupon Operations ---

// CheckIntroEligibility checks if a user is eligible for intro pricing.
func (r *StripeRepository) CheckIntroEligibility(ctx context.Context, email string) (bool, error) {
	normalizedEmail := NormalizeEmail(email)
	var hasUsed bool
	err := r.db.QueryRowContext(ctx, `
		SELECT COALESCE(has_used_intro, FALSE)
		FROM users
		WHERE LOWER(email) = $1
	`, normalizedEmail).Scan(&hasUsed)
	if err == sql.ErrNoRows {
		return true, nil // New user is eligible
	}
	if err != nil {
		return false, err
	}
	return !hasUsed, nil
}

// MarkIntroUsed records that a user has used their intro coupon.
func (r *StripeRepository) MarkIntroUsed(ctx context.Context, email, customerID, couponID, planTier, subscriptionID string) error {
	normalizedEmail := NormalizeEmail(email)

	tx, err := r.db.BeginTx(ctx, nil)
	if err != nil {
		return err
	}
	defer func() { _ = tx.Rollback() }()

	// Upsert user record
	_, err = tx.ExecContext(ctx, `
		INSERT INTO users (email, has_used_intro, stripe_customer_id, created_at, updated_at)
		VALUES ($1, TRUE, $2, NOW(), NOW())
		ON CONFLICT (email) DO UPDATE SET
			has_used_intro = TRUE,
			stripe_customer_id = COALESCE(EXCLUDED.stripe_customer_id, users.stripe_customer_id),
			updated_at = NOW()
	`, normalizedEmail, customerID)
	if err != nil {
		return err
	}

	// Record usage in tracking table
	_, err = tx.ExecContext(ctx, `
		INSERT INTO intro_coupon_usage (email, stripe_customer_id, coupon_id, plan_tier, subscription_id, used_at)
		VALUES ($1, $2, $3, $4, $5, NOW())
	`, normalizedEmail, customerID, couponID, planTier, subscriptionID)
	if err != nil {
		return err
	}

	return tx.Commit()
}

// --- User Operations ---

// LinkUserToStripeCustomer links a user email to a Stripe customer ID.
func (r *StripeRepository) LinkUserToStripeCustomer(email, customerID string) error {
	normalizedEmail := NormalizeEmail(email)
	_, err := r.db.Exec(`
		INSERT INTO users (email, stripe_customer_id, updated_at)
		VALUES ($1, $2, NOW())
		ON CONFLICT (email) DO UPDATE SET
			stripe_customer_id = EXCLUDED.stripe_customer_id,
			updated_at = NOW()
	`, normalizedEmail, customerID)
	return err
}

// GetOldEmailForCustomer finds the previous email for a customer ID.
func (r *StripeRepository) GetOldEmailForCustomer(customerID string) (string, error) {
	var oldEmail string
	err := r.db.QueryRow(`
		SELECT email FROM users WHERE stripe_customer_id = $1 LIMIT 1
	`, customerID).Scan(&oldEmail)
	if err == sql.ErrNoRows {
		return "", nil
	}
	return oldEmail, err
}

// MigrateCustomerEmail updates email across all tables atomically.
func (r *StripeRepository) MigrateCustomerEmail(ctx context.Context, oldEmail, newEmail, customerID string) error {
	normalizedOld := NormalizeEmail(oldEmail)
	normalizedNew := NormalizeEmail(newEmail)

	tx, err := r.db.BeginTx(ctx, nil)
	if err != nil {
		return err
	}
	defer func() { _ = tx.Rollback() }()

	// Update users table
	_, err = tx.ExecContext(ctx, `
		UPDATE users SET email = $1, updated_at = NOW()
		WHERE email = $2 OR stripe_customer_id = $3
	`, normalizedNew, normalizedOld, customerID)
	if err != nil {
		return err
	}

	// Update subscriptions table
	_, err = tx.ExecContext(ctx, `
		UPDATE subscriptions SET customer_email = $1, updated_at = NOW()
		WHERE customer_email = $2 OR customer_id = $3
	`, normalizedNew, normalizedOld, customerID)
	if err != nil {
		return err
	}

	// Update credit_wallets table
	_, err = tx.ExecContext(ctx, `
		UPDATE credit_wallets SET customer_email = $1, updated_at = NOW()
		WHERE customer_email = $2
	`, normalizedNew, normalizedOld)
	if err != nil {
		return err
	}

	// Update credit_transactions table
	_, err = tx.ExecContext(ctx, `
		UPDATE credit_transactions SET customer_email = $1
		WHERE customer_email = $2
	`, normalizedNew, normalizedOld)
	if err != nil {
		return err
	}

	// Update intro_coupon_usage table
	_, err = tx.ExecContext(ctx, `
		UPDATE intro_coupon_usage SET email = $1
		WHERE email = $2 OR stripe_customer_id = $3
	`, normalizedNew, normalizedOld, customerID)
	if err != nil {
		return err
	}

	return tx.Commit()
}

// --- Invoice Status Operations ---

// UpsertSubscriptionFromInvoice updates subscription status from invoice data.
func (r *StripeRepository) UpsertSubscriptionFromInvoice(subscriptionID, customerID, customerEmail, priceID, status, planTier, bundleKey string) error {
	_, err := r.db.Exec(`
		INSERT INTO subscriptions (subscription_id, customer_id, customer_email, status, plan_tier, price_id, bundle_key, created_at, updated_at)
		VALUES ($1, $2, $3, $4, $5, $6, $7, NOW(), NOW())
		ON CONFLICT (subscription_id) DO UPDATE SET
			customer_id = COALESCE(EXCLUDED.customer_id, subscriptions.customer_id),
			customer_email = COALESCE(EXCLUDED.customer_email, subscriptions.customer_email),
			status = EXCLUDED.status,
			plan_tier = COALESCE(NULLIF(EXCLUDED.plan_tier, ''), subscriptions.plan_tier),
			price_id = COALESCE(NULLIF(EXCLUDED.price_id, ''), subscriptions.price_id),
			bundle_key = COALESCE(NULLIF(EXCLUDED.bundle_key, ''), subscriptions.bundle_key),
			updated_at = NOW()
	`, subscriptionID, customerID, NormalizeEmail(customerEmail), status, planTier, priceID, bundleKey)
	return err
}

// --- Helper Functions ---

// nullString returns the string value from sql.NullString or empty string if not valid.
func nullString(ns sql.NullString) interface{} {
	if ns.Valid {
		return ns.String
	}
	return nil
}

// ErrAlreadyProcessed indicates an idempotent operation was already completed.
var ErrAlreadyProcessed = errors.New("already processed")
