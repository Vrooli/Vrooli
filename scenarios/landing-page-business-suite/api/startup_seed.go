package main

import (
	"context"
	"database/sql"
	"encoding/json"
	"errors"
	"fmt"
	"strings"

	"landing-page-business-suite-api/internal/analytics"
	"landing-page-business-suite-api/internal/commerce"
	"landing-page-business-suite-api/internal/delivery"
	"landing-page-business-suite-api/internal/logx"

	"github.com/vrooli/api-core/database"
)

// applyRuntimeSchema applies the ordered, domain-owned declarative schemas.
// api-core performs only additive column reconciliation for existing tables;
// there are no migration or data-move statements at application startup.
func applyRuntimeSchema(db StartupStore) error {
	concrete, ok := db.(*sql.DB)
	if !ok {
		return fmt.Errorf("schema initialization requires a concrete database connection")
	}
	ctx := context.Background()
	provider := database.SchemaProviderFunc(runtimeSchema)
	// Reconcile additive columns before executing the domain DDL. Existing
	// deployments can contain indexes in CREATE TABLE blobs that reference
	// columns added after the table was first created; applying those blobs
	// first would fail before api-core gets a chance to repair the drift.
	if err := database.ReconcileDeclaredColumns(ctx, concrete, provider); err != nil {
		return err
	}
	if err := database.ApplySchemas(ctx, concrete, provider); err != nil {
		return err
	}
	if err := database.VerifyDeclaredColumns(ctx, concrete, provider); err != nil {
		return err
	}
	if _, err := concrete.ExecContext(ctx, analytics.IndexesSchema()); err != nil {
		return fmt.Errorf("apply analytics indexes: %w", err)
	}
	if _, err := concrete.ExecContext(ctx, commerce.FinancialIndexesSchema()); err != nil {
		return fmt.Errorf("apply financial indexes: %w", err)
	}
	return nil
}

// seedDefaultData sets up baseline records that are not variant-specific.
func seedDefaultData(db StartupStore) error {
	if err := applyRuntimeSchema(db); err != nil {
		return fmt.Errorf("failed to apply schema: %w", err)
	}

	adminEmail, adminPasswordHash, err := getAdminDefaults()
	if err != nil {
		return fmt.Errorf("failed to get admin defaults: %w", err)
	}
	if _, err := db.Exec(seedDeleteDuplicateAdminSQL, adminEmail, seededAdminID); err != nil {
		return fmt.Errorf("failed to cleanup admin duplicates: %w", err)
	}
	// The seeded admin is bootstrap state: an administrator who changes their
	// account through the profile surface owns it, and a reseed must never
	// revert that change. So the operator-supplied password credential is
	// re-applied only while the stored account still holds the bootstrap email.
	// That lets an operator rotate the declared credential to recover a lost
	// password without clobbering a self-managed account.
	adminSeedSQL := seedAdminSQL
	if adminPasswordConfigured() && seededAdminHoldsBootstrapEmail(db, adminEmail) {
		adminSeedSQL = seedAdminUpsertSQL
	}
	if _, err := db.Exec(adminSeedSQL, seededAdminID, adminEmail, adminPasswordHash); err != nil {
		return fmt.Errorf("failed to seed admin user: %w", err)
	}
	if _, err := db.Exec(seedAdminSequenceSQL); err != nil {
		return fmt.Errorf("synchronize admin user sequence: %w", err)
	}
	if adminEmail != defaultAdminEmail {
		logx.Info("admin_user_seeded_custom", map[string]interface{}{"level": "info", "email": adminEmail})
	}
	if _, err := db.Exec(seedPaymentSettingsSQL); err != nil {
		return fmt.Errorf("failed to seed payment settings: %w", err)
	}
	downloads, err := delivery.DefaultDownloadSeed()
	if err != nil {
		return fmt.Errorf("load delivery seed: %w", err)
	}
	if err := seedDownloadDefaults(db, downloads); err != nil {
		return err
	}
	return seedTierLimitsDefaults(db)
}

// seededAdminHoldsBootstrapEmail reports whether the seeded administrator still
// carries the bootstrap email (or does not exist yet). A different email means
// the account is operator-managed and must not be reconciled.
func seededAdminHoldsBootstrapEmail(db StartupStore, bootstrapEmail string) bool {
	var storedEmail string
	if err := db.QueryRow(`SELECT email FROM admin_users WHERE id = $1`, seededAdminID).Scan(&storedEmail); err != nil {
		// No row yet (first start) uses the insert path; any other read failure
		// falls back to the non-destructive insert.
		return errors.Is(err, sql.ErrNoRows)
	}
	return strings.EqualFold(strings.TrimSpace(storedEmail), strings.TrimSpace(bootstrapEmail))
}

