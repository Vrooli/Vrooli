package main

import (
	"context"
	"database/sql"
	"testing"
)

func TestLandingConfigServiceReturnsFallbackWhenVariantMissing(t *testing.T) {
	db := setupTestDB(t)
	defer db.Close()

	service := NewLandingConfigService(
		NewVariantService(db, defaultVariantSpace),
		NewContentService(db),
		NewPlanService(db),
		NewDownloadService(db),
		NewBrandingService(db),
	)

	// [REQ:AB-FALLBACK] Backend should return baked fallback config when variant lookup fails
	resp, err := service.GetLandingConfig(context.Background(), "nonexistent-variant")
	if err != nil {
		t.Fatalf("GetLandingConfig failed: %v", err)
	}

	if !resp.Fallback {
		t.Fatalf("expected fallback response, got active variant %s", resp.Variant.Slug)
	}
	if resp.Variant.Slug != "fallback" {
		t.Fatalf("expected fallback slug, got %s", resp.Variant.Slug)
	}
	if len(resp.Sections) == 0 {
		t.Fatal("expected fallback sections")
	}
}

func TestLandingConfigServiceFallsBackWhenVariantHasNoSections(t *testing.T) {
	db := setupTestDB(t)
	defer db.Close()

	variantID := createTestVariant(t, db)
	clearSectionsForVariant(t, db, variantID)

	service := NewLandingConfigService(
		NewVariantService(db, defaultVariantSpace),
		NewContentService(db),
		NewPlanService(db),
		NewDownloadService(db),
		NewBrandingService(db),
	)

	resp, err := service.GetLandingConfig(context.Background(), "test-variant")
	if err != nil {
		t.Fatalf("GetLandingConfig failed: %v", err)
	}

	if !resp.Fallback {
		t.Fatalf("expected fallback response when sections missing, got %s", resp.Variant.Slug)
	}
}

func TestLandingConfigServiceFallsBackWhenHeroMissing(t *testing.T) {
	db := setupTestDB(t)
	defer db.Close()

	variantID := createTestVariant(t, db)
	clearSectionsForVariant(t, db, variantID)
	insertTestSection(t, db, variantID, "features", 1)

	service := NewLandingConfigService(
		NewVariantService(db, defaultVariantSpace),
		NewContentService(db),
		NewPlanService(db),
		NewDownloadService(db),
		NewBrandingService(db),
	)

	resp, err := service.GetLandingConfig(context.Background(), "test-variant")
	if err != nil {
		t.Fatalf("GetLandingConfig failed: %v", err)
	}

	if !resp.Fallback {
		t.Fatalf("expected fallback when hero missing, got active slug %s", resp.Variant.Slug)
	}
}

func TestLandingConfigServiceAllowsVariantsWithHero(t *testing.T) {
	db := setupTestDB(t)
	defer db.Close()

	variantID := createTestVariant(t, db)
	clearSectionsForVariant(t, db, variantID)
	insertTestSection(t, db, variantID, "hero", 1)
	insertTestSection(t, db, variantID, "features", 2)

	service := NewLandingConfigService(
		NewVariantService(db, defaultVariantSpace),
		NewContentService(db),
		NewPlanService(db),
		NewDownloadService(db),
		NewBrandingService(db),
	)

	resp, err := service.GetLandingConfig(context.Background(), "test-variant")
	if err != nil {
		t.Fatalf("GetLandingConfig failed: %v", err)
	}

	if resp.Fallback {
		t.Fatal("expected live variant when hero section present")
	}
	if resp.Variant.Slug != "test-variant" {
		t.Fatalf("expected test-variant slug, got %s", resp.Variant.Slug)
	}
	if len(resp.Sections) != 2 {
		t.Fatalf("expected 2 sections, got %d", len(resp.Sections))
	}
}

