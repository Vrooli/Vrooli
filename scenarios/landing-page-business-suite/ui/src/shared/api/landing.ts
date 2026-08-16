import { createClient } from '@connectrpc/connect';
import { toJson, type JsonValue } from '@bufbuild/protobuf';
import {
  PricingService,
  type GetPricingResponse,
} from '@vrooli/proto-types/landing-page-business-suite/v1/pricing_pb';
import {
  LandingConfigResponseSchema as LandingConfigMessageSchema,
  LandingConfigService,
  type LandingConfigResponse as LandingConfigMessage,
} from '@vrooli/proto-types/landing-page-business-suite/v1/config_pb';
import { BillingInterval, IntroPricingType, PlanKind } from '@vrooli/proto-types/landing-page-business-suite/v1/shared/commerce_pb';
import { CONNECT_API_BASE } from './common';
import { createScenarioConnectTransport } from '@vrooli/api-base';
import type { LandingConfigResponse, PlanOption, PricingOverview } from './types';
import { normalizeTimestampOrNow } from '../lib/protobuf-utils';
import { isRecord } from '../lib/utils';
import { LandingConfigResponseSchema, PlanOptionSchema, PricingOverviewSchema } from './schemas';
import { parseOrNull, parseOrThrow, safeParse } from './safeParse';

const pricingClient = createClient(
  PricingService,
  createScenarioConnectTransport({ baseUrl: CONNECT_API_BASE }),
);
const landingConfigClient = createClient(
  LandingConfigService,
  createScenarioConnectTransport({ baseUrl: CONNECT_API_BASE }),
);

type JsonRecord = Record<string, unknown>;

function asRecord(value: unknown): JsonRecord {
  return isRecord(value) ? value : {};
}

function field(record: JsonRecord, snakeName: string, camelName: string): unknown {
  return record[snakeName] ?? record[camelName];
}

function stringValue(value: unknown, fallback = ''): string {
  return typeof value === 'string' ? value : fallback;
}

function numberValue(value: unknown, fallback = 0): number {
  if (typeof value === 'number' && Number.isFinite(value)) return value;
  if (typeof value === 'string' && value.trim() !== '') {
    const parsed = Number(value);
    if (Number.isFinite(parsed)) return parsed;
  }
  return fallback;
}

function booleanValue(value: unknown, fallback = false): boolean {
  return typeof value === 'boolean' ? value : fallback;
}

function arrayValue(value: unknown): unknown[] {
  return Array.isArray(value) ? value : [];
}

function normalizeEnum(value: unknown, prefix: string, fallback = ''): string {
  const raw = stringValue(value);
  if (!raw) return fallback;
  const normalizedPrefix = `${prefix}_`;
  return raw.startsWith(normalizedPrefix) ? raw.slice(normalizedPrefix.length).toLowerCase() : raw.toLowerCase();
}

function normalizeStruct(value: unknown): JsonRecord {
  const record = asRecord(value);
  const fields = record.fields;
  return isRecord(fields) ? fields : record;
}

