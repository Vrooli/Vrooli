package main

import (
	"testing"
)

// NOTE: Tests that used NewLandingConfigService with database-backed services have been removed.
// Variant configuration is stored in tracked config JSON files and loaded via ConfigStore.
// See config_store_test.go for tests of the new architecture.

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

// [REQ:LANDING-CONFIG] The baked fallback remains renderable when live configuration is unavailable.
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
