import { createClient } from '@connectrpc/connect';
import { create, fromJsonString, toJson } from '@bufbuild/protobuf';
import {
  PricingService,
  GetPricingResponseSchema,
  type GetPricingResponse,
} from '@vrooli/proto-types/landing-page-business-suite/v1/pricing_pb';
import {
  LandingConfigResponseSchema as LandingConfigMessageSchema,
  GetLandingConfigRequestSchema,
  LandingConfigService,
  RecordPresentationExposureRequestSchema,
  type RecordPresentationExposureRequest,
  type LandingConfigResponse as LandingConfigMessage,
} from '@vrooli/proto-types/landing-page-business-suite/v1/config_pb';
import { CONNECT_API_BASE } from './common';
import { createScenarioConnectTransport } from '@vrooli/api-base';
import type { LandingConfigResponse, PlanOption, PricingOverview } from './types';
import { normalizeTimestampOrNow } from '../lib/protobuf-utils';
import { isRecord } from '../lib/utils';
import { LandingConfigResponseSchema, PricingOverviewSchema } from './schemas';
import { parseOrThrow } from './safeParse';

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

// common.v1.JsonValue is not google.protobuf.Value: protobuf JSON retains
// its typed oneof envelope. Decode that envelope after the generated codec.
function metadataValue(value: unknown): unknown {
  if (!isRecord(value)) return value;
  for (const key of ['string_value', 'bool_value', 'double_value', 'bytes_value']) {
    if (key in value) return value[key];
  }
  if ('null_value' in value) return null;
  if ('int_value' in value) {
    const number = Number(value.int_value);
    return Number.isSafeInteger(number) ? number : value.int_value;
  }
  if (isRecord(value.object_value)) return normalizeMetadata(value.object_value.fields);
  if (isRecord(value.list_value)) return arrayValue(value.list_value.values).map(metadataValue);
  return value;
}

function normalizeMetadata(value: unknown): JsonRecord | undefined {
  if (!isRecord(value)) return undefined;
  return Object.fromEntries(Object.entries(value).map(([key, item]) => [key, metadataValue(item)]));
}

function normalizePlanOption(value: unknown): PlanOption {
  const plan = asRecord(value);
  // Monetary input must never become a plausible zero-price offer on a
  // malformed response. Generated protobuf normally prevents this; retain a
  // fail-closed check at the JSON normalization boundary as well.
  for (const [snake, camel] of [['amount_cents', 'amountCents'], ['intro_amount_cents', 'introAmountCents']] as const) {
    const amount = field(plan, snake, camel);
    if (amount != null && (!Number.isSafeInteger(Number(amount)) || Number(amount) < 0 || amount === '')) {
      throw new Error(`Invalid pricing amount: ${snake}`);
    }
  }
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
    metadata: normalizeMetadata(metadata),
  };
}

function normalizePricing(value: unknown): PricingOverview | undefined {
  if (value == null) return undefined;
  const pricing = asRecord(value);
  const bundle = asRecord(field(pricing, 'bundle', 'bundle'));
  const normalizePlans = (name: string, camelName = name) => arrayValue(field(pricing, name, camelName)).map(normalizePlanOption);
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
      metadata: normalizeMetadata(field(bundle, 'metadata', 'metadata')),
    },
    monthly: normalizePlans('monthly'),
    yearly: normalizePlans('yearly'),
    credit_topups: normalizePlans('credit_topups', 'creditTopups'),
    updated_at: normalizeTimestampOrNow(field(pricing, 'updated_at', 'updatedAt')),
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

export function getLandingConfig(variantSlug?: string, visitorId?: string, options: { route?: string; locale?: string; signal?: AbortSignal } = {}) {
  return landingConfigClient.getLandingConfig(create(GetLandingConfigRequestSchema, { variantSlug: variantSlug ?? '', visitorId: visitorId ?? '', route: options.route ?? '/', locale: options.locale ?? '' }), { signal: options.signal }).then((response: LandingConfigMessage) => {
    return decodeLandingConfig(response);
  });
}

export function decodeLandingConfig(response: LandingConfigMessage): LandingConfigResponse {
  const raw = asRecord(toJson(LandingConfigMessageSchema, response, { useProtoFieldName: true }));
  return parseOrThrow(LandingConfigResponseSchema, {
    presentation: response.presentation,
    pricing: normalizePricing(raw.pricing),
    downloads: normalizeDownloads(raw.downloads),
    fallback: response.fallback,
  }, 'LandingConfigResponse');
}

/** Explicit rendered-exposure write; configuration reads never imply exposure. */
export function recordPresentationExposure(proof: Omit<RecordPresentationExposureRequest, '$typeName' | '$unknown'>) {
  return landingConfigClient.recordPresentationExposure(create(RecordPresentationExposureRequestSchema, proof), { timeoutMs: 10000 });
}

/** Server bootstrap and network responses share the installed generated contract. */
export function parseLandingConfigJson(text: string): LandingConfigResponse {
  return decodeLandingConfig(fromJsonString(LandingConfigMessageSchema, text, { ignoreUnknownFields: false }));
}

export function getPlans(): Promise<PricingOverview> {
  return pricingClient.getPricing({}).then((message: GetPricingResponse) => {
    const raw = asRecord(toJson(GetPricingResponseSchema, message, { useProtoFieldName: true }));
    return parseOrThrow(PricingOverviewSchema, normalizePricing(raw.pricing), 'PricingOverview');
  });
}
