import { fromJson, type JsonValue, type DescMessage } from '@bufbuild/protobuf';
import { z } from 'zod';
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
import type { BillingPortalResponse, BundleCatalogEntry, CheckoutSession, PlanOption } from './types';
import { normalizeTimestamp } from '../lib/protobuf-utils';
import { parseOrNull } from './safeParse';
import {
  BundleCatalogResponseSchema,
  CheckoutSessionSchema,
  BillingPortalResponseSchema,
  VerifyStripePriceResponseSchema,
  StripeImportPreviewSchema,
  StripeImportResultSchema,
} from './schemas/billing.schema';
import { PlanOptionSchema } from './schemas/landing.schema';

type BundleCatalogResponseParsed = z.infer<typeof BundleCatalogResponseSchema>;

const normalizeBundleCatalog = (response: BundleCatalogResponseParsed): BundleCatalogResponse => ({
  bundles: response.bundles.map((entry) => ({
    bundle: entry.bundle,
    prices: entry.prices.map((plan): PlanOption => ({
      ...plan,
      intro_enabled: plan.intro_enabled ?? false,
      monthly_included_credits: plan.monthly_included_credits ?? 0,
      one_time_bonus_credits: plan.one_time_bonus_credits ?? 0,
      display_enabled: plan.display_enabled ?? false,
      display_weight: plan.display_weight ?? 0,
    })),
  })),
});

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

export type RevealStripeSecretField = 'secret_key' | 'webhook_secret' | 'publishable_key';

export interface RevealStripeSecretResponse {
  field: RevealStripeSecretField;
  value: string;
}

export function revealStripeSecret(field: RevealStripeSecretField): Promise<RevealStripeSecretResponse> {
  const params = new URLSearchParams({ field });
  return apiCall<RevealStripeSecretResponse>(`/admin/settings/stripe/reveal?${params.toString()}`);
}

export function getBundleCatalog(): Promise<BundleCatalogResponse> {
  return apiCall<BundleCatalogResponse>('/admin/bundles').then((resp) => {
    const validated = parseOrNull(BundleCatalogResponseSchema, resp, 'BundleCatalogResponse');
    if (!validated) {
      throw new Error('Invalid bundle catalog response');
    }
    return normalizeBundleCatalog(validated);
  });
}

export function updateBundlePrice(bundleKey: string, priceId: string, payload: UpdateBundlePricePayload) {
  return apiCall(`/admin/bundles/${encodeURIComponent(bundleKey)}/prices/${encodeURIComponent(priceId)}`, {
    method: 'PATCH',
    body: JSON.stringify(payload),
  }).then((resp) => {
    const validated = parseOrNull(PlanOptionSchema, resp, 'PlanOption');
    if (!validated) {
      throw new Error('Invalid plan response from update');
    }
    return validated;
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

// Stripe Import Types
export interface StripePriceImport {
  price_id: string;
  lookup_key?: string;
  currency: string;
  amount_cents: number;
  interval?: string;
  product_id: string;
  product_name: string;
  active: boolean;
  exists_locally: boolean;
}

export interface StripeProductWithPrices {
  product_id: string;
  product_name: string;
  is_current_bundle?: boolean;
  prices: StripePriceImport[];
}

export interface StripeImportPreview {
  bundle_key?: string;
  bundle_product_id?: string;
  bundle_product_found?: boolean;
  bundle_plan_count?: number;
  products: StripeProductWithPrices[];
  total_prices: number;
  conflict_count: number;
  new_count: number;
}

export interface ImportPlanSelection {
  price_id: string;
  action: 'import' | 'overwrite' | 'skip';
}

export interface StripeImportRequest {
  bundle_product_id: string;
  mode?: 'merge' | 'replace';
  selections: ImportPlanSelection[];
}

export interface StripeImportResult {
  imported: number;
  overwritten: number;
  skipped: number;
  errors?: string[];
}

/**
 * Get a preview of products/prices available to import from Stripe.
 */
export function getStripeImportPreview(): Promise<StripeImportPreview> {
  return apiCall<StripeImportPreview>('/admin/stripe/import-preview').then((resp) => {
    const validated = parseOrNull(StripeImportPreviewSchema, resp, 'StripeImportPreview');
    if (!validated) {
      throw new Error('Invalid Stripe import preview response');
    }
    return validated;
  });
}

/**
 * Import selected prices from Stripe into the local plan store.
 */
export function importStripePlans(request: StripeImportRequest): Promise<StripeImportResult> {
  return apiCall<StripeImportResult>('/admin/stripe/import', {
    method: 'POST',
    body: JSON.stringify(request),
  }).then((resp) => {
    const validated = parseOrNull(StripeImportResultSchema, resp, 'StripeImportResult');
    if (!validated) {
      throw new Error('Invalid Stripe import result');
    }
    return validated;
  });
}

// Create Plan Types
export interface CreateBundlePricePayload {
  stripe_price_id: string;
  plan_name: string;
  plan_tier: string;
  billing_interval: string;
  amount_cents?: number;
  currency?: string;
  display_weight?: number;
  display_enabled?: boolean;
  monthly_included_credits?: number;
  subtitle?: string;
  badge?: string;
  cta_label?: string;
  highlight?: boolean;
  features?: string[];
}

/**
 * Create a new plan in the plan store.
 */
export function createBundlePrice(bundleKey: string, payload: CreateBundlePricePayload) {
  return apiCall(`/admin/bundles/${encodeURIComponent(bundleKey)}/prices`, {
    method: 'POST',
    body: JSON.stringify(payload),
  }).then((resp) => {
    const validated = parseOrNull(PlanOptionSchema, resp, 'PlanOption');
    if (!validated) {
      throw new Error('Invalid plan response from create');
    }
    return validated;
  });
}

/**
 * Delete a plan from the plan store.
 */
export function deleteBundlePrice(bundleKey: string, priceId: string) {
  return apiCall(`/admin/bundles/${encodeURIComponent(bundleKey)}/prices/${encodeURIComponent(priceId)}`, {
    method: 'DELETE',
  });
}