function normalizePlanOption(value: unknown): PlanOption {
  const plan = asRecord(value);
  const metadata = field(plan, 'metadata', 'metadata');
  const introType = normalizeEnum(field(plan, 'intro_type', 'introType'), 'INTRO_PRICING_TYPE');
  return {
    plan_name: stringValue(field(plan, 'plan_name', 'planName')),
    plan_tier: stringValue(field(plan, 'plan_tier', 'planTier')),
    billing_interval: normalizeEnum(field(plan, 'billing_interval', 'billingInterval'), 'BILLING_INTERVAL', 'month') as PlanOption['billing_interval'],
    amount_cents: numberValue(field(plan, 'amount_cents', 'amountCents')),
    currency: stringValue(field(plan, 'currency', 'currency'), 'usd'),
    intro_enabled: booleanValue(field(plan, 'intro_enabled', 'introEnabled')),
    intro_type: (introType || undefined) as PlanOption['intro_type'],
    intro_amount_cents: field(plan, 'intro_amount_cents', 'introAmountCents') == null ? undefined : numberValue(field(plan, 'intro_amount_cents', 'introAmountCents')),
    intro_periods: field(plan, 'intro_periods', 'introPeriods') == null ? undefined : numberValue(field(plan, 'intro_periods', 'introPeriods')),
    intro_price_lookup_key: stringValue(field(plan, 'intro_price_lookup_key', 'introPriceLookupKey')) || undefined,
    stripe_price_id: stringValue(field(plan, 'stripe_price_id', 'stripePriceId')),
    monthly_included_credits: numberValue(field(plan, 'monthly_included_credits', 'monthlyIncludedCredits')),
    one_time_bonus_credits: numberValue(field(plan, 'one_time_bonus_credits', 'oneTimeBonusCredits')),
    plan_rank: field(plan, 'plan_rank', 'planRank') == null ? undefined : numberValue(field(plan, 'plan_rank', 'planRank')),
    bonus_type: stringValue(field(plan, 'bonus_type', 'bonusType')) || undefined,
    kind: normalizeEnum(field(plan, 'kind', 'kind'), 'PLAN_KIND', 'subscription'),
    is_variable_amount: booleanValue(field(plan, 'is_variable_amount', 'isVariableAmount')),
    display_enabled: booleanValue(field(plan, 'display_enabled', 'displayEnabled')),
    bundle_key: stringValue(field(plan, 'bundle_key', 'bundleKey')) || undefined,
    display_weight: numberValue(field(plan, 'display_weight', 'displayWeight')),
    metadata: isRecord(metadata) ? metadata : undefined,
  };
}

function normalizePricing(value: unknown): PricingOverview | undefined {
  if (value == null) return undefined;
  const pricing = asRecord(value);
  const bundle = asRecord(field(pricing, 'bundle', 'bundle'));
  const normalizePlans = (name: string) => arrayValue(field(pricing, name, name === 'monthly' ? 'monthly' : 'yearly')).map(normalizePlanOption);
  return {
    bundle: {
      id: field(bundle, 'id', 'id') == null ? undefined : numberValue(field(bundle, 'id', 'id')),
      bundle_key: stringValue(field(bundle, 'bundle_key', 'bundleKey')),
      name: stringValue(field(bundle, 'name', 'name')),
      stripe_product_id: stringValue(field(bundle, 'stripe_product_id', 'stripeProductId')),
      credits_per_usd: numberValue(field(bundle, 'credits_per_usd', 'creditsPerUsd')),
      display_credits_multiplier: numberValue(field(bundle, 'display_credits_multiplier', 'displayCreditsMultiplier')),
      display_credits_label: stringValue(field(bundle, 'display_credits_label', 'displayCreditsLabel'), 'credits'),
      environment: stringValue(field(bundle, 'environment', 'environment')) || undefined,
      metadata: isRecord(field(bundle, 'metadata', 'metadata')) ? field(bundle, 'metadata', 'metadata') as JsonRecord : undefined,
    },
    monthly: normalizePlans('monthly'),
    yearly: normalizePlans('yearly'),
    updated_at: normalizeTimestampOrNow(field(pricing, 'updated_at', 'updatedAt')),
  };
}

function normalizeHeaderLink(value: unknown): JsonRecord {
  const link = asRecord(value);
  const visible = asRecord(field(link, 'visible_on', 'visibleOn'));
  return {
    id: stringValue(field(link, 'id', 'id')),
    type: normalizeEnum(field(link, 'type', 'type'), 'HEADER_NAV_LINK_TYPE', 'custom'),
    label: stringValue(field(link, 'label', 'label')),
    section_type: stringValue(field(link, 'section_type', 'sectionType')) || undefined,
    section_id: field(link, 'section_id', 'sectionId') == null ? undefined : numberValue(field(link, 'section_id', 'sectionId')),
    anchor: stringValue(field(link, 'anchor', 'anchor')) || undefined,
    href: stringValue(field(link, 'href', 'href')) || undefined,
    visible_on: {
      desktop: booleanValue(field(visible, 'desktop', 'desktop'), true),
      mobile: booleanValue(field(visible, 'mobile', 'mobile'), true),
    },
    children: arrayValue(field(link, 'children', 'children')).map(normalizeHeaderLink),
  };
}

