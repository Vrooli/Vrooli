/** Synthetic owner records for tests and the isolated review entry only. */
import type { DownloadAsset, PlanOption, PricingOverview } from '../../../shared/api/types';
export const downloadOptions: DownloadAsset[] = [
  { id: 11, bundle_key: 'example', app_key: 'example-app', platform: 'linux', release_version: '2.4.1', requires_entitlement: false, artifact_url: '/never-use-catalog-url', artifact_filename: 'example-linux-x64.AppImage', checksum: 'a'.repeat(64), release_notes: 'Configured release notes for this installer.' },
  { id: 12, bundle_key: 'example', app_key: 'example-app', platform: 'mac', release_version: '2.4.0', requires_entitlement: true, artifact_url: '/never-use-catalog-url', artifact_filename: 'example-mac-arm64.dmg' },
];
export const ownerPlan: PlanOption = { plan_name: 'Configured monthly plan', plan_tier: 'example', billing_interval: 'month', amount_cents: 1234, currency: 'usd', intro_enabled: false, stripe_price_id: 'price-month', monthly_included_credits: 0, one_time_bonus_credits: 0, display_enabled: true, display_weight: 0, bundle_key: 'example' };
export const ownerPricing: PricingOverview = { bundle: { bundle_key: 'example', name: 'Configured owner', stripe_product_id: 'prod-example', credits_per_usd: 100, display_credits_multiplier: 1, display_credits_label: 'units' }, monthly: [ownerPlan], yearly: [{ ...ownerPlan, plan_name: 'Configured yearly plan', stripe_price_id: 'price-year', billing_interval: 'year', amount_cents: 12345 }], updated_at: '2026-09-15T00:00:00Z' };
