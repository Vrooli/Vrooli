import { apiCall } from './common';
import type { BundleCatalogEntry } from './types';

export interface StripeSettingsResponse {
  publishable_key_preview?: string;
  publishable_key_set: boolean;
  secret_key_set: boolean;
  webhook_secret_set: boolean;
  dashboard_url?: string;
  updated_at?: string;
  source: 'env' | 'database' | string;
}

export interface StripeSettingsUpdatePayload {
  publishable_key?: string;
  secret_key?: string;
  webhook_secret?: string;
  dashboard_url?: string;
}

export interface BundleCatalogResponse {
  bundles: BundleCatalogEntry[];
}

export interface UpdateBundlePricePayload {
  plan_name?: string;
  display_weight?: number;
  display_enabled?: boolean;
  subtitle?: string;
  badge?: string;
  cta_label?: string;
  highlight?: boolean;
  features?: string[];
}

export function getStripeSettings() {
  return apiCall<StripeSettingsResponse>('/admin/settings/stripe');
}

export function updateStripeSettings(payload: StripeSettingsUpdatePayload) {
  return apiCall<StripeSettingsResponse>('/admin/settings/stripe', {
    method: 'PUT',
    body: JSON.stringify(payload),
  });
}

export function getBundleCatalog() {
  return apiCall<BundleCatalogResponse>('/admin/bundles');
}

export function updateBundlePrice(bundleKey: string, priceId: string, payload: UpdateBundlePricePayload) {
  return apiCall(`/admin/bundles/${encodeURIComponent(bundleKey)}/prices/${encodeURIComponent(priceId)}`, {
    method: 'PATCH',
    body: JSON.stringify(payload),
  });
}