func TestParseFallbackLandingConfigRequiresVariant(t *testing.T) {
	payloadJSON := []byte(`{
		"variant": { "name": "Missing Slug" },
		"sections": [
			{ "section_type": "hero", "content": { "title": "Hi" } }
		],
		"pricing": {
			"bundle": {
				"id": 1,
				"bundle_key": "bundle",
				"name": "Bundle",
				"stripe_product_id": "prod_123",
				"credits_per_usd": 1,
				"display_credits_multiplier": 1,
				"display_credits_label": "credits",
				"environment": "production"
			},
			"monthly": [],
			"yearly": [],
			"updated_at": "2024-01-01T00:00:00Z"
		},
		"downloads": []
	}`)

	if _, err := parseFallbackLandingConfig(payloadJSON); err == nil {
		t.Fatal("expected error for missing variant slug")
	}
}

func TestParseFallbackLandingConfigNormalizesSectionsAndAxes(t *testing.T) {
	payloadJSON := []byte(`{
		"variant": {
			"slug": "fallback",
			"name": "Fallback"
		},
		"axes": {
			"persona": "ops",
			"jtbd": "demo"
		},
		"sections": [
			{ "section_type": "hero", "content": { "title": "Hero" } },
			{ "section_type": "pricing", "order": 5, "enabled": false, "content": {} },
			{ "section_type": "", "content": { "title": "ignored" } }
		],
		"pricing": {
			"bundle": {
				"id": 1,
				"bundle_key": "bundle",
				"name": "Bundle",
				"stripe_product_id": "prod_123",
				"credits_per_usd": 1,
				"display_credits_multiplier": 1,
				"display_credits_label": "credits",
				"environment": "production"
			},
			"monthly": [],
			"yearly": [],
			"updated_at": "2024-01-01T00:00:00Z"
		},
		"downloads": [
			{
				"bundle_key": "bundle",
				"app_key": "desktop",
				"name": "Desktop App",
				"platforms": [
					{
						"id": 10,
						"bundle_key": "bundle",
						"app_key": "desktop",
						"platform": "windows",
						"artifact_url": "https://example.com",
						"release_version": "1.0.0",
						"requires_entitlement": true
					}
				]
			}
		]
	}`)

	payload, err := parseFallbackLandingConfig(payloadJSON)
	if err != nil {
		t.Fatalf("expected payload, got %v", err)
	}

	if payload.Variant.Axes["persona"] != "ops" {
		t.Fatalf("expected axes propagation, got %v", payload.Variant.Axes)
	}

	if len(payload.Sections) != 2 {
		t.Fatalf("expected 2 usable sections, got %d", len(payload.Sections))
	}

	if payload.Sections[0].Order != 1 || !payload.Sections[0].Enabled {
		t.Fatalf("expected inferred order=1 enabled=true, got order=%d enabled=%v", payload.Sections[0].Order, payload.Sections[0].Enabled)
	}

	if payload.Sections[1].Order != 5 || payload.Sections[1].Enabled {
		t.Fatalf("expected explicit order/enabled preserved, got order=%d enabled=%v", payload.Sections[1].Order, payload.Sections[1].Enabled)
	}

	if len(payload.Downloads) != 1 || len(payload.Downloads[0].Platforms) != 1 || payload.Downloads[0].Platforms[0].Platform != "windows" {
		t.Fatalf("expected downloads copied, got %+v", payload.Downloads)
	}
}

func clearSectionsForVariant(t *testing.T, db *sql.DB, variantID int64) {
	t.Helper()
	if _, err := db.Exec(`DELETE FROM content_sections WHERE variant_id = $1`, variantID); err != nil {
		t.Fatalf("failed to clear sections: %v", err)
	}
}

func insertTestSection(t *testing.T, db *sql.DB, variantID int64, sectionType string, order int) {
	t.Helper()
	payload := `{"title":"Test"}`
	if _, err := db.Exec(`
		INSERT INTO content_sections (variant_id, section_type, content, "order", enabled, created_at, updated_at)
		VALUES ($1, $2, $3::jsonb, $4, TRUE, NOW(), NOW())
	`, variantID, sectionType, payload, order); err != nil {
		t.Fatalf("failed to insert %s section: %v", sectionType, err)
	}
}
