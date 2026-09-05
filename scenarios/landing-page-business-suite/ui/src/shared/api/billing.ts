import { createClient } from '@connectrpc/connect';
import { z } from 'zod';
import {
  ConfigSource,
  StripeSettingsService,
  type StripeConfigSnapshot,
  type StripeSettings,
} from '@vrooli/proto-types/landing-page-business-suite/v1/settings_pb';
import { LandingPagePaymentsService, SessionKind, type CreateCheckoutSessionResponse } from '@vrooli/proto-types/landing-page-business-suite/v1/billing_pb';
import { BundleAdminService, type ListBundleCatalogResponse, type UpdateBundlePriceResponse } from '@vrooli/proto-types/landing-page-business-suite/v1/bundles_pb';
import { CouponAdminService, CouponDuration, type Coupon, type CouponImportPreviewItem as GeneratedCouponImportPreviewItem } from '@vrooli/proto-types/landing-page-business-suite/v1/coupons_pb';
import { BillingInterval, IntroPricingType, PlanKind } from '@vrooli/proto-types/landing-page-business-suite/v1/shared/commerce_pb';
import { apiCall, CONNECT_API_BASE } from './common';
import { createScenarioConnectTransport } from '@vrooli/api-base';
import type { BundleCatalogEntry, CheckoutSession, PlanOption } from './types';
import { normalizeTimestamp } from '../lib/protobuf-utils';
import { parseOrNull } from './safeParse';
import {
  BundleCatalogResponseSchema,
  CheckoutSessionSchema,
  BillingPortalResponseSchema,
  VerifyStripePriceResponseSchema,
  StripeImportPreviewSchema,
  StripeImportResultSchema,
  ListCouponsResponseSchema,
  StripeCouponSchema,
  CouponUsageStatsListSchema,
  type StripeCoupon,
  type ListCouponsResponse,
  type CouponUsageStats,
} from './schemas/billing.schema';
import { PlanOptionSchema } from './schemas/landing.schema';

type BundleCatalogResponseParsed = z.infer<typeof BundleCatalogResponseSchema>;

const paymentsClient = createClient(
  LandingPagePaymentsService,
  createScenarioConnectTransport({ baseUrl: CONNECT_API_BASE }),
);
const stripeSettingsClient = createClient(
  StripeSettingsService,
  createScenarioConnectTransport({ baseUrl: CONNECT_API_BASE }),
);
const bundleAdminClient = createClient(
  BundleAdminService,
  createScenarioConnectTransport({ baseUrl: CONNECT_API_BASE }),
);
const couponAdminClient = createClient(
  CouponAdminService,
  createScenarioConnectTransport({ baseUrl: CONNECT_API_BASE }),
);

function normalizeCheckoutSession(session: CreateCheckoutSessionResponse['session']): CheckoutSession | null {
  if (!session) return null;
  return {
    session_id: session.sessionId,
    url: session.url,
    ...(session.customerEmail ? { customer_email: session.customerEmail } : {}),
    ...(session.stripePriceId ? { stripe_price_id: session.stripePriceId } : {}),
    ...(session.amountCents ? { amount_cents: Number(session.amountCents) } : {}),
    ...(session.currency ? { currency: session.currency } : {}),
    ...(session.successUrl ? { success_url: session.successUrl } : {}),
    ...(session.cancelUrl ? { cancel_url: session.cancelUrl } : {}),
  };
}