function normalizeHeader(value: unknown): JsonRecord {
  const header = asRecord(value);
  const branding = asRecord(field(header, 'branding', 'branding'));
  const nav = asRecord(field(header, 'nav', 'nav'));
  const ctas = asRecord(field(header, 'ctas', 'ctas'));
  const behavior = asRecord(field(header, 'behavior', 'behavior'));
  const normalizeCTA = (value: unknown): JsonRecord => {
    const cta = asRecord(value);
    return {
      mode: normalizeEnum(field(cta, 'mode', 'mode'), 'HEADER_CTA_MODE', 'inherit_hero'),
      label: stringValue(field(cta, 'label', 'label')) || undefined,
      href: stringValue(field(cta, 'href', 'href')) || undefined,
      variant: stringValue(field(cta, 'variant', 'variant')) || undefined,
    };
  };
  return {
    branding: {
      mode: normalizeEnum(field(branding, 'mode', 'mode'), 'HEADER_BRANDING_MODE', 'none'),
      label: stringValue(field(branding, 'label', 'label')) || undefined,
      subtitle: stringValue(field(branding, 'subtitle', 'subtitle')) || undefined,
      mobile_preference: stringValue(field(branding, 'mobile_preference', 'mobilePreference')) || undefined,
    },
    nav: { links: arrayValue(field(nav, 'links', 'links')).map(normalizeHeaderLink) },
    ctas: {
      primary: normalizeCTA(field(ctas, 'primary', 'primary')),
      secondary: normalizeCTA(field(ctas, 'secondary', 'secondary')),
    },
    behavior: {
      sticky: booleanValue(field(behavior, 'sticky', 'sticky')),
      hide_on_scroll: booleanValue(field(behavior, 'hide_on_scroll', 'hideOnScroll')),
    },
  };
}

function normalizeDownloads(value: unknown): JsonRecord[] {
  return arrayValue(value).map((entry) => {
    const app = asRecord(entry);
    const normalizeAsset = (assetValue: unknown): JsonRecord => {
      const asset = asRecord(assetValue);
      return {
        id: field(asset, 'id', 'id') == null ? undefined : numberValue(field(asset, 'id', 'id')),
        bundle_key: stringValue(field(asset, 'bundle_key', 'bundleKey')),
        app_key: stringValue(field(asset, 'app_key', 'appKey')),
        platform: stringValue(field(asset, 'platform', 'platform')),
        artifact_url: stringValue(field(asset, 'artifact_url', 'artifactUrl')),
        artifact_source: stringValue(field(asset, 'artifact_source', 'artifactSource')) || undefined,
        artifact_id: field(asset, 'artifact_id', 'artifactId') == null ? undefined : numberValue(field(asset, 'artifact_id', 'artifactId')),
        release_version: stringValue(field(asset, 'release_version', 'releaseVersion')),
        release_notes: stringValue(field(asset, 'release_notes', 'releaseNotes')) || undefined,
        checksum: stringValue(field(asset, 'checksum', 'checksum')) || undefined,
        requires_entitlement: booleanValue(field(asset, 'requires_entitlement', 'requiresEntitlement')),
        metadata: normalizeStruct(field(asset, 'metadata', 'metadata')),
        artifact_filename: stringValue(field(asset, 'artifact_filename', 'artifactFilename')) || undefined,
        artifact_size_bytes: field(asset, 'artifact_size_bytes', 'artifactSizeBytes') == null ? undefined : numberValue(field(asset, 'artifact_size_bytes', 'artifactSizeBytes')),
        artifact_count: field(asset, 'artifact_count', 'artifactCount') == null ? undefined : numberValue(field(asset, 'artifact_count', 'artifactCount')),
      };
    };
    return {
      bundle_key: stringValue(field(app, 'bundle_key', 'bundleKey')),
      app_key: stringValue(field(app, 'app_key', 'appKey')),
      name: stringValue(field(app, 'name', 'name')),
      tagline: stringValue(field(app, 'tagline', 'tagline')) || undefined,
      description: stringValue(field(app, 'description', 'description')) || undefined,
      icon_url: stringValue(field(app, 'icon_url', 'iconUrl')) || undefined,
      screenshot_url: stringValue(field(app, 'screenshot_url', 'screenshotUrl')) || undefined,
      install_overview: stringValue(field(app, 'install_overview', 'installOverview')) || undefined,
      install_steps: arrayValue(field(app, 'install_steps', 'installSteps')).filter((step): step is string => typeof step === 'string'),
      storefronts: arrayValue(field(app, 'storefronts', 'storefronts')).map((storefront) => {
        const value = asRecord(storefront);
        return { store: stringValue(field(value, 'store', 'store')), label: stringValue(field(value, 'label', 'label')), url: stringValue(field(value, 'url', 'url')), badge: stringValue(field(value, 'badge', 'badge')) || undefined };
      }),
      metadata: normalizeStruct(field(app, 'metadata', 'metadata')),
      display_order: field(app, 'display_order', 'displayOrder') == null ? undefined : numberValue(field(app, 'display_order', 'displayOrder')),
      platforms: arrayValue(field(app, 'platforms', 'platforms')).map(normalizeAsset),
    };
  });
}

