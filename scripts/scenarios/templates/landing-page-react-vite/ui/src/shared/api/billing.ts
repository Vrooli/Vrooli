import { fromJson } from '@bufbuild/protobuf';
import { GetStripeSettingsResponseSchema, UpdateStripeSettingsResponseSchema, type StripeConfigSnapshot, type StripeSettings } from '@proto-lprv/settings_pb';
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

function flattenStripeSettings(snapshot?: StripeConfigSnapshot, settings?: StripeSettings): StripeSettingsResponse {
  return {
    publishable_key_preview: snapshot?.publishableKeyPreview,
    publishable_key_set: Boolean(snapshot?.publishableKeySet),
    secret_key_set: Boolean(snapshot?.secretKeySet),
    webhook_secret_set: Boolean(snapshot?.webhookSecretSet),
    dashboard_url: settings?.dashboardUrl,
    updated_at: settings?.updatedAt?.toJsonString(),
    source: snapshot?.source ?? 'env',
  };
}

export function getStripeSettings() {
  return apiCall('/admin/settings/stripe').then((resp) => {
    const message = fromJson(GetStripeSettingsResponseSchema, resp, {
      ignoreUnknownFields: true,
      protoFieldName: true,
    });
    return flattenStripeSettings(message.snapshot, message.settings);
  });
}

export function updateStripeSettings(payload: StripeSettingsUpdatePayload) {
  return apiCall('/admin/settings/stripe', {
    method: 'PUT',
    body: JSON.stringify(payload),
  }).then((resp) => {
    const message = fromJson(UpdateStripeSettingsResponseSchema, resp, {
      ignoreUnknownFields: true,
      protoFieldName: true,
    });
    return flattenStripeSettings(message.snapshot, message.settings);
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
