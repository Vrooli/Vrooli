package main

const (
	seedDeleteDuplicateAdminSQL = `DELETE FROM admin_users WHERE LOWER(email) = LOWER($1) AND id <> $2`
	seedAdminSQL                = `INSERT INTO admin_users (id, email, password_hash) VALUES ($1, $2, $3)
		 ON CONFLICT (id) DO NOTHING`
	seedAdminSequenceSQL   = `SELECT setval(pg_get_serial_sequence('admin_users', 'id'), (SELECT COALESCE(MAX(id), 1) FROM admin_users), true)`
	seedPaymentSettingsSQL = `INSERT INTO payment_settings (id, dashboard_url, updated_at)
		VALUES (1, NULL, NOW())
		ON CONFLICT (id) DO NOTHING`
	seedDownloadAppCountSQL = `SELECT COUNT(*) FROM download_apps`
	seedDownloadAppSQL      = `INSERT INTO download_apps (bundle_key, app_key, name, tagline, description, install_overview, install_steps, storefronts, metadata, display_order)
		VALUES ($1,$2,$3,$4,$5,$6,$7::jsonb,$8::jsonb,$9::jsonb,$10)
		ON CONFLICT (bundle_key, app_key) DO UPDATE SET
			name = EXCLUDED.name,
			tagline = EXCLUDED.tagline,
			description = EXCLUDED.description,
			install_overview = EXCLUDED.install_overview,
			install_steps = EXCLUDED.install_steps,
			storefronts = EXCLUDED.storefronts,
			metadata = EXCLUDED.metadata,
			display_order = EXCLUDED.display_order,
			updated_at = NOW()`
	seedDownloadAssetSQL = `INSERT INTO download_assets (bundle_key, app_key, platform, artifact_url, release_version, release_notes, checksum, requires_entitlement, metadata, variant_key)
		VALUES ($1,$2,$3,$4,$5,$6,$7,$8,$9::jsonb,'default')
		ON CONFLICT (bundle_key, app_key, platform, variant_key)
		DO UPDATE SET artifact_url = EXCLUDED.artifact_url,
			release_version = EXCLUDED.release_version,
			release_notes = EXCLUDED.release_notes,
			checksum = EXCLUDED.checksum,
			requires_entitlement = EXCLUDED.requires_entitlement,
			metadata = EXCLUDED.metadata,
			updated_at = NOW()`
	seedTierLimitCountSQL = `SELECT COUNT(*) FROM subscription_tier_limits`
	seedTierLimitSQL      = `INSERT INTO subscription_tier_limits (tier_id, limit_type, limit_key, limit_value, app_bundle_key)
		VALUES ($1, $2, $3, $4, $5)
		ON CONFLICT (tier_id, limit_type, limit_key, app_bundle_key) DO NOTHING`
)
