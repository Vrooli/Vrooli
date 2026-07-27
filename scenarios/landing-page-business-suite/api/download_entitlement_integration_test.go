package main

import (
	"database/sql"
	"errors"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// ============================================================================
// Download Entitlement Integration Tests
// Full 8-Status Subscription Matrix × Download Authorization
// ============================================================================

// setupDownloadEntitlementDB creates the minimal tables needed for
// subscription + download authorization integration tests.
func setupDownloadEntitlementDB(t *testing.T) (*sql.DB, *DownloadAuthorizer) {
	t.Helper()

	db := setupTestDB(t)
	t.Cleanup(func() { db.Close() })
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
			bonus_credits BIGINT DEFAULT 0,
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

	productID := upsertTestBundleProduct(t, db, "business_suite", "Business Suite", "prod_dl_test", "production", 1000000, 0.001, "credits")
	insertBundlePrice(t, db, productID, "price_pro_dl", "Pro Plan", "pro", "month", "usd", 2900, false, "", 0, 0, "", 1000000, 0, 1, 10, "none", sessionTypeSubscription, map[string]interface{}{})

	// Seed gated asset
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
	// Disable subscription cache so status changes in lifecycle tests are immediately visible
	accountSvc.cacheTTL = 0
	downloadSvc := &DownloadService{db: db}
	authorizer := NewDownloadAuthorizer(downloadSvc, accountSvc, "business_suite")

	return db, authorizer
}

// seedSubscription inserts a subscription row with the given status for testing.
func seedSubscription(t *testing.T, db *sql.DB, email, status, subID string) {
	t.Helper()
	_, err := db.Exec(`
		INSERT INTO users (email, stripe_customer_id) VALUES ($1, $2)
		ON CONFLICT (email) DO NOTHING
	`, email, "cus_"+subID)
	require.NoError(t, err)
	_, err = db.Exec(`
		INSERT INTO subscriptions (subscription_id, customer_id, customer_email, status, plan_tier, price_id, bundle_key)
		VALUES ($1, $2, $3, $4, $5, $6, $7)
		ON CONFLICT (subscription_id) DO UPDATE SET status = $4
	`, subID, "cus_"+subID, email, status, "pro", "price_pro_dl", "business_suite")
	require.NoError(t, err)
}

// ============================================================================
// Full 8-Status Matrix
// ============================================================================

func TestIntegration_Status_Active_Download_Allowed(t *testing.T) {
	db, authorizer := setupDownloadEntitlementDB(t)
	seedSubscription(t, db, "active@example.com", "active", "sub_active")

	asset, err := authorizer.Authorize(testRequestContext, "test-app", "windows", "active@example.com")
	require.NoError(t, err)
	assert.Equal(t, "windows", asset.Platform)
}

func TestIntegration_Status_PastDue_Download_Denied(t *testing.T) {
	db, authorizer := setupDownloadEntitlementDB(t)
	seedSubscription(t, db, "pastdue@example.com", "past_due", "sub_pastdue")

	_, err := authorizer.Authorize(testRequestContext, "test-app", "windows", "pastdue@example.com")
	assert.True(t, errors.Is(err, ErrDownloadRequiresActiveSubscription),
		"past_due subscription should deny downloads, got: %v", err)
}

func TestIntegration_Status_Canceled_Download_Denied(t *testing.T) {
	db, authorizer := setupDownloadEntitlementDB(t)
	seedSubscription(t, db, "canceled@example.com", "canceled", "sub_canceled")

	_, err := authorizer.Authorize(testRequestContext, "test-app", "windows", "canceled@example.com")
	assert.True(t, errors.Is(err, ErrDownloadRequiresActiveSubscription),
		"canceled subscription should deny downloads, got: %v", err)
}

func TestIntegration_Status_Unpaid_Download_Denied(t *testing.T) {
	db, authorizer := setupDownloadEntitlementDB(t)
	seedSubscription(t, db, "unpaid@example.com", "unpaid", "sub_unpaid")

	_, err := authorizer.Authorize(testRequestContext, "test-app", "windows", "unpaid@example.com")
	assert.True(t, errors.Is(err, ErrDownloadRequiresActiveSubscription),
		"unpaid subscription should deny downloads, got: %v", err)
}

func TestIntegration_Status_Incomplete_Download_Denied(t *testing.T) {
	db, authorizer := setupDownloadEntitlementDB(t)
	seedSubscription(t, db, "incomplete@example.com", "incomplete", "sub_incomplete")

	_, err := authorizer.Authorize(testRequestContext, "test-app", "windows", "incomplete@example.com")
	assert.True(t, errors.Is(err, ErrDownloadRequiresActiveSubscription),
		"incomplete subscription should deny downloads, got: %v", err)
}

func TestIntegration_Status_IncompleteExpired_Download_Denied(t *testing.T) {
	db, authorizer := setupDownloadEntitlementDB(t)
	seedSubscription(t, db, "incexp@example.com", "incomplete_expired", "sub_incexp")

	_, err := authorizer.Authorize(testRequestContext, "test-app", "windows", "incexp@example.com")
	assert.True(t, errors.Is(err, ErrDownloadRequiresActiveSubscription),
		"incomplete_expired subscription should deny downloads, got: %v", err)
}

func TestIntegration_Status_Trialing_Download_Allowed(t *testing.T) {
	db, authorizer := setupDownloadEntitlementDB(t)
	seedSubscription(t, db, "trialing@example.com", "trialing", "sub_trialing")

	asset, err := authorizer.Authorize(testRequestContext, "test-app", "windows", "trialing@example.com")
	require.NoError(t, err)
	assert.Equal(t, "windows", asset.Platform)
}

func TestIntegration_Status_Paused_Download_Denied(t *testing.T) {
	db, authorizer := setupDownloadEntitlementDB(t)
	seedSubscription(t, db, "paused@example.com", "paused", "sub_paused")

	_, err := authorizer.Authorize(testRequestContext, "test-app", "windows", "paused@example.com")
	assert.True(t, errors.Is(err, ErrDownloadRequiresActiveSubscription),
		"paused subscription should deny downloads, got: %v", err)
}

// ============================================================================
// Subscription Lifecycle Transition Tests
// ============================================================================

// TestIntegration_Subscription_Active_To_Canceled_Download_Denied verifies that
// canceling an active subscription blocks download access.
func TestIntegration_Subscription_Active_To_Canceled_Download_Denied(t *testing.T) {
	db, authorizer := setupDownloadEntitlementDB(t)
	seedSubscription(t, db, "lifecycle@example.com", "active", "sub_lifecycle_cancel")

	// Verify access while active
	asset, err := authorizer.Authorize(testRequestContext, "test-app", "windows", "lifecycle@example.com")
	require.NoError(t, err)
	assert.Equal(t, "windows", asset.Platform)

	// Cancel subscription
	_, err = db.Exec(`UPDATE subscriptions SET status = 'canceled', canceled_at = NOW() WHERE subscription_id = $1`, "sub_lifecycle_cancel")
	require.NoError(t, err)

	// Download should now be denied
	_, err = authorizer.Authorize(testRequestContext, "test-app", "windows", "lifecycle@example.com")
	assert.True(t, errors.Is(err, ErrDownloadRequiresActiveSubscription),
		"canceled subscription should deny downloads, got: %v", err)
}

// TestIntegration_Subscription_Canceled_To_Resubscribed_Download_Restored verifies
// that re-subscribing after cancellation restores download access.
func TestIntegration_Subscription_Canceled_To_Resubscribed_Download_Restored(t *testing.T) {
	db, authorizer := setupDownloadEntitlementDB(t)
	seedSubscription(t, db, "resub@example.com", "canceled", "sub_resub_old")

	// Verify access denied while canceled
	_, err := authorizer.Authorize(testRequestContext, "test-app", "windows", "resub@example.com")
	assert.True(t, errors.Is(err, ErrDownloadRequiresActiveSubscription))

	// Re-subscribe (new subscription ID, same email)
	_, err = db.Exec(`
		INSERT INTO subscriptions (subscription_id, customer_id, customer_email, status, plan_tier, price_id, bundle_key, created_at, updated_at)
		VALUES ($1, $2, $3, $4, $5, $6, $7, NOW(), NOW())
	`, "sub_resub_new", "cus_sub_resub_old", "resub@example.com", "active", "pro", "price_pro_dl", "business_suite")
	require.NoError(t, err)

	// Download should be restored
	asset, err := authorizer.Authorize(testRequestContext, "test-app", "windows", "resub@example.com")
	require.NoError(t, err)
	assert.Equal(t, "windows", asset.Platform)
}

// TestIntegration_Subscription_Active_To_PastDue_To_Active verifies that past_due
// degrades access and recovery restores it.
func TestIntegration_Subscription_Active_To_PastDue_To_Active(t *testing.T) {
	db, authorizer := setupDownloadEntitlementDB(t)
	seedSubscription(t, db, "pastdue-recovery@example.com", "active", "sub_pastdue_recovery")

	// Active: access granted
	_, err := authorizer.Authorize(testRequestContext, "test-app", "windows", "pastdue-recovery@example.com")
	require.NoError(t, err)

	// Transition to past_due
	_, err = db.Exec(`UPDATE subscriptions SET status = 'past_due' WHERE subscription_id = $1`, "sub_pastdue_recovery")
	require.NoError(t, err)

	// Past due: access denied
	_, err = authorizer.Authorize(testRequestContext, "test-app", "windows", "pastdue-recovery@example.com")
	assert.True(t, errors.Is(err, ErrDownloadRequiresActiveSubscription))

	// Recovery: back to active
	_, err = db.Exec(`UPDATE subscriptions SET status = 'active' WHERE subscription_id = $1`, "sub_pastdue_recovery")
	require.NoError(t, err)

	// Active again: access restored
	asset, err := authorizer.Authorize(testRequestContext, "test-app", "windows", "pastdue-recovery@example.com")
	require.NoError(t, err)
	assert.Equal(t, "windows", asset.Platform)
}

// TestIntegration_Subscription_Trial_Expired_No_Payment_Download_Denied verifies
// that when a trial ends without payment, download access is denied.
func TestIntegration_Subscription_Trial_Expired_No_Payment_Download_Denied(t *testing.T) {
	db, authorizer := setupDownloadEntitlementDB(t)
	seedSubscription(t, db, "trial-expired@example.com", "trialing", "sub_trial_expire")

	// Trialing: access granted
	asset, err := authorizer.Authorize(testRequestContext, "test-app", "windows", "trial-expired@example.com")
	require.NoError(t, err)
	assert.Equal(t, "windows", asset.Platform)

	// Trial expires without payment → incomplete_expired
	_, err = db.Exec(`UPDATE subscriptions SET status = 'incomplete_expired' WHERE subscription_id = $1`, "sub_trial_expire")
	require.NoError(t, err)

	// Access denied
	_, err = authorizer.Authorize(testRequestContext, "test-app", "windows", "trial-expired@example.com")
	assert.True(t, errors.Is(err, ErrDownloadRequiresActiveSubscription),
		"expired trial should deny downloads, got: %v", err)
}

// ============================================================================
// Error Propagation Tests
// ============================================================================

// TestIntegration_Entitlement_LookupError_Download_Denied verifies fail-closed
// behavior: when the entitlement lookup errors, download is denied.
func TestIntegration_Entitlement_LookupError_Download_Denied(t *testing.T) {
	fakeDownloads := &fakeDownloads{
		assets: map[string]*DownloadAsset{
			"bundle:app:windows": {Platform: "windows", RequiresEntitlement: true},
		},
	}
	failingEntitlements := &trackingEntitlements{
		err: errors.New("database connection refused"),
	}

	authorizer := NewDownloadAuthorizer(fakeDownloads, failingEntitlements, "bundle")
	_, err := authorizer.Authorize(testRequestContext, "app", "windows", "user@example.com")
	require.Error(t, err, "download should be denied when entitlement lookup fails")
	assert.Contains(t, err.Error(), "database connection refused")
	assert.Equal(t, 1, failingEntitlements.calls)
}
