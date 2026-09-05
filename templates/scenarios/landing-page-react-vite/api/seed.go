package main

import (
	"context"
	"database/sql"
	"fmt"
)

// defaultAdminHash is the bcrypt hash for the seeded admin@localhost operator
// (password "admin"). Operators are expected to rotate this immediately in any
// non-demo deployment.
const defaultAdminHash = "$2a$10$nhmpbhFPQUZZwEH.qaYHCeiKBWDvr8z5Z7eM4v62MmNwm.0N.5xeG"

// seedDefaultData installs idempotent default/demo data after schemas are
// applied: the default admin user, singleton payment-settings and branding
// rows, the control/variant-a variants with their hero sections and axes, and
// the default bundle pricing and download catalog.
//
// Every insert is written to be safe to re-run (ON CONFLICT DO NOTHING / DO
// UPDATE), so this runs on every boot and is also the body reused by the admin
// demo-reset after a TRUNCATE. The ordering respects cross-table foreign keys
// (variants before sections/axes, bundle products before prices, download apps
// before assets).
func seedDefaultData(ctx context.Context, db *sql.DB) error {
	if err := seedAdmin(ctx, db); err != nil {
		return err
	}
	if err := seedBranding(ctx, db); err != nil {
		return err
	}
	if err := seedVariants(ctx, db); err != nil {
		return err
	}
	if err := seedBundles(ctx, db); err != nil {
		return err
	}
	return seedDownloads(ctx, db)
}

func seedAdmin(ctx context.Context, db *sql.DB) error {
	if _, err := db.ExecContext(ctx, `
		INSERT INTO admin_users (email, password_hash) VALUES ('admin@localhost', $1)
		ON CONFLICT (email) DO NOTHING
	`, defaultAdminHash); err != nil {
		return fmt.Errorf("seed admin: %w", err)
	}
	if _, err := db.ExecContext(ctx, `
		INSERT INTO payment_settings (id) VALUES (1) ON CONFLICT (id) DO NOTHING
	`); err != nil {
		return fmt.Errorf("seed payment settings: %w", err)
	}
	return nil
}

func seedBranding(ctx context.Context, db *sql.DB) error {
	_, err := db.ExecContext(ctx, `
		INSERT INTO site_branding (id, site_name, tagline)
		VALUES (1, 'My Landing', 'Ship your idea faster')
		ON CONFLICT (id) DO NOTHING
	`)
	if err != nil {
		return fmt.Errorf("seed branding: %w", err)
	}
	return nil
}

func seedVariants(ctx context.Context, db *sql.DB) error {
	for _, v := range []struct {
		slug, name, description, headline string
	}{
		{"control", "Control", "Default landing experience", "Build and ship, faster"},
		{"variant-a", "Variant A", "Alternate hero copy", "Your idea, live today"},
	} {
		var id int64
		err := db.QueryRowContext(ctx, `
			INSERT INTO variants (slug, name, description, weight, status, header_config, seo_config)
			VALUES ($1, $2, $3, 50, 'active', '{}'::jsonb, '{}'::jsonb)
			ON CONFLICT (slug) DO NOTHING
			RETURNING id
		`, v.slug, v.name, v.description).Scan(&id)
		if err == sql.ErrNoRows {
			if err = db.QueryRowContext(ctx, `SELECT id FROM variants WHERE slug = $1`, v.slug).Scan(&id); err != nil {
				return fmt.Errorf("seed variant lookup %s: %w", v.slug, err)
			}
		} else if err != nil {
			return fmt.Errorf("seed variant %s: %w", v.slug, err)
		}

		content := fmt.Sprintf(`{"headline": %q, "subheadline": "A production-ready landing page template."}`, v.headline)
		if _, err := db.ExecContext(ctx, `
			INSERT INTO content_sections (variant_id, section_type, content, "order", enabled)
			SELECT $1, 'hero', $2::jsonb, 0, TRUE
			WHERE NOT EXISTS (SELECT 1 FROM content_sections WHERE variant_id = $1 AND section_type = 'hero')
		`, id, content); err != nil {
			return fmt.Errorf("seed hero section %s: %w", v.slug, err)
		}
		if _, err := db.ExecContext(ctx, `
			INSERT INTO variant_axes (variant_id, axis_id, variant_value)
			VALUES ($1, 'hero_copy', $2)
			ON CONFLICT (variant_id, axis_id) DO NOTHING
		`, id, v.slug); err != nil {
			return fmt.Errorf("seed variant axes %s: %w", v.slug, err)
		}
	}
	return nil
}

func seedBundles(ctx context.Context, db *sql.DB) error {
	var productID int64
	err := db.QueryRowContext(ctx, `
		INSERT INTO bundle_products (bundle_key, bundle_name, stripe_product_id, credits_per_usd, display_credits_multiplier, display_credits_label, environment)
		VALUES ('business_suite', 'Business Suite', 'prod_business_suite', 1000000, 0.001, 'credits', 'production')
		ON CONFLICT (bundle_key) DO NOTHING
		RETURNING id
	`).Scan(&productID)
	if err == sql.ErrNoRows {
		if err = db.QueryRowContext(ctx, `SELECT id FROM bundle_products WHERE bundle_key = 'business_suite'`).Scan(&productID); err != nil {
			return fmt.Errorf("seed bundle product lookup: %w", err)
		}
	} else if err != nil {
		return fmt.Errorf("seed bundle product: %w", err)
	}

	for _, p := range []struct {
		priceID, name, interval string
		amount, rank, weight    int
	}{
		{"price_business_monthly", "Monthly", "month", 4900, 1, 20},
		{"price_business_yearly", "Yearly", "year", 49000, 2, 10},
	} {
		if _, err := db.ExecContext(ctx, `
			INSERT INTO bundle_prices (
				product_id, stripe_price_id, plan_name, plan_tier, billing_interval,
				amount_cents, currency, plan_rank, kind, display_enabled, display_weight
			) VALUES ($1, $2, $3, 'pro', $4, $5, 'usd', $6, 'subscription', TRUE, $7)
			ON CONFLICT (stripe_price_id) DO NOTHING
		`, productID, p.priceID, p.name, p.interval, p.amount, p.rank, p.weight); err != nil {
			return fmt.Errorf("seed bundle price %s: %w", p.priceID, err)
		}
	}
	return nil
}

func seedDownloads(ctx context.Context, db *sql.DB) error {
	if _, err := db.ExecContext(ctx, `
		INSERT INTO download_apps (bundle_key, app_key, name, tagline, display_order)
		VALUES ('business_suite', 'desktop', 'Desktop App', 'Install the desktop client', 0)
		ON CONFLICT (bundle_key, app_key) DO NOTHING
	`); err != nil {
		return fmt.Errorf("seed download app: %w", err)
	}
	if _, err := db.ExecContext(ctx, `
		INSERT INTO download_assets (bundle_key, app_key, platform, artifact_url, release_version, requires_entitlement)
		VALUES ('business_suite', 'desktop', 'mac', 'https://downloads.example.com/desktop-mac.dmg', '1.0.0', FALSE)
		ON CONFLICT (bundle_key, app_key, platform) DO NOTHING
	`); err != nil {
		return fmt.Errorf("seed download asset: %w", err)
	}
	return nil
}
