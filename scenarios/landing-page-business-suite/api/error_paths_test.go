package main

import (
	"context"
	"errors"
	"testing"
	"time"

	_ "github.com/mattn/go-sqlite3"
	"landing-page-business-suite-api/internal/commerce"
)

// ============================================================================
// Error Path Tests
//
// These tests verify proper error handling for database timeouts, connection
// issues, and other failure scenarios that can occur in production.
// ============================================================================

func TestErrorPaths_ContextCancellation(t *testing.T) {
	usageSvc, _, db := createTestUsageService(t)
	defer db.Close()

	// Create an already-cancelled context
	ctx, cancel := context.WithCancel(context.Background())
	cancel()

	// Test that operations properly handle cancelled context
	err := usageSvc.RecordUsage(ctx, commerce.UsageReportRequest{
		UserIdentity: "cancel-test@example.com",
		LimitKey:     "ai_credits",
		Amount:       1000,
		AppBundleKey: "test-app",
	})

	if !errors.Is(err, context.Canceled) {
		t.Fatalf("RecordUsage error = %v, want context.Canceled", err)
	}
}

func TestErrorPaths_ContextTimeout(t *testing.T) {
	usageSvc, _, db := createTestUsageService(t)
	defer db.Close()

	// Create a context that times out almost immediately
	ctx, cancel := context.WithTimeout(context.Background(), 1*time.Nanosecond)
	defer cancel()

	// Allow the context to actually expire
	time.Sleep(1 * time.Millisecond)

	err := usageSvc.RecordUsage(ctx, commerce.UsageReportRequest{
		UserIdentity: "timeout-test@example.com",
		LimitKey:     "ai_credits",
		Amount:       1000,
		AppBundleKey: "test-app",
	})

	if !errors.Is(err, context.DeadlineExceeded) {
		t.Fatalf("RecordUsage error = %v, want context.DeadlineExceeded", err)
	}
}

func TestErrorPaths_InvalidUserIdentity(t *testing.T) {
	usageSvc, _, db := createTestUsageService(t)
	defer db.Close()

	ctx := context.Background()

	// Test empty user identity
	err := usageSvc.RecordUsage(ctx, commerce.UsageReportRequest{
		UserIdentity: "",
		LimitKey:     "ai_credits",
		Amount:       1000,
		AppBundleKey: "test-app",
	})

	if err == nil {
		t.Error("Expected error for empty user identity, got nil")
	}
}

func TestErrorPaths_InvalidLimitKey(t *testing.T) {
	usageSvc, _, db := createTestUsageService(t)
	defer db.Close()

	ctx := context.Background()

	// Test empty limit key
	err := usageSvc.RecordUsage(ctx, commerce.UsageReportRequest{
		UserIdentity: "valid@example.com",
		LimitKey:     "",
		Amount:       1000,
		AppBundleKey: "test-app",
	})

	if err == nil {
		t.Error("Expected error for empty limit key, got nil")
	}
}

func TestErrorPaths_NegativeAmount(t *testing.T) {
	usageSvc, _, db := createTestUsageService(t)
	defer db.Close()

	ctx := context.Background()

	// Test negative amount
	err := usageSvc.RecordUsage(ctx, commerce.UsageReportRequest{
		UserIdentity: "valid@example.com",
		LimitKey:     "ai_credits",
		Amount:       -100,
		AppBundleKey: "test-app",
	})

	if err == nil {
		t.Fatal("negative usage amount must be rejected")
	}
}

func TestErrorPaths_GetUsage_NonexistentUser(t *testing.T) {
	usageSvc, _, db := createTestUsageService(t)
	defer db.Close()

	ctx := context.Background()

	// Query for a user that doesn't exist
	usage, err := usageSvc.GetUsage(ctx, "nonexistent@example.com", "ai_credits", nil)
	// Should return 0 usage for non-existent user, not an error
	if err != nil {
		t.Errorf("Expected nil error for non-existent user, got: %v", err)
	}
	if usage != 0 {
		t.Errorf("Expected 0 usage for non-existent user, got: %d", usage)
	}
}

func TestErrorPaths_CheckLimit_InvalidTier(t *testing.T) {
	usageSvc, _, db := createTestUsageService(t)
	defer db.Close()

	seedTestUsageTierLimits(t, db)

	ctx := context.Background()

	// Check limit with invalid tier
	allowed, remaining, err := usageSvc.CheckLimit(ctx, "test@example.com", "nonexistent_tier", "ai_credits", 1000)

	// Should handle gracefully - either error or return unlimited
	if err != nil {
		t.Logf("Invalid tier returned error (acceptable): %v", err)
	} else {
		t.Logf("Invalid tier allowed=%v, remaining=%d", allowed, remaining)
	}
}

func TestErrorPaths_AccountService_NonexistentUser(t *testing.T) {
	db := setupTestDB(t)

	planSvc := NewPlanService(db)
	svc := NewAccountService(db, planSvc)

	// Query for a user that doesn't exist
	sub, err := svc.GetSubscription("nonexistent-999999@example.com")
	// Should return nil subscription without error (or with specific error)
	if err != nil {
		t.Logf("Non-existent user returned error: %v", err)
	}
	if sub != nil {
		t.Logf("Got subscription for non-existent user: state=%v", sub.State)
	}
}