function normalizeIntroOffers(value: unknown): JsonRecord[] {
  return arrayValue(value).map((entry) => {
    const offer = asRecord(entry);
    const optionalNumber = (snake: string, camel: string, zeroMeansUndefined = false) => {
      if (field(offer, snake, camel) == null) return undefined;
      const parsed = numberValue(field(offer, snake, camel));
      return zeroMeansUndefined && parsed <= 0 ? undefined : parsed;
    };
    return {
      id: stringValue(field(offer, 'id', 'id')),
      name: stringValue(field(offer, 'name', 'name')) || undefined,
      amount_off: optionalNumber('amount_off', 'amountOff'),
      percent_off: optionalNumber('percent_off', 'percentOff'),
      currency: stringValue(field(offer, 'currency', 'currency')) || undefined,
      duration: stringValue(field(offer, 'duration', 'duration'), 'once'),
      duration_in_months: optionalNumber('duration_in_months', 'durationInMonths'),
      max_redemptions: optionalNumber('max_redemptions', 'maxRedemptions'),
      redeem_by: optionalNumber('redeem_by', 'redeemBy', true),
      times_redeemed: numberValue(field(offer, 'times_redeemed', 'timesRedeemed')),
      valid: booleanValue(field(offer, 'valid', 'valid')),
      created: numberValue(field(offer, 'created', 'created')),
      is_intro_coupon: booleanValue(field(offer, 'is_intro_coupon', 'isIntroCoupon')),
      intro_tier: stringValue(field(offer, 'intro_tier', 'introTier')) || undefined,
    };
  });
}