const normalizeBundleCatalog = (response: BundleCatalogResponseParsed): BundleCatalogResponse => ({
  bundles: response.bundles.map((entry) => ({
    bundle: entry.bundle,
    prices: entry.prices.map((plan): PlanOption => ({
      ...plan,
      intro_enabled: plan.intro_enabled,
      monthly_included_credits: typeof plan.monthly_included_credits === 'number' ? plan.monthly_included_credits : 0,
      one_time_bonus_credits: typeof plan.one_time_bonus_credits === 'number' ? plan.one_time_bonus_credits : 0,
      display_enabled: plan.display_enabled,
      display_weight: typeof plan.display_weight === 'number' ? plan.display_weight : 0,
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
  source: string;
}

export interface StripeSettingsUpdatePayload {
  publishable_key?: string;
  secret_key?: string;
  webhook_secret?: string;
  dashboard_url?: string;
	/** JSON object encoded as text to retain the proto's optional-field semantics. */
	anomaly_webhook_url?: string;
	anomaly_webhook_enabled?: boolean;
	anomaly_rate_limits?: string;
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

function objectMap(input?: Record<string, { toJson?: () => unknown }>): Record<string, unknown> | undefined {
  if (!input) return undefined;
  return Object.fromEntries(Object.entries(input).map(([key, value]) => [key, value.toJson?.() ?? null]));
}

function bundlePlanKind(kind?: PlanKind): PlanOption['kind'] {
  switch (kind) {
    case PlanKind.CREDITS_TOPUP: return 'credits_topup';
    case PlanKind.SUPPORTER_CONTRIBUTION: return 'supporter_contribution';
    default: return 'subscription';
  }
}

function bundleBillingInterval(interval?: BillingInterval): PlanOption['billing_interval'] {
  switch (interval) {
    case BillingInterval.YEAR: return 'year';
    case BillingInterval.ONE_TIME: return 'one_time';
    default: return 'month';
  }
}

function bundleIntroType(type?: IntroPricingType): PlanOption['intro_type'] {
  switch (type) {
    case IntroPricingType.PERCENTAGE: return 'percentage';
    case IntroPricingType.FLAT_AMOUNT: return 'flat_amount';
    default: return undefined;
  }
}

function couponDuration(duration?: CouponDuration): 'once' | 'repeating' | 'forever' | undefined {
  switch (duration) {
    case CouponDuration.ONCE: return 'once';
    case CouponDuration.REPEATING: return 'repeating';
    case CouponDuration.FOREVER: return 'forever';
    default: return undefined;
  }
}

function couponDurationRequest(duration: CreateCouponPayload['duration']): CouponDuration {
  switch (duration) {
    case 'once': return CouponDuration.ONCE;
    case 'repeating': return CouponDuration.REPEATING;
    case 'forever': return CouponDuration.FOREVER;
  }
}

function normalizeCoupon(coupon: Coupon | undefined): StripeCoupon | null {
  const duration = couponDuration(coupon?.duration);
  if (!coupon || !duration) return null;
  return {
    id: coupon.id, name: coupon.name, amount_off: coupon.amountOff == null ? undefined : Number(coupon.amountOff),
    percent_off: coupon.percentOff, currency: coupon.currency, duration,
    duration_in_months: coupon.durationInMonths, max_redemptions: coupon.maxRedemptions,
    redeem_by: coupon.redeemBy == null ? undefined : Number(coupon.redeemBy), times_redeemed: coupon.timesRedeemed,
    valid: coupon.valid, created: Number(coupon.created), is_intro_coupon: coupon.isIntroCoupon, intro_tier: coupon.introTier,
  };
}

function requireCoupon(coupon: Coupon | undefined, operation: string): StripeCoupon {
  const normalized = normalizeCoupon(coupon);
  const validated = normalized && parseOrNull(StripeCouponSchema, normalized, 'StripeCoupon');
  if (!validated) throw new Error(`Invalid coupon response from ${operation}`);
  return validated;
}

type GeneratedPlan = {
  planName?: string; planTier?: string; billingInterval?: BillingInterval; amountCents?: string | number | bigint;
  currency?: string; introEnabled?: boolean; introType?: IntroPricingType; introAmountCents?: string | number;
  introPeriods?: string | number | bigint; introPriceLookupKey?: string; stripePriceId?: string;
  monthlyIncludedCredits?: string | number | bigint; oneTimeBonusCredits?: string | number | bigint; planRank?: string | number | bigint;
  bonusType?: string; kind?: PlanKind; isVariableAmount?: boolean; displayEnabled?: boolean; bundleKey?: string;
  displayWeight?: string | number | bigint; metadata?: Record<string, { toJson?: () => unknown }>;
};

function normalizeBundlePlan(plan: GeneratedPlan): PlanOption {
  return {
    plan_name: plan.planName ?? '', plan_tier: plan.planTier ?? '', billing_interval: bundleBillingInterval(plan.billingInterval),
    amount_cents: Number(plan.amountCents ?? 0), currency: plan.currency ?? 'usd', intro_enabled: Boolean(plan.introEnabled),
    intro_type: bundleIntroType(plan.introType), intro_amount_cents: plan.introAmountCents == null ? undefined : Number(plan.introAmountCents),
    intro_periods: plan.introPeriods == null ? undefined : Number(plan.introPeriods), intro_price_lookup_key: plan.introPriceLookupKey,
    stripe_price_id: plan.stripePriceId ?? '', monthly_included_credits: Number(plan.monthlyIncludedCredits ?? 0),
    one_time_bonus_credits: Number(plan.oneTimeBonusCredits ?? 0), plan_rank: plan.planRank == null ? undefined : Number(plan.planRank),
    bonus_type: plan.bonusType, kind: bundlePlanKind(plan.kind), is_variable_amount: Boolean(plan.isVariableAmount),
    display_enabled: Boolean(plan.displayEnabled), bundle_key: plan.bundleKey, display_weight: Number(plan.displayWeight ?? 0), metadata: objectMap(plan.metadata),
  };
}

function normalizeBundleCatalogMessage(response: ListBundleCatalogResponse): BundleCatalogResponse {
  return {
    bundles: (response.bundles ?? []).map((entry) => ({
      bundle: {
        bundle_key: entry.bundle?.bundleKey ?? '', name: entry.bundle?.name ?? '', stripe_product_id: entry.bundle?.stripeProductId ?? '',
        credits_per_usd: Number(entry.bundle?.creditsPerUsd ?? 0), display_credits_multiplier: entry.bundle?.displayCreditsMultiplier ?? 0,
        display_credits_label: entry.bundle?.displayCreditsLabel ?? 'credits', environment: entry.bundle?.environment || undefined,
        metadata: objectMap(entry.bundle?.metadata),
      },
      prices: (entry.prices ?? []).map((price) => normalizeBundlePlan(price as GeneratedPlan)),
    })),
  };
}

function flattenStripeSettings(snapshot?: StripeConfigSnapshot, settings?: StripeSettings): StripeSettingsResponse {
  const normalizeSource = (source?: unknown): string => {
    switch (source) {
      case ConfigSource.DATABASE:
        return 'database';
      case ConfigSource.ENV:
      case ConfigSource.UNSPECIFIED:
        return 'env';
      default:
        return typeof source === 'string' || typeof source === 'number' ? String(source) : 'env';
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
  return stripeSettingsClient.getStripeSettings({}).then((message) => flattenStripeSettings(message.snapshot, message.settings));
}

export function updateStripeSettings(payload: StripeSettingsUpdatePayload) {
  return stripeSettingsClient.updateStripeSettings({
    publishableKey: payload.publishable_key,
    secretKey: payload.secret_key,
    webhookSecret: payload.webhook_secret,
    dashboardUrl: payload.dashboard_url,
    anomalyWebhookUrl: payload.anomaly_webhook_url,
    anomalyWebhookEnabled: payload.anomaly_webhook_enabled,
    anomalyRateLimits: payload.anomaly_rate_limits,
  }).then((message) => flattenStripeSettings(message.snapshot, message.settings));
}

export type RevealStripeSecretField = 'secret_key' | 'webhook_secret' | 'publishable_key' | 'anomaly_webhook_url';

export interface RevealStripeSecretResponse {
  field: RevealStripeSecretField;
  value: string;
}

export function revealStripeSecret(field: RevealStripeSecretField): Promise<RevealStripeSecretResponse> {
  return stripeSettingsClient.revealStripeSecret({ field }).then((response) => ({ field: response.field as RevealStripeSecretField, value: response.value }));
}

export function getBundleCatalog(): Promise<BundleCatalogResponse> {
  return bundleAdminClient.listBundleCatalog({}).then((response: ListBundleCatalogResponse) => {
    const normalized = normalizeBundleCatalogMessage(response);
    const validated = parseOrNull(BundleCatalogResponseSchema, normalized, 'BundleCatalogResponse');
    if (!validated) {
      throw new Error('Invalid bundle catalog response');
    }
    return normalizeBundleCatalog(validated);
  });
}

export function updateBundlePrice(bundleKey: string, priceId: string, payload: UpdateBundlePricePayload) {
  return bundleAdminClient.updateBundlePrice({
    bundleKey, priceId, stripePriceId: payload.stripe_price_id, planName: payload.plan_name,
    displayWeight: payload.display_weight, displayEnabled: payload.display_enabled, subtitle: payload.subtitle,
    badge: payload.badge, ctaLabel: payload.cta_label, highlight: payload.highlight,
    features: payload.features, featuresPresent: payload.features !== undefined,
  }).then((response: UpdateBundlePriceResponse) => {
    if (!response.price) {
      throw new Error('Invalid plan response from update');
    }
    const validated = parseOrNull(PlanOptionSchema, normalizeBundlePlan(response.price as GeneratedPlan), 'PlanOption');
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

  return paymentsClient.createCheckoutSession({
    priceId: body.price_id ?? '', customerEmail: body.customer_email ?? '', successUrl: body.success_url ?? '', cancelUrl: body.cancel_url ?? '', sessionKind: SessionKind.SUBSCRIPTION,
  }).then((resp: CreateCheckoutSessionResponse) => {
    const validated = parseOrNull(CheckoutSessionSchema, normalizeCheckoutSession(resp.session), 'CheckoutSession');
    if (!validated) {
      throw new Error('Invalid checkout session response');
    }
    return validated;
  });
}

export function createCreditsCheckoutSession(payload: { price_id: string; customer_email: string; success_url?: string; cancel_url?: string }) {
  return paymentsClient.createCheckoutSession({
    priceId: payload.price_id, customerEmail: payload.customer_email, successUrl: payload.success_url ?? '', cancelUrl: payload.cancel_url ?? '', sessionKind: SessionKind.CREDITS_TOPUP,
  }).then((resp: CreateCheckoutSessionResponse) => {
    const validated = parseOrNull(CheckoutSessionSchema, normalizeCheckoutSession(resp.session), 'CheckoutSession');
    if (!validated) {
      throw new Error('Invalid credits checkout session response');
    }
    return validated;
  });
}

export function createBillingPortalSession(returnUrl?: string, userEmail?: string) {
	void userEmail;
	return paymentsClient.getBillingPortal({ returnUrl: returnUrl ?? '' }).then((resp) => {
		const validated = parseOrNull(BillingPortalResponseSchema, { url: resp.url }, 'BillingPortalResponse');
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

// Coupon Management Types
export interface CreateCouponPayload {
  id?: string;
  name?: string;
  amount_off?: number;
  percent_off?: number;
  currency?: string;
  duration: 'once' | 'repeating' | 'forever';
  duration_in_months?: number;
  max_redemptions?: number;
  redeem_by?: number;
}

export type { StripeCoupon, ListCouponsResponse, CouponUsageStats };

/**
 * List all coupons from Stripe.
 */
export function listCoupons(): Promise<ListCouponsResponse> {
  return couponAdminClient.listCoupons({}).then((response) => {
    const coupons = (response.coupons ?? []).map((coupon) => normalizeCoupon(coupon));
    if (coupons.some((coupon) => coupon === null)) {
      throw new Error('Invalid coupons list response');
    }
    const normalized = { coupons, intro_coupon_map: response.introCouponMap };
    const validated = parseOrNull(ListCouponsResponseSchema, normalized, 'ListCouponsResponse');
    if (!validated) {
      throw new Error('Invalid coupons list response');
    }
    return validated;
  });
}

/**
 * Create a new coupon in Stripe.
 */
export function createCoupon(payload: CreateCouponPayload): Promise<StripeCoupon> {
  return couponAdminClient.createCoupon({ id: payload.id, name: payload.name, amountOff: payload.amount_off == null ? undefined : BigInt(payload.amount_off), percentOff: payload.percent_off, currency: payload.currency, duration: couponDurationRequest(payload.duration), durationInMonths: payload.duration_in_months, maxRedemptions: payload.max_redemptions, redeemBy: payload.redeem_by == null ? undefined : BigInt(payload.redeem_by) }).then((response) => requireCoupon(response.coupon, 'create'));
}

/**
 * Get a single coupon from Stripe.
 */
export function getCoupon(couponId: string): Promise<StripeCoupon> {
  return couponAdminClient.getCoupon({ couponId }).then((response) => requireCoupon(response.coupon, 'get'));
}

/**
 * Delete a coupon from Stripe.
 */
export function deleteCoupon(couponId: string): Promise<void> {
  return couponAdminClient.deleteCoupon({ couponId }).then((response) => {
    if (!response.deleted) throw new Error('Coupon deletion was not confirmed');
  });
}

/**
 * Update coupon payload. Note: Stripe only allows updating the name.
 */
export interface UpdateCouponPayload {
  name?: string;
}

/**
 * Update a coupon in Stripe (only name can be updated).
 */
export function updateCoupon(couponId: string, payload: UpdateCouponPayload): Promise<StripeCoupon> {
  return couponAdminClient.updateCoupon({ couponId, name: payload.name }).then((response) => requireCoupon(response.coupon, 'update'));
}

/**
 * Get coupon usage statistics from local database.
 */
export function getCouponUsage(): Promise<CouponUsageStats[]> {
  return couponAdminClient.listCouponUsage({}).then((response) => {
    const normalized = (response.usage ?? []).map((entry) => ({ coupon_id: entry.couponId, total_uses: Number(entry.totalUses), last_used_at: entry.lastUsedAt ?? null }));
    const validated = parseOrNull(CouponUsageStatsListSchema, normalized, 'CouponUsageStats[]');
    if (!validated) {
      throw new Error('Invalid coupon usage response');
    }
    return validated;
  });
}

// Coupon-Plan Mapping Types
export interface CouponMappingsResponse {
  mappings: Record<string, string>; // priceID -> couponID
}

const CouponMappingsResponseSchema = z.object({
  mappings: z.record(z.string(), z.string()),
});

/**
 * Get all coupon-to-plan mappings.
 */
export function getCouponMappings(): Promise<CouponMappingsResponse> {
  return couponAdminClient.getCouponMappings({}).then((response) => {
    const validated = parseOrNull(CouponMappingsResponseSchema, { mappings: response.mappings ?? {} }, 'CouponMappingsResponse');
    if (!validated) {
      throw new Error('Invalid coupon mappings response');
    }
    return validated;
  });
}

/**
 * Assign a coupon to a specific plan.
 */
export function setCouponForPlan(priceId: string, couponId: string): Promise<void> {
  return couponAdminClient.setCouponForPlan({ priceId, couponId }).then((response) => {
    if (!response.assigned) throw new Error('Coupon assignment was not confirmed');
  });
}

/**
 * Remove the coupon assignment from a specific plan.
 */
export function removeCouponFromPlan(priceId: string): Promise<void> {
  return couponAdminClient.removeCouponFromPlan({ priceId }).then((response) => {
    if (!response.removed) throw new Error('Coupon removal was not confirmed');
  });
}

// Stripe Coupon Import Types
export interface CouponImportPreviewItem {
  id: string;
  name?: string;
  amount_off?: number | null;
  percent_off?: number | null;
  currency?: string;
  duration: 'once' | 'repeating' | 'forever';
  duration_in_months?: number | null;
  times_redeemed: number;
  valid: boolean;
  exists_locally: boolean;
}

export interface CouponImportPreview {
  coupons: CouponImportPreviewItem[];
  total_coupons: number;
  existing_count: number;
  new_count: number;
}

const CouponImportPreviewItemSchema = z.object({
  id: z.string().min(1),
  name: z.string().optional(),
  amount_off: z.number().nullable().optional(),
  percent_off: z.number().nullable().optional(),
  currency: z.string().optional(),
  duration: z.enum(['once', 'repeating', 'forever']),
  duration_in_months: z.number().nullable().optional(),
  times_redeemed: z.number().int().nonnegative(),
  valid: z.boolean(),
  exists_locally: z.boolean(),
});

const CouponImportPreviewSchema = z.object({
  coupons: z.array(CouponImportPreviewItemSchema),
  total_coupons: z.number().int().nonnegative(),
  existing_count: z.number().int().nonnegative(),
  new_count: z.number().int().nonnegative(),
});

/**
 * Get a preview of coupons available to import from Stripe.
 */
export function getStripeCouponPreview(): Promise<CouponImportPreview> {
  return couponAdminClient.getCouponImportPreview({}).then((response) => {
    const coupons = (response.coupons ?? []).map((coupon: GeneratedCouponImportPreviewItem) => {
      const duration = couponDuration(coupon.duration);
      if (!duration) return null;
      return { id: coupon.id, name: coupon.name, amount_off: coupon.amountOff == null ? undefined : Number(coupon.amountOff), percent_off: coupon.percentOff, currency: coupon.currency, duration, duration_in_months: coupon.durationInMonths, times_redeemed: coupon.timesRedeemed, valid: coupon.valid, exists_locally: coupon.existsLocally };
    }).filter((coupon): coupon is NonNullable<typeof coupon> => coupon !== null);
    const normalized = { coupons, total_coupons: response.totalCoupons, existing_count: response.existingCount, new_count: response.newCount };
    const validated = parseOrNull(CouponImportPreviewSchema, normalized, 'CouponImportPreview');
    if (!validated) {
      throw new Error('Invalid coupon import preview response');
    }
    return validated;
  });
}
