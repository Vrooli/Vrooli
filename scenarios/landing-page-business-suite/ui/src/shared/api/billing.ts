import { fromJson, type JsonValue, type DescMessage } from '@bufbuild/protobuf';
import {
  ConfigSource,
  GetStripeSettingsResponseSchema,
  UpdateStripeSettingsResponseSchema,
  type GetStripeSettingsResponse,
  type UpdateStripeSettingsResponse,
  type StripeConfigSnapshot,
  type StripeSettings,
} from '@proto-lprv/settings_pb';
import { apiCall } from './common';
import type { BillingPortalResponse, BundleCatalogEntry, CheckoutSession } from './types';
import { normalizeTimestamp } from '../lib/protobuf-utils';
import { parseOrNull } from './safeParse';
import {
  BundleCatalogResponseSchema,
  CheckoutSessionSchema,
  BillingPortalResponseSchema,
  VerifyStripePriceResponseSchema,
} from './schemas/billing.schema';

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
  stripe_price_id?: string;
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
    const message = fromJson(GetStripeSettingsResponseSchema as DescMessage, resp as JsonValue, {
      ignoreUnknownFields: true,
    }) as GetStripeSettingsResponse;
    return flattenStripeSettings(message.snapshot, message.settings);
  });
}

export function updateStripeSettings(payload: StripeSettingsUpdatePayload) {
  return apiCall('/admin/settings/stripe', {
    method: 'PUT',
    body: JSON.stringify(payload),
  }).then((resp) => {
    const message = fromJson(UpdateStripeSettingsResponseSchema as DescMessage, resp as JsonValue, {
      ignoreUnknownFields: true,
    }) as UpdateStripeSettingsResponse;
    return flattenStripeSettings(message.snapshot, message.settings);
  });
}

export function getBundleCatalog() {
  return apiCall<BundleCatalogResponse>('/admin/bundles').then((resp) => {
    const validated = parseOrNull(BundleCatalogResponseSchema, resp, 'BundleCatalogResponse');
    if (!validated) {
      // Return empty bundles array if validation fails but log error
      return { bundles: [] };
    }
    return validated;
  });
}

export function updateBundlePrice(bundleKey: string, priceId: string, payload: UpdateBundlePricePayload) {
  return apiCall(`/admin/bundles/${encodeURIComponent(bundleKey)}/prices/${encodeURIComponent(priceId)}`, {
    method: 'PATCH',
    body: JSON.stringify(payload),
  });
}

export function verifyStripePrice(key: string) {
  const params = new URLSearchParams({ key });
  return apiCall<{ id: string; lookup_key?: string; currency?: string; amount_cents?: number; interval?: string; active?: boolean; product?: string }>(
    `/admin/stripe/verify-price?${params.toString()}`,
  ).then((resp) => {
    const validated = parseOrNull(VerifyStripePriceResponseSchema, resp, 'VerifyStripePriceResponse');
    if (!validated) {
      throw new Error('Invalid price verification response from Stripe');
    }
    return validated;
  });
}

export function createCheckoutSession(payload: {
  price_id: string;
  customer_email?: string;
  success_url?: string;
  cancel_url?: string;
}) {
  const body: Record<string, string | undefined> = {
    price_id: payload.price_id,
    success_url: payload.success_url,
    cancel_url: payload.cancel_url,
  };

  if (payload.customer_email) {
    body.customer_email = payload.customer_email;
  }

  return apiCall<{ session: CheckoutSession }>('/billing/create-checkout-session', {
    method: 'POST',
    body: JSON.stringify(body),
  }).then((resp) => {
    const validated = parseOrNull(CheckoutSessionSchema, resp.session, 'CheckoutSession');
    if (!validated) {
      throw new Error('Invalid checkout session response');
    }
    return validated;
  });
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
  }).then((resp) => {
    const validated = parseOrNull(CheckoutSessionSchema, resp.session, 'CheckoutSession');
    if (!validated) {
      throw new Error('Invalid credits checkout session response');
    }
    return validated;
  });
}

export function createBillingPortalSession(returnUrl?: string, userEmail?: string) {
  const params = new URLSearchParams();
  if (returnUrl) params.set('return_url', returnUrl);
  if (userEmail) params.set('user', userEmail);
  const suffix = params.toString() ? `?${params.toString()}` : '';
  return apiCall<BillingPortalResponse>(`/billing/portal-url${suffix}`).then((resp) => {
    const validated = parseOrNull(BillingPortalResponseSchema, resp, 'BillingPortalResponse');
    if (!validated) {
      throw new Error('Invalid billing portal response');
    }
    return validated;
  });
}