func seedDownloadDefaults(db StartupStore, downloads []delivery.App) error {
	if len(downloads) == 0 {
		return nil
	}
	for idx, app := range downloads {
		bundleKey := strings.TrimSpace(app.BundleKey)
		appKey := strings.TrimSpace(app.AppKey)
		if bundleKey == "" {
			bundleKey = "business_suite"
		}
		if appKey == "" {
			appKey = fmt.Sprintf("bundle_app_%d", idx+1)
		}
		installSteps, err := json.Marshal(app.InstallSteps)
		if err != nil {
			return fmt.Errorf("marshal install steps for %s: %w", appKey, err)
		}
		storefronts, err := json.Marshal(app.Storefronts)
		if err != nil {
			return fmt.Errorf("marshal storefronts for %s: %w", appKey, err)
		}
		metadata, err := json.Marshal(app.Metadata)
		if err != nil {
			return fmt.Errorf("marshal metadata for %s: %w", appKey, err)
		}
		displayOrder := app.DisplayOrder
		if displayOrder == 0 {
			displayOrder = idx + 1
		}
		var iconURL, screenshotURL interface{}
		if strings.TrimSpace(app.IconURL) != "" {
			iconURL = app.IconURL
		}
		if strings.TrimSpace(app.ScreenshotURL) != "" {
			screenshotURL = app.ScreenshotURL
		}
		if _, err := db.Exec(seedDownloadAppSQL, bundleKey, appKey, app.Name, app.Tagline, app.Description, iconURL, screenshotURL, app.InstallOverview, installSteps, storefronts, metadata, displayOrder); err != nil {
			return fmt.Errorf("seed download app %s: %w", appKey, err)
		}

		for _, asset := range app.Platforms {
			platform := strings.TrimSpace(asset.Platform)
			if platform == "" {
				continue
			}
			assetMeta, err := json.Marshal(asset.Metadata)
			if err != nil {
				return fmt.Errorf("marshal asset metadata %s:%s: %w", appKey, platform, err)
			}
			assetBundle := asset.BundleKey
			if strings.TrimSpace(assetBundle) == "" {
				assetBundle = bundleKey
			}
			assetAppKey := asset.AppKey
			if strings.TrimSpace(assetAppKey) == "" {
				assetAppKey = appKey
			}
			if _, err := db.Exec(seedDownloadAssetSQL, assetBundle, assetAppKey, platform, asset.ArtifactURL, asset.ReleaseVersion, asset.ReleaseNotes, asset.Checksum, asset.RequiresEntitlement, assetMeta); err != nil {
				return fmt.Errorf("seed download asset %s:%s: %w", appKey, platform, err)
			}
		}
	}
	// Legacy rows predate app-level visibility metadata. Keep them configured
	// and recoverable in the admin CMS, but do not publish them accidentally.
	if _, err := db.Exec(seedDownloadVisibilityMigrationSQL, "business_suite"); err != nil {
		return fmt.Errorf("migrate download app visibility: %w", err)
	}
	return nil
}

// seedTierLimitsDefaults seeds default subscription tier limits for the cost-based credit system.
func seedTierLimitsDefaults(db StartupStore) error {
	const bundleKey = "business_suite"
	tierLimits := []struct {
		tierID       string
		limitType    string
		limitKey     string
		limitValue   int64
		appBundleKey *string
	}{
		{"free", "cost_based", "ai_credits", 0, stringPointer(bundleKey)},
		{"solo", "cost_based", "ai_credits", 500000000, stringPointer(bundleKey)},
		{"pro", "cost_based", "ai_credits", 2000000000, stringPointer(bundleKey)},
		{"studio", "cost_based", "ai_credits", 10000000000, stringPointer(bundleKey)},
		{"business", "cost_based", "ai_credits", -1, stringPointer(bundleKey)},
		{"free", "count_based", "workflow_executions", 0, stringPointer(bundleKey)},
		{"solo", "count_based", "workflow_executions", 100, stringPointer(bundleKey)},
		{"pro", "count_based", "workflow_executions", 1000, stringPointer(bundleKey)},
		{"studio", "count_based", "workflow_executions", 5000, stringPointer(bundleKey)},
		{"business", "count_based", "workflow_executions", -1, stringPointer(bundleKey)},
		{"free", "count_based", "voice_minutes", 0, stringPointer(bundleKey)},
		{"solo", "count_based", "voice_minutes", 60, stringPointer(bundleKey)},
		{"pro", "count_based", "voice_minutes", 600, stringPointer(bundleKey)},
		{"studio", "count_based", "voice_minutes", 3000, stringPointer(bundleKey)},
		{"business", "count_based", "voice_minutes", -1, stringPointer(bundleKey)},
		{"free", "count_based", "compute_minutes", 0, stringPointer(bundleKey)},
		{"solo", "count_based", "compute_minutes", 60, stringPointer(bundleKey)},
		{"pro", "count_based", "compute_minutes", 600, stringPointer(bundleKey)},
		{"studio", "count_based", "compute_minutes", 3000, stringPointer(bundleKey)},
		{"business", "count_based", "compute_minutes", -1, stringPointer(bundleKey)},
	}
	for _, limit := range tierLimits {
		if _, err := db.Exec(seedTierLimitSQL, limit.tierID, limit.limitType, limit.limitKey, limit.limitValue, limit.appBundleKey); err != nil {
			return fmt.Errorf("seed tier limit %s/%s: %w", limit.tierID, limit.limitKey, err)
		}
	}
	logx.Info("tier_limits_seeded", map[string]interface{}{"level": "info", "count": len(tierLimits), "bundle_key": bundleKey})
	return nil
}

func stringPointer(value string) *string { return &value }