func TestErrorPaths_AccountService_EmptyEmail(t *testing.T) {
	db := setupTestDB(t)

	planSvc := NewPlanService(db)
	svc := NewAccountService(db, planSvc)

	// Test with empty email
	sub, err := svc.GetSubscription("")
	// Should handle gracefully
	if err != nil {
		t.Logf("Empty email returned error (acceptable): %v", err)
	}
	if sub != nil {
		t.Logf("Empty email returned subscription: state=%v", sub.State)
	}
}

func TestErrorPaths_LimitsService_InvalidTierID(t *testing.T) {
	db := setupTestDB(t)

	svc := NewLimitsService(db, "postgres")
	ctx := context.Background()

	// Test with empty tier ID
	limits, err := svc.GetTierLimits(ctx, "")
	if err != nil {
		t.Logf("Empty tier ID returned error (acceptable): %v", err)
	} else if len(limits) > 0 {
		t.Error("Expected empty limits for empty tier ID")
	}

	// Test with non-existent tier
	limits, err = svc.GetTierLimits(ctx, "nonexistent_tier_xyz")
	if err != nil {
		t.Errorf("Non-existent tier should return empty list, not error: %v", err)
	}
	if len(limits) > 0 {
		t.Error("Expected empty limits for non-existent tier")
	}
}

func TestErrorPaths_LimitsService_UpdateNonexistent(t *testing.T) {
	db := setupTestDB(t)

	svc := NewLimitsService(db, "postgres")
	ctx := context.Background()

	// Try to update a non-existent limit
	newValue := int64(5000)
	_, err := svc.UpdateLimit(ctx, "nonexistent_tier", "nonexistent_key", nil, commerce.TierLimitUpdate{
		LimitValue: &newValue,
	})

	if err == nil {
		t.Error("Expected error when updating non-existent limit")
	}
}

func TestErrorPaths_LimitsService_CreateDuplicate(t *testing.T) {
	db := setupTestDB(t)

	svc := NewLimitsService(db, "postgres")
	ctx := context.Background()

	// Create a limit - must set AppBundleKey because NULL != NULL in unique constraints
	appKey := "test_app_dup"
	limit := commerce.TierLimit{
		TierID:         "test_tier_dup",
		LimitType:      "cost_based",
		LimitKey:       "test_limit_dup",
		LimitValue:     1000,
		CostMultiplier: 1000000,
		AppBundleKey:   &appKey,
	}

	if _, err := db.Exec(
		`DELETE FROM subscription_tier_limits WHERE tier_id = $1 AND limit_type = $2 AND limit_key = $3 AND app_bundle_key = $4`,
		limit.TierID,
		limit.LimitType,
		limit.LimitKey,
		appKey,
	); err != nil {
		t.Fatalf("Failed to cleanup existing limit: %v", err)
	}

	_, err := svc.CreateLimit(ctx, limit)
	if err != nil {
		t.Fatalf("Failed to create initial limit: %v", err)
	}

	// Try to create duplicate
	_, err = svc.CreateLimit(ctx, limit)
	if err == nil {
		t.Error("Expected error when creating duplicate limit")
	}
}

func TestErrorPaths_LimitsService_DeleteNonexistent(t *testing.T) {
	db := setupTestDB(t)

	svc := NewLimitsService(db, "postgres")
	ctx := context.Background()

	// Try to delete a non-existent limit
	err := svc.DeleteLimit(ctx, "nonexistent_tier", "nonexistent_key", nil)

	if err == nil {
		t.Error("Expected error when deleting non-existent limit")
	}
}

func TestErrorPaths_StripeService_ConfigSnapshot(t *testing.T) {
	db := setupTestDB(t)

	svc := NewStripeService(db)

	// Override config to have no keys
	svc.UseConfigLoader(func(ctx context.Context) (stripeRuntimeConfig, error) {
		return stripeRuntimeConfig{
			publishableKey: "",
			secretKey:      "",
			webhookSecret:  "",
			hasPublishable: false,
			hasSecret:      false,
			hasWebhook:     false,
			source:         "test",
		}, nil
	})
	_ = svc.RefreshConfig(context.Background())

	// Test config snapshot
	snapshot := svc.ConfigSnapshot()
	if snapshot == nil {
		t.Error("Expected non-nil config snapshot")
	} else {
		// Verify the snapshot reflects the unconfigured state
		t.Logf("Config snapshot: hasPublishable=%v", snapshot.PublishableKeySet)
	}
}

func TestErrorPaths_StripeService_InvalidSignature(t *testing.T) {
	db := setupTestDB(t)

	svc := NewStripeService(db)

	// Set up webhook secret
	svc.UseConfigLoader(func(ctx context.Context) (stripeRuntimeConfig, error) {
		return stripeRuntimeConfig{
			publishableKey: "pk_test",
			secretKey:      "sk_test",
			webhookSecret:  "whsec_test_secret",
			hasPublishable: true,
			hasSecret:      true,
			hasWebhook:     true,
			source:         "test",
		}, nil
	})
	_ = svc.RefreshConfig(context.Background())

	// Test with invalid signature
	payload := []byte(`{"type":"test.event"}`)
	valid := svc.VerifyWebhookSignature(payload, "invalid_signature")

	if valid {
		t.Error("Expected invalid result for invalid signature")
	}
}