function normalizeLandingConfig(value: JsonValue): LandingConfigResponse {
  const raw = asRecord(value);
  const variant = asRecord(field(raw, 'variant', 'variant'));
  const sections = arrayValue(field(raw, 'sections', 'sections')).map((entry) => {
    const section = asRecord(entry);
    return {
      id: field(section, 'id', 'id') == null ? undefined : numberValue(field(section, 'id', 'id')),
      key: stringValue(field(section, 'key', 'key')) || stringValue(field(section, 'section_key', 'sectionKey')) || undefined,
      section_type: stringValue(field(section, 'section_type', 'sectionType')),
      content: normalizeStruct(field(section, 'content', 'content')),
      order: numberValue(field(section, 'order', 'order')),
      enabled: booleanValue(field(section, 'enabled', 'enabled'), true),
    };
  });
  return parseOrThrow(
    LandingConfigResponseSchema,
    {
      variant: {
        id: field(variant, 'id', 'id') == null ? undefined : numberValue(field(variant, 'id', 'id')),
        slug: stringValue(field(variant, 'slug', 'slug')),
        name: stringValue(field(variant, 'name', 'name')),
        description: stringValue(field(variant, 'description', 'description')) || undefined,
        axes: isRecord(field(variant, 'axes', 'axes')) ? field(variant, 'axes', 'axes') : {},
      },
      sections,
      pricing: normalizePricing(field(raw, 'pricing', 'pricing')),
      downloads: normalizeDownloads(field(raw, 'downloads', 'downloads')),
      header: normalizeHeader(field(raw, 'header', 'header')),
      branding: isRecord(field(raw, 'branding', 'branding')) ? {
        site_name: stringValue(field(asRecord(field(raw, 'branding', 'branding')), 'site_name', 'siteName')),
        tagline: stringValue(field(asRecord(field(raw, 'branding', 'branding')), 'tagline', 'tagline')) || undefined,
        logo_url: stringValue(field(asRecord(field(raw, 'branding', 'branding')), 'logo_url', 'logoUrl')) || undefined,
        logo_icon_url: stringValue(field(asRecord(field(raw, 'branding', 'branding')), 'logo_icon_url', 'logoIconUrl')) || undefined,
        favicon_url: stringValue(field(asRecord(field(raw, 'branding', 'branding')), 'favicon_url', 'faviconUrl')) || undefined,
        theme_primary_color: stringValue(field(asRecord(field(raw, 'branding', 'branding')), 'theme_primary_color', 'themePrimaryColor')) || undefined,
        theme_background_color: stringValue(field(asRecord(field(raw, 'branding', 'branding')), 'theme_background_color', 'themeBackgroundColor')) || undefined,
        support_chat_url: stringValue(field(asRecord(field(raw, 'branding', 'branding')), 'support_chat_url', 'supportChatUrl')) || undefined,
        support_email: stringValue(field(asRecord(field(raw, 'branding', 'branding')), 'support_email', 'supportEmail')) || undefined,
        coming_soon_enabled: field(asRecord(field(raw, 'branding', 'branding')), 'coming_soon_enabled', 'comingSoonEnabled') as boolean | undefined,
        coming_soon_message: stringValue(field(asRecord(field(raw, 'branding', 'branding')), 'coming_soon_message', 'comingSoonMessage')) || undefined,
      } : undefined,
      coupon_mappings: isRecord(field(raw, 'coupon_mappings', 'couponMappings')) ? field(raw, 'coupon_mappings', 'couponMappings') : undefined,
      intro_offers: normalizeIntroOffers(field(raw, 'intro_offers', 'introOffers')),
      fallback: booleanValue(field(raw, 'fallback', 'fallback')),
    },
    'LandingConfigResponse',
  );
}

export function getLandingConfig(variantSlug?: string) {
  return landingConfigClient.getLandingConfig({ variantSlug }).then((response: LandingConfigMessage) => {
    const raw = toJson(LandingConfigMessageSchema, response, { useProtoFieldName: true });
    return normalizeLandingConfig(raw);
  });
}

