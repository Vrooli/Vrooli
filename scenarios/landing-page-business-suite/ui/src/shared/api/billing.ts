import { fromJson } from '@bufbuild/protobuf';
import {
  ConfigSource,
  GetStripeSettingsResponseSchema,
  UpdateStripeSettingsResponseSchema,
  type StripeConfigSnapshot,
  type StripeSettings,
} from '@proto-lprv/settings_pb';
import { apiCall } from './common';
import type { BillingPortalResponse, BundleCatalogEntry, CheckoutSession } from './types';

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
  const normalizeSource = (source?: ConfigSource | string | number): 'env' | 'database' | string => {
    switch (source) {
      case ConfigSource.CONFIG_SOURCE_DATABASE:
        return 'database';
      case ConfigSource.CONFIG_SOURCE_ENV:
        return 'env';
      default:
        return typeof source === 'number' ? String(source) : source ?? 'env';
    }
  };

  const normalizeTimestamp = (value: unknown): string | undefined => {
    if (!value) return undefined;
    // Buf Timestamp with toJsonString
    if (typeof (value as { toJsonString?: () => string }).toJsonString === 'function') {
      return (value as { toJsonString: () => string }).toJsonString();
    }
    if (typeof value === 'string') return value;
    if (value instanceof Date) return value.toISOString();
    // Fallback for plain objects with seconds/nanos
    const maybe = value as { seconds?: number; nanos?: number };
    if (typeof maybe.seconds === 'number') {
      const ms = maybe.seconds * 1000 + (maybe.nanos ? maybe.nanos / 1_000_000 : 0);
      return new Date(ms).toISOString();
    }
    return undefined;
  };

  return {
    publishable_key_preview: snapshot?.publishableKeyPreview,
    publishable_key_set: Boolean(snapshot?.publishableKeySet),
    secret_key_set: Boolean(snapshot?.secretKeySet),
    webhook_secret_set: Boolean(snapshot?.webhookSecretSet),
    dashboard_url: settings?.dashboardUrl,
    updated_at: normalizeTimestamp(settings?.updatedAt),
    source: normalizeSource(snapshot?.source),
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

export function createCheckoutSession(payload: { price_id: string; customer_email: string; success_url?: string; cancel_url?: string }) {
  return apiCall<{ session: CheckoutSession }>('/billing/create-checkout-session', {
    method: 'POST',
    body: JSON.stringify({
      price_id: payload.price_id,
      customer_email: payload.customer_email,
      success_url: payload.success_url,
      cancel_url: payload.cancel_url,
    }),
  }).then((resp) => resp.session);
}

export function createCreditsCheckoutSession(payload: { price_id: string; customer_email: string; success_url?: string; cancel_url?: string }) {
  return apiCall<{ session: CheckoutSession }>('/billing/create-credits-checkout-session', {
    method: 'POST',
    body: JSON.stringify({
      price_id: payload.price_id,
      customer_email: payload.customer_email,
      success_url: payload.success_url,
      cancel_url: payload.cancel_url,
    }),
  }).then((resp) => resp.session);
}

export function createBillingPortalSession(returnUrl?: string, userEmail?: string) {
  const params = new URLSearchParams();
  if (returnUrl) params.set('return_url', returnUrl);
  if (userEmail) params.set('user', userEmail);
  const suffix = params.toString() ? `?${params.toString()}` : '';
  return apiCall<BillingPortalResponse>(`/billing/portal-url${suffix}`);
}