export function getPlans() {
  return pricingClient.getPricing({}).then((message: GetPricingResponse) => {
    const toObjectMap = (input?: Record<string, { toJson?: () => unknown }>) => {
      if (!input) return undefined;
      return Object.fromEntries(
        Object.entries(input).map(([key, value]) => [key, value.toJson?.() ?? null])
      );
    };
    const planKind = (kind?: PlanKind): PlanOption['kind'] => {
      switch (kind) {
        case PlanKind.CREDITS_TOPUP:
          return 'credits_topup';
        case PlanKind.SUPPORTER_CONTRIBUTION:
          return 'supporter_contribution';
        default:
          return 'subscription';
      }
    };

    const billingInterval = (interval?: BillingInterval): PlanOption['billing_interval'] => {
      switch (interval) {
        case BillingInterval.YEAR:
          return 'year';
        case BillingInterval.ONE_TIME:
          return 'one_time';
        default:
          return 'month';
      }
    };

    const introType = (type?: IntroPricingType): PlanOption['intro_type'] => {
      switch (type) {
        case IntroPricingType.PERCENTAGE:
          return 'percentage';
        case IntroPricingType.FLAT_AMOUNT:
          return 'flat_amount';
        default:
          return undefined;
      }
    };

    // Define the shape of raw protobuf plan data
    interface RawPlan {
      planName?: string;
      planTier?: string;
      billingInterval?: BillingInterval;
      amountCents?: string | number;
      currency?: string;
      introEnabled?: boolean;
      introType?: IntroPricingType;
      introAmountCents?: string | number;
      introPeriods?: string | number;
      introPriceLookupKey?: string;
      stripePriceId?: string;
      monthlyIncludedCredits?: string | number;
      oneTimeBonusCredits?: string | number;
      planRank?: string | number;
      bonusType?: string;
      kind?: PlanKind;
      isVariableAmount?: boolean;
      displayEnabled?: boolean;
      bundleKey?: string;
      displayWeight?: string | number;
      metadata?: Record<string, { toJson?: () => unknown }>;
    }

    const normalizePlan = (plan: RawPlan): PlanOption | null => {
      const normalized: PlanOption = {
        plan_name: plan.planName ?? '',
        plan_tier: plan.planTier ?? '',
        billing_interval: billingInterval(plan.billingInterval),
        amount_cents: Number(plan.amountCents ?? 0),
        currency: plan.currency ?? 'usd',
        intro_enabled: Boolean(plan.introEnabled),
        intro_type: introType(plan.introType),
        intro_amount_cents: plan.introAmountCents != null ? Number(plan.introAmountCents) : undefined,
        intro_periods: plan.introPeriods != null ? Number(plan.introPeriods) : undefined,
        intro_price_lookup_key: plan.introPriceLookupKey,
        stripe_price_id: plan.stripePriceId ?? '',
        monthly_included_credits: Number(plan.monthlyIncludedCredits ?? 0),
        one_time_bonus_credits: Number(plan.oneTimeBonusCredits ?? 0),
        plan_rank: plan.planRank != null ? Number(plan.planRank) : undefined,
        bonus_type: plan.bonusType,
        kind: planKind(plan.kind),
        is_variable_amount: Boolean(plan.isVariableAmount),
        display_enabled: Boolean(plan.displayEnabled),
        bundle_key: plan.bundleKey,
        display_weight: Number(plan.displayWeight ?? 0),
        metadata: toObjectMap(plan.metadata),
      };

      // Validate the normalized plan against the schema
      const validated = parseOrNull(PlanOptionSchema, normalized, 'PlanOption');
      if (!validated) return null;
      return normalized;
    };

    const pricing = message.pricing;
    const updatedAt = normalizeTimestampOrNow(pricing?.updatedAt);

    // Normalize and filter out invalid plans
    const monthlyPlans = (pricing?.monthly ?? [])
      .map((p) => normalizePlan(p))
      .filter((p): p is PlanOption => p !== null);
    const yearlyPlans = (pricing?.yearly ?? [])
      .map((p) => normalizePlan(p))
      .filter((p): p is PlanOption => p !== null);

    const overview: PricingOverview = {
      bundle: {
        bundle_key: pricing?.bundle?.bundleKey ?? '',
        name: pricing?.bundle?.name ?? '',
        stripe_product_id: pricing?.bundle?.stripeProductId ?? '',
        credits_per_usd: Number(pricing?.bundle?.creditsPerUsd ?? 0),
        display_credits_multiplier: Number(pricing?.bundle?.displayCreditsMultiplier ?? 0),
        display_credits_label: pricing?.bundle?.displayCreditsLabel ?? 'credits',
        environment: pricing?.bundle?.environment ?? 'production',
        metadata: toObjectMap(pricing?.bundle?.metadata),
      },
      monthly: monthlyPlans,
      yearly: yearlyPlans,
      updated_at: updatedAt,
    };

    // Validate the final overview
    const validationResult = safeParse(PricingOverviewSchema, overview, 'PricingOverview');
    if (!validationResult.success) {
      console.warn('[getPlans] Pricing overview validation failed, returning as-is:', validationResult.error);
    }

    return overview;
  });
}
